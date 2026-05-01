package main

import (
	"context"
	"errors"
	"fmt"

	"lurus-switch/internal/appconfig"
	"lurus-switch/internal/hub/admin"
)

// ============================
// Hub Admin Bindings (S-Xa.5) — Reseller-mode console talks to lurus-newhub.
// ============================
//
// All methods here build a fresh admin.Client from the current AppSettings
// each call. That keeps the surface stateless (no manager struct on App),
// and means changing HubURL / AdminToken in Settings takes effect on the
// next request. The cost is one Client allocation per call — negligible for
// admin operations that already round-trip the network.

// hubClient builds an admin.Client from saved Reseller settings. Returns a
// caller-friendly error when the URL or token are missing — frontend can
// surface "请先在设置中配置 Hub" to the user.
func hubClient() (*admin.Client, error) {
	s, err := appconfig.LoadAppSettings()
	if err != nil {
		return nil, fmt.Errorf("load app settings: %w", err)
	}
	if s.Reseller.HubURL == "" {
		return nil, errors.New("Hub URL 未配置 — 请先在 Reseller 设置中填写")
	}
	if s.Reseller.AdminToken == "" {
		return nil, errors.New("Hub 管理员 Token 未配置")
	}
	return admin.New(admin.Config{
		BaseURL: s.Reseller.HubURL,
		Token:   s.Reseller.AdminToken,
	})
}

// hubCtx is the cancellation context for Wails-invoked hub calls. Wails
// doesn't pass a per-call ctx today; we use the App's ctx so app shutdown
// cancels in-flight requests cleanly.
func (a *App) hubCtx() context.Context {
	if a == nil || a.ctx == nil {
		return context.Background()
	}
	return a.ctx
}

// ── Channels ──────────────────────────────────────────────────────────────

// HubListChannels returns paginated channels. statusFilter accepts Hub's
// -1 / 0 / 1 string codes (or "" to use Hub default).
func (a *App) HubListChannels(page, pageSize int, statusFilter string) (*admin.ChannelPage, error) {
	c, err := hubClient()
	if err != nil {
		return nil, err
	}
	opts := admin.ListOpts{Page: page, PageSize: pageSize}
	if statusFilter != "" {
		opts.Extra = map[string]string{"status": statusFilter}
	}
	return c.ListChannels(a.hubCtx(), &opts)
}

// HubGetChannel returns a single channel.
func (a *App) HubGetChannel(id int) (*admin.Channel, error) {
	c, err := hubClient()
	if err != nil {
		return nil, err
	}
	return c.GetChannel(a.hubCtx(), id)
}

// HubSearchChannels runs Hub's name/tag/remark search. Pagination is not
// honored by the upstream search endpoint — Hub returns the full match set
// in one shot, so the binding signature drops page params.
func (a *App) HubSearchChannels(keyword string) ([]admin.Channel, error) {
	c, err := hubClient()
	if err != nil {
		return nil, err
	}
	return c.SearchChannels(a.hubCtx(), keyword)
}

// HubCopyChannel duplicates a channel server-side. Hub names the copy
// "<original> (copy)" — the frontend should reload the list after success.
func (a *App) HubCopyChannel(id int) error {
	c, err := hubClient()
	if err != nil {
		return err
	}
	return c.CopyChannel(a.hubCtx(), id)
}

// HubAddChannel creates a channel from a free-form payload (Reseller UI
// composes it from form fields; backend doesn't need to enforce shape since
// Hub validates).
func (a *App) HubAddChannel(input map[string]any) error {
	c, err := hubClient()
	if err != nil {
		return err
	}
	return c.AddChannel(a.hubCtx(), admin.CreateChannelInput(input))
}

// HubUpdateChannel applies partial changes.
func (a *App) HubUpdateChannel(input map[string]any) error {
	c, err := hubClient()
	if err != nil {
		return err
	}
	return c.UpdateChannel(a.hubCtx(), input)
}

// HubDeleteChannel removes a single channel.
func (a *App) HubDeleteChannel(id int) error {
	c, err := hubClient()
	if err != nil {
		return err
	}
	return c.DeleteChannel(a.hubCtx(), id)
}

// HubDeleteChannelBatch removes a list of channels.
func (a *App) HubDeleteChannelBatch(ids []int) error {
	c, err := hubClient()
	if err != nil {
		return err
	}
	return c.DeleteChannelBatch(a.hubCtx(), ids)
}

// HubTestChannel pings a channel via Hub's test endpoint.
func (a *App) HubTestChannel(id int, model string) error {
	c, err := hubClient()
	if err != nil {
		return err
	}
	return c.TestChannel(a.hubCtx(), id, model)
}

// ── Tokens ────────────────────────────────────────────────────────────────

// HubListTokens returns paginated tokens.
func (a *App) HubListTokens(page, pageSize int) (*admin.TokenPage, error) {
	c, err := hubClient()
	if err != nil {
		return nil, err
	}
	return c.ListTokens(a.hubCtx(), &admin.ListOpts{Page: page, PageSize: pageSize})
}

// HubAddToken creates a token.
func (a *App) HubAddToken(input map[string]any) error {
	c, err := hubClient()
	if err != nil {
		return err
	}
	return c.AddToken(a.hubCtx(), admin.CreateTokenInput(input))
}

