package main

import (
	"context"

	"lurus-switch/internal/mcp"
	"lurus-switch/internal/mcpmarket"
)

// McpMarketResult is the standard {success, message} envelope used by
// mutation bindings.  Wails generates a TypeScript interface from this struct.
type McpMarketResult struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// McpMarketInstallResult extends McpMarketResult with per-tool install statuses.
type McpMarketInstallResult struct {
	Success  bool                          `json:"success"`
	Message  string                        `json:"message"`
	Statuses []mcpmarket.ToolInstallStatus `json:"statuses,omitempty"`
}

// McpMarketPresetResult extends McpMarketResult with the saved preset ID.
type McpMarketPresetResult struct {
	Success  bool   `json:"success"`
	Message  string `json:"message"`
	PresetID string `json:"presetId,omitempty"`
}

// --------------------------------------------------------------------------
// Read bindings
// --------------------------------------------------------------------------

// McpMarketList returns the merged list of builtin + cached registry servers.
// This call never blocks on a network request.
func (a *App) McpMarketList() ([]mcpmarket.MarketServer, error) {
	return mcpmarket.NewMarket().ListServers()
}

// --------------------------------------------------------------------------
// Write / mutation bindings
// --------------------------------------------------------------------------

// McpMarketRefresh fetches the first page of servers from the public registry
// and updates the local disk cache.  Network errors are swallowed; builtins
// remain usable offline.  The caller must check result.success.
func (a *App) McpMarketRefresh(query string) McpMarketResult {
	ctx := a.ctx
	if ctx == nil {
		ctx = context.Background()
	}
	if err := mcpmarket.NewMarket().RefreshFromRegistry(ctx, query); err != nil {
		return McpMarketResult{Success: false, Message: err.Error()}
	}
	return McpMarketResult{Success: true, Message: "ok"}
}

// McpMarketInstall writes an MCP server entry into each selected target tool's
// config file.  targetTools is a JSON array of tool identifiers such as
// ["claude_code","cursor","gemini","antigravity"].  userConfig holds
// user-supplied values for the server's configSchema fields (env vars, etc.).
// Each tool is attempted independently; the caller must inspect each status.
func (a *App) McpMarketInstall(
	serverID string,
	userConfig map[string]string,
	targetTools []string,
) McpMarketInstallResult {
	ctx := a.ctx
	if ctx == nil {
		ctx = context.Background()
	}

	if serverID == "" {
		return McpMarketInstallResult{Success: false, Message: "serverID required"}
	}
	if len(targetTools) == 0 {
		return McpMarketInstallResult{Success: false, Message: "at least one target tool required"}
	}

	m := mcpmarket.NewMarket()
	server, err := m.GetServer(ctx, serverID)
	if err != nil {
		return McpMarketInstallResult{Success: false, Message: err.Error()}
	}

	// Convert string slice to TargetTool slice with validation.
	targets := make([]mcpmarket.TargetTool, 0, len(targetTools))
	for _, t := range targetTools {
		targets = append(targets, mcpmarket.TargetTool(t))
	}

	report, err := m.InstallToTools(ctx, *server, userConfig, targets)
	if err != nil {
		return McpMarketInstallResult{Success: false, Message: err.Error()}
	}

	// Overall success = all tools succeeded.
	allOK := true
	for _, st := range report.Statuses {
		if !st.OK {
			allOK = false
			break
		}
	}
	msg := "installed"
	if !allOK {
		msg = "partial install — see statuses"
	}
	return McpMarketInstallResult{
		Success:  allOK,
		Message:  msg,
		Statuses: report.Statuses,
	}
}

// McpMarketSavePreset saves a market server as a reusable MCP preset.
// The caller must check result.success.
func (a *App) McpMarketSavePreset(serverID string, userConfig map[string]string) McpMarketPresetResult {
	ctx := a.ctx
	if ctx == nil {
		ctx = context.Background()
	}

	if serverID == "" {
		return McpMarketPresetResult{Success: false, Message: "serverID required"}
	}

	m := mcpmarket.NewMarket()
	server, err := m.GetServer(ctx, serverID)
	if err != nil {
		return McpMarketPresetResult{Success: false, Message: err.Error()}
	}

	store := a.mcpStr
	if store == nil {
		var storeErr error
		store, storeErr = mcp.NewStore()
		if storeErr != nil {
			return McpMarketPresetResult{Success: false, Message: storeErr.Error()}
		}
	}

	preset, err := m.SaveAsPreset(store, *server, userConfig)
	if err != nil {
		return McpMarketPresetResult{Success: false, Message: err.Error()}
	}
	return McpMarketPresetResult{Success: true, Message: "preset saved", PresetID: preset.ID}
}
