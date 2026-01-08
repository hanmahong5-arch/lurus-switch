package biz

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"go.uber.org/zap"
)

// MockLogRepo is a mock implementation of LogRepo
type MockLogRepo struct {
	logs []*RequestLog
}

func NewMockLogRepo() *MockLogRepo {
	return &MockLogRepo{
		logs: make([]*RequestLog, 0),
	}
}

func (m *MockLogRepo) Insert(ctx context.Context, log *RequestLog) error {
	m.logs = append(m.logs, log)
	return nil
}

func (m *MockLogRepo) InsertBatch(ctx context.Context, logs []*RequestLog) error {
	m.logs = append(m.logs, logs...)
	return nil
}

func (m *MockLogRepo) Query(ctx context.Context, filter *LogFilter) ([]*RequestLog, int64, error) {
	var result []*RequestLog
	for _, log := range m.logs {
		if filter.UserID != "" && log.UserID != filter.UserID {
			continue
		}
		if filter.Platform != "" && log.Platform != filter.Platform {
			continue
		}
		if filter.Provider != "" && log.Provider != filter.Provider {
			continue
		}
		if filter.Model != "" && log.Model != filter.Model {
			continue
		}
		if filter.TraceID != "" && log.TraceID != filter.TraceID {
			continue
		}
		if !filter.StartTime.IsZero() && log.CreatedAt.Before(filter.StartTime) {
			continue
		}
		if !filter.EndTime.IsZero() && log.CreatedAt.After(filter.EndTime) {
			continue
		}
		result = append(result, log)
	}

	// Apply limit and offset
	total := int64(len(result))
	if filter.Offset > 0 && filter.Offset < len(result) {
		result = result[filter.Offset:]
	}
	if filter.Limit > 0 && filter.Limit < len(result) {
		result = result[:filter.Limit]
	}

	return result, total, nil
}

func (m *MockLogRepo) GetStats(ctx context.Context, filter *StatsFilter) (*LogStats, error) {
	stats := &LogStats{}
	for _, log := range m.logs {
		if filter.UserID != "" && log.UserID != filter.UserID {
			continue
		}
		if filter.Platform != "" && log.Platform != filter.Platform {
			continue
		}
		stats.TotalRequests++
		if log.HTTPCode < 400 {
			stats.SuccessRequests++
		} else {
			stats.FailedRequests++
		}
		stats.TotalTokens += int64(log.InputTokens + log.OutputTokens)
		stats.TotalCost += log.TotalCost
	}
	return stats, nil
}

func (m *MockLogRepo) GetHourlyStats(ctx context.Context, filter *StatsFilter) ([]*HourlyStat, error) {
	return []*HourlyStat{
		{Hour: time.Now().Truncate(time.Hour), RequestCount: 100},
	}, nil
}

func (m *MockLogRepo) GetDailyStats(ctx context.Context, filter *StatsFilter) ([]*DailyStat, error) {
	return []*DailyStat{
		{Date: time.Now().Truncate(24 * time.Hour), RequestCount: 1000},
	}, nil
}

func (m *MockLogRepo) GetModelUsage(ctx context.Context, filter *StatsFilter, limit int) ([]*ModelUsageStat, error) {
	return []*ModelUsageStat{
		{Model: "claude-3-opus", RequestCount: 500},
	}, nil
}

func TestRequestLog_JSON(t *testing.T) {
	log := &RequestLog{
		ID:           "log-1",
		TraceID:      "trace-1",
		UserID:       "user-1",
		Platform:     "claude",
		Model:        "claude-3-opus",
		Provider:     "anthropic",
		IsStream:     true,
		HTTPCode:     200,
		DurationSec:  1.5,
		InputTokens:  1000,
		OutputTokens: 500,
		TotalCost:    0.05,
		CreatedAt:    time.Now(),
	}

	data, err := json.Marshal(log)
	if err != nil {
		t.Fatalf("Failed to marshal log: %v", err)
	}

	var decoded RequestLog
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal log: %v", err)
	}

	if decoded.TraceID != log.TraceID {
		t.Errorf("TraceID mismatch: got %s, want %s", decoded.TraceID, log.TraceID)
	}
	if decoded.InputTokens != log.InputTokens {
		t.Errorf("InputTokens mismatch: got %d, want %d", decoded.InputTokens, log.InputTokens)
	}
}

