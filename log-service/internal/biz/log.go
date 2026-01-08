package biz

import (
	"context"
	"time"

	"go.uber.org/zap"
)

// RequestLog represents a request log entry
type RequestLog struct {
	ID        string    `json:"id"`
	TraceID   string    `json:"trace_id"`
	RequestID string    `json:"request_id"`
	UserID    string    `json:"user_id"`
	CreatedAt time.Time `json:"created_at"`

	Platform      string `json:"platform"`
	Model         string `json:"model"`
	Provider      string `json:"provider"`
	ProviderModel string `json:"provider_model"`

	RequestMethod string `json:"request_method"`
	RequestPath   string `json:"request_path"`
	IsStream      bool   `json:"is_stream"`
	UserAgent     string `json:"user_agent"`
	ClientIP      string `json:"client_ip"`

	HTTPCode     int     `json:"http_code"`
	DurationSec  float64 `json:"duration_sec"`
	FinishReason string  `json:"finish_reason"`

	InputTokens       int `json:"input_tokens"`
	OutputTokens      int `json:"output_tokens"`
	CacheCreateTokens int `json:"cache_create_tokens"`
	CacheReadTokens   int `json:"cache_read_tokens"`
	ReasoningTokens   int `json:"reasoning_tokens"`

	InputCost       float64 `json:"input_cost"`
	OutputCost      float64 `json:"output_cost"`
	CacheCreateCost float64 `json:"cache_create_cost"`
	CacheReadCost   float64 `json:"cache_read_cost"`
	TotalCost       float64 `json:"total_cost"`

	ErrorType         string `json:"error_type"`
	ErrorMessage      string `json:"error_message"`
	ProviderErrorCode string `json:"provider_error_code"`
}

// LogFilter for querying logs
type LogFilter struct {
	UserID     string
	Platform   string
	Provider   string
	Model      string
	TraceID    string
	StartTime  time.Time
	EndTime    time.Time
	Limit      int
	Offset     int
	OrderBy    string
	Descending bool
}

// StatsFilter for statistics queries
type StatsFilter struct {
	UserID    string
	Platform  string
	StartTime time.Time
	EndTime   time.Time
}

// LogStats represents aggregated statistics
type LogStats struct {
	TotalRequests   int64   `json:"total_requests"`
	SuccessRequests int64   `json:"success_requests"`
	FailedRequests  int64   `json:"failed_requests"`
	TotalTokens     int64   `json:"total_tokens"`
	TotalCost       float64 `json:"total_cost"`
	AvgLatencyMs    float64 `json:"avg_latency_ms"`
	P50LatencyMs    float64 `json:"p50_latency_ms"`
	P95LatencyMs    float64 `json:"p95_latency_ms"`
	P99LatencyMs    float64 `json:"p99_latency_ms"`
}

// HourlyStat represents hourly statistics
type HourlyStat struct {
	Hour          time.Time `json:"hour"`
	Platform      string    `json:"platform"`
	Provider      string    `json:"provider"`
	Model         string    `json:"model"`
	RequestCount  int64     `json:"request_count"`
	SuccessCount  int64     `json:"success_count"`
	TotalTokens   int64     `json:"total_tokens"`
	TotalCost     float64   `json:"total_cost"`
	AvgDurationMs float64   `json:"avg_duration_ms"`
}

// DailyStat represents daily statistics
type DailyStat struct {
	Date          time.Time `json:"date"`
	Platform      string    `json:"platform"`
	Provider      string    `json:"provider"`
	Model         string    `json:"model"`
	RequestCount  int64     `json:"request_count"`
	SuccessCount  int64     `json:"success_count"`
	TotalTokens   int64     `json:"total_tokens"`
	TotalCost     float64   `json:"total_cost"`
	AvgDurationMs float64   `json:"avg_duration_ms"`
}

// ModelUsageStat represents per-model usage statistics
type ModelUsageStat struct {
	Model        string  `json:"model"`
	RequestCount int64   `json:"request_count"`
	TokenCount   int64   `json:"token_count"`
	TotalCost    float64 `json:"total_cost"`
	AvgLatencyMs float64 `json:"avg_latency_ms"`
	ErrorRate    float64 `json:"error_rate"`
}

// LogRepo is the log repository interface
type LogRepo interface {
	Insert(ctx context.Context, log *RequestLog) error
	InsertBatch(ctx context.Context, logs []*RequestLog) error
	Query(ctx context.Context, filter *LogFilter) ([]*RequestLog, int64, error)
	GetStats(ctx context.Context, filter *StatsFilter) (*LogStats, error)
	GetHourlyStats(ctx context.Context, filter *StatsFilter) ([]*HourlyStat, error)
	GetDailyStats(ctx context.Context, filter *StatsFilter) ([]*DailyStat, error)
	GetModelUsage(ctx context.Context, filter *StatsFilter, limit int) ([]*ModelUsageStat, error)
}

// LogUsecase is the log business logic
type LogUsecase struct {
	repo   LogRepo
	logger *zap.Logger
}

// NewLogUsecase creates a new log usecase
func NewLogUsecase(repo LogRepo, logger *zap.Logger) *LogUsecase {
	return &LogUsecase{
		repo:   repo,
		logger: logger,
	}
}

// WriteLog writes a single log entry
func (uc *LogUsecase) WriteLog(ctx context.Context, log *RequestLog) error {
	if log.CreatedAt.IsZero() {
		log.CreatedAt = time.Now()
	}
	return uc.repo.Insert(ctx, log)
}

// WriteLogs writes multiple log entries (batch)
func (uc *LogUsecase) WriteLogs(ctx context.Context, logs []*RequestLog) (int, int, error) {
	if len(logs) == 0 {
		return 0, 0, nil
	}

	now := time.Now()
	for _, log := range logs {
		if log.CreatedAt.IsZero() {
			log.CreatedAt = now
		}
	}

	err := uc.repo.InsertBatch(ctx, logs)
	if err != nil {
		uc.logger.Error("Failed to write log batch", zap.Error(err), zap.Int("count", len(logs)))
		return 0, len(logs), err
	}

	return len(logs), 0, nil
}

// QueryLogs queries logs with filters
func (uc *LogUsecase) QueryLogs(ctx context.Context, filter *LogFilter) ([]*RequestLog, int64, error) {
	return uc.repo.Query(ctx, filter)
}

// GetStats returns aggregated statistics
func (uc *LogUsecase) GetStats(ctx context.Context, filter *StatsFilter) (*LogStats, error) {
	return uc.repo.GetStats(ctx, filter)
}

// GetHourlyStats returns hourly statistics
func (uc *LogUsecase) GetHourlyStats(ctx context.Context, filter *StatsFilter) ([]*HourlyStat, error) {
	return uc.repo.GetHourlyStats(ctx, filter)
}

// GetDailyStats returns daily statistics
func (uc *LogUsecase) GetDailyStats(ctx context.Context, filter *StatsFilter) ([]*DailyStat, error) {
	return uc.repo.GetDailyStats(ctx, filter)
}

// GetModelUsage returns per-model usage statistics
func (uc *LogUsecase) GetModelUsage(ctx context.Context, filter *StatsFilter, limit int) ([]*ModelUsageStat, error) {
	return uc.repo.GetModelUsage(ctx, filter, limit)
}
