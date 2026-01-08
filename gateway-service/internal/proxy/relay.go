package proxy

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/pocketzworld/lurus-common/models"
	"github.com/pocketzworld/lurus-switch/gateway-service/internal/client"
	"github.com/pocketzworld/lurus-switch/gateway-service/internal/conf"
	"github.com/pocketzworld/lurus-switch/gateway-service/internal/middleware"
	"github.com/pocketzworld/lurus-switch/gateway-service/pkg/nats"
	"github.com/tidwall/gjson"
	"go.uber.org/zap"
)

// Platform represents the LLM platform type
type Platform string

const (
	PlatformClaude Platform = "claude"
	PlatformCodex  Platform = "codex"
	PlatformGemini Platform = "gemini"
)

// RelayService handles request forwarding to LLM providers
type RelayService struct {
	config         *conf.Bootstrap
	providerClient *client.ProviderClient
	billingClient  *client.BillingClient
	natsPublisher  *nats.Publisher
	httpClient     *http.Client
	logger         *zap.Logger
	rrCounter      uint64 // round-robin counter
}

// NewRelayService creates a new relay service
func NewRelayService(
	config *conf.Bootstrap,
	providerClient *client.ProviderClient,
	billingClient *client.BillingClient,
	natsPublisher *nats.Publisher,
	logger *zap.Logger,
) *RelayService {
	return &RelayService{
		config:         config,
		providerClient: providerClient,
		billingClient:  billingClient,
		natsPublisher:  natsPublisher,
		httpClient: &http.Client{
			Timeout: config.Proxy.RequestTimeout,
			Transport: &http.Transport{
				MaxIdleConns:        100,
				MaxIdleConnsPerHost: 20,
				IdleConnTimeout:     90 * time.Second,
			},
		},
		logger: logger,
	}
}

// RequestLog represents a request log entry
type RequestLog struct {
	ID              string    `json:"id"`
	TraceID         string    `json:"trace_id"`
	RequestID       string    `json:"request_id"`
	UserID          string    `json:"user_id"`
	Platform        string    `json:"platform"`
	Model           string    `json:"model"`
	Provider        string    `json:"provider"`
	ProviderModel   string    `json:"provider_model"`
	IsStream        bool      `json:"is_stream"`
	HTTPCode        int       `json:"http_code"`
	DurationSec     float64   `json:"duration_sec"`
	FinishReason    string    `json:"finish_reason"`
	InputTokens     int       `json:"input_tokens"`
	OutputTokens    int       `json:"output_tokens"`
	CacheReadTokens int       `json:"cache_read_tokens"`
	TotalCost       float64   `json:"total_cost"`
	ErrorType       string    `json:"error_type"`
	ErrorMessage    string    `json:"error_message"`
	CreatedAt       time.Time `json:"created_at"`
}

