package main

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"lurus-switch/internal/appconfig"
	"lurus-switch/internal/billing"
	"lurus-switch/internal/modelcatalog"
	"lurus-switch/internal/netproxy"
	"lurus-switch/internal/proxy"
	"lurus-switch/internal/proxydetect"
)

const pingTimeout = 10 * time.Second

// ============================
// Proxy / NewAPI Methods
// ============================

// GetProxySettings returns the saved NewAPI proxy settings
func (a *App) GetProxySettings() *proxy.ProxySettings {
	if a.proxyMgr == nil {
		return &proxy.ProxySettings{}
	}
	return a.proxyMgr.GetSettings()
}

// SaveProxySettings saves NewAPI proxy settings to disk and updates billing client
func (a *App) SaveProxySettings(s *proxy.ProxySettings) error {
	if a.proxyMgr == nil {
		return fmt.Errorf("proxy manager not initialized")
	}
	if endpoint := strings.TrimSpace(s.APIEndpoint); endpoint != "" {
		if u, err := url.Parse(endpoint); err != nil || (u.Scheme != "http" && u.Scheme != "https") {
			return fmt.Errorf("invalid API endpoint URL: must start with http:// or https://")
		}
	}
	// Validate the upstream proxy config before persisting so a typo
	// doesn't get saved and silently break every outbound request.
	if s.UpstreamProxy != nil && s.UpstreamProxy.Enabled {
		if _, err := netproxy.BuildTransport(*s.UpstreamProxy); err != nil {
			return fmt.Errorf("upstream proxy config invalid: %w", err)
		}
	}
	if err := a.proxyMgr.SaveSettings(s); err != nil {
		return err
	}
	// Live-apply: subsequent outbound requests pick up the new transport.
	up := netproxy.Settings{}
	if s.UpstreamProxy != nil {
		up = *s.UpstreamProxy
	}
	_ = netproxy.Apply(up) // already validated above

	a.billingMu.Lock()
	if s.UserToken != "" && s.APIEndpoint != "" {
		a.billingClient = billing.NewClient(s.APIEndpoint, s.TenantSlug, s.UserToken)
	} else {
		a.billingClient = nil
	}
	a.billingMu.Unlock()

	// Re-sync a RUNNING gateway so a mid-session endpoint/token edit takes
	// effect immediately instead of leaving every proxied request on the
	// stale upstream until a manual restart. UpdateUpstream locks + persists;
	// the guard is a no-op when the gateway hasn't been built yet.
	if a.gatewaySrv != nil {
		a.syncGatewayUpstream()
	}
	return nil
}

// ConfigureAllProxy applies the saved proxy settings to all installed tools
func (a *App) ConfigureAllProxy() map[string]string {
	if a.proxyMgr == nil {
		return map[string]string{"error": "proxy manager not initialized"}
	}
	settings := a.proxyMgr.GetSettings()
	errs := a.instMgr.ConfigureAllProxy(a.ctx, settings.APIEndpoint, settings.APIKey)
	result := make(map[string]string)
	for name, err := range errs {
		result[name] = err.Error()
	}
	return result
}

// ConfigureAllToolsRelay applies the Lurus relay endpoint and API key (with UserToken fallback)
// to every installed tool's config file. Returns a per-tool error map (empty = all succeeded).
func (a *App) ConfigureAllToolsRelay() map[string]string {
	if a.proxyMgr == nil {
		return map[string]string{"error": "proxy manager not initialized"}
	}
	settings := a.proxyMgr.GetSettings()
	if settings.APIEndpoint == "" {
		return map[string]string{"error": "API endpoint not configured"}
	}
	apiKey := settings.BuildToolAPIKey()
	if apiKey == "" {
		return map[string]string{"error": "no API key or user token configured"}
	}
	errs := a.instMgr.ConfigureAllProxy(a.ctx, settings.APIEndpoint, apiKey)
	result := make(map[string]string)
	for name, err := range errs {
		result[name] = err.Error()
	}
	return result
}

// ============================
// Proxy Auto-Detection Methods
// ============================

// DetectSystemProxy runs all proxy detection methods (env vars, common ports, system settings)
func (a *App) DetectSystemProxy() []proxydetect.DetectedProxy {
	return proxydetect.DetectAll()
}

// ============================
// App Settings Methods (Phase C)
// ============================

// GetAppSettings returns current application settings
func (a *App) GetAppSettings() (*appconfig.AppSettings, error) {
	return appconfig.LoadAppSettings()
}

// SaveAppSettings persists application settings
func (a *App) SaveAppSettings(s *appconfig.AppSettings) error {
	return appconfig.SaveAppSettings(s)
}

