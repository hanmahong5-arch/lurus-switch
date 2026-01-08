package models

import "time"

// RequestLog represents an LLM request log entry
// Optimized for ClickHouse OLAP queries
type RequestLog struct {
	// Primary identifiers
	ID        string    `json:"id" ch:"id"`
	TraceID   string    `json:"trace_id" ch:"trace_id"`
	RequestID string    `json:"request_id" ch:"request_id"`
	UserID    string    `json:"user_id" ch:"user_id"`
	CreatedAt time.Time `json:"created_at" ch:"created_at"`

	// Request metadata
	Platform      string `json:"platform" ch:"platform"`           // claude, codex, gemini
	Model         string `json:"model" ch:"model"`                 // requested model
	Provider      string `json:"provider" ch:"provider"`           // provider name
	ProviderModel string `json:"provider_model" ch:"provider_model"` // actual provider model

	// Request details
	RequestMethod string `json:"request_method" ch:"request_method"`
	RequestPath   string `json:"request_path" ch:"request_path"`
	IsStream      bool   `json:"is_stream" ch:"is_stream"`
	UserAgent     string `json:"user_agent" ch:"user_agent"`
	ClientIP      string `json:"client_ip" ch:"client_ip"`

	// Response details
	HTTPCode     int     `json:"http_code" ch:"http_code"`
	DurationSec  float64 `json:"duration_sec" ch:"duration_sec"`
	FinishReason string  `json:"finish_reason" ch:"finish_reason"`

	// Token usage
	InputTokens       int `json:"input_tokens" ch:"input_tokens"`
	OutputTokens      int `json:"output_tokens" ch:"output_tokens"`
	CacheCreateTokens int `json:"cache_create_tokens" ch:"cache_create_tokens"`
	CacheReadTokens   int `json:"cache_read_tokens" ch:"cache_read_tokens"`
	ReasoningTokens   int `json:"reasoning_tokens" ch:"reasoning_tokens"`

	// Pre-calculated costs (USD)
	InputCost       float64 `json:"input_cost" ch:"input_cost"`
	OutputCost      float64 `json:"output_cost" ch:"output_cost"`
	CacheCreateCost float64 `json:"cache_create_cost" ch:"cache_create_cost"`
	CacheReadCost   float64 `json:"cache_read_cost" ch:"cache_read_cost"`
	Ephemeral5mCost float64 `json:"ephemeral_5m_cost" ch:"ephemeral_5m_cost"`
	Ephemeral1hCost float64 `json:"ephemeral_1h_cost" ch:"ephemeral_1h_cost"`
	TotalCost       float64 `json:"total_cost" ch:"total_cost"`

	// Error information
	ErrorType         string `json:"error_type,omitempty" ch:"error_type"`
	ErrorMessage      string `json:"error_message,omitempty" ch:"error_message"`
	ProviderErrorCode string `json:"provider_error_code,omitempty" ch:"provider_error_code"`
}

// RequestLogStats represents aggregated statistics
type RequestLogStats struct {
	TotalRequests   int64   `json:"total_requests"`
	SuccessRequests int64   `json:"success_requests"`
	FailedRequests  int64   `json:"failed_requests"`
	TotalTokens     int64   `json:"total_tokens"`
	TotalCost       float64 `json:"total_cost"`
	AvgLatency      float64 `json:"avg_latency_ms"`
	P50Latency      float64 `json:"p50_latency_ms"`
	P95Latency      float64 `json:"p95_latency_ms"`
	P99Latency      float64 `json:"p99_latency_ms"`
}

// HourlyStats represents hourly aggregated statistics
type HourlyStats struct {
	Hour          time.Time `json:"hour" ch:"hour"`
	Platform      string    `json:"platform" ch:"platform"`
	Model         string    `json:"model" ch:"model"`
	Provider      string    `json:"provider" ch:"provider"`
	RequestCount  int64     `json:"request_count" ch:"request_count"`
	SuccessCount  int64     `json:"success_count" ch:"success_count"`
	TotalTokens   int64     `json:"total_tokens" ch:"total_tokens"`
	TotalCost     float64   `json:"total_cost" ch:"total_cost"`
	AvgDurationMs float64   `json:"avg_duration_ms" ch:"avg_duration_ms"`
}

// DailyStats represents daily aggregated statistics
type DailyStats struct {
	Date          time.Time `json:"date" ch:"date"`
	Platform      string    `json:"platform" ch:"platform"`
	Model         string    `json:"model" ch:"model"`
	Provider      string    `json:"provider" ch:"provider"`
	RequestCount  int64     `json:"request_count" ch:"request_count"`
	SuccessCount  int64     `json:"success_count" ch:"success_count"`
	TotalTokens   int64     `json:"total_tokens" ch:"total_tokens"`
	TotalCost     float64   `json:"total_cost" ch:"total_cost"`
	AvgDurationMs float64   `json:"avg_duration_ms" ch:"avg_duration_ms"`
}

// ModelUsageStats represents per-model usage statistics
type ModelUsageStats struct {
	Model        string  `json:"model"`
	RequestCount int64   `json:"request_count"`
	TokenCount   int64   `json:"token_count"`
	TotalCost    float64 `json:"total_cost"`
	AvgLatency   float64 `json:"avg_latency_ms"`
	ErrorRate    float64 `json:"error_rate"`
}
