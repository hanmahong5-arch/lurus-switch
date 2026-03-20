package main

import (
	"encoding/json"
	"fmt"
	"strings"

	"lurus-switch/internal/installer"
	"lurus-switch/internal/toolconfig"
	"lurus-switch/internal/toolhealth"

	"github.com/BurntSushi/toml"
)

// ============================
// Switch Orchestration Methods
// ============================
// These bridge tool detection, app registry, config writing, and the local
// gateway — enabling one-click "connect all tools", unified environment
// diagnostics, and full setup orchestration.

// ToolConfigResult is the per-tool outcome of auto-configure.
type ToolConfigResult struct {
	Tool    string `json:"tool"`
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// ToolDiagnostic holds the unified environment status for a single tool.
type ToolDiagnostic struct {
	Tool            string   `json:"tool"`
	Installed       bool     `json:"installed"`
	Version         string   `json:"version"`
	Path            string   `json:"path"`
	ConfigExists    bool     `json:"configExists"`
	HealthStatus    string   `json:"healthStatus"` // "green" | "yellow" | "red" | "unknown"
	HealthIssues    []string `json:"healthIssues"`
	GatewayBound    bool     `json:"gatewayBound"`    // config points to Switch gateway
	Connected       bool     `json:"connected"`        // app marked connected in registry
	CurrentEndpoint string   `json:"currentEndpoint"`  // the API base URL currently in the config
	CurrentModel    string   `json:"currentModel"`     // the model currently in the config
}

// RuntimeDiagnostic holds a runtime dependency check result.
type RuntimeDiagnostic struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Installed bool   `json:"installed"`
	Version   string `json:"version"`
	Required  bool   `json:"required"`
}

// EnvironmentCheck is the unified diagnostic report.
type EnvironmentCheck struct {
	Tools          []ToolDiagnostic    `json:"tools"`
	Runtimes       []RuntimeDiagnostic `json:"runtimes"`
	GatewayRunning bool                `json:"gatewayRunning"`
	GatewayURL     string              `json:"gatewayUrl"`
	AllToolsBound  bool                `json:"allToolsBound"`
	InstalledCount int                 `json:"installedCount"`
	BoundCount     int                 `json:"boundCount"`
}

// FullSetupResult is the outcome of the complete setup orchestration.
type FullSetupResult struct {
	GatewayStarted bool               `json:"gatewayStarted"`
	SnapshotsTaken int                `json:"snapshotsTaken"`
	ConfigResults  []ToolConfigResult `json:"configResults"`
	GatewayURL     string             `json:"gatewayUrl"`
	Errors         []string           `json:"errors"`
}

// Ordered list of tools that Switch can auto-configure.
var managedTools = []string{
	installer.ToolClaude, installer.ToolCodex, installer.ToolGemini,
	installer.ToolPicoClaw, installer.ToolNullClaw, installer.ToolZeroClaw, installer.ToolOpenClaw,
}

// gatewayBaseURL returns the gateway endpoint URL for tool configuration.
func (a *App) gatewayBaseURL() string {
	if a.gatewaySrv != nil {
		st := a.gatewaySrv.Status()
		if st.Running && st.URL != "" {
			return st.URL
		}
		cfg := a.gatewaySrv.GetConfig()
		if cfg.Port > 0 {
			return fmt.Sprintf("http://localhost:%d", cfg.Port)
		}
	}
	return "http://localhost:19090"
}

// AutoConfigureToolsForGateway detects all installed tools and writes their
// config files to point at the local Switch gateway with per-app tokens.
func (a *App) AutoConfigureToolsForGateway() []ToolConfigResult {
	if a.appRegistry == nil || a.instMgr == nil {
		return []ToolConfigResult{{Tool: "*", Success: false, Message: "services not initialized"}}
	}

	gwURL := a.gatewayBaseURL()
	statuses, _ := a.instMgr.DetectAll(a.ctx)
	var results []ToolConfigResult

	for _, tool := range managedTools {
		status, ok := statuses[tool]
		if !ok || !status.Installed {
			continue
		}

		app := a.appRegistry.Get(tool)
		if app == nil {
			results = append(results, ToolConfigResult{
				Tool: tool, Success: false,
				Message: "no matching app in registry",
			})
			continue
		}

		toolEP := installer.ToolEndpoint(tool, gwURL)
		if err := a.instMgr.ConfigureTool(a.ctx, tool, toolEP, app.Token); err != nil {
			results = append(results, ToolConfigResult{
				Tool: tool, Success: false,
				Message: fmt.Sprintf("config write failed: %v", err),
			})
			continue
		}

		_ = a.appRegistry.SetConnected(tool, true)
		results = append(results, ToolConfigResult{
			Tool: tool, Success: true,
			Message: fmt.Sprintf("configured → %s", toolEP),
		})
	}

	return results
}

