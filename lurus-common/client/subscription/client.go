// Package subscription provides a client for interacting with the subscription-service.
package subscription

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/lurus-ai/lurus-common/types"
)

// Client is the subscription-service HTTP client.
type Client struct {
	baseURL    string
	authToken  string
	httpClient *http.Client
}

// NewClient creates a new subscription-service client.
func NewClient(baseURL, authToken string) *Client {
	return &Client{
		baseURL:   baseURL,
		authToken: authToken,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// WithHTTPClient sets a custom HTTP client.
func (c *Client) WithHTTPClient(httpClient *http.Client) *Client {
	c.httpClient = httpClient
	return c
}

// doRequest performs an HTTP request to subscription-service.
func (c *Client) doRequest(ctx context.Context, method, path string, body interface{}) (*http.Response, error) {
	var bodyReader io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(jsonBody)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if c.authToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.authToken)
	}

	return c.httpClient.Do(req)
}

// Response wraps a subscription-service response.
type Response struct {
	Success bool            `json:"success"`
	Message string          `json:"message,omitempty"`
	Data    json.RawMessage `json:"data,omitempty"`
}

// parseResponse parses the response body.
func parseResponse(resp *http.Response, target interface{}) error {
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("subscription-service returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var r Response
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}

	if !r.Success {
		return fmt.Errorf("subscription-service request failed: %s", r.Message)
	}

	if target != nil && len(r.Data) > 0 {
		if err := json.Unmarshal(r.Data, target); err != nil {
			return fmt.Errorf("unmarshal data: %w", err)
		}
	}

	return nil
}

// GetPlans returns all available subscription plans.
func (c *Client) GetPlans(ctx context.Context) ([]*types.Plan, error) {
	resp, err := c.doRequest(ctx, http.MethodGet, "/api/v1/plans", nil)
	if err != nil {
		return nil, err
	}

	var plans []*types.Plan
	if err := parseResponse(resp, &plans); err != nil {
		return nil, err
	}

	return plans, nil
}

// GetPlan returns a subscription plan by code.
func (c *Client) GetPlan(ctx context.Context, code string) (*types.Plan, error) {
	resp, err := c.doRequest(ctx, http.MethodGet, fmt.Sprintf("/api/v1/plans/%s", code), nil)
	if err != nil {
		return nil, err
	}

	var plan types.Plan
	if err := parseResponse(resp, &plan); err != nil {
		return nil, err
	}

	return &plan, nil
}

// GetUserSubscription returns the active subscription for a user.
func (c *Client) GetUserSubscription(ctx context.Context, userID int) (*types.Subscription, error) {
	resp, err := c.doRequest(ctx, http.MethodGet, fmt.Sprintf("/api/v1/subscriptions/user/%d", userID), nil)
	if err != nil {
		return nil, err
	}

	var sub types.Subscription
	if err := parseResponse(resp, &sub); err != nil {
		return nil, err
	}

	return &sub, nil
}

// SubscribeRequest is the request to create a subscription.
type SubscribeRequest struct {
	UserID   int    `json:"user_id"`
	PlanCode string `json:"plan_code"`
}

// Subscribe creates a new subscription for a user.
func (c *Client) Subscribe(ctx context.Context, userID int, planCode string) (*types.Subscription, error) {
	body := SubscribeRequest{
		UserID:   userID,
		PlanCode: planCode,
	}

	resp, err := c.doRequest(ctx, http.MethodPost, "/api/v1/subscriptions", body)
	if err != nil {
		return nil, err
	}

	var sub types.Subscription
	if err := parseResponse(resp, &sub); err != nil {
		return nil, err
	}

	return &sub, nil
}

// CancelSubscription cancels a subscription.
func (c *Client) CancelSubscription(ctx context.Context, subscriptionID int64, reason string) error {
	body := map[string]interface{}{
		"reason": reason,
	}

	resp, err := c.doRequest(ctx, http.MethodPost, fmt.Sprintf("/api/v1/subscriptions/%d/cancel", subscriptionID), body)
	if err != nil {
		return err
	}

	return parseResponse(resp, nil)
}

// GetQuotaStatus returns the current quota status for a user.
// NOTE: Daily quota status is now fetched from new-api as the single source of truth.
// This method returns subscription-level quota info combined with new-api daily quota.
func (c *Client) GetQuotaStatus(ctx context.Context, userID int) (*types.QuotaStatus, error) {
	resp, err := c.doRequest(ctx, http.MethodGet, fmt.Sprintf("/api/v1/subscriptions/user/%d/quota", userID), nil)
	if err != nil {
		return nil, err
	}

	var status types.QuotaStatus
	if err := parseResponse(resp, &status); err != nil {
		return nil, err
	}

	return &status, nil
}
