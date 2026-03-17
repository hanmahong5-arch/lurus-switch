package main

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"lurus-switch/internal/appconfig"
	"lurus-switch/internal/billing"
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
	if err := a.proxyMgr.SaveSettings(s); err != nil {
		return err
	}

	a.billingMu.Lock()
	if s.UserToken != "" && s.APIEndpoint != "" {
		a.billingClient = billing.NewClient(s.APIEndpoint, s.TenantSlug, s.UserToken)
	} else {
		a.billingClient = nil
	}
	a.billingMu.Unlock()
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
