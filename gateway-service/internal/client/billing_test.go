package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/pocketzworld/lurus-switch/gateway-service/internal/conf"
	"go.uber.org/zap"
)

func TestBillingClient_CheckBalance_Allowed(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Path should be /api/v1/billing/check/{user_id}
		if !strings.HasPrefix(r.URL.Path, "/api/v1/billing/check/") {
			t.Errorf("Expected path /api/v1/billing/check/{user_id}, got %s", r.URL.Path)
		}

		response := BalanceCheckResult{
			Allowed: true,
			Balance: 100.0,
			Quota:   1000000,
			Used:    50000,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	config := &conf.Billing{
		Endpoint: server.URL,
		Timeout:  5 * time.Second,
		Enabled:  true,
	}
	logger := zap.NewNop()
	client := NewBillingClient(config, logger)

	ctx := context.Background()
	err := client.CheckBalance(ctx, "user-1")
	if err != nil {
		t.Errorf("CheckBalance should not return error for allowed user: %v", err)
	}
}

func TestBillingClient_CheckBalance_NotAllowed(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := BalanceCheckResult{
			Allowed: false,
			Message: "Quota exceeded",
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	config := &conf.Billing{
		Endpoint: server.URL,
		Timeout:  5 * time.Second,
		Enabled:  true,
	}
	logger := zap.NewNop()
	client := NewBillingClient(config, logger)

	ctx := context.Background()
	err := client.CheckBalance(ctx, "user-1")
	if err == nil {
		t.Error("CheckBalance should return error for not allowed user")
	}
}

func TestBillingClient_CheckBalance_PaymentRequired(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusPaymentRequired)
	}))
	defer server.Close()

	config := &conf.Billing{
		Endpoint: server.URL,
		Timeout:  5 * time.Second,
		Enabled:  true,
	}
	logger := zap.NewNop()
	client := NewBillingClient(config, logger)

	ctx := context.Background()
	err := client.CheckBalance(ctx, "user-1")
	if err == nil {
		t.Error("CheckBalance should return error for 402 response")
	}
}

func TestBillingClient_CheckBalance_Disabled(t *testing.T) {
	config := &conf.Billing{
		Endpoint: "http://should-not-be-called",
		Timeout:  5 * time.Second,
		Enabled:  false,
	}
	logger := zap.NewNop()
	client := NewBillingClient(config, logger)

	ctx := context.Background()
	err := client.CheckBalance(ctx, "user-1")
	if err != nil {
		t.Errorf("CheckBalance should not return error when disabled: %v", err)
	}
}

func TestBillingClient_CheckBalance_FailOpen(t *testing.T) {
	// Server that returns error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	config := &conf.Billing{
		Endpoint: server.URL,
		Timeout:  5 * time.Second,
		Enabled:  true,
	}
	logger := zap.NewNop()
	client := NewBillingClient(config, logger)

	ctx := context.Background()
	err := client.CheckBalance(ctx, "user-1")
	// Fail-open: should not return error for server errors
	if err != nil {
		t.Errorf("CheckBalance should fail-open on server error: %v", err)
	}
}

func TestBillingClient_ReportUsage(t *testing.T) {
	received := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/api/v1/billing/usage") {
			t.Errorf("Expected path /api/v1/billing/usage, got %s", r.URL.Path)
		}
		if r.Method != "POST" {
			t.Errorf("Expected POST method, got %s", r.Method)
		}

		var report UsageReport
		if err := json.NewDecoder(r.Body).Decode(&report); err != nil {
			t.Errorf("Failed to decode request body: %v", err)
		}

		if report.UserID != "user-1" {
			t.Errorf("Expected user-1, got %s", report.UserID)
		}
		if report.InputTokens != 1000 {
			t.Errorf("Expected 1000 input tokens, got %d", report.InputTokens)
		}

		received = true
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	config := &conf.Billing{
		Endpoint: server.URL,
		Timeout:  5 * time.Second,
		Enabled:  true,
	}
	logger := zap.NewNop()
	client := NewBillingClient(config, logger)

	ctx := context.Background()
	report := &UsageReport{
		UserID:       "user-1",
		Platform:     "claude",
		Model:        "claude-3-opus",
		InputTokens:  1000,
		OutputTokens: 500,
		TotalCost:    0.05,
		TraceID:      "trace-1",
	}

	err := client.ReportUsage(ctx, report)
	if err != nil {
		t.Errorf("ReportUsage failed: %v", err)
	}

	if !received {
		t.Error("Server did not receive the request")
	}
}

