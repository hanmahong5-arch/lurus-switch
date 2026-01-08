package http

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

// Client is an enhanced HTTP client with retry, timeout, and observability
type Client struct {
	client  *http.Client
	config  *Config
	logger  *zap.Logger
	tracer  trace.Tracer
}

// Config represents HTTP client configuration
type Config struct {
	// Timeout for non-streaming requests
	Timeout time.Duration `json:"timeout" yaml:"timeout"`

	// StreamTimeout for streaming requests (SSE)
	StreamTimeout time.Duration `json:"stream_timeout" yaml:"stream_timeout"`

	// Connection pool settings
	MaxIdleConns        int           `json:"max_idle_conns" yaml:"max_idle_conns"`
	MaxIdleConnsPerHost int           `json:"max_idle_conns_per_host" yaml:"max_idle_conns_per_host"`
	IdleConnTimeout     time.Duration `json:"idle_conn_timeout" yaml:"idle_conn_timeout"`

	// TLS configuration
	InsecureSkipVerify bool `json:"insecure_skip_verify" yaml:"insecure_skip_verify"`

	// Retry configuration
	RetryCount    int           `json:"retry_count" yaml:"retry_count"`
	RetryWaitMin  time.Duration `json:"retry_wait_min" yaml:"retry_wait_min"`
	RetryWaitMax  time.Duration `json:"retry_wait_max" yaml:"retry_wait_max"`
	RetryableStatusCodes []int  `json:"retryable_status_codes" yaml:"retryable_status_codes"`

	// User agent
	UserAgent string `json:"user_agent" yaml:"user_agent"`
}

// DefaultConfig returns default HTTP client configuration
func DefaultConfig() *Config {
	return &Config{
		Timeout:             60 * time.Second,
		StreamTimeout:       5 * time.Minute,
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 10,
		IdleConnTimeout:     90 * time.Second,
		RetryCount:          3,
		RetryWaitMin:        100 * time.Millisecond,
		RetryWaitMax:        2 * time.Second,
		RetryableStatusCodes: []int{502, 503, 504},
		UserAgent:           "lurus-switch/1.0",
	}
}

// NewClient creates a new HTTP client
func NewClient(cfg *Config, logger *zap.Logger) *Client {
	if cfg == nil {
		cfg = DefaultConfig()
	}
	if logger == nil {
		logger, _ = zap.NewProduction()
	}

	transport := &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		MaxIdleConns:        cfg.MaxIdleConns,
		MaxIdleConnsPerHost: cfg.MaxIdleConnsPerHost,
		IdleConnTimeout:     cfg.IdleConnTimeout,
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: cfg.InsecureSkipVerify,
		},
		ForceAttemptHTTP2: true,
	}

	return &Client{
		client: &http.Client{
			Transport: transport,
			Timeout:   cfg.Timeout,
		},
		config: cfg,
		logger: logger,
		tracer: otel.Tracer("lurus-common/http"),
	}
}

// Request represents an HTTP request
type Request struct {
	Method  string
	URL     string
	Headers map[string]string
	Body    interface{}
	Timeout time.Duration
	Stream  bool
}

// Response represents an HTTP response
type Response struct {
	StatusCode int
	Headers    http.Header
	Body       []byte
	Duration   time.Duration
}