func TestLogUsecase_WriteLog(t *testing.T) {
	repo := NewMockLogRepo()
	logger := zap.NewNop()
	uc := NewLogUsecase(repo, logger)

	ctx := context.Background()
	log := &RequestLog{
		ID:       "log-1",
		TraceID:  "trace-1",
		UserID:   "user-1",
		Platform: "claude",
	}

	err := uc.WriteLog(ctx, log)
	if err != nil {
		t.Fatalf("WriteLog failed: %v", err)
	}

	if len(repo.logs) != 1 {
		t.Errorf("Expected 1 log, got %d", len(repo.logs))
	}
	if repo.logs[0].CreatedAt.IsZero() {
		t.Error("CreatedAt should be set automatically")
	}
}

func TestLogUsecase_WriteLog_PreservesCreatedAt(t *testing.T) {
	repo := NewMockLogRepo()
	logger := zap.NewNop()
	uc := NewLogUsecase(repo, logger)

	ctx := context.Background()
	customTime := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	log := &RequestLog{
		ID:        "log-1",
		CreatedAt: customTime,
	}

	err := uc.WriteLog(ctx, log)
	if err != nil {
		t.Fatalf("WriteLog failed: %v", err)
	}

	if !repo.logs[0].CreatedAt.Equal(customTime) {
		t.Error("Should preserve custom CreatedAt")
	}
}

func TestLogUsecase_WriteLogs_Batch(t *testing.T) {
	repo := NewMockLogRepo()
	logger := zap.NewNop()
	uc := NewLogUsecase(repo, logger)

	ctx := context.Background()
	logs := []*RequestLog{
		{ID: "log-1", Platform: "claude"},
		{ID: "log-2", Platform: "codex"},
		{ID: "log-3", Platform: "gemini"},
	}

	success, failed, err := uc.WriteLogs(ctx, logs)
	if err != nil {
		t.Fatalf("WriteLogs failed: %v", err)
	}

	if success != 3 {
		t.Errorf("Expected 3 success, got %d", success)
	}
	if failed != 0 {
		t.Errorf("Expected 0 failed, got %d", failed)
	}
	if len(repo.logs) != 3 {
		t.Errorf("Expected 3 logs in repo, got %d", len(repo.logs))
	}
}

func TestLogUsecase_WriteLogs_Empty(t *testing.T) {
	repo := NewMockLogRepo()
	logger := zap.NewNop()
	uc := NewLogUsecase(repo, logger)

	ctx := context.Background()
	success, failed, err := uc.WriteLogs(ctx, nil)
	if err != nil {
		t.Fatalf("WriteLogs failed: %v", err)
	}

	if success != 0 || failed != 0 {
		t.Errorf("Expected 0 success and 0 failed for empty batch")
	}
}

func TestLogUsecase_QueryLogs(t *testing.T) {
	repo := NewMockLogRepo()
	logger := zap.NewNop()
	uc := NewLogUsecase(repo, logger)

	ctx := context.Background()
	// Add some logs
	uc.WriteLog(ctx, &RequestLog{ID: "1", UserID: "user-1", Platform: "claude"})
	uc.WriteLog(ctx, &RequestLog{ID: "2", UserID: "user-1", Platform: "codex"})
	uc.WriteLog(ctx, &RequestLog{ID: "3", UserID: "user-2", Platform: "claude"})

	// Query by user
	logs, total, err := uc.QueryLogs(ctx, &LogFilter{UserID: "user-1"})
	if err != nil {
		t.Fatalf("QueryLogs failed: %v", err)
	}

	if total != 2 {
		t.Errorf("Expected 2 total, got %d", total)
	}
	if len(logs) != 2 {
		t.Errorf("Expected 2 logs, got %d", len(logs))
	}

	// Query by platform
	logs, total, err = uc.QueryLogs(ctx, &LogFilter{Platform: "claude"})
	if err != nil {
		t.Fatalf("QueryLogs failed: %v", err)
	}

	if total != 2 {
		t.Errorf("Expected 2 total for claude platform, got %d", total)
	}
}

