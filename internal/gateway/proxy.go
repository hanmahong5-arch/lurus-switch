package gateway

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"lurus-switch/internal/metering"
)

const (
	maxRequestBodySize  = 10 << 20 // 10 MB
	upstreamTimeout     = 5 * time.Minute
)

// handleProxy forwards the request to the upstream LLM provider,
// swapping the per-app token for the user's Lurus Cloud token.
// Supports both streaming (SSE) and non-streaming responses.
func (s *Server) handleProxy(w http.ResponseWriter, r *http.Request) {
	s.activeReqs.Add(1)
	defer s.activeReqs.Add(-1)
	s.totalReqs.Add(1)

	meta := getMeta(r)

	// Read request body (needed to parse model name for metering).
	body, err := io.ReadAll(io.LimitReader(r.Body, maxRequestBodySize))
	if err != nil {
		writeOpenAIError(w, http.StatusBadRequest, "invalid_request", "failed to read request body")
		return
	}
	r.Body.Close()

	// Extract model from request body for metering.
	model := extractModelFromBody(body)
	if meta != nil {
		meta.Model = model
	}

	// Check upstream is configured.
	s.mu.Lock()
	upstreamURL := s.cfg.UpstreamURL
	userToken := s.cfg.UserToken
	s.mu.Unlock()

	if upstreamURL == "" {
		writeOpenAIError(w, http.StatusServiceUnavailable, "gateway_not_configured",
			"Gateway upstream not configured. Open Lurus Switch settings to set your API endpoint.")
		return
	}
	if userToken == "" {
		writeOpenAIError(w, http.StatusPaymentRequired, "no_balance",
			"No Lurus account connected. Open Lurus Switch to log in and add balance.")
		return
	}

	// Build upstream request.
	targetURL := strings.TrimRight(upstreamURL, "/") + r.URL.Path
	if r.URL.RawQuery != "" {
		targetURL += "?" + r.URL.RawQuery
	}

	upstreamReq, err := http.NewRequestWithContext(r.Context(), r.Method, targetURL, bytes.NewReader(body))
	if err != nil {
		writeOpenAIError(w, http.StatusInternalServerError, "internal_error",
			fmt.Sprintf("failed to create upstream request: %v", err))
		return
	}

	// Copy relevant headers, swap auth token.
	copyRequestHeaders(upstreamReq, r)
	upstreamReq.Header.Set("Authorization", "Bearer "+userToken)

	// Send to upstream.
	client := &http.Client{Timeout: upstreamTimeout}
	resp, err := client.Do(upstreamReq)
	if err != nil {
		writeOpenAIError(w, http.StatusBadGateway, "upstream_error",
			fmt.Sprintf("upstream request failed: %v", err))
		s.recordError(meta, model, err.Error())
		return
	}
	defer resp.Body.Close()

	// Determine if this is a streaming response.
	isStreaming := strings.Contains(resp.Header.Get("Content-Type"), "text/event-stream")

	if isStreaming {
		s.proxyStreaming(w, resp, meta, model)
	} else {
		s.proxyBuffered(w, resp, meta, model)
	}
}

// proxyBuffered handles non-streaming responses: read full body, extract usage, forward.
func (s *Server) proxyBuffered(w http.ResponseWriter, resp *http.Response, meta *RequestMeta, model string) {
	respBody, err := io.ReadAll(io.LimitReader(resp.Body, maxRequestBodySize))
	if err != nil {
		writeOpenAIError(w, http.StatusBadGateway, "upstream_read_error",
			"failed to read upstream response")
		return
	}

	// Copy response headers.
	copyResponseHeaders(w, resp)
	w.WriteHeader(resp.StatusCode)
	w.Write(respBody)

	// Extract usage for metering.
	usage := extractUsageFromBody(respBody)
	s.recordUsage(meta, model, usage, resp.StatusCode)
}

// proxyStreaming pipes SSE chunks from upstream to client in real-time.
func (s *Server) proxyStreaming(w http.ResponseWriter, resp *http.Response, meta *RequestMeta, model string) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		// Fallback to buffered if flushing is not supported.
		s.proxyBuffered(w, resp, meta, model)
		return
	}

	// Copy response headers.
	copyResponseHeaders(w, resp)
	w.WriteHeader(resp.StatusCode)

	var totalUsage UsageFromResponse
	buf := make([]byte, 4096)

	for {
		n, readErr := resp.Body.Read(buf)
		if n > 0 {
			chunk := buf[:n]
			w.Write(chunk)
			flusher.Flush()

			// Try to extract usage from the final SSE data chunk.
			// OpenAI sends usage in the last "data: {...}" line when stream_options.include_usage is set.
			if u := extractUsageFromSSEChunk(chunk); u.TotalTokens > 0 {
				totalUsage = u
			}
		}
		if readErr != nil {
			break
		}
	}

	s.recordUsage(meta, model, totalUsage, resp.StatusCode)
}

