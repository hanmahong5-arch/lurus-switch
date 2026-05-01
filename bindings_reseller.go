package main

import (
	"context"
	"errors"
	"fmt"
	"time"

	"lurus-switch/internal/appconfig"
	"lurus-switch/internal/hub/admin"
	"lurus-switch/internal/hub/deploy"
)

// ============================
// Reseller Setup (S-Xb.1) — wizard backend.
// ============================
//
// The wizard frontend collects (kind, hub_url, admin_token, tenant_slug,
// display_name) and walks through three Wails entrypoints:
//
//   - ListResellerDeployKinds: enumerate Provider options for the picker.
//   - TestHubConnection:       smoke-test the URL/token before saving.
//   - ProvisionResellerHub:    orchestrate Provider.Provision +
//                              persistResellerConfig. Idempotent — re-running
//                              with the same inputs is a no-op.
//
// HasResellerConfig is read on every render to decide whether to gate the
// main UI behind the wizard.

// resellerKindEntry is the JSON payload Wails generates for the picker.
// Kept as a flat struct (not the deploy.Kind type) so the frontend doesn't
// have to import the Go enum.
type resellerKindEntry struct {
	Kind           string `json:"kind"`
	Implemented    bool   `json:"implemented"`
	LabelZH        string `json:"labelZh"`
	LabelEN        string `json:"labelEn"`
	DescriptionZH  string `json:"descriptionZh"`
	DescriptionEN  string `json:"descriptionEn"`
}

// resellerKindCatalog is the static label set rendered in the wizard
// picker. Real cloud-deploy adapters will flip Implemented to true once
// their integrations land. Centralized here so i18n + behavior stay in lock-step.
var resellerKindCatalog = []resellerKindEntry{
	{
		Kind:          string(deploy.KindManual),
		Implemented:   true,
		LabelZH:       "手动接入",
		LabelEN:       "Manual",
		DescriptionZH: "我已自行部署 lurus-newhub，只需告诉 Switch 它在哪里。",
		DescriptionEN: "I've already deployed lurus-newhub somewhere — just record the URL.",
	},
	{
		Kind:          string(deploy.KindSealos),
		Implemented:   false,
		LabelZH:       "Sealos 一键部署",
		LabelEN:       "Sealos (one-click)",
		DescriptionZH: "（即将上线）在 Sealos 集群拉起 newhub。当前请用「手动接入」。",
		DescriptionEN: "(Coming soon) Provision newhub on Sealos. For now use Manual.",
	},
	{
		Kind:          string(deploy.KindAliyun),
		Implemented:   false,
		LabelZH:       "阿里云 ECS",
		LabelEN:       "Aliyun ECS",
		DescriptionZH: "（即将上线）开 ECS + docker-compose 部署 newhub。",
		DescriptionEN: "(Coming soon) Spin up an ECS instance and bootstrap newhub.",
	},
}

// ListResellerDeployKinds returns the picker options for the wizard.
func (a *App) ListResellerDeployKinds() []resellerKindEntry {
	return resellerKindCatalog
}

// hubConnectionCheckTimeout is the budget for TestHubConnection's HTTP
// round-trip. Generous so a slow VPS doesn't false-fail, capped so a
// genuinely-down Hub doesn't hang the wizard for minutes.
const hubConnectionCheckTimeout = 8 * time.Second