func TestLogUsecase_QueryLogs_WithPagination(t *testing.T) {
	repo := NewMockLogRepo()
	logger := zap.NewNop()
	uc := NewLogUsecase(repo, logger)

	ctx := context.Background()
	// Add 10 logs
	for i := 0; i < 10; i++ {
		uc.WriteLog(ctx, &RequestLog{ID: string(rune('0' + i))})
	}

	// Query with limit
	logs, total, err := uc.QueryLogs(ctx, &LogFilter{Limit: 5})
	if err != nil {
		t.Fatalf("QueryLogs failed: %v", err)
	}

	if total != 10 {
		t.Errorf("Expected 10 total, got %d", total)
	}
	if len(logs) != 5 {
		t.Errorf("Expected 5 logs with limit, got %d", len(logs))
	}

	// Query with offset
	logs, _, err = uc.QueryLogs(ctx, &LogFilter{Offset: 5, Limit: 5})
	if err != nil {
		t.Fatalf("QueryLogs failed: %v", err)
	}

	if len(logs) != 5 {
		t.Errorf("Expected 5 logs with offset, got %d", len(logs))
	}
}

func TestLogUsecase_GetStats(t *testing.T) {
	repo := NewMockLogRepo()
	logger := zap.NewNop()
	uc := NewLogUsecase(repo, logger)

	ctx := context.Background()
	// Add logs with different statuses
	uc.WriteLog(ctx, &RequestLog{UserID: "user-1", HTTPCode: 200, InputTokens: 100, OutputTokens: 50, TotalCost: 0.01})
	uc.WriteLog(ctx, &RequestLog{UserID: "user-1", HTTPCode: 200, InputTokens: 200, OutputTokens: 100, TotalCost: 0.02})
	uc.WriteLog(ctx, &RequestLog{UserID: "user-1", HTTPCode: 500, InputTokens: 100, OutputTokens: 0, TotalCost: 0.00})

	stats, err := uc.GetStats(ctx, &StatsFilter{UserID: "user-1"})
	if err != nil {
		t.Fatalf("GetStats failed: %v", err)
	}

	if stats.TotalRequests != 3 {
		t.Errorf("Expected 3 total requests, got %d", stats.TotalRequests)
	}
	if stats.SuccessRequests != 2 {
		t.Errorf("Expected 2 success requests, got %d", stats.SuccessRequests)
	}
	if stats.FailedRequests != 1 {
		t.Errorf("Expected 1 failed request, got %d", stats.FailedRequests)
	}
	if stats.TotalTokens != 550 {
		t.Errorf("Expected 550 total tokens, got %d", stats.TotalTokens)
	}
}

func TestLogUsecase_GetHourlyStats(t *testing.T) {
	repo := NewMockLogRepo()
	logger := zap.NewNop()
	uc := NewLogUsecase(repo, logger)

	ctx := context.Background()
	stats, err := uc.GetHourlyStats(ctx, &StatsFilter{})
	if err != nil {
		t.Fatalf("GetHourlyStats failed: %v", err)
	}

	if len(stats) != 1 {
		t.Errorf("Expected 1 hourly stat, got %d", len(stats))
	}
}

