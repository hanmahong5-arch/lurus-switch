package admin

import (
	"context"
	"fmt"
	"net/http"
)

// ListRedemptions returns paginated redemption codes for the Reseller's
// tenant. Filter by status via opts.Extra["status"]: 1 unused, 2 used,
// 3 disabled, -1 all.
func (c *Client) ListRedemptions(ctx context.Context, opts *ListOpts) (*RedemptionPage, error) {
	var out RedemptionPage
	if err := c.do(ctx, http.MethodGet, "/api/redemption/", opts.pageQuery(), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// SearchRedemptions matches name + key prefix.
func (c *Client) SearchRedemptions(ctx context.Context, keyword string) ([]Redemption, error) {
	var out []Redemption
	if err := c.do(ctx, http.MethodGet, "/api/redemption/search", newQuery("keyword", keyword), nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// GetRedemption returns a single code by ID.
func (c *Client) GetRedemption(ctx context.Context, id int) (*Redemption, error) {
	var out Redemption
	if err := c.do(ctx, http.MethodGet, fmt.Sprintf("/api/redemption/%d", id), nil, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// CreateRedemptions issues a batch of codes. Returns the generated codes so
// the UI (CSV export) can hand them off without a follow-up list call.
func (c *Client) CreateRedemptions(ctx context.Context, input CreateRedemptionInput) ([]Redemption, error) {
	if input.Count <= 0 {
		input.Count = 1
	}
	var out []Redemption
	if err := c.do(ctx, http.MethodPost, "/api/redemption/", nil, input, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// UpdateRedemption changes name / status / quota of an existing code.
func (c *Client) UpdateRedemption(ctx context.Context, redemption map[string]any) error {
	return c.do(ctx, http.MethodPut, "/api/redemption/", nil, redemption, nil)
}

// DeleteRedemption removes a single code.
func (c *Client) DeleteRedemption(ctx context.Context, id int) error {
	return c.do(ctx, http.MethodDelete, fmt.Sprintf("/api/redemption/%d", id), nil, nil, nil)
}

// DeleteInvalidRedemptions removes all expired / used codes (Hub server-side
// determines what "invalid" means).
func (c *Client) DeleteInvalidRedemptions(ctx context.Context) error {
	return c.do(ctx, http.MethodDelete, "/api/redemption/invalid", nil, nil, nil)
}

// Redemption status integer values surfaced for UI rendering.
const (
	RedemptionStatusUnused   = 1
	RedemptionStatusUsed     = 2
	RedemptionStatusDisabled = 3
)
