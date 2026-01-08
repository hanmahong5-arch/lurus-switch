// Package newapi provides a client for interacting with the new-api service.
package newapi

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

// Client is the new-api HTTP client.
type Client struct {
	baseURL    string
	adminToken string
	httpClient *http.Client
}

// NewClient creates a new new-api client.
func NewClient(baseURL, adminToken string) *Client {
	return &Client{
		baseURL:    baseURL,
		adminToken: adminToken,
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

// doRequest performs an HTTP request to new-api.
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
	req.Header.Set("Authorization", "Bearer "+c.adminToken)

	return c.httpClient.Do(req)
}

// Response wraps a new-api response.
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
		return fmt.Errorf("new-api returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var r Response
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}

	if !r.Success {
		return fmt.Errorf("new-api request failed: %s", r.Message)
	}

	if target != nil && len(r.Data) > 0 {
		if err := json.Unmarshal(r.Data, target); err != nil {
			return fmt.Errorf("unmarshal data: %w", err)
		}
	}

	return nil
}

// GetUser gets a user by ID.
func (c *Client) GetUser(ctx context.Context, userID int) (*types.User, error) {
	resp, err := c.doRequest(ctx, http.MethodGet, fmt.Sprintf("/api/user/%d", userID), nil)
	if err != nil {
		return nil, err
	}

	var user types.User
	if err := parseResponse(resp, &user); err != nil {
		return nil, err
	}

	return &user, nil
}

// UpdateUserQuota updates a user's quota.
func (c *Client) UpdateUserQuota(ctx context.Context, userID int, quota int64) error {
	body := map[string]interface{}{
		"quota": quota,
	}

	resp, err := c.doRequest(ctx, http.MethodPut, fmt.Sprintf("/api/user/%d", userID), body)
	if err != nil {
		return err
	}

	return parseResponse(resp, nil)
}

// UpdateUserGroup updates a user's group.
func (c *Client) UpdateUserGroup(ctx context.Context, userID int, group string) error {
	body := map[string]interface{}{
		"group": group,
	}

	resp, err := c.doRequest(ctx, http.MethodPut, fmt.Sprintf("/api/user/%d", userID), body)
	if err != nil {
		return err
	}

	return parseResponse(resp, nil)
}

// UpdateUserSubscriptionConfig updates user subscription configuration.
// This is the primary method for syncing subscription state to new-api.
func (c *Client) UpdateUserSubscriptionConfig(ctx context.Context, userID int, config *types.SubscriptionConfig) error {
	resp, err := c.doRequest(ctx, http.MethodPut, fmt.Sprintf("/api/user/%d/subscription", userID), config)
	if err != nil {
		return err
	}

	return parseResponse(resp, nil)
}

// GetUserDailyQuotaStatus gets user daily quota status.
func (c *Client) GetUserDailyQuotaStatus(ctx context.Context, userID int) (*types.DailyQuotaInfo, error) {
	resp, err := c.doRequest(ctx, http.MethodGet, fmt.Sprintf("/api/user/%d/daily-quota", userID), nil)
	if err != nil {
		return nil, err
	}

	var info types.DailyQuotaInfo
	if err := parseResponse(resp, &info); err != nil {
		return nil, err
	}

	return &info, nil
}

// ResetUserDailyQuota resets user daily quota.
func (c *Client) ResetUserDailyQuota(ctx context.Context, userID int) error {
	resp, err := c.doRequest(ctx, http.MethodPost, fmt.Sprintf("/api/user/%d/daily-quota/reset", userID), nil)
	if err != nil {
		return err
	}

	return parseResponse(resp, nil)
}

// CreateToken creates a new API token for a user.
func (c *Client) CreateToken(ctx context.Context, userID int, name string, quota int64, expiredTime int64) (*types.Token, error) {
	body := map[string]interface{}{
		"user_id":      userID,
		"name":         name,
		"remain_quota": quota,
		"expired_time": expiredTime,
	}

	resp, err := c.doRequest(ctx, http.MethodPost, "/api/token/", body)
	if err != nil {
		return nil, err
	}

	var token types.Token
	if err := parseResponse(resp, &token); err != nil {
		return nil, err
	}

	return &token, nil
}