func TestLogUsecase_GetDailyStats(t *testing.T) {
	repo := NewMockLogRepo()
	logger := zap.NewNop()
	uc := NewLogUsecase(repo, logger)

	ctx := context.Background()
	stats, err := uc.GetDailyStats(ctx, &StatsFilter{})
	if err != nil {
		t.Fatalf("GetDailyStats failed: %v", err)
	}

	if len(stats) != 1 {
		t.Errorf("Expected 1 daily stat, got %d", len(stats))
	}
}

func TestLogUsecase_GetModelUsage(t *testing.T) {
	repo := NewMockLogRepo()
	logger := zap.NewNop()
	uc := NewLogUsecase(repo, logger)

	ctx := context.Background()
	stats, err := uc.GetModelUsage(ctx, &StatsFilter{}, 10)
	if err != nil {
		t.Fatalf("GetModelUsage failed: %v", err)
	}

	if len(stats) != 1 {
		t.Errorf("Expected 1 model usage stat, got %d", len(stats))
	}
	if stats[0].Model != "claude-3-opus" {
		t.Errorf("Expected claude-3-opus, got %s", stats[0].Model)
	}
}

func TestLogFilter(t *testing.T) {
	filter := &LogFilter{
		UserID:     "user-1",
		Platform:   "claude",
		Provider:   "anthropic",
		Model:      "claude-3-opus",
		TraceID:    "trace-1",
		StartTime:  time.Now().Add(-24 * time.Hour),
		EndTime:    time.Now(),
		Limit:      100,
		Offset:     0,
		OrderBy:    "created_at",
		Descending: true,
	}

	if filter.UserID != "user-1" {
		t.Error("UserID not set correctly")
	}
	if filter.Limit != 100 {
		t.Error("Limit not set correctly")
	}
}

func TestStatsFilter(t *testing.T) {
	filter := &StatsFilter{
		UserID:    "user-1",
		Platform:  "claude",
		StartTime: time.Now().Add(-7 * 24 * time.Hour),
		EndTime:   time.Now(),
	}

	if filter.UserID != "user-1" {
		t.Error("UserID not set correctly")
	}
}

func TestLogStats(t *testing.T) {
	stats := &LogStats{
		TotalRequests:   1000,
		SuccessRequests: 950,
		FailedRequests:  50,
		TotalTokens:     5000000,
		TotalCost:       250.0,
		AvgLatencyMs:    150.0,
		P50LatencyMs:    100.0,
		P95LatencyMs:    300.0,
		P99LatencyMs:    500.0,
	}

	if stats.SuccessRequests+stats.FailedRequests != stats.TotalRequests {
		t.Error("Success + Failed should equal Total")
	}
}

func TestHourlyStat(t *testing.T) {
	stat := &HourlyStat{
		Hour:          time.Now().Truncate(time.Hour),
		Platform:      "claude",
		Provider:      "anthropic",
		Model:         "claude-3-opus",
		RequestCount:  500,
		SuccessCount:  490,
		TotalTokens:   250000,
		TotalCost:     12.50,
		AvgDurationMs: 120.0,
	}

	if stat.RequestCount < stat.SuccessCount {
		t.Error("RequestCount should be >= SuccessCount")
	}
}

func TestDailyStat(t *testing.T) {
	stat := &DailyStat{
		Date:          time.Now().Truncate(24 * time.Hour),
		Platform:      "claude",
		RequestCount:  5000,
		TotalTokens:   2500000,
		TotalCost:     125.0,
		AvgDurationMs: 150.0,
	}

	if stat.RequestCount == 0 {
		t.Error("RequestCount should not be zero")
	}
}

func TestModelUsageStat(t *testing.T) {
	stat := &ModelUsageStat{
		Model:        "claude-3-opus",
		RequestCount: 10000,
		TokenCount:   50000000,
		TotalCost:    500.0,
		AvgLatencyMs: 180.0,
		ErrorRate:    0.02,
	}

	if stat.ErrorRate < 0 || stat.ErrorRate > 1 {
		t.Error("ErrorRate should be between 0 and 1")
	}
}
