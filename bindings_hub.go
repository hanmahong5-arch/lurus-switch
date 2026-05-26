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

// hubClientFactory is the seam tests use to inject a fake admin client.
// Production paths use defaultHubClient which reads from disk.
var hubClientFactory = defaultHubClient

// hubClient is the production accessor — calls through the factory so
// tests can swap it. Both bindings_hub.go and app_audit_undo.go go
// through here.
func hubClient() (*admin.Client, error) {
	return hubClientFactory()
}

// defaultHubClient builds an admin.Client from saved Reseller settings.
// Returns a caller-friendly error when the URL or token are missing —
// frontend can surface "请先在设置中配置 Hub" to the user.
func defaultHubClient() (*admin.Client, error) {
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
func (a *App) HubAddChannel(input map[string]any) (err error) {
	if err = a.requireAndAudit(capChannelWrite(), auditOpChannelCreate, "", input); err != nil {
		return err
	}
	defer func() { a.recordOutcome(auditOpChannelCreate, "", input, err) }()
	c, err := hubClient()
	if err != nil {
		return err
	}
	return c.AddChannel(a.hubCtx(), admin.CreateChannelInput(input))
}

// HubUpdateChannel applies partial changes. Captures the prior channel
// state into the audit journal's Before field so the Undo handler can
// restore it.
func (a *App) HubUpdateChannel(input map[string]any) (err error) {
	target := stringField(input, "id")
	if err = a.requireAndAudit(capChannelWrite(), auditOpChannelUpdate, target, input); err != nil {
		return err
	}
	c, cerr := hubClient()
	if cerr != nil {
		a.recordOutcome(auditOpChannelUpdate, target, input, cerr)
		return cerr
	}
	// Best-effort prior-state capture. A failed fetch leaves Before nil
	// — the entry stays Reversible=true (the op IS reversible if a
	// future Before is found), but Undo at runtime will refuse cleanly
	// with a "missing Before snapshot" error.
	var before any
	if id := intField(input, "id"); id > 0 {
		if ch, gerr := c.GetChannel(a.hubCtx(), id); gerr == nil {
			before = ch
		}
	}
	defer func() { a.recordOutcomeFull(auditOpChannelUpdate, target, before, input, err) }()
	return c.UpdateChannel(a.hubCtx(), input)
}

// HubDeleteChannel removes a single channel. Captures the prior
// channel state so Undo can re-create it.
func (a *App) HubDeleteChannel(id int) (err error) {
	target := fmtIntID(id)
	if err = a.requireAndAudit(capChannelWrite(), auditOpChannelDelete, target, map[string]any{"id": id}); err != nil {
		return err
	}
	c, cerr := hubClient()
	if cerr != nil {
		a.recordOutcome(auditOpChannelDelete, target, map[string]any{"id": id}, cerr)
		return cerr
	}
	var before any
	if ch, gerr := c.GetChannel(a.hubCtx(), id); gerr == nil {
		before = ch
	}
	defer func() {
		a.recordOutcomeFull(auditOpChannelDelete, target, before, map[string]any{"id": id}, err)
	}()
	return c.DeleteChannel(a.hubCtx(), id)
}

// HubDeleteChannelBatch removes a list of channels. Snapshots all
// affected rows for batch undo.
func (a *App) HubDeleteChannelBatch(ids []int) (err error) {
	if err = a.requireAndAudit(capChannelWrite(), auditOpChannelDeleteBatch, "", map[string]any{"ids": ids}); err != nil {
		return err
	}
	c, cerr := hubClient()
	if cerr != nil {
		a.recordOutcome(auditOpChannelDeleteBatch, "", map[string]any{"ids": ids}, cerr)
		return cerr
	}
	var before []*admin.Channel
	for _, id := range ids {
		if ch, gerr := c.GetChannel(a.hubCtx(), id); gerr == nil {
			before = append(before, ch)
		}
	}
	defer func() {
		a.recordOutcomeFull(auditOpChannelDeleteBatch, "", before, map[string]any{"ids": ids}, err)
	}()
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

// ── Channel batch / tag ops (Wave 5 W5.3) ─────────────────────────────────

// HubBatchSetChannelTag tags multiple channels in one Hub call.
func (a *App) HubBatchSetChannelTag(ids []int, tag string) error {
	c, err := hubClient()
	if err != nil {
		return err
	}
	return c.BatchSetChannelTag(a.hubCtx(), ids, tag)
}

// HubEnableChannelsByTag enables every channel carrying tag.
func (a *App) HubEnableChannelsByTag(tag string) error {
	c, err := hubClient()
	if err != nil {
		return err
	}
	return c.EnableChannelsByTag(a.hubCtx(), tag)
}

// HubDisableChannelsByTag disables every channel carrying tag.
func (a *App) HubDisableChannelsByTag(tag string) error {
	c, err := hubClient()
	if err != nil {
		return err
	}
	return c.DisableChannelsByTag(a.hubCtx(), tag)
}

// HubEditChannelTag renames a tag everywhere it appears.
func (a *App) HubEditChannelTag(oldTag, newTag string) error {
	c, err := hubClient()
	if err != nil {
		return err
	}
	return c.EditChannelTag(a.hubCtx(), oldTag, newTag)
}

// HubFetchChannelModels asks Hub to pull upstream's model catalogue.
func (a *App) HubFetchChannelModels(id int) ([]string, error) {
	c, err := hubClient()
	if err != nil {
		return nil, err
	}
	return c.FetchChannelModels(a.hubCtx(), id)
}

// HubFixChannelAbilities reconciles channel abilities rows server-side.
func (a *App) HubFixChannelAbilities() error {
	c, err := hubClient()
	if err != nil {
		return err
	}
	return c.FixChannelAbilities(a.hubCtx())
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
func (a *App) HubAddToken(input map[string]any) (err error) {
	if err = a.requireAndAudit(capTokenCreate(), auditOpTokenCreate, "", input); err != nil {
		return err
	}
	defer func() { a.recordOutcome(auditOpTokenCreate, "", input, err) }()
	c, err := hubClient()
	if err != nil {
		return err
	}
	return c.AddToken(a.hubCtx(), admin.CreateTokenInput(input))
}

// HubUpdateToken applies partial changes. Captures prior token state
// so Undo can revert.
func (a *App) HubUpdateToken(input map[string]any) (err error) {
	target := stringField(input, "id")
	if err = a.requireAndAudit(capTokenCreate(), auditOpTokenUpdate, target, input); err != nil {
		return err
	}
	c, cerr := hubClient()
	if cerr != nil {
		a.recordOutcome(auditOpTokenUpdate, target, input, cerr)
		return cerr
	}
	var before any
	if id := intField(input, "id"); id > 0 {
		if tk, gerr := c.GetToken(a.hubCtx(), id); gerr == nil {
			before = tk
		}
	}
	defer func() { a.recordOutcomeFull(auditOpTokenUpdate, target, before, input, err) }()
	return c.UpdateToken(a.hubCtx(), input)
}

// HubDeleteToken removes a single token. Captures prior state for
// Undo (note: re-created tokens get a new key — Hub regenerates
// secrets to avoid resurrecting compromised credentials).
func (a *App) HubDeleteToken(id int) (err error) {
	target := fmtIntID(id)
	if err = a.requireAndAudit(capTokenRevoke(), auditOpTokenDelete, target, map[string]any{"id": id}); err != nil {
		return err
	}
	c, cerr := hubClient()
	if cerr != nil {
		a.recordOutcome(auditOpTokenDelete, target, map[string]any{"id": id}, cerr)
		return cerr
	}
	var before any
	if tk, gerr := c.GetToken(a.hubCtx(), id); gerr == nil {
		before = tk
	}
	defer func() {
		a.recordOutcomeFull(auditOpTokenDelete, target, before, map[string]any{"id": id}, err)
	}()
	return c.DeleteToken(a.hubCtx(), id)
}

// HubDeleteTokenBatch removes a list of tokens. Snapshots affected
// rows for batch undo.
func (a *App) HubDeleteTokenBatch(ids []int) (err error) {
	if err = a.requireAndAudit(capTokenRevoke(), auditOpTokenDeleteBatch, "", map[string]any{"ids": ids}); err != nil {
		return err
	}
	c, cerr := hubClient()
	if cerr != nil {
		a.recordOutcome(auditOpTokenDeleteBatch, "", map[string]any{"ids": ids}, cerr)
		return cerr
	}
	var before []*admin.Token
	for _, id := range ids {
		if tk, gerr := c.GetToken(a.hubCtx(), id); gerr == nil {
			before = append(before, tk)
		}
	}
	defer func() {
		a.recordOutcomeFull(auditOpTokenDeleteBatch, "", before, map[string]any{"ids": ids}, err)
	}()
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

// HubDeleteRedemption removes a single code. Captures the prior
// redemption struct so Undo can re-issue an equivalent code.
func (a *App) HubDeleteRedemption(id int) (err error) {
	target := fmtIntID(id)
	if err = a.requireAndAudit(capRedemptionDelete(), auditOpRedemptionDelete, target, map[string]any{"id": id}); err != nil {
		return err
	}
	c, cerr := hubClient()
	if cerr != nil {
		a.recordOutcome(auditOpRedemptionDelete, target, map[string]any{"id": id}, cerr)
		return cerr
	}
	var before any
	if r, gerr := c.GetRedemption(a.hubCtx(), id); gerr == nil {
		before = r
	}
	defer func() {
		a.recordOutcomeFull(auditOpRedemptionDelete, target, before, map[string]any{"id": id}, err)
	}()
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

// ── Wallet (Reseller financial visibility, Wave 5 W5.1) ───────────────────

// HubGetWalletInfo returns the Hub-side wallet snapshot used by the
// Reseller Wallet page top KPI cards. Source = "platform" when backed by
// lurus-platform, "internal" when the account isn't bridged (frontend
// surfaces a banner in that case).
func (a *App) HubGetWalletInfo() (*admin.WalletInfo, error) {
	c, err := hubClient()
	if err != nil {
		return nil, err
	}
	return c.GetWalletInfo(a.hubCtx())
}

// HubListWalletTransactions returns paginated wallet transactions for the
// admin's bound platform account. q.Page defaults to 1, q.PageSize to 20
// (clamped to 200 max on the Hub side).
func (a *App) HubListWalletTransactions(q admin.WalletQuery) (*admin.WalletTransactionPage, error) {
	c, err := hubClient()
	if err != nil {
		return nil, err
	}
	return c.ListWalletTransactions(a.hubCtx(), q)
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

