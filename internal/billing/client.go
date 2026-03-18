package billing

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const (
	defaultTimeout  = 15 * time.Second
	maxResponseSize = 10 << 20 // 10 MB limit for API responses
)

// Client communicates with the lurus-api V2 billing API
type Client struct {
	baseURL    string
	tenantSlug string
	token      string
	httpClient *http.Client
}

// NewClient creates a new billing API client
func NewClient(baseURL, tenantSlug, token string) *Client {
	return &Client{
		baseURL:    strings.TrimRight(baseURL, "/"),
		tenantSlug: tenantSlug,
		token:      token,
		httpClient: &http.Client{Timeout: defaultTimeout},
	}
}

// GetUserInfo retrieves user account and quota information
func (c *Client) GetUserInfo(ctx context.Context) (*UserInfo, error) {
	var info UserInfo
	if err := c.doGet(ctx, "/api/v2/user/info", &info); err != nil {
		return nil, fmt.Errorf("get user info: %w", err)
	}
	return &info, nil
}

// GetQuotaSummary retrieves a lightweight quota summary
func (c *Client) GetQuotaSummary(ctx context.Context) (*QuotaSummary, error) {
	info, err := c.GetUserInfo(ctx)
	if err != nil {
		return nil, err
	}
	return &QuotaSummary{
		Quota:          info.Quota,
		UsedQuota:      info.UsedQuota,
		RemainingQuota: info.RemainingQuota,
		DailyQuota:     info.DailyQuota,
		DailyUsed:      info.DailyUsed,
		Username:       info.Username,
	}, nil
}

// GetTopUpInfo retrieves available top-up methods and options
func (c *Client) GetTopUpInfo(ctx context.Context) (*TopUpInfo, error) {
	var info TopUpInfo
	if err := c.doGet(ctx, "/api/v2/user/topup/info", &info); err != nil {
		return nil, fmt.Errorf("get top-up info: %w", err)
	}
	return &info, nil
}

// GetPlans retrieves available subscription plans
func (c *Client) GetPlans(ctx context.Context) ([]SubscriptionPlan, error) {
	var plans []SubscriptionPlan
	if err := c.doGet(ctx, "/api/v2/subscription/plans", &plans); err != nil {
		return nil, fmt.Errorf("get plans: %w", err)
	}
	return plans, nil
}

// GetSubscriptions retrieves the user's current subscriptions
func (c *Client) GetSubscriptions(ctx context.Context) ([]SubscriptionInfo, error) {
	var subs []SubscriptionInfo
	if err := c.doGet(ctx, "/api/v2/subscription/list", &subs); err != nil {
		return nil, fmt.Errorf("get subscriptions: %w", err)
	}
	return subs, nil
}

// CreateTopUp creates a top-up payment request
func (c *Client) CreateTopUp(ctx context.Context, amount int64, method string) (*PaymentResult, error) {
	reqBody := struct {
		Amount        int64  `json:"amount"`
		PaymentMethod string `json:"payment_method"`
	}{Amount: amount, PaymentMethod: method}
	var result PaymentResult
	if err := c.doPost(ctx, "/api/v2/user/topup", reqBody, &result); err != nil {
		return nil, fmt.Errorf("create top-up: %w", err)
	}
	return &result, nil
}

// Subscribe creates a subscription request
func (c *Client) Subscribe(ctx context.Context, planCode, method string) (*PaymentResult, error) {
	reqBody := struct {
		PlanCode      string `json:"plan_code"`
		PaymentMethod string `json:"payment_method"`
	}{PlanCode: planCode, PaymentMethod: method}
	var result PaymentResult
	if err := c.doPost(ctx, "/api/v2/subscription/subscribe", reqBody, &result); err != nil {
		return nil, fmt.Errorf("subscribe: %w", err)
	}
	return &result, nil
}

// CancelSubscription cancels an active subscription
func (c *Client) CancelSubscription(ctx context.Context, id int) error {
	reqBody := struct {
		ID int `json:"id"`
	}{ID: id}
	if err := c.doPost(ctx, "/api/v2/subscription/cancel", reqBody, nil); err != nil {
		return fmt.Errorf("cancel subscription: %w", err)
	}
	return nil
}

// RedeemCode redeems a top-up code and returns the credited amount
func (c *Client) RedeemCode(ctx context.Context, code string) (int64, error) {
	reqBody := struct {
		Code string `json:"code"`
	}{Code: code}
	var result struct {
		Amount int64 `json:"amount"`
	}
	if err := c.doPost(ctx, "/api/v2/user/redeem", reqBody, &result); err != nil {
		return 0, fmt.Errorf("redeem code: %w", err)
	}
	return result.Amount, nil
}