// FetchModelCatalog retrieves the model catalog from the gateway API.
func (a *App) FetchModelCatalog() (*modelcatalog.Catalog, error) {
	if a.catalogMgr == nil {
		return modelcatalog.DefaultCatalog(), nil
	}
	op := a.activityBus.Op("fetch-models", "拉取模型目录", "Fetching model catalog")
	apiBase := ""
	apiKey := ""
	if a.proxyMgr != nil {
		s := a.proxyMgr.GetSettings()
		apiBase = s.APIEndpoint
		apiKey = s.BuildToolAPIKey()
	}
	cat, err := a.catalogMgr.Fetch(a.ctx, apiBase, apiKey)
	if err != nil {
		op.Error(err.Error())
	} else if cat != nil {
		op.Done(fmt.Sprintf("收到 %d 个模型", len(cat.Models)), fmt.Sprintf("Received %d models", len(cat.Models)))
	} else {
		op.Done("", "")
	}
	return cat, err
}

// QuickSetup performs one-click configuration: saves the model, applies endpoint+key+model
// to all installed tools. Returns a per-tool error map (empty = all succeeded).
func (a *App) QuickSetup(model string) map[string]string {
	result := make(map[string]string)

	if a.proxyMgr == nil {
		result["error"] = "proxy manager not initialized"
		return result
	}

	settings := a.proxyMgr.GetSettings()
	settings.Model = model
	if err := a.proxyMgr.SaveSettings(settings); err != nil {
		result["error"] = fmt.Sprintf("failed to save settings: %v", err)
		return result
	}

	apiKey := settings.BuildToolAPIKey()
	if settings.APIEndpoint == "" {
		result["error"] = "API endpoint not configured"
		return result
	}
	if apiKey == "" {
		result["error"] = "no API key or user token configured"
		return result
	}

	// Apply endpoint + key
	proxyErrs := a.instMgr.ConfigureAllProxy(a.ctx, settings.APIEndpoint, apiKey)
	for name, err := range proxyErrs {
		result[name] = fmt.Sprintf("proxy: %v", err)
	}

	// Apply model
	if model != "" {
		modelErrs := a.instMgr.ConfigureAllModels(a.ctx, model, settings.ToolModels)
		for name, err := range modelErrs {
			if existing, ok := result[name]; ok {
				result[name] = existing + "; " + fmt.Sprintf("model: %v", err)
			} else {
				result[name] = fmt.Sprintf("model: %v", err)
			}
		}
	}

	return result
}

// SwitchModel changes the model for all installed tools without reconfiguring endpoint/key.
// Captures the prior model into the audit Before snapshot so the Undo
// handler can revert to the previous selection.
func (a *App) SwitchModel(model string) map[string]string {
	result := make(map[string]string)

	// Audit-record + cap-gate. Model switching affects every CLI bound to
	// the gateway — definitely a write operation.
	if err := a.requireAndAudit(capPricingWrite(), auditOpModelSwitch, model, map[string]any{"model": model}); err != nil {
		result["error"] = err.Error()
		return result
	}

	prevModel := ""
	if a.proxyMgr != nil {
		prevModel = a.proxyMgr.GetSettings().Model
	}

	var swErr error
	defer func() {
		a.recordOutcomeFull(
			auditOpModelSwitch,
			model,
			map[string]any{"model": prevModel},
			map[string]any{"model": model, "result": result},
			swErr,
		)
	}()

	if a.proxyMgr == nil {
		result["error"] = "proxy manager not initialized"
		swErr = fmt.Errorf("proxy manager not initialized")
		return result
	}

	settings := a.proxyMgr.GetSettings()
	settings.Model = model
	if err := a.proxyMgr.SaveSettings(settings); err != nil {
		result["error"] = fmt.Sprintf("failed to save settings: %v", err)
		swErr = err
		return result
	}

	if model != "" {
		modelErrs := a.instMgr.ConfigureAllModels(a.ctx, model, settings.ToolModels)
		for name, err := range modelErrs {
			result[name] = err.Error()
		}
	}

	return result
}

// PingEndpoint tests connectivity to the given endpoint URL.
// Returns the round-trip latency in milliseconds, or -1 on failure.
func (a *App) PingEndpoint(endpoint string) (int64, error) {
	endpoint = strings.TrimSpace(endpoint)
	if endpoint == "" {
		return -1, fmt.Errorf("endpoint is empty")
	}
	if u, err := url.Parse(endpoint); err != nil || (u.Scheme != "http" && u.Scheme != "https") {
		return -1, fmt.Errorf("invalid endpoint URL")
	}

	client := &http.Client{
		Timeout: pingTimeout,
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	req, err := http.NewRequestWithContext(a.ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return -1, fmt.Errorf("build request: %w", err)
	}

	start := time.Now()
	resp, err := client.Do(req)
	latency := time.Since(start).Milliseconds()
	if err != nil {
		return -1, err
	}
	resp.Body.Close()
	return latency, nil
}