// TestHubConnection issues a minimal admin call (ListChannels page=1
// pageSize=1) against the supplied URL+token. Returns a friendly
// surface-able message when auth fails or the URL is unreachable, so the
// wizard can render it inline without parsing.
func (a *App) TestHubConnection(hubURL, token string) (string, error) {
	if hubURL == "" {
		return "", errors.New("HubURL 必填")
	}
	if token == "" {
		return "", errors.New("管理员 Token 必填")
	}

	c, err := admin.New(admin.Config{
		BaseURL: hubURL,
		Token:   token,
		Timeout: hubConnectionCheckTimeout,
	})
	if err != nil {
		return "", fmt.Errorf("Hub URL 校验失败：%w", err)
	}

	ctx, cancel := context.WithTimeout(a.hubCtx(), hubConnectionCheckTimeout)
	defer cancel()

	page, err := c.ListChannels(ctx, &admin.ListOpts{Page: 1, PageSize: 1})
	if err != nil {
		if admin.IsUnauthorized(err) {
			return "", fmt.Errorf("Token 无权限或已过期：%w", err)
		}
		return "", fmt.Errorf("无法连接到 Hub：%w", err)
	}

	// Minimal success surface — return a human note the wizard can show.
	return fmt.Sprintf("连接成功 · 当前 Hub 共 %d 个 channel", page.Total), nil
}

// ProvisionResellerHub runs the chosen Provider and atomically persists
// the resulting Reseller config when it succeeds.
//
// Idempotency: when the saved config already matches the inputs, returns
// without redeploying. This means re-running the wizard with the same
// answers is safe — no double-provision risk.
func (a *App) ProvisionResellerHub(kindRaw, displayName, hubURL, adminToken, tenantSlug string) (*deploy.Result, error) {
	kind, err := deploy.ParseKind(kindRaw)
	if err != nil {
		return nil, err
	}

	provider, err := deploy.New(kind)
	if err != nil {
		return nil, err
	}

	// Short-circuit: same coordinates already saved → return existing.
	if existing, err := appconfig.LoadAppSettings(); err == nil {
		if existing.Reseller.HubURL == hubURL && existing.Reseller.AdminToken == adminToken {
			return &deploy.Result{
				Kind:        kind,
				HubURL:      existing.Reseller.HubURL,
				AdminToken:  existing.Reseller.AdminToken,
				TenantSlug:  existing.Reseller.TenantSlug,
				DisplayName: existing.Reseller.DisplayName,
				Notes:       "已是当前配置，跳过部署。",
			}, nil
		}
	}

	res, err := provider.Provision(a.hubCtx(), deploy.Inputs{
		Kind:        kind,
		DisplayName: displayName,
		Manual: deploy.ManualInputs{
			HubURL:     hubURL,
			AdminToken: adminToken,
			TenantSlug: tenantSlug,
		},
	})
	if err != nil {
		return nil, err
	}

	if err := persistResellerConfig(res); err != nil {
		return nil, fmt.Errorf("save reseller config: %w", err)
	}
	return res, nil
}

// HasResellerConfig reports whether a Hub URL + admin token are already
// saved. The frontend gate uses this to decide between wizard vs main UI.
func (a *App) HasResellerConfig() (bool, error) {
	s, err := appconfig.LoadAppSettings()
	if err != nil {
		return false, fmt.Errorf("load app settings: %w", err)
	}
	return s.Reseller.HubURL != "" && s.Reseller.AdminToken != "", nil
}

// ClearResellerConfig wipes saved Reseller coordinates. Used by the
// "重置配置" button in Settings — primarily for support/dev troubleshooting.
func (a *App) ClearResellerConfig() error {
	s, err := appconfig.LoadAppSettings()
	if err != nil {
		return fmt.Errorf("load app settings: %w", err)
	}
	s.Reseller = appconfig.ResellerConfig{}
	return appconfig.SaveAppSettings(s)
}

// persistResellerConfig is the single mutation point for the Reseller
// block. Keeps Save semantics consistent (mode normalization, lock check)
// in one place.
func persistResellerConfig(res *deploy.Result) error {
	if res == nil {
		return errors.New("nil deploy result")
	}
	s, err := appconfig.LoadAppSettings()
	if err != nil {
		return err
	}
	s.Reseller = appconfig.ResellerConfig{
		HubURL:      res.HubURL,
		AdminToken:  res.AdminToken,
		TenantSlug:  res.TenantSlug,
		DisplayName: res.DisplayName,
	}
	return appconfig.SaveAppSettings(s)
}
