package main

import (
	"fmt"
	"os"
	"path/filepath"

	"lurus-switch/internal/mcp"
)

// ============================
// MCP Server Methods (Phase D)
// ============================

// ListMCPPresets returns user-saved MCP presets
func (a *App) ListMCPPresets() ([]mcp.MCPPreset, error) {
	if a.mcpStr == nil {
		return mcp.BuiltinPresets(), nil
	}
	return a.mcpStr.ListPresets()
}

// SaveMCPPreset persists a user MCP preset
func (a *App) SaveMCPPreset(p mcp.MCPPreset) error {
	if a.mcpStr == nil {
		return fmt.Errorf("mcp store not initialized")
	}
	return a.mcpStr.SavePreset(p)
}

// DeleteMCPPreset removes a user MCP preset by ID
func (a *App) DeleteMCPPreset(id string) error {
	if a.mcpStr == nil {
		return fmt.Errorf("mcp store not initialized")
	}
	return a.mcpStr.DeletePreset(id)
}

// GetBuiltinMCPPresets returns the bundled MCP server presets
func (a *App) GetBuiltinMCPPresets() []mcp.MCPPreset {
	return mcp.BuiltinPresets()
}

// ApplyMCPServerToTool upserts an MCP server entry into a tool's settings file
func (a *App) ApplyMCPServerToTool(tool string, server mcp.MCPServer) error {
	return applyMCPToTool(tool, server)
}

// GetClaudeHooks reads the hooks section from ~/.claude/settings.json
func (a *App) GetClaudeHooks() (map[string]interface{}, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	return readJSONSection(filepath.Join(home, ".claude", "settings.json"), "hooks")
}

// SaveClaudeHooks writes the hooks section to ~/.claude/settings.json
func (a *App) SaveClaudeHooks(hooks map[string]interface{}) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	return writeJSONSection(filepath.Join(home, ".claude", "settings.json"), "hooks", hooks)
}
