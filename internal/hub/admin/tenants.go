package admin

import (
	"context"
	"fmt"
	"net/http"
)

// V2 multi-tenant management (RootJWTAuth). These calls require a Lurus
// Platform admin session, not a tenant-scoped token — they fail with
// HubError{HTTPStatus: 401} when authenticated only as a tenant admin.

// ListTenants returns all tenants (platform admin scope).
func (c *Client) ListTenants(ctx context.Context) (*TenantList, error) {
	var out TenantList
	if err := c.do(ctx, http.MethodGet, "/api/v2/admin/tenants", nil, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// GetTenant returns a single tenant by ID.
func (c *Client) GetTenant(ctx context.Context, id string) (*Tenant, error) {
	var out Tenant
	if err := c.do(ctx, http.MethodGet, fmt.Sprintf("/api/v2/admin/tenants/%s", id), nil, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// CreateTenant provisions a new tenant. Slug must be URL-safe (Hub validates).
type CreateTenantInput struct {
	Slug string `json:"slug"`
	Name string `json:"name"`
}

// CreateTenant calls the Hub admin API. Returns the created Tenant.
func (c *Client) CreateTenant(ctx context.Context, input CreateTenantInput) (*Tenant, error) {
	var out Tenant
	if err := c.do(ctx, http.MethodPost, "/api/v2/admin/tenants", nil, input, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// EnableTenant lifts an admin-imposed lock.
func (c *Client) EnableTenant(ctx context.Context, id string) error {
	return c.do(ctx, http.MethodPost, fmt.Sprintf("/api/v2/admin/tenants/%s/enable", id), nil, nil, nil)
}

// DisableTenant blocks all activity for the tenant.
func (c *Client) DisableTenant(ctx context.Context, id string) error {
	return c.do(ctx, http.MethodPost, fmt.Sprintf("/api/v2/admin/tenants/%s/disable", id), nil, nil, nil)
}

// SuspendTenant freezes the tenant pending review (distinct from disable —
// data is preserved but reads return 403 to tenant users).
func (c *Client) SuspendTenant(ctx context.Context, id string) error {
	return c.do(ctx, http.MethodPost, fmt.Sprintf("/api/v2/admin/tenants/%s/suspend", id), nil, nil, nil)
}

// TenantStats — opaque map (Hub returns a record-shaped JSON tailored to
// the dashboard; UI consumes verbatim).
type TenantStats map[string]any

// GetTenantStats fetches usage statistics for a specific tenant.
func (c *Client) GetTenantStats(ctx context.Context, id string) (TenantStats, error) {
	var out TenantStats
	if err := c.do(ctx, http.MethodGet, fmt.Sprintf("/api/v2/admin/tenants/%s/stats", id), nil, nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// SwitchPreset is one of the provider presets surfaced via Hub's public
// /api/v2/switch/presets endpoint. Switch's ProviderPicker consumes these.
type SwitchPreset struct {
	ID          string         `json:"id"`
	Name        string         `json:"name"`
	Provider    string         `json:"provider"`
	Description string         `json:"description"`
	Logo        string         `json:"logo"`
	Config      map[string]any `json:"config"`
	IsOfficial  bool           `json:"is_official"`
}

// ListSwitchPresets fetches the public preset catalog (no auth required).
func (c *Client) ListSwitchPresets(ctx context.Context) ([]SwitchPreset, error) {
	var out []SwitchPreset
	if err := c.do(ctx, http.MethodGet, "/api/v2/switch/presets", nil, nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}
