package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/lurus-ai/subscription-service/internal/biz"
)

// NewAPIClient implements biz.NewAPIClient for new-api integration
type NewAPIClient struct {
	baseURL    string
	adminToken string
	httpClient *http.Client
}

// NewNewAPIClient creates a new new-api client
func NewNewAPIClient(baseURL, adminToken string) *NewAPIClient {
	return &NewAPIClient{
		baseURL:    baseURL,
		adminToken: adminToken,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// doRequest performs an HTTP request to new-api
func (c *NewAPIClient) doRequest(ctx context.Context, method, path string, body interface{}) (*http.Response, error) {
	var bodyReader io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		bodyReader = bytes.NewReader(jsonBody)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, bodyReader)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.adminToken)

	return c.httpClient.Do(req)
}

// GetUser gets a user from new-api
func (c *NewAPIClient) GetUser(ctx context.Context, userID int) (*biz.NewAPIUser, error) {
	resp, err := c.doRequest(ctx, http.MethodGet, fmt.Sprintf("/api/user/%d", userID), nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("new-api returned status %d", resp.StatusCode)
	}

	var result struct {
		Success bool           `json:"success"`
		Data    biz.NewAPIUser `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	if !result.Success {
		return nil, fmt.Errorf("new-api request failed")
	}

	return &result.Data, nil
}

// UpdateUserQuota updates a user's quota in new-api
func (c *NewAPIClient) UpdateUserQuota(ctx context.Context, userID int, quota int64) error {
	body := map[string]interface{}{
		"quota": quota,
	}

	resp, err := c.doRequest(ctx, http.MethodPut, fmt.Sprintf("/api/user/%d", userID), body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("new-api returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	return nil
}

// UpdateUserGroup updates a user's group in new-api
func (c *NewAPIClient) UpdateUserGroup(ctx context.Context, userID int, group string) error {
	body := map[string]interface{}{
		"group": group,
	}

	resp, err := c.doRequest(ctx, http.MethodPut, fmt.Sprintf("/api/user/%d", userID), body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("new-api returned status %d", resp.StatusCode)
	}

	return nil
}

// CreateToken creates a new token for a user in new-api
func (c *NewAPIClient) CreateToken(ctx context.Context, userID int, name string, quota int64, expiredTime int64) (*biz.NewAPIToken, error) {
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
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("new-api returned status %d", resp.StatusCode)
	}

	var result struct {
		Success bool            `json:"success"`
		Data    biz.NewAPIToken `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return &result.Data, nil
}

// AddUserQuota adds quota to a user (for renewal)
func (c *NewAPIClient) AddUserQuota(ctx context.Context, userID int, quota int64) error {
	// First get current quota
	user, err := c.GetUser(ctx, userID)
	if err != nil {
		return err
	}

	// Add to existing quota
	return c.UpdateUserQuota(ctx, userID, user.Quota+quota)
}

// ResetUserQuota resets user quota to a specific value (for monthly reset)
func (c *NewAPIClient) ResetUserQuota(ctx context.Context, userID int, quota int64) error {
	return c.UpdateUserQuota(ctx, userID, quota)
}

// UpdateUserSubscriptionConfig updates user subscription config in new-api
// This is the primary method for subscription-service to sync subscription state to new-api
func (c *NewAPIClient) UpdateUserSubscriptionConfig(ctx context.Context, userID int, config *biz.SubscriptionConfig) error {
	resp, err := c.doRequest(ctx, http.MethodPut, fmt.Sprintf("/api/user/%d/subscription", userID), config)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("new-api returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	return nil
}

// GetUserDailyQuotaStatus gets user daily quota status from new-api
func (c *NewAPIClient) GetUserDailyQuotaStatus(ctx context.Context, userID int) (*biz.DailyQuotaStatus, error) {
	resp, err := c.doRequest(ctx, http.MethodGet, fmt.Sprintf("/api/user/%d/daily-quota", userID), nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("new-api returned status %d", resp.StatusCode)
	}

	var result struct {
		Success bool                  `json:"success"`
		Data    biz.DailyQuotaStatus `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	if !result.Success {
		return nil, fmt.Errorf("new-api request failed")
	}

	return &result.Data, nil
}

// ResetUserDailyQuota resets user daily quota in new-api
func (c *NewAPIClient) ResetUserDailyQuota(ctx context.Context, userID int) error {
	resp, err := c.doRequest(ctx, http.MethodPost, fmt.Sprintf("/api/user/%d/daily-quota/reset", userID), nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("new-api returned status %d", resp.StatusCode)
	}

	return nil
}