// ForwardRequest forwards a request to the appropriate provider
func (s *RelayService) ForwardRequest(ctx context.Context, c *app.RequestContext, platform Platform, endpoint string) error {
	startTime := time.Now()
	traceID := generateTraceID()
	requestID := string(c.GetHeader("X-Request-ID"))
	if requestID == "" {
		requestID = traceID
	}

	// Track active requests
	middleware.IncrementActiveRequests(string(platform))
	defer middleware.DecrementActiveRequests(string(platform))

	// Read request body
	bodyBytes := c.Request.Body()

	// Parse stream mode and model from request
	isStream := gjson.GetBytes(bodyBytes, "stream").Bool()
	requestedModel := gjson.GetBytes(bodyBytes, "model").String()

	s.logger.Debug("Processing request",
		zap.String("trace_id", traceID),
		zap.String("platform", string(platform)),
		zap.String("model", requestedModel),
		zap.Bool("stream", isStream),
	)

	// Check billing (if enabled)
	userID := s.extractUserID(c)
	if s.config.Features.BillingCheck && s.billingClient != nil {
		if err := s.billingClient.CheckBalance(ctx, userID); err != nil {
			s.logger.Warn("Balance check failed", zap.Error(err), zap.String("user_id", userID))
			c.JSON(http.StatusPaymentRequired, map[string]interface{}{
				"error": map[string]interface{}{
					"type":    "insufficient_balance",
					"message": "Insufficient balance to process request",
				},
			})
			return err
		}
	}

	// Try NEW-API first if enabled
	if s.config.Features.NewAPIEnabled && s.config.Features.NewAPIURL != "" {
		success, err := s.forwardToNewAPI(ctx, c, platform, endpoint, bodyBytes, isStream, requestedModel, traceID)
		if success {
			return nil
		}
		if err != nil {
			s.logger.Warn("NEW-API failed, falling back to local providers", zap.Error(err))
		}
	}

	// Get providers from Provider Service
	providers, err := s.providerClient.GetProviders(ctx, string(platform))
	if err != nil {
		s.logger.Error("Failed to get providers", zap.Error(err))
		c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": map[string]interface{}{
				"type":    "provider_error",
				"message": "Failed to load provider configuration",
			},
		})
		return err
	}

	// Filter providers that support the requested model
	activeProviders := s.filterProviders(providers, requestedModel)
	if len(activeProviders) == 0 {
		c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error": map[string]interface{}{
				"type":    "model_not_found",
				"message": fmt.Sprintf("No provider supports model: %s", requestedModel),
			},
		})
		return fmt.Errorf("no provider supports model: %s", requestedModel)
	}

	// Try providers with fallback
	var lastErr error
	startIdx := 0
	if s.config.Proxy.RoundRobin {
		startIdx = int(atomic.AddUint64(&s.rrCounter, 1)-1) % len(activeProviders)
	}

	retryCount := s.config.Proxy.RetryCount
	if retryCount > len(activeProviders) {
		retryCount = len(activeProviders)
	}

	for i := 0; i < retryCount; i++ {
		idx := (startIdx + i) % len(activeProviders)
		provider := activeProviders[idx]

		// Get effective model (with mapping)
		effectiveModel := s.getEffectiveModel(provider, requestedModel)

		// Create request log
		reqLog := &RequestLog{
			ID:            generateID(),
			TraceID:       traceID,
			RequestID:     requestID,
			UserID:        userID,
			Platform:      string(platform),
			Model:         requestedModel,
			Provider:      provider.Name,
			ProviderModel: effectiveModel,
			IsStream:      isStream,
			CreatedAt:     time.Now(),
		}

		// Forward request
		err := s.forwardToProvider(ctx, c, provider, platform, endpoint, bodyBytes, isStream, effectiveModel, reqLog)
		if err == nil {
			reqLog.DurationSec = time.Since(startTime).Seconds()
			reqLog.HTTPCode = c.Response.StatusCode()

			// Record metrics
			middleware.RecordLLMRequest(string(platform), provider.Name, requestedModel, strconv.Itoa(reqLog.HTTPCode), reqLog.DurationSec, isStream)
			middleware.RecordTokens(string(platform), provider.Name, reqLog.InputTokens, reqLog.OutputTokens)
			middleware.RecordCost(string(platform), provider.Name, requestedModel, reqLog.TotalCost)

			// Publish log event
			if s.config.Features.AsyncLogging && s.natsPublisher != nil {
				go s.natsPublisher.PublishLogEvent(reqLog)
			}

			s.logger.Info("Request succeeded",
				zap.String("trace_id", traceID),
				zap.String("provider", provider.Name),
				zap.Float64("duration", reqLog.DurationSec),
			)
			return nil
		}

		// Record error metrics
		middleware.RecordProviderError(string(platform), provider.Name, reqLog.ErrorType)

		// Record failover if there's a next provider
		if i+1 < retryCount {
			nextIdx := (startIdx + i + 1) % len(activeProviders)
			middleware.RecordFailover(string(platform), provider.Name, activeProviders[nextIdx].Name)
		}

		lastErr = err
		s.logger.Warn("Provider failed, trying next",
			zap.String("provider", provider.Name),
			zap.Error(err),
		)
	}

	// All providers failed
	c.JSON(http.StatusBadGateway, map[string]interface{}{
		"error": map[string]interface{}{
			"type":    "all_providers_failed",
			"message": fmt.Sprintf("All %d providers failed: %v", retryCount, lastErr),
		},
	})
	return lastErr
}

