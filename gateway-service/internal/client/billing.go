package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/pocketzworld/lurus-switch/gateway-service/internal/conf"
	"go.uber.org/zap"
)

// BillingClient is the client for Billing Service
type BillingClient struct {
	config     *conf.Billing
	httpClient *http.Client
	logger     *zap.Logger
}

// NewBillingClient creates a new billing client
func NewBillingClient(config *conf.Billing, logger *zap.Logger) *BillingClient {
	return &BillingClient{
		config: config,
		httpClient: &http.Client{
			Timeout: config.Timeout,
		},
		logger: logger,
	}
}

// BalanceCheckResult represents the result of a balance check
type BalanceCheckResult struct {
	Allowed bool    `json:"allowed"`
	Balance float64 `json:"balance"`
	Quota   int64   `json:"quota"`
	Used    int64   `json:"used"`
	Message string  `json:"message,omitempty"`
}

// CheckBalance checks if user has sufficient balance
func (c *BillingClient) CheckBalance(ctx context.Context, userID string) error {
	if !c.config.Enabled {
		return nil
	}

	url := fmt.Sprintf("%s/api/v1/billing/check/%s", c.config.Endpoint, userID)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.logger.Warn("Billing service unavailable, allowing request", zap.Error(err))
		return nil // Fail-open for availability
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusPaymentRequired {
		return fmt.Errorf("insufficient balance")
	}

	if resp.StatusCode != http.StatusOK {
		c.logger.Warn("Billing service error, allowing request",
			zap.Int("status", resp.StatusCode))
		return nil // Fail-open
	}

	var result BalanceCheckResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		c.logger.Warn("Failed to decode billing response, allowing request", zap.Error(err))
		return nil
	}

	if !result.Allowed {
		return fmt.Errorf("request not allowed: %s", result.Message)
	}

	return nil
}

// UsageReport represents a usage report to billing service
type UsageReport struct {
	UserID       string  `json:"user_id"`
	Platform     string  `json:"platform"`
	Model        string  `json:"model"`
	InputTokens  int     `json:"input_tokens"`
	OutputTokens int     `json:"output_tokens"`
	TotalCost    float64 `json:"total_cost"`
	TraceID      string  `json:"trace_id"`
}

// ReportUsage reports usage to billing service
func (c *BillingClient) ReportUsage(ctx context.Context, report *UsageReport) error {
	if !c.config.Enabled {
		return nil
	}

	url := fmt.Sprintf("%s/api/v1/billing/usage", c.config.Endpoint)
	body, err := json.Marshal(report)
	if err != nil {
		return fmt.Errorf("failed to marshal report: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.logger.Warn("Failed to report usage", zap.Error(err))
		return nil // Don't fail the request for billing errors
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		c.logger.Warn("Billing service returned error for usage report",
			zap.Int("status", resp.StatusCode))
	}

	return nil
}

// GetQuota gets user quota information
func (c *BillingClient) GetQuota(ctx context.Context, userID string) (*BalanceCheckResult, error) {
	if !c.config.Enabled {
		return &BalanceCheckResult{Allowed: true}, nil
	}

	url := fmt.Sprintf("%s/api/v1/billing/quota/%s", c.config.Endpoint, userID)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get quota: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("billing service returned %d", resp.StatusCode)
	}

	var result BalanceCheckResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}