// HubUpdateToken applies partial changes.
func (a *App) HubUpdateToken(input map[string]any) error {
	c, err := hubClient()
	if err != nil {
		return err
	}
	return c.UpdateToken(a.hubCtx(), input)
}

// HubDeleteToken removes a single token.
func (a *App) HubDeleteToken(id int) error {
	c, err := hubClient()
	if err != nil {
		return err
	}
	return c.DeleteToken(a.hubCtx(), id)
}

// HubDeleteTokenBatch removes a list of tokens.
func (a *App) HubDeleteTokenBatch(ids []int) error {
	c, err := hubClient()
	if err != nil {
		return err
	}
	return c.DeleteTokenBatch(a.hubCtx(), ids)
}

// ── Redemptions ───────────────────────────────────────────────────────────

// HubListRedemptions returns paginated activation codes.
func (a *App) HubListRedemptions(page, pageSize int) (*admin.RedemptionPage, error) {
	c, err := hubClient()
	if err != nil {
		return nil, err
	}
	return c.ListRedemptions(a.hubCtx(), &admin.ListOpts{Page: page, PageSize: pageSize})
}

// HubCreateRedemptions issues a batch of activation codes.
func (a *App) HubCreateRedemptions(name string, quota int64, count int, expiredTime int64) ([]admin.Redemption, error) {
	c, err := hubClient()
	if err != nil {
		return nil, err
	}
	return c.CreateRedemptions(a.hubCtx(), admin.CreateRedemptionInput{
		Name:        name,
		Quota:       quota,
		Count:       count,
		ExpiredTime: expiredTime,
	})
}

// HubDeleteRedemption removes a single code.
func (a *App) HubDeleteRedemption(id int) error {
	c, err := hubClient()
	if err != nil {
		return err
	}
	return c.DeleteRedemption(a.hubCtx(), id)
}

// HubDeleteInvalidRedemptions purges expired/used codes server-side.
func (a *App) HubDeleteInvalidRedemptions() error {
	c, err := hubClient()
	if err != nil {
		return err
	}
	return c.DeleteInvalidRedemptions(a.hubCtx())
}

// ── Dashboard data ────────────────────────────────────────────────────────

// HubGetDashboardSummary returns the top-level counters used by the
// Reseller dashboard cards (user/channel/token counts + today's totals).
func (a *App) HubGetDashboardSummary() (*admin.DashboardSummary, error) {
	c, err := hubClient()
	if err != nil {
		return nil, err
	}
	return c.GetDashboardSummary(a.hubCtx())
}

// HubGetQuotaDates returns the 14-day usage rollup for the chart. Dates
// are inclusive and formatted "YYYY-MM-DD" — caller passes whatever Hub
// expects.
func (a *App) HubGetQuotaDates(startDate, endDate string) ([]admin.QuotaDate, error) {
	c, err := hubClient()
	if err != nil {
		return nil, err
	}
	return c.GetQuotaDates(a.hubCtx(), startDate, endDate)
}

// HubGetPerformanceStats returns the Hub process runtime metrics
// (goroutines / memory / uptime / req-rate). Refreshes on demand.
func (a *App) HubGetPerformanceStats() (*admin.PerformanceStats, error) {
	c, err := hubClient()
	if err != nil {
		return nil, err
	}
	return c.GetPerformanceStats(a.hubCtx())
}

// ── Logs ──────────────────────────────────────────────────────────────────

// HubListLogs queries the log table.
func (a *App) HubListLogs(query admin.LogQuery) (*admin.LogPage, error) {
	c, err := hubClient()
	if err != nil {
		return nil, err
	}
	return c.ListLogs(a.hubCtx(), query)
}

// ── Tenants (V2, root role required) ──────────────────────────────────────

// HubListTenants returns all tenants. Only platform admins succeed —
// regular Reseller tokens get IsUnauthorized.
func (a *App) HubListTenants() (*admin.TenantList, error) {
	c, err := hubClient()
	if err != nil {
		return nil, err
	}
	return c.ListTenants(a.hubCtx())
}

// HubCreateTenant provisions a new tenant.
func (a *App) HubCreateTenant(slug, name string) (*admin.Tenant, error) {
	c, err := hubClient()
	if err != nil {
		return nil, err
	}
	return c.CreateTenant(a.hubCtx(), admin.CreateTenantInput{Slug: slug, Name: name})
}

// ── Switch presets (public, no auth) ──────────────────────────────────────

// HubListSwitchPresets pulls the public preset catalog. Works in any mode
// (no admin token required).
func (a *App) HubListSwitchPresets() ([]admin.SwitchPreset, error) {
	// Public endpoint — fall back to base URL only.
	s, err := appconfig.LoadAppSettings()
	if err != nil {
		return nil, fmt.Errorf("load app settings: %w", err)
	}
	hubURL := s.Reseller.HubURL
	if hubURL == "" {
		hubURL = s.LockedHubURL // EndUser white-label fallback
	}
	if hubURL == "" {
		return nil, errors.New("Hub URL 未配置")
	}
	c, err := admin.New(admin.Config{BaseURL: hubURL})
	if err != nil {
		return nil, err
	}
	return c.ListSwitchPresets(a.hubCtx())
}

