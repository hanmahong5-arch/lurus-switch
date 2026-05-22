package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	goruntime "runtime"
	"strings"
	"time"

	"lurus-switch/internal/appconfig"
	"lurus-switch/internal/diagnostics"
)

// DiagnosticCheck is one row in the "Run Diagnostics" report.
// Status is the traffic-light: ok | warn | fail.
type DiagnosticCheck struct {
	ID     string `json:"id"`
	Label  string `json:"label"`
	Status string `json:"status"`
	Detail string `json:"detail"`
}

// DiagnosticsReport is the full payload returned by RunDiagnostics.
type DiagnosticsReport struct {
	GeneratedAt string            `json:"generatedAt"`
	AppVersion  string            `json:"appVersion"`
	OS          string            `json:"os"`
	Arch        string            `json:"arch"`
	ConfigDir   string            `json:"configDir"`
	Checks      []DiagnosticCheck `json:"checks"`
}

// CompetingInstall is a detected install of a similar tool.
// Used by the migration banner to offer a one-click import path.
type CompetingInstall struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Path string `json:"path"`
}

// RunDiagnostics performs health checks across auth/network/gateway/config.
// Synchronous. Worst-case ~15s if every external probe times out.
func (a *App) RunDiagnostics() DiagnosticsReport {
	checks := make([]DiagnosticCheck, 0, 8)

	// OIDC client_id presence.
	clientID := ""
	issuer := "https://auth.lurus.cn"
	if settings, err := appconfig.LoadAppSettings(); err == nil && settings != nil {
		clientID = settings.AuthClientID
		if settings.AuthIssuer != "" {
			issuer = settings.AuthIssuer
		}
	}
	if clientID == "" {
		checks = append(checks, DiagnosticCheck{ID: "oidc-config", Label: "OIDC Client ID", Status: "warn", Detail: "未配置 — 无法登录 Lurus 账号"})
	} else {
		checks = append(checks, DiagnosticCheck{ID: "oidc-config", Label: "OIDC Client ID", Status: "ok", Detail: "已配置 (" + maskID(clientID) + ")"})
	}

	// OIDC issuer reachability via the well-known discovery doc.
	{
		status, detail := probeHTTP(strings.TrimRight(issuer, "/")+"/.well-known/openid-configuration", 5*time.Second)
		checks = append(checks, DiagnosticCheck{ID: "oidc-issuer", Label: "OIDC Issuer 可达", Status: status, Detail: issuer + " — " + detail})
	}

	// Auth session.
	if a.authSession != nil {
		st := a.authSession.GetAuthState()
		switch {
		case !st.IsLoggedIn:
			checks = append(checks, DiagnosticCheck{ID: "auth-state", Label: "登录态", Status: "warn", Detail: "未登录"})
		case !st.HasGatewayToken:
			checks = append(checks, DiagnosticCheck{ID: "auth-state", Label: "登录态", Status: "warn", Detail: "已登录但 Gateway token 未发放"})
		default:
			checks = append(checks, DiagnosticCheck{ID: "auth-state", Label: "登录态", Status: "ok", Detail: "已登录 + Gateway token 已发放"})
		}
	}

	// Local gateway.
	if a.gatewaySrv != nil {
		st := a.gatewaySrv.Status()
		if st.Running {
			checks = append(checks, DiagnosticCheck{ID: "gateway", Label: "本地网关", Status: "ok", Detail: fmt.Sprintf("运行中, port %d", st.Port)})
		} else {
			checks = append(checks, DiagnosticCheck{ID: "gateway", Label: "本地网关", Status: "warn", Detail: fmt.Sprintf("未启动 (port %d 待用)", st.Port)})
		}
	}

	// Upstream API reachability.
	upstreamURL := "https://api.lurus.cn"
	if a.proxyMgr != nil {
		if s := a.proxyMgr.GetSettings(); s != nil && s.APIEndpoint != "" {
			upstreamURL = s.APIEndpoint
		}
	}
	{
		status, detail := probeHTTP(upstreamURL, 5*time.Second)
		checks = append(checks, DiagnosticCheck{ID: "upstream", Label: "Upstream API", Status: status, Detail: upstreamURL + " — " + detail})
	}

	// Public-internet connectivity — distinguishes "Lurus is down" from "user has no internet".
	if hasInternet() {
		checks = append(checks, DiagnosticCheck{ID: "internet", Label: "公网连接", Status: "ok", Detail: "可访问外网"})
	} else {
		checks = append(checks, DiagnosticCheck{ID: "internet", Label: "公网连接", Status: "fail", Detail: "无法访问外网"})
	}

	// Config directory writability.
	cfgDir := appDataBaseDir()
	if err := canWriteDir(cfgDir); err != nil {
		checks = append(checks, DiagnosticCheck{ID: "config-dir", Label: "配置目录", Status: "fail", Detail: fmt.Sprintf("%s 不可写: %v", cfgDir, err)})
	} else {
		checks = append(checks, DiagnosticCheck{ID: "config-dir", Label: "配置目录", Status: "ok", Detail: cfgDir})
	}

	return DiagnosticsReport{
		GeneratedAt: time.Now().Format(time.RFC3339),
		AppVersion:  AppVersion,
		OS:          goruntime.GOOS,
		Arch:        goruntime.GOARCH,
		ConfigDir:   cfgDir,
		Checks:      checks,
	}
}