// ConfigPreset is a cloud-hosted configuration template for an AI CLI tool.
type ConfigPreset struct {
	ID          string                 `json:"id"`
	Tool        string                 `json:"tool"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Category    string                 `json:"category"`
	ConfigJSON  map[string]interface{} `json:"config_json"`
	IsOfficial  bool                   `json:"is_official"`
}

// FetchPresets calls GET <baseURL>/api/v2/switch/presets?tool=<tool> and returns the preset list.
// No authentication header is required — presets are publicly readable.
func (c *Client) FetchPresets(ctx context.Context, tool string) ([]ConfigPreset, error) {
	path := "/api/v2/switch/presets"
	if tool != "" {
		path += "?tool=" + tool
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseSize))
	if err != nil {
		return nil, fmt.Errorf("read response body: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("api error: HTTP %d", resp.StatusCode)
	}

	var envelope struct {
		Success bool           `json:"success"`
		Data    []ConfigPreset `json:"data"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	if !envelope.Success {
		return nil, fmt.Errorf("api returned unsuccessful response")
	}
	return envelope.Data, nil
}

// GetAffiliateStats retrieves referral/affiliate statistics from the newapi gateway.
// Returns zeroed stats on any error — callers should treat this as non-critical.
func (c *Client) GetAffiliateStats(ctx context.Context) (*AffiliateStats, error) {
	var stats AffiliateStats
	if err := c.doGet(ctx, "/api/v2/user/aff", &stats); err != nil {
		return nil, fmt.Errorf("get affiliate stats: %w", err)
	}
	return &stats, nil
}

// GetIdentityOverview retrieves the aggregated identity overview (VIP, wallet, subscription)
// from lurus-api GET /api/v2/user/identity-overview, which proxies lurus-identity.
// The endpoint returns a direct JSON object (not wrapped in the standard API envelope).
func (c *Client) GetIdentityOverview(ctx context.Context, productID string) (*IdentityOverview, error) {
	path := "/api/v2/user/identity-overview"
	if productID != "" {
		path += "?product_id=" + productID
	}
	var ov IdentityOverview
	if err := c.doGetRaw(ctx, path, &ov); err != nil {
		return nil, fmt.Errorf("get identity overview: %w", err)
	}
	return &ov, nil
}

// doGet performs a GET request and decodes the response data
func (c *Client) doGet(ctx context.Context, path string, target interface{}) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	return c.doRequest(req, target)
}

// doPost performs a POST request with a JSON body and decodes the response data
func (c *Client) doPost(ctx context.Context, path string, body interface{}, target interface{}) error {
	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshal request body: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bytes.NewReader(bodyBytes))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	return c.doRequest(req, target)
}

// doGetRaw performs a GET request and decodes the response directly (no envelope check).
// Used for endpoints that return a plain JSON object, not the standard API envelope.
func (c *Client) doGetRaw(ctx context.Context, path string, target interface{}) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	if c.tenantSlug != "" {
		req.Header.Set("X-Tenant-Slug", c.tenantSlug)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("http request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseSize))
	if err != nil {
		return fmt.Errorf("read response body: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("api error: HTTP %d", resp.StatusCode)
	}

	if target != nil {
		if err := json.Unmarshal(respBody, target); err != nil {
			return fmt.Errorf("decode response: %w", err)
		}
	}
	return nil
}

// doRequest executes the HTTP request, checks the API envelope, and decodes data
func (c *Client) doRequest(req *http.Request, target interface{}) error {
	req.Header.Set("Authorization", "Bearer "+c.token)
	if c.tenantSlug != "" {
		req.Header.Set("X-Tenant-Slug", c.tenantSlug)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("http request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseSize))
	if err != nil {
		return fmt.Errorf("read response body: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		// Try to parse error message from API
		var apiErr apiResponse
		if json.Unmarshal(respBody, &apiErr) == nil && apiErr.Message != "" {
			return fmt.Errorf("api error (HTTP %d): %s", resp.StatusCode, apiErr.Message)
		}
		return fmt.Errorf("api error: HTTP %d", resp.StatusCode)
	}

	var envelope apiResponse
	if err := json.Unmarshal(respBody, &envelope); err != nil {
		return fmt.Errorf("decode response envelope: %w", err)
	}

	if !envelope.Success {
		return fmt.Errorf("api returned error: %s", envelope.Message)
	}

	if target != nil && len(envelope.Data) > 0 {
		if err := json.Unmarshal(envelope.Data, target); err != nil {
			return fmt.Errorf("decode response data: %w", err)
		}
	}

	return nil
}
