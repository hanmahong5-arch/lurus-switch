package admin

import (
	"context"
	"fmt"
	"net/http"
)

// ListTokens returns paginated tokens belonging to the authenticated user
// (admin sees all). Hub's /api/token/ accepts the standard PageInfo query.
func (c *Client) ListTokens(ctx context.Context, opts *ListOpts) (*TokenPage, error) {
	var out TokenPage
	if err := c.do(ctx, http.MethodGet, "/api/token/", opts.pageQuery(), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// SearchTokens matches name + key prefix.
func (c *Client) SearchTokens(ctx context.Context, keyword string) ([]Token, error) {
	var out []Token
	if err := c.do(ctx, http.MethodGet, "/api/token/search", newQuery("keyword", keyword), nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// GetToken returns a single token by ID.
func (c *Client) GetToken(ctx context.Context, id int) (*Token, error) {
	var out Token
	if err := c.do(ctx, http.MethodGet, fmt.Sprintf("/api/token/%d", id), nil, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// AddToken creates a new token. Required fields per Hub: name, expired_time
// (-1 for never), remain_quota, unlimited_quota, model_limits.
func (c *Client) AddToken(ctx context.Context, input CreateTokenInput) error {
	return c.do(ctx, http.MethodPost, "/api/token/", nil, input, nil)
}

// UpdateToken applies partial changes; Hub merges.
func (c *Client) UpdateToken(ctx context.Context, token map[string]any) error {
	return c.do(ctx, http.MethodPut, "/api/token/", nil, token, nil)
}

// DeleteToken removes a single token.
func (c *Client) DeleteToken(ctx context.Context, id int) error {
	return c.do(ctx, http.MethodDelete, fmt.Sprintf("/api/token/%d", id), nil, nil, nil)
}

// DeleteTokenBatch removes multiple tokens in one call.
func (c *Client) DeleteTokenBatch(ctx context.Context, ids []int) error {
	body := map[string]any{"ids": ids}
	return c.do(ctx, http.MethodPost, "/api/token/batch", nil, body, nil)
}