// DetectCompetingInstalls scans well-known competitor config dirs under $HOME.
// Returns only non-empty directories — a stale empty dir post-migration won't
// trigger a false-positive banner.
func (a *App) DetectCompetingInstalls() []CompetingInstall {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}
	candidates := []CompetingInstall{
		{ID: "cc-switch", Name: "cc-switch", Path: filepath.Join(home, ".cc-switch")},
		{ID: "opcode", Name: "opcode", Path: filepath.Join(home, ".opcode")},
		{ID: "ccr", Name: "claude-code-router", Path: filepath.Join(home, ".ccr")},
		{ID: "hermes", Name: "Hermes Agent", Path: filepath.Join(home, ".hermes")},
		{ID: "openclaw", Name: "OpenClaw", Path: filepath.Join(home, ".openclaw")},
	}
	found := make([]CompetingInstall, 0, len(candidates))
	for _, c := range candidates {
		info, err := os.Stat(c.Path)
		if err != nil || !info.IsDir() {
			continue
		}
		entries, _ := os.ReadDir(c.Path)
		if len(entries) == 0 {
			continue
		}
		found = append(found, c)
	}
	return found
}

// WriteDebugDump writes a redacted JSON of diagnostics + app state to
// %APPDATA%/lurus-switch/dumps/debug-<ts>.json and returns the absolute
// path so the UI can offer "open file location".
//
// Redaction rules: tokens are never written, only "set/not-set" booleans;
// the OIDC client_id is masked to last-4. Designed so the user can email
// the file to support without leaking credentials.
func (a *App) WriteDebugDump() (string, error) {
	dir := filepath.Join(appDataBaseDir(), "dumps")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("mkdir dumps: %w", err)
	}
	path := filepath.Join(dir, fmt.Sprintf("debug-%s.json", time.Now().Format("20060102-150405")))

	diag := a.RunDiagnostics()
	dump := map[string]any{
		"generatedAt": diag.GeneratedAt,
		"appVersion":  AppVersion,
		"platform": map[string]string{
			"os":   goruntime.GOOS,
			"arch": goruntime.GOARCH,
		},
		"configDir":         diag.ConfigDir,
		"diagnostics":       diag.Checks,
		"competingInstalls": a.DetectCompetingInstalls(),
	}

	if a.authSession != nil {
		st := a.authSession.GetAuthState()
		dump["auth"] = map[string]any{
			"isLoggedIn":      st.IsLoggedIn,
			"hasGatewayToken": st.HasGatewayToken,
		}
	}
	if settings, err := appconfig.LoadAppSettings(); err == nil && settings != nil {
		dump["appSettings"] = map[string]any{
			"appMode":         settings.AppMode,
			"authIssuer":      settings.AuthIssuer,
			"authClientIdSet": settings.AuthClientID != "",
			"lockedHubUrlSet": settings.LockedHubURL != "",
			"brandName":       settings.BrandName,
		}
	}
	if a.gatewaySrv != nil {
		st := a.gatewaySrv.Status()
		dump["gateway"] = map[string]any{
			"running":       st.Running,
			"port":          st.Port,
			"totalRequests": st.TotalRequests,
		}
	}
	if a.proxyMgr != nil {
		if s := a.proxyMgr.GetSettings(); s != nil {
			dump["proxy"] = map[string]any{
				"apiEndpoint":  s.APIEndpoint,
				"tenantSlug":   s.TenantSlug,
				"userTokenSet": s.UserToken != "",
				"apiKeySet":    s.APIKey != "",
			}
		}
	}

	out, err := json.MarshalIndent(dump, "", "  ")
	if err != nil {
		return "", fmt.Errorf("marshal dump: %w", err)
	}
	if err := os.WriteFile(path, out, 0644); err != nil {
		return "", fmt.Errorf("write dump: %w", err)
	}
	return path, nil
}

