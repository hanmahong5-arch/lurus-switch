package models

import (
	"encoding/json"
	"testing"
	"time"
)

func TestRequestLog_JSON(t *testing.T) {
	log := &RequestLog{
		ID:           "log-123",
		TraceID:      "trace-abc",
		RequestID:    "req-xyz",
		UserID:       "user-1",
		CreatedAt:    time.Now(),
		Platform:     PlatformClaude,
		Model:        "claude-3-opus",
		Provider:     "anthropic",
		IsStream:     true,
		HTTPCode:     200,
		DurationSec:  1.5,
		InputTokens:  1000,
		OutputTokens: 500,
		TotalCost:    0.05,
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
	if decoded.Platform != log.Platform {
		t.Errorf("Platform mismatch: got %s, want %s", decoded.Platform, log.Platform)
	}
	if decoded.InputTokens != log.InputTokens {
		t.Errorf("InputTokens mismatch: got %d, want %d", decoded.InputTokens, log.InputTokens)
	}
	if decoded.TotalCost != log.TotalCost {
		t.Errorf("TotalCost mismatch: got %f, want %f", decoded.TotalCost, log.TotalCost)
	}
}

func TestRequestLog_WithErrors(t *testing.T) {
	log := &RequestLog{
		ID:                "log-err",
		TraceID:           "trace-err",
		UserID:            "user-1",
		CreatedAt:         time.Now(),
		Platform:          PlatformCodex,
		HTTPCode:          429,
		ErrorType:         "rate_limit",
		ErrorMessage:      "Too many requests",
		ProviderErrorCode: "429",
	}

	data, err := json.Marshal(log)
	if err != nil {
		t.Fatalf("Failed to marshal error log: %v", err)
	}

	var decoded RequestLog
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal error log: %v", err)
	}

	if decoded.ErrorType != log.ErrorType {
		t.Errorf("ErrorType mismatch: got %s, want %s", decoded.ErrorType, log.ErrorType)
	}
	if decoded.ErrorMessage != log.ErrorMessage {
		t.Errorf("ErrorMessage mismatch: got %s, want %s", decoded.ErrorMessage, log.ErrorMessage)
	}
}

func TestRequestLogStats(t *testing.T) {
	stats := &RequestLogStats{
		TotalRequests:   10000,
		SuccessRequests: 9500,
		FailedRequests:  500,
		TotalTokens:     5000000,
		TotalCost:       250.50,
		AvgLatency:      150.0,
		P50Latency:      100.0,
		P95Latency:      300.0,
		P99Latency:      500.0,
	}

	data, err := json.Marshal(stats)
	if err != nil {
		t.Fatalf("Failed to marshal stats: %v", err)
	}

	var decoded RequestLogStats
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal stats: %v", err)
	}

	if decoded.TotalRequests != stats.TotalRequests {
		t.Errorf("TotalRequests mismatch: got %d, want %d", decoded.TotalRequests, stats.TotalRequests)
	}
	if decoded.P99Latency != stats.P99Latency {
		t.Errorf("P99Latency mismatch: got %f, want %f", decoded.P99Latency, stats.P99Latency)
	}
}

func TestHourlyStats(t *testing.T) {
	stats := &HourlyStats{
		Hour:          time.Now().Truncate(time.Hour),
		Platform:      PlatformGemini,
		Model:         "gemini-pro",
		Provider:      "google",
		RequestCount:  500,
		SuccessCount:  490,
		TotalTokens:   250000,
		TotalCost:     12.50,
		AvgDurationMs: 120.5,
	}

	data, err := json.Marshal(stats)
	if err != nil {
		t.Fatalf("Failed to marshal hourly stats: %v", err)
	}

	var decoded HourlyStats
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal hourly stats: %v", err)
	}

	if decoded.RequestCount != stats.RequestCount {
		t.Errorf("RequestCount mismatch: got %d, want %d", decoded.RequestCount, stats.RequestCount)
	}
}

func TestDailyStats(t *testing.T) {
	stats := &DailyStats{
		Date:          time.Now().Truncate(24 * time.Hour),
		Platform:      PlatformClaude,
		Model:         "claude-3-sonnet",
		Provider:      "anthropic",
		RequestCount:  5000,
		SuccessCount:  4900,
		TotalTokens:   2500000,
		TotalCost:     125.00,
		AvgDurationMs: 200.0,
	}

	data, err := json.Marshal(stats)
	if err != nil {
		t.Fatalf("Failed to marshal daily stats: %v", err)
	}

	var decoded DailyStats
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal daily stats: %v", err)
	}

	if decoded.TotalCost != stats.TotalCost {
		t.Errorf("TotalCost mismatch: got %f, want %f", decoded.TotalCost, stats.TotalCost)
	}
}

func TestModelUsageStats(t *testing.T) {
	stats := &ModelUsageStats{
		Model:        "gpt-4",
		RequestCount: 10000,
		TokenCount:   50000000,
		TotalCost:    500.00,
		AvgLatency:   180.0,
		ErrorRate:    0.02,
	}

	data, err := json.Marshal(stats)
	if err != nil {
		t.Fatalf("Failed to marshal model usage stats: %v", err)
	}

	var decoded ModelUsageStats
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal model usage stats: %v", err)
	}

	if decoded.ErrorRate != stats.ErrorRate {
		t.Errorf("ErrorRate mismatch: got %f, want %f", decoded.ErrorRate, stats.ErrorRate)
	}
}