// forwardToProvider forwards the request to a specific provider
func (s *RelayService) forwardToProvider(
	ctx context.Context,
	c *app.RequestContext,
	provider *models.Provider,
	platform Platform,
	endpoint string,
	body []byte,
	isStream bool,
	effectiveModel string,
	reqLog *RequestLog,
) error {
	// Build target URL
	targetURL := strings.TrimSuffix(provider.APIURL, "/") + endpoint

	// Modify body with effective model if different
	if effectiveModel != gjson.GetBytes(body, "model").String() {
		body = s.replaceModel(body, effectiveModel)
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", targetURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	s.setRequestHeaders(req, c, provider, platform)

	// Execute request
	resp, err := s.httpClient.Do(req)
	if err != nil {
		reqLog.ErrorType = "network_error"
		reqLog.ErrorMessage = err.Error()
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	reqLog.HTTPCode = resp.StatusCode

	// Handle error responses
	if resp.StatusCode >= 400 {
		reqLog.ErrorType = classifyHTTPError(resp.StatusCode)
		bodyBytes, _ := io.ReadAll(resp.Body)
		reqLog.ErrorMessage = string(bodyBytes)
		return fmt.Errorf("upstream error: %d - %s", resp.StatusCode, string(bodyBytes))
	}

	// Forward response
	if isStream {
		return s.streamResponse(c, resp, reqLog)
	}
	return s.normalResponse(c, resp, reqLog)
}

// streamResponse handles SSE streaming response
func (s *RelayService) streamResponse(c *app.RequestContext, resp *http.Response, reqLog *RequestLog) error {
	// Set SSE headers
	c.Response.Header.Set("Content-Type", "text/event-stream")
	c.Response.Header.Set("Cache-Control", "no-cache")
	c.Response.Header.Set("Connection", "keep-alive")
	c.Response.Header.Set("X-Accel-Buffering", "no")
	c.Response.Header.Set("Transfer-Encoding", "chunked")

	// Get the response writer for streaming
	c.Response.SetStatusCode(resp.StatusCode)

	// Read and forward the stream
	scanner := bufio.NewScanner(resp.Body)
	// Increase buffer size for large chunks
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	for scanner.Scan() {
		line := scanner.Bytes()

		// Parse token usage from SSE events
		s.parseSSETokenUsage(line, reqLog)

		// Write line to client
		c.Response.AppendBody(line)
		c.Response.AppendBodyString("\n")
	}

	if err := scanner.Err(); err != nil {
		s.logger.Error("Stream scan error", zap.Error(err))
		return err
	}

	return nil
}

// normalResponse handles non-streaming response
func (s *RelayService) normalResponse(c *app.RequestContext, resp *http.Response, reqLog *RequestLog) error {
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	// Parse token usage from response
	s.parseNormalTokenUsage(bodyBytes, reqLog)

	// Copy response headers
	for key, values := range resp.Header {
		for _, value := range values {
			c.Response.Header.Set(key, value)
		}
	}

	c.Response.SetStatusCode(resp.StatusCode)
	c.Response.SetBody(bodyBytes)
	return nil
}

// parseSSETokenUsage extracts token usage from SSE events
func (s *RelayService) parseSSETokenUsage(line []byte, reqLog *RequestLog) {
	if !bytes.HasPrefix(line, []byte("data: ")) {
		return
	}

	data := bytes.TrimPrefix(line, []byte("data: "))
	if bytes.Equal(data, []byte("[DONE]")) {
		return
	}

	// Try to parse usage from the event
	usage := gjson.GetBytes(data, "usage")
	if usage.Exists() {
		reqLog.InputTokens = int(usage.Get("input_tokens").Int())
		reqLog.OutputTokens = int(usage.Get("output_tokens").Int())
		reqLog.CacheReadTokens = int(usage.Get("cache_read_input_tokens").Int())
	}

	// Parse finish reason
	finishReason := gjson.GetBytes(data, "delta.stop_reason").String()
	if finishReason == "" {
		finishReason = gjson.GetBytes(data, "choices.0.finish_reason").String()
	}
	if finishReason != "" {
		reqLog.FinishReason = finishReason
	}
}

// parseNormalTokenUsage extracts token usage from non-streaming response
func (s *RelayService) parseNormalTokenUsage(body []byte, reqLog *RequestLog) {
	usage := gjson.GetBytes(body, "usage")
	if usage.Exists() {
		reqLog.InputTokens = int(usage.Get("input_tokens").Int())
		reqLog.OutputTokens = int(usage.Get("output_tokens").Int())
		reqLog.CacheReadTokens = int(usage.Get("cache_read_input_tokens").Int())
	}

	// OpenAI format
	if !usage.Exists() {
		usage = gjson.GetBytes(body, "usage")
		reqLog.InputTokens = int(usage.Get("prompt_tokens").Int())
		reqLog.OutputTokens = int(usage.Get("completion_tokens").Int())
	}

	finishReason := gjson.GetBytes(body, "stop_reason").String()
	if finishReason == "" {
		finishReason = gjson.GetBytes(body, "choices.0.finish_reason").String()
	}
	reqLog.FinishReason = finishReason
}

// setRequestHeaders sets headers for upstream request
func (s *RelayService) setRequestHeaders(req *http.Request, c *app.RequestContext, provider *models.Provider, platform Platform) {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	// Platform-specific authorization
	switch platform {
	case PlatformGemini:
		// Gemini uses URL query parameter for API key
		q := req.URL.Query()
		q.Set("key", provider.APIKey)
		req.URL.RawQuery = q.Encode()
	default:
		// OpenAI/Claude style Bearer token
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", provider.APIKey))
	}

	// Claude-specific headers
	if platform == PlatformClaude {
		req.Header.Set("x-api-key", provider.APIKey)
		req.Header.Set("anthropic-version", "2023-06-01")
	}

	// Forward some client headers
	if userAgent := string(c.GetHeader("User-Agent")); userAgent != "" {
		req.Header.Set("X-Original-User-Agent", userAgent)
	}
}

// filterProviders filters providers that support the model
func (s *RelayService) filterProviders(providers []*models.Provider, model string) []*models.Provider {
	var active []*models.Provider
	for _, p := range providers {
		if !p.Enabled {
			continue
		}
		if p.APIURL == "" || p.APIKey == "" {
			continue
		}
		if s.isModelSupported(p, model) {
			active = append(active, p)
		}
	}
	return active
}

// isModelSupported checks if provider supports the model
func (s *RelayService) isModelSupported(p *models.Provider, model string) bool {
	// If no models specified, support all
	if len(p.SupportedModels) == 0 {
		return true
	}

	// Check exact match
	if p.SupportedModels[model] {
		return true
	}

	// Check wildcard patterns
	for m := range p.SupportedModels {
		if strings.HasSuffix(m, "*") {
			prefix := strings.TrimSuffix(m, "*")
			if strings.HasPrefix(model, prefix) {
				return true
			}
		}
	}
	return false
}

// getEffectiveModel applies model mapping
func (s *RelayService) getEffectiveModel(p *models.Provider, model string) string {
	if len(p.ModelMapping) == 0 {
		return model
	}

	// Check for exact mapping
	if mapped, ok := p.ModelMapping[model]; ok {
		return mapped
	}

	// Check for wildcard mapping
	for pattern, replacement := range p.ModelMapping {
		if strings.HasSuffix(pattern, "*") {
			prefix := strings.TrimSuffix(pattern, "*")
			if strings.HasPrefix(model, prefix) {
				// Apply wildcard replacement
				if strings.Contains(replacement, "*") {
					suffix := strings.TrimPrefix(model, prefix)
					return strings.Replace(replacement, "*", suffix, 1)
				}
				return replacement
			}
		}
	}

	return model
}

// replaceModel replaces the model field in request body
func (s *RelayService) replaceModel(body []byte, newModel string) []byte {
	var data map[string]interface{}
	if err := json.Unmarshal(body, &data); err != nil {
		return body
	}
	data["model"] = newModel
	newBody, err := json.Marshal(data)
	if err != nil {
		return body
	}
	return newBody
}

// extractUserID extracts user ID from request
func (s *RelayService) extractUserID(c *app.RequestContext) string {
	// Try various header names
	userID := string(c.GetHeader("X-User-ID"))
	if userID == "" {
		userID = string(c.GetHeader("X-User-Id"))
	}
	if userID == "" {
		// Extract from Authorization header (JWT sub claim)
		// For now, return empty
		userID = "anonymous"
	}
	return userID
}

// forwardToNewAPI forwards request to NEW-API gateway
func (s *RelayService) forwardToNewAPI(
	ctx context.Context,
	c *app.RequestContext,
	platform Platform,
	endpoint string,
	body []byte,
	isStream bool,
	model string,
	traceID string,
) (bool, error) {
	targetURL := strings.TrimSuffix(s.config.Features.NewAPIURL, "/") + endpoint

	req, err := http.NewRequestWithContext(ctx, "POST", targetURL, bytes.NewReader(body))
	if err != nil {
		return false, err
	}

	// Copy headers from client
	req.Header.Set("Content-Type", "application/json")
	if auth := string(c.GetHeader("Authorization")); auth != "" {
		req.Header.Set("Authorization", auth)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 500 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return false, fmt.Errorf("NEW-API error: %d - %s", resp.StatusCode, string(bodyBytes))
	}

	// Forward response
	if isStream {
		c.Response.Header.Set("Content-Type", "text/event-stream")
		c.Response.Header.Set("Cache-Control", "no-cache")
		c.Response.Header.Set("Connection", "keep-alive")
		c.Response.Header.Set("Transfer-Encoding", "chunked")
		c.Response.SetStatusCode(resp.StatusCode)

		scanner := bufio.NewScanner(resp.Body)
		for scanner.Scan() {
			c.Response.AppendBody(scanner.Bytes())
			c.Response.AppendBodyString("\n")
		}
		if scanner.Err() != nil {
			return false, scanner.Err()
		}
	} else {
		bodyBytes, _ := io.ReadAll(resp.Body)
		for key, values := range resp.Header {
			for _, value := range values {
				c.Response.Header.Set(key, value)
			}
		}
		c.Response.SetStatusCode(resp.StatusCode)
		c.Response.SetBody(bodyBytes)
	}

	return true, nil
}

// Helper functions

func generateTraceID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

func generateID() string {
	return fmt.Sprintf("req_%d", time.Now().UnixNano())
}

func classifyHTTPError(statusCode int) string {
	switch {
	case statusCode == 401 || statusCode == 403:
		return "auth_error"
	case statusCode == 429:
		return "rate_limit"
	case statusCode >= 400 && statusCode < 500:
		return "client_error"
	case statusCode >= 500:
		return "server_error"
	default:
		return "unknown_error"
	}
}