// OpenDebugDumpDir opens the dumps folder in the OS file explorer so
// the user can grab the latest debug file to attach to a support email.
func (a *App) OpenDebugDumpDir() error {
	return openDirectory(filepath.Join(appDataBaseDir(), "dumps"))
}

// ─── helpers ────────────────────────────────────────────────────────────

// probeHTTP returns ("ok"|"warn"|"fail", detail). 2xx-4xx counts as
// reachable; only DNS / refused / timeout / 5xx are flagged.
func probeHTTP(url string, timeout time.Duration) (string, string) {
	client := &http.Client{Timeout: timeout}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, "HEAD", url, nil)
	if err != nil {
		return "fail", err.Error()
	}
	resp, err := client.Do(req)
	if err != nil {
		// HEAD can be 405 on some endpoints — retry with GET.
		getReq, gerr := http.NewRequestWithContext(ctx, "GET", url, nil)
		if gerr != nil {
			return "fail", err.Error()
		}
		resp, err = client.Do(getReq)
		if err != nil {
			return "fail", err.Error()
		}
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 200 && resp.StatusCode < 500 {
		return "ok", fmt.Sprintf("HTTP %d", resp.StatusCode)
	}
	return "warn", fmt.Sprintf("HTTP %d", resp.StatusCode)
}

// hasInternet probes Cloudflare's 1.1.1.1:443 — TCP-only, no DNS, so
// it isolates "is there a route to the internet at all" from any
// DNS / proxy interference that might affect the Lurus probes above.
func hasInternet() bool {
	conn, err := net.DialTimeout("tcp", "1.1.1.1:443", 3*time.Second)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

func canWriteDir(dir string) error {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	probe := filepath.Join(dir, ".write-test")
	if err := os.WriteFile(probe, []byte("ok"), 0644); err != nil {
		return err
	}
	return os.Remove(probe)
}

func maskID(s string) string {
	if len(s) <= 4 {
		return strings.Repeat("*", len(s))
	}
	return strings.Repeat("*", len(s)-4) + s[len(s)-4:]
}

// GetStartupTrace returns the current process's startup timeline — phase
// breakdown plus the GUI-ready and cold-start milestones. Powers the
// "startup performance" card in Settings.
func (a *App) GetStartupTrace() diagnostics.Trace {
	return diagnostics.Default.Snapshot()
}

// GetStartupHistory returns the last few persisted startup traces (newest
// first) so the UI can show a "vs. last launch" delta. The current trace
// is index 0 only after it has been persisted; callers that want "current
// vs previous" should pair this with GetStartupTrace.
func (a *App) GetStartupHistory() []diagnostics.Trace {
	return diagnostics.History(appDataBaseDir())
}