// Do executes an HTTP request with retry logic
func (c *Client) Do(ctx context.Context, req *Request) (*Response, error) {
	ctx, span := c.tracer.Start(ctx, "http.request",
		trace.WithAttributes(
			attribute.String("http.method", req.Method),
			attribute.String("http.url", req.URL),
		))
	defer span.End()

	start := time.Now()

	// Build request body
	var bodyReader io.Reader
	if req.Body != nil {
		switch v := req.Body.(type) {
		case []byte:
			bodyReader = bytes.NewReader(v)
		case string:
			bodyReader = bytes.NewReader([]byte(v))
		case io.Reader:
			bodyReader = v
		default:
			jsonBody, err := json.Marshal(req.Body)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal request body: %w", err)
			}
			bodyReader = bytes.NewReader(jsonBody)
		}
	}

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, req.Method, req.URL, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	if c.config.UserAgent != "" {
		httpReq.Header.Set("User-Agent", c.config.UserAgent)
	}
	for k, v := range req.Headers {
		httpReq.Header.Set(k, v)
	}

	// Execute with retry
	var resp *http.Response
	var lastErr error

	for attempt := 0; attempt <= c.config.RetryCount; attempt++ {
		if attempt > 0 {
			// Calculate backoff
			wait := c.calculateBackoff(attempt)
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(wait):
			}
			c.logger.Debug("Retrying request",
				zap.Int("attempt", attempt),
				zap.String("url", req.URL))
		}

		resp, lastErr = c.client.Do(httpReq)
		if lastErr != nil {
			continue
		}

		// Check if retryable
		if !c.isRetryable(resp.StatusCode) {
			break
		}

		resp.Body.Close()
	}

	if lastErr != nil {
		span.RecordError(lastErr)
		return nil, lastErr
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	duration := time.Since(start)
	span.SetAttributes(
		attribute.Int("http.status_code", resp.StatusCode),
		attribute.Int64("http.response_size", int64(len(body))),
		attribute.Int64("http.duration_ms", duration.Milliseconds()),
	)

	return &Response{
		StatusCode: resp.StatusCode,
		Headers:    resp.Header,
		Body:       body,
		Duration:   duration,
	}, nil
}

// DoStream executes a streaming HTTP request
func (c *Client) DoStream(ctx context.Context, req *Request) (*http.Response, error) {
	ctx, span := c.tracer.Start(ctx, "http.stream",
		trace.WithAttributes(
			attribute.String("http.method", req.Method),
			attribute.String("http.url", req.URL),
		))
	// Note: span.End() should be called by the caller after consuming the stream

	// Build request body
	var bodyReader io.Reader
	if req.Body != nil {
		switch v := req.Body.(type) {
		case []byte:
			bodyReader = bytes.NewReader(v)
		case string:
			bodyReader = bytes.NewReader([]byte(v))
		case io.Reader:
			bodyReader = v
		default:
			jsonBody, err := json.Marshal(req.Body)
			if err != nil {
				span.End()
				return nil, fmt.Errorf("failed to marshal request body: %w", err)
			}
			bodyReader = bytes.NewReader(jsonBody)
		}
	}

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, req.Method, req.URL, bodyReader)
	if err != nil {
		span.End()
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	if c.config.UserAgent != "" {
		httpReq.Header.Set("User-Agent", c.config.UserAgent)
	}
	for k, v := range req.Headers {
		httpReq.Header.Set(k, v)
	}

	// Use stream timeout
	client := &http.Client{
		Transport: c.client.Transport,
		Timeout:   c.config.StreamTimeout,
	}

	resp, err := client.Do(httpReq)
	if err != nil {
		span.End()
		return nil, err
	}

	span.SetAttributes(attribute.Int("http.status_code", resp.StatusCode))
	return resp, nil
}

// Get performs a GET request
func (c *Client) Get(ctx context.Context, url string, headers map[string]string) (*Response, error) {
	return c.Do(ctx, &Request{
		Method:  http.MethodGet,
		URL:     url,
		Headers: headers,
	})
}

// Post performs a POST request
func (c *Client) Post(ctx context.Context, url string, body interface{}, headers map[string]string) (*Response, error) {
	return c.Do(ctx, &Request{
		Method:  http.MethodPost,
		URL:     url,
		Headers: headers,
		Body:    body,
	})
}

// PostJSON performs a POST request with JSON body
func (c *Client) PostJSON(ctx context.Context, url string, body interface{}) (*Response, error) {
	return c.Post(ctx, url, body, map[string]string{
		"Content-Type": "application/json",
	})
}

// calculateBackoff calculates exponential backoff duration
func (c *Client) calculateBackoff(attempt int) time.Duration {
	wait := c.config.RetryWaitMin * time.Duration(1<<uint(attempt-1))
	if wait > c.config.RetryWaitMax {
		wait = c.config.RetryWaitMax
	}
	return wait
}

// isRetryable checks if the status code is retryable
func (c *Client) isRetryable(statusCode int) bool {
	for _, code := range c.config.RetryableStatusCodes {
		if statusCode == code {
			return true
		}
	}
	return false
}