// AutoConfigureToolForGateway configures a single tool to use the Switch gateway.
func (a *App) AutoConfigureToolForGateway(tool string) (*ToolConfigResult, error) {
	if a.appRegistry == nil || a.instMgr == nil {
		return nil, fmt.Errorf("services not initialized")
	}

	app := a.appRegistry.Get(tool)
	if app == nil {
		return &ToolConfigResult{Tool: tool, Success: false, Message: "no matching app in registry"}, nil
	}

	gwURL := a.gatewayBaseURL()
	toolEP := installer.ToolEndpoint(tool, gwURL)
	if err := a.instMgr.ConfigureTool(a.ctx, tool, toolEP, app.Token); err != nil {
		return &ToolConfigResult{
			Tool: tool, Success: false,
			Message: fmt.Sprintf("config write failed: %v", err),
		}, nil
	}

	_ = a.appRegistry.SetConnected(tool, true)
	return &ToolConfigResult{
		Tool: tool, Success: true,
		Message: fmt.Sprintf("configured → %s", toolEP),
	}, nil
}

// FullSetupForGateway orchestrates the complete setup flow:
//  1. Start gateway if not running
//  2. Snapshot all installed tool configs
//  3. Auto-configure all installed tools
//
// Returns a comprehensive result.
func (a *App) FullSetupForGateway() *FullSetupResult {
	result := &FullSetupResult{}

	// Step 1: Start gateway if not running.
	if a.gatewaySrv != nil {
		st := a.gatewaySrv.Status()
		if !st.Running {
			a.syncGatewayUpstream()
			if err := a.gatewaySrv.Start(a.ctx); err != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("gateway start: %v", err))
			} else {
				result.GatewayStarted = true
			}
		}
	}

	result.GatewayURL = a.gatewayBaseURL()

	// Step 2: Snapshot all installed tool configs before overwriting.
	if a.snapshotStr != nil {
		statuses, _ := a.instMgr.DetectAll(a.ctx)
		for _, tool := range managedTools {
			ts, ok := statuses[tool]
			if !ok || !ts.Installed {
				continue
			}
			info, err := toolconfig.ReadConfig(tool)
			if err != nil || !info.Exists {
				continue
			}
			if err := a.snapshotStr.Take(tool, "pre-switch-setup", info.Content); err == nil {
				result.SnapshotsTaken++
			}
		}
	}

	// Step 3: Auto-configure all installed tools.
	result.ConfigResults = a.AutoConfigureToolsForGateway()
	return result
}

// RunEnvironmentCheck performs a unified diagnostic of all tools, runtimes,
// and gateway status. Returns actionable information including current
// endpoint and model for each tool.
func (a *App) RunEnvironmentCheck() *EnvironmentCheck {
	result := &EnvironmentCheck{AllToolsBound: true}

	if a.gatewaySrv != nil {
		st := a.gatewaySrv.Status()
		result.GatewayRunning = st.Running
		result.GatewayURL = a.gatewayBaseURL()
	}

	statuses, _ := a.instMgr.DetectAll(a.ctx)
	healthResults := toolhealth.CheckAll()
	gwURL := a.gatewayBaseURL()

	for _, tool := range managedTools {
		diag := ToolDiagnostic{
			Tool:         tool,
			HealthIssues: []string{},
		}

		if ts, ok := statuses[tool]; ok {
			diag.Installed = ts.Installed
			diag.Version = ts.Version
			diag.Path = ts.Path
		}

		if hr, ok := healthResults[tool]; ok {
			diag.HealthStatus = string(hr.Status)
			diag.ConfigExists = true
			diag.HealthIssues = hr.Issues
			for _, issue := range hr.Issues {
				if strings.Contains(issue, "config file not found") ||
					strings.Contains(issue, "config read error") {
					diag.ConfigExists = false
				}
			}
		} else {
			diag.HealthStatus = "unknown"
		}

		// Extract current endpoint and model from config.
		if diag.ConfigExists {
			ep, model := extractToolEndpointAndModel(tool)
			diag.CurrentEndpoint = ep
			diag.CurrentModel = model
		}

		if diag.Installed {
			diag.GatewayBound = isToolBoundToGateway(tool, gwURL)
			if diag.GatewayBound {
				result.BoundCount++
			}
			result.InstalledCount++
		}

		if a.appRegistry != nil {
			if app := a.appRegistry.Get(tool); app != nil {
				diag.Connected = app.Connected
			}
		}

		if diag.Installed && !diag.GatewayBound {
			result.AllToolsBound = false
		}

		result.Tools = append(result.Tools, diag)
	}

	if deps, err := a.instMgr.CheckDependencies(a.ctx); err == nil && deps != nil {
		for _, rt := range deps.Runtimes {
			result.Runtimes = append(result.Runtimes, RuntimeDiagnostic{
				ID:        rt.ID,
				Name:      rt.Name,
				Installed: rt.Installed,
				Version:   rt.Version,
				Required:  rt.Required,
			})
		}
	}

	return result
}