func TestBillingClient_ReportUsage_Disabled(t *testing.T) {
	config := &conf.Billing{
		Endpoint: "http://should-not-be-called",
		Timeout:  5 * time.Second,
		Enabled:  false,
	}
	logger := zap.NewNop()
	client := NewBillingClient(config, logger)

	ctx := context.Background()
	report := &UsageReport{UserID: "user-1"}

	err := client.ReportUsage(ctx, report)
	if err != nil {
		t.Errorf("ReportUsage should not return error when disabled: %v", err)
	}
}

func TestBillingClient_GetQuota(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Path should be /api/v1/billing/quota/{user_id}
		if !strings.HasPrefix(r.URL.Path, "/api/v1/billing/quota/") {
			t.Errorf("Expected path /api/v1/billing/quota/{user_id}, got %s", r.URL.Path)
		}

		response := BalanceCheckResult{
			Allowed: true,
			Balance: 50.0,
			Quota:   1000000,
			Used:    250000,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	config := &conf.Billing{
		Endpoint: server.URL,
		Timeout:  5 * time.Second,
		Enabled:  true,
	}
	logger := zap.NewNop()
	client := NewBillingClient(config, logger)

	ctx := context.Background()
	result, err := client.GetQuota(ctx, "user-1")
	if err != nil {
		t.Fatalf("GetQuota failed: %v", err)
	}

	if result.Quota != 1000000 {
		t.Errorf("Expected quota 1000000, got %d", result.Quota)
	}
	if result.Used != 250000 {
		t.Errorf("Expected used 250000, got %d", result.Used)
	}
}

func TestBillingClient_GetQuota_Disabled(t *testing.T) {
	config := &conf.Billing{
		Endpoint: "http://should-not-be-called",
		Timeout:  5 * time.Second,
		Enabled:  false,
	}
	logger := zap.NewNop()
	client := NewBillingClient(config, logger)

	ctx := context.Background()
	result, err := client.GetQuota(ctx, "user-1")
	if err != nil {
		t.Errorf("GetQuota should not return error when disabled: %v", err)
	}

	if !result.Allowed {
		t.Error("GetQuota should return Allowed=true when disabled")
	}
}

func TestBalanceCheckResult(t *testing.T) {
	result := &BalanceCheckResult{
		Allowed: true,
		Balance: 100.0,
		Quota:   1000000,
		Used:    500000,
		Message: "",
	}

	if !result.Allowed {
		t.Error("Expected Allowed to be true")
	}
	if result.Balance != 100.0 {
		t.Errorf("Expected Balance 100.0, got %f", result.Balance)
	}
}

func TestUsageReport(t *testing.T) {
	report := &UsageReport{
		UserID:       "user-1",
		Platform:     "claude",
		Model:        "claude-3-opus",
		InputTokens:  1000,
		OutputTokens: 500,
		TotalCost:    0.05,
		TraceID:      "trace-123",
	}

	data, err := json.Marshal(report)
	if err != nil {
		t.Fatalf("Failed to marshal UsageReport: %v", err)
	}

	var decoded UsageReport
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal UsageReport: %v", err)
	}

	if decoded.UserID != report.UserID {
		t.Errorf("UserID mismatch: got %s, want %s", decoded.UserID, report.UserID)
	}
	if decoded.InputTokens != report.InputTokens {
		t.Errorf("InputTokens mismatch: got %d, want %d", decoded.InputTokens, report.InputTokens)
	}
}