// --- metering helpers ---

func (s *Server) recordUsage(meta *RequestMeta, model string, usage UsageFromResponse, statusCode int) {
	if s.meter == nil || meta == nil {
		return
	}
	if usage.Model != "" {
		model = usage.Model
	}
	rec := metering.Record{
		AppID:      meta.AppID,
		Model:      model,
		TokensIn:   usage.PromptTokens,
		TokensOut:  usage.CompletionTokens,
		LatencyMs:  time.Since(meta.StartTime).Milliseconds(),
		StatusCode: statusCode,
		Timestamp:  time.Now(),
	}
	s.meter.Record(rec)
}

func (s *Server) recordError(meta *RequestMeta, model, errMsg string) {
	if s.meter == nil || meta == nil {
		return
	}
	rec := metering.Record{
		AppID:        meta.AppID,
		Model:        model,
		LatencyMs:    time.Since(meta.StartTime).Milliseconds(),
		StatusCode:   502,
		ErrorMessage: errMsg,
		Timestamp:    time.Now(),
	}
	s.meter.Record(rec)
}

// --- request/response helpers ---

func copyRequestHeaders(dst, src *http.Request) {
	for _, key := range []string{
		"Content-Type", "Accept", "User-Agent",
		"X-Request-ID", "X-Stainless-Arch", "X-Stainless-Lang",
		"X-Stainless-OS", "X-Stainless-Package-Version",
		"X-Stainless-Runtime", "X-Stainless-Runtime-Version",
	} {
		if v := src.Header.Get(key); v != "" {
			dst.Header.Set(key, v)
		}
	}
	dst.Header.Set("Content-Length", src.Header.Get("Content-Length"))
}

func copyResponseHeaders(w http.ResponseWriter, resp *http.Response) {
	for _, key := range []string{
		"Content-Type", "X-Request-ID", "X-RateLimit-Limit-Requests",
		"X-RateLimit-Limit-Tokens", "X-RateLimit-Remaining-Requests",
		"X-RateLimit-Remaining-Tokens", "X-RateLimit-Reset-Requests",
		"X-RateLimit-Reset-Tokens", "OpenAI-Processing-Ms",
	} {
		if v := resp.Header.Get(key); v != "" {
			w.Header().Set(key, v)
		}
	}
}

// --- JSON extraction helpers ---

func extractModelFromBody(body []byte) string {
	var req struct {
		Model string `json:"model"`
	}
	if json.Unmarshal(body, &req) == nil {
		return req.Model
	}
	return ""
}

func extractUsageFromBody(body []byte) UsageFromResponse {
	var resp struct {
		Model string `json:"model"`
		Usage struct {
			PromptTokens     int64 `json:"prompt_tokens"`
			CompletionTokens int64 `json:"completion_tokens"`
			TotalTokens      int64 `json:"total_tokens"`
		} `json:"usage"`
	}
	if json.Unmarshal(body, &resp) == nil {
		return UsageFromResponse{
			Model:            resp.Model,
			PromptTokens:     resp.Usage.PromptTokens,
			CompletionTokens: resp.Usage.CompletionTokens,
			TotalTokens:      resp.Usage.TotalTokens,
		}
	}
	return UsageFromResponse{}
}

// extractUsageFromSSEChunk attempts to parse usage from an SSE data line.
// SSE format: "data: {json}\n\n"
func extractUsageFromSSEChunk(chunk []byte) UsageFromResponse {
	lines := bytes.Split(chunk, []byte("\n"))
	for _, line := range lines {
		line = bytes.TrimSpace(line)
		if !bytes.HasPrefix(line, []byte("data: ")) {
			continue
		}
		data := line[6:] // strip "data: "
		if bytes.Equal(data, []byte("[DONE]")) {
			continue
		}
		var msg struct {
			Model string `json:"model"`
			Usage *struct {
				PromptTokens     int64 `json:"prompt_tokens"`
				CompletionTokens int64 `json:"completion_tokens"`
				TotalTokens      int64 `json:"total_tokens"`
			} `json:"usage"`
		}
		if json.Unmarshal(data, &msg) == nil && msg.Usage != nil {
			return UsageFromResponse{
				Model:            msg.Model,
				PromptTokens:     msg.Usage.PromptTokens,
				CompletionTokens: msg.Usage.CompletionTokens,
				TotalTokens:      msg.Usage.TotalTokens,
			}
		}
	}
	return UsageFromResponse{}
}