// SyncToolConnectionStatus reads all tool configs, checks which ones
// point to the Switch gateway, and updates the app registry's connected
// flag accordingly. Call this periodically or after config changes.
func (a *App) SyncToolConnectionStatus() {
	if a.appRegistry == nil {
		return
	}
	gwURL := a.gatewayBaseURL()
	for _, tool := range managedTools {
		bound := isToolBoundToGateway(tool, gwURL)
		if app := a.appRegistry.Get(tool); app != nil {
			if app.Connected != bound {
				_ = a.appRegistry.SetConnected(tool, bound)
			}
		}
	}
}

// ── Export & Fix ────────────────────────────────────────────────────

// ExportDiagnostics generates a human-readable text report of the environment
// check result, suitable for pasting into support tickets or bug reports.
func (a *App) ExportDiagnostics() string {
	check := a.RunEnvironmentCheck()
	if check == nil {
		return "Environment check unavailable"
	}

	var sb strings.Builder
	sb.WriteString("=== Lurus Switch Environment Report ===\n\n")

	// Gateway
	if check.GatewayRunning {
		sb.WriteString(fmt.Sprintf("Gateway: RUNNING at %s\n", check.GatewayURL))
	} else {
		sb.WriteString("Gateway: STOPPED\n")
	}
	sb.WriteString(fmt.Sprintf("Tools: %d installed, %d connected\n\n", check.InstalledCount, check.BoundCount))

	// Tools
	for _, t := range check.Tools {
		status := "not installed"
		if t.Installed {
			status = fmt.Sprintf("v%s", t.Version)
			if t.GatewayBound {
				status += " [connected]"
			} else {
				status += " [disconnected]"
			}
			status += fmt.Sprintf(" health=%s", t.HealthStatus)
			if t.CurrentEndpoint != "" {
				status += fmt.Sprintf(" endpoint=%s", t.CurrentEndpoint)
			}
			if t.CurrentModel != "" {
				status += fmt.Sprintf(" model=%s", t.CurrentModel)
			}
		}
		sb.WriteString(fmt.Sprintf("  %-10s %s\n", t.Tool, status))

		if len(t.HealthIssues) > 0 {
			for _, issue := range t.HealthIssues {
				sb.WriteString(fmt.Sprintf("             ! %s\n", issue))
			}
		}
	}

	// Runtimes
	if len(check.Runtimes) > 0 {
		sb.WriteString("\nRuntimes:\n")
		for _, rt := range check.Runtimes {
			installed := "missing"
			if rt.Installed {
				installed = fmt.Sprintf("v%s", rt.Version)
			}
			req := ""
			if rt.Required {
				req = " (required)"
			}
			sb.WriteString(fmt.Sprintf("  %-10s %s%s\n", rt.Name, installed, req))
		}
	}

	return sb.String()
}

// AutoFixToolConfig attempts to fix common configuration issues for a tool.
// For example: creates missing config dir/file, ensures the config is valid JSON/TOML.
// Returns a result indicating what was fixed.
func (a *App) AutoFixToolConfig(tool string) (*ToolConfigResult, error) {
	info, err := toolconfig.ReadConfig(tool)
	if err != nil {
		return &ToolConfigResult{Tool: tool, Success: false, Message: fmt.Sprintf("read error: %v", err)}, nil
	}

	// Fix 1: Config file doesn't exist — create from template.
	if !info.Exists {
		if err := toolconfig.WriteConfig(tool, info.Content); err != nil {
			return &ToolConfigResult{Tool: tool, Success: false, Message: fmt.Sprintf("create failed: %v", err)}, nil
		}
		return &ToolConfigResult{Tool: tool, Success: true, Message: "created config file from template"}, nil
	}

	// Fix 2: Config file exists but is empty.
	if strings.TrimSpace(info.Content) == "" {
		template := getToolDefaultTemplate(tool)
		if err := toolconfig.WriteConfig(tool, template); err != nil {
			return &ToolConfigResult{Tool: tool, Success: false, Message: fmt.Sprintf("write failed: %v", err)}, nil
		}
		return &ToolConfigResult{Tool: tool, Success: true, Message: "replaced empty config with template"}, nil
	}

	// Fix 3: Validate JSON/TOML syntax.
	switch info.Language {
	case "json":
		var check any
		if jsonErr := json.Unmarshal([]byte(info.Content), &check); jsonErr != nil {
			// JSON is broken — overwrite with template.
			if a.snapshotStr != nil {
				_ = a.snapshotStr.Take(tool, "pre-autofix", info.Content)
			}
			template := getToolDefaultTemplate(tool)
			if err := toolconfig.WriteConfig(tool, template); err != nil {
				return &ToolConfigResult{Tool: tool, Success: false, Message: fmt.Sprintf("write failed: %v", err)}, nil
			}
			return &ToolConfigResult{Tool: tool, Success: true, Message: "fixed broken JSON (reset to template, old config saved as snapshot)"}, nil
		}
	case "toml":
		var check any
		if _, tomlErr := toml.Decode(info.Content, &check); tomlErr != nil {
			if a.snapshotStr != nil {
				_ = a.snapshotStr.Take(tool, "pre-autofix", info.Content)
			}
			template := getToolDefaultTemplate(tool)
			if err := toolconfig.WriteConfig(tool, template); err != nil {
				return &ToolConfigResult{Tool: tool, Success: false, Message: fmt.Sprintf("write failed: %v", err)}, nil
			}
			return &ToolConfigResult{Tool: tool, Success: true, Message: "fixed broken TOML (reset to template, old config saved as snapshot)"}, nil
		}
	}

	return &ToolConfigResult{Tool: tool, Success: true, Message: "config is valid, no fix needed"}, nil
}

// ── Disconnect & Restore ────────────────────────────────────────────

// ToolSnapshotInfo is a lightweight snapshot entry returned to the frontend.
type ToolSnapshotInfo struct {
	ID        string `json:"id"`
	Tool      string `json:"tool"`
	Label     string `json:"label"`
	CreatedAt string `json:"createdAt"`
	Size      int    `json:"size"`
}

// DisconnectToolFromGateway restores a tool's config to its pre-switch-setup
// snapshot (if available) or resets to the default template, and marks it
// disconnected in the registry.
func (a *App) DisconnectToolFromGateway(tool string) (*ToolConfigResult, error) {
	if a.appRegistry == nil {
		return nil, fmt.Errorf("services not initialized")
	}

	restored := false

	// Try to restore the most recent pre-switch-setup snapshot.
	if a.snapshotStr != nil {
		metas, err := a.snapshotStr.List(tool)
		if err == nil {
			for _, m := range metas {
				if strings.Contains(m.Label, "pre-switch-setup") {
					content, err := a.snapshotStr.Restore(tool, m.ID)
					if err == nil && content != "" {
						if err := toolconfig.WriteConfig(tool, content); err == nil {
							restored = true
							break
						}
					}
				}
			}
		}
	}

	// Fallback: write the default template (clears endpoint + key).
	if !restored {
		info, err := toolconfig.ReadConfig(tool)
		if err == nil && !info.Exists {
			// Config doesn't exist — nothing to disconnect.
			_ = a.appRegistry.SetConnected(tool, false)
			return &ToolConfigResult{Tool: tool, Success: true, Message: "no config to disconnect"}, nil
		}
		// Write default template to clear the gateway binding.
		if err := toolconfig.WriteConfig(tool, defaultConfigFor(tool)); err != nil {
			return &ToolConfigResult{
				Tool: tool, Success: false,
				Message: fmt.Sprintf("config reset failed: %v", err),
			}, nil
		}
	}

	_ = a.appRegistry.SetConnected(tool, false)

	msg := "disconnected (restored from snapshot)"
	if !restored {
		msg = "disconnected (reset to default)"
	}
	return &ToolConfigResult{Tool: tool, Success: true, Message: msg}, nil
}

// DisconnectAllToolsFromGateway disconnects all installed tools.
func (a *App) DisconnectAllToolsFromGateway() []ToolConfigResult {
	if a.instMgr == nil {
		return []ToolConfigResult{{Tool: "*", Success: false, Message: "services not initialized"}}
	}

	statuses, _ := a.instMgr.DetectAll(a.ctx)
	var results []ToolConfigResult

	for _, tool := range managedTools {
		ts, ok := statuses[tool]
		if !ok || !ts.Installed {
			continue
		}
		gwURL := a.gatewayBaseURL()
		if !isToolBoundToGateway(tool, gwURL) {
			continue // already disconnected
		}
		r, err := a.DisconnectToolFromGateway(tool)
		if err != nil {
			results = append(results, ToolConfigResult{Tool: tool, Success: false, Message: err.Error()})
		} else {
			results = append(results, *r)
		}
	}

	return results
}

// ListToolSnapshots returns available config snapshots for a tool.
func (a *App) ListToolSnapshots(tool string) ([]ToolSnapshotInfo, error) {
	if a.snapshotStr == nil {
		return nil, fmt.Errorf("snapshot store not initialized")
	}

	metas, err := a.snapshotStr.List(tool)
	if err != nil {
		return nil, err
	}

	out := make([]ToolSnapshotInfo, 0, len(metas))
	for _, m := range metas {
		out = append(out, ToolSnapshotInfo{
			ID:        m.ID,
			Tool:      m.Tool,
			Label:     m.Label,
			CreatedAt: m.CreatedAt,
			Size:      m.Size,
		})
	}
	return out, nil
}

// RestoreToolSnapshot restores a specific snapshot to a tool's config file.
// Takes a snapshot before overwriting so the user can undo.
func (a *App) RestoreToolSnapshot(tool, snapshotID string) (*ToolConfigResult, error) {
	if a.snapshotStr == nil {
		return nil, fmt.Errorf("snapshot store not initialized")
	}

	content, err := a.snapshotStr.Restore(tool, snapshotID)
	if err != nil {
		return &ToolConfigResult{Tool: tool, Success: false, Message: fmt.Sprintf("snapshot read: %v", err)}, nil
	}

	// Snapshot the current config before overwriting.
	if info, err := toolconfig.ReadConfig(tool); err == nil && info.Exists {
		_ = a.snapshotStr.Take(tool, "pre-restore", info.Content)
	}

	if err := toolconfig.WriteConfig(tool, content); err != nil {
		return &ToolConfigResult{Tool: tool, Success: false, Message: fmt.Sprintf("write failed: %v", err)}, nil
	}

	// Update connection status based on new config content.
	a.SyncToolConnectionStatus()

	return &ToolConfigResult{Tool: tool, Success: true, Message: fmt.Sprintf("restored snapshot %s", snapshotID)}, nil
}

// defaultConfigFor returns the default config template for a tool.
// Used as fallback when no snapshot is available for disconnect.
func defaultConfigFor(tool string) string {
	info, err := toolconfig.ReadConfig(tool)
	if err == nil && !info.Exists {
		// The ReadConfig function returns the default template when the file doesn't exist.
		return info.Content
	}
	// For an existing config, re-read via the "not found" path to get the template.
	// This is a simple approach — the toolconfig package embeds templates.
	return getToolDefaultTemplate(tool)
}

// getToolDefaultTemplate returns the built-in default template for a tool.
func getToolDefaultTemplate(tool string) string {
	// Create a temp ReadConfig scenario to extract the template.
	// toolconfig.ReadConfig returns Content = defaultTemplate when file doesn't exist,
	// but we can't rely on that if the file DOES exist. Instead, use a direct map.
	templates := map[string]string{
		"claude": `{
  "env": {
    "ANTHROPIC_API_KEY": "",
    "ANTHROPIC_BASE_URL": ""
  }
}`,
		"codex": `model = ""
[model_providers.custom]
name = "Custom Proxy"
base_url = ""
env_key = "OPENAI_API_KEY"
wire_api = "chat"
`,
		"gemini": `{
  "model": {
    "name": "gemini-2.5-flash"
  }
}`,
		"picoclaw": `{
  "model_list": [
    {
      "name": "default",
      "api_base": "",
      "api_key": "",
      "model_name": ""
    }
  ]
}`,
		"nullclaw": `{
  "model_list": [
    {
      "name": "default",
      "api_base": "",
      "api_key": "",
      "model_name": ""
    }
  ]
}`,
		"zeroclaw": `[provider]
type = "anthropic"
api_key = ""
model = ""
base_url = ""
`,
		"openclaw": `{
  "provider": {
    "type": "anthropic",
    "api_key": "",
    "model": ""
  }
}`,
	}
	if t, ok := templates[tool]; ok {
		return t
	}
	return "{}"
}

// ── Config parsing helpers ──────────────────────────────────────────

// isToolBoundToGateway reads a tool's config and checks if the API endpoint
// points to the Switch gateway.
func isToolBoundToGateway(tool, gwURL string) bool {
	info, err := toolconfig.ReadConfig(tool)
	if err != nil || !info.Exists {
		return false
	}

	gwNorm := strings.TrimRight(gwURL, "/")
	gwV1 := gwNorm + "/v1"
	lower := strings.ToLower(info.Content)
	return strings.Contains(lower, strings.ToLower(gwNorm)) ||
		strings.Contains(lower, strings.ToLower(gwV1))
}

// extractToolEndpointAndModel reads a tool's config and extracts the
// currently configured API endpoint and model. Best-effort; returns ""
// for fields that cannot be parsed.
func extractToolEndpointAndModel(tool string) (endpoint, model string) {
	info, err := toolconfig.ReadConfig(tool)
	if err != nil || !info.Exists || info.Content == "" {
		return "", ""
	}

	switch tool {
	case "claude":
		return parseClaudeConfig(info.Content)
	case "codex":
		return parseCodexConfig(info.Content)
	case "gemini":
		return parseGeminiConfig(info.Content)
	case "picoclaw", "nullclaw":
		return parseClawConfig(info.Content)
	case "zeroclaw":
		return parseZeroClawConfig(info.Content)
	case "openclaw":
		return parseOpenClawConfig(info.Content)
	}
	return "", ""
}

// parseClaudeConfig extracts endpoint and model from Claude's JSON config.
// Claude stores: env.ANTHROPIC_BASE_URL and optionally a model field.
func parseClaudeConfig(content string) (string, string) {
	var data map[string]any
	if json.Unmarshal([]byte(content), &data) != nil {
		return "", ""
	}
	env, _ := data["env"].(map[string]any)
	ep, _ := env["ANTHROPIC_BASE_URL"].(string)
	model, _ := data["model"].(string)
	return ep, model
}

// parseCodexConfig extracts endpoint from Codex's TOML config.
func parseCodexConfig(content string) (string, string) {
	var data map[string]any
	if _, err := toml.Decode(content, &data); err != nil {
		return "", ""
	}
	model, _ := data["model"].(string)
	// model_providers.custom.base_url
	if mp, ok := data["model_providers"].(map[string]any); ok {
		if custom, ok := mp["custom"].(map[string]any); ok {
			if ep, ok := custom["base_url"].(string); ok {
				return ep, model
			}
		}
	}
	return "", model
}

// parseGeminiConfig extracts endpoint and model from Gemini's JSON config.
func parseGeminiConfig(content string) (string, string) {
	var data map[string]any
	if json.Unmarshal([]byte(content), &data) != nil {
		return "", ""
	}
	ep, _ := data["apiEndpoint"].(string)
	if modelObj, ok := data["model"].(map[string]any); ok {
		model, _ := modelObj["name"].(string)
		return ep, model
	}
	return ep, ""
}

// parseClawConfig extracts endpoint and model from PicoClaw/NullClaw JSON.
func parseClawConfig(content string) (string, string) {
	var data map[string]any
	if json.Unmarshal([]byte(content), &data) != nil {
		return "", ""
	}
	list, _ := data["model_list"].([]any)
	if len(list) == 0 {
		return "", ""
	}
	first, _ := list[0].(map[string]any)
	ep, _ := first["api_base"].(string)
	model, _ := first["model_name"].(string)
	return ep, model
}

// parseZeroClawConfig extracts endpoint and model from ZeroClaw TOML.
func parseZeroClawConfig(content string) (string, string) {
	var data map[string]any
	if _, err := toml.Decode(content, &data); err != nil {
		return "", ""
	}
	if provider, ok := data["provider"].(map[string]any); ok {
		ep, _ := provider["base_url"].(string)
		model, _ := provider["model"].(string)
		return ep, model
	}
	return "", ""
}

// parseOpenClawConfig extracts endpoint and model from OpenClaw JSON.
func parseOpenClawConfig(content string) (string, string) {
	var data map[string]any
	if json.Unmarshal([]byte(content), &data) != nil {
		return "", ""
	}
	if provider, ok := data["provider"].(map[string]any); ok {
		ep, _ := provider["base_url"].(string)
		model, _ := provider["model"].(string)
		return ep, model
	}
	return "", ""
}
