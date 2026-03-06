package main

import (
	"fmt"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"lurus-switch/internal/analytics"
	"lurus-switch/internal/docmgr"
	"lurus-switch/internal/envmgr"
	"lurus-switch/internal/snapshot"
	"lurus-switch/internal/toolconfig"
)

// ============================
// Document / Context File Methods (Phase G)
// ============================

// GetContextFile reads a tool's context file (e.g. CLAUDE.md)
func (a *App) GetContextFile(tool, scope string) (*docmgr.ContextFile, error) {
	return a.docMgr.GetContextFile(tool, scope, "")
}

// SaveContextFile writes a tool's context file
func (a *App) SaveContextFile(f *docmgr.ContextFile) error {
	return a.docMgr.SaveContextFile(f)
}

// OpenFolderAndScanContext opens a folder dialog and scans for context files
func (a *App) OpenFolderAndScanContext() ([]docmgr.ContextFile, error) {
	dir, err := runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Select Project Folder",
	})
	if err != nil {
		return nil, err
	}
	if dir == "" {
		return nil, fmt.Errorf("no directory selected")
	}
	return a.docMgr.ScanProjectDir(dir)
}

// ============================
// Snapshot Methods (Phase H.1)
// ============================

// TakeConfigSnapshot saves a snapshot of the current tool config file
func (a *App) TakeConfigSnapshot(tool, label string) error {
	if a.snapshotStr == nil {
		return fmt.Errorf("snapshot store not initialized")
	}
	info, err := toolconfig.ReadConfig(tool)
	if err != nil {
		return fmt.Errorf("failed to read config for snapshot: %w", err)
	}
	return a.snapshotStr.Take(tool, label, info.Content)
}

// ListConfigSnapshots returns all snapshots for a tool
func (a *App) ListConfigSnapshots(tool string) ([]snapshot.SnapshotMeta, error) {
	if a.snapshotStr == nil {
		return nil, fmt.Errorf("snapshot store not initialized")
	}
	return a.snapshotStr.List(tool)
}

// RestoreConfigSnapshot restores a snapshot to the tool's config file
func (a *App) RestoreConfigSnapshot(tool, id string) error {
	if a.snapshotStr == nil {
		return fmt.Errorf("snapshot store not initialized")
	}
	content, err := a.snapshotStr.Restore(tool, id)
	if err != nil {
		return err
	}
	return toolconfig.WriteConfig(tool, content)
}

// DeleteConfigSnapshot removes a snapshot
func (a *App) DeleteConfigSnapshot(tool, id string) error {
	if a.snapshotStr == nil {
		return fmt.Errorf("snapshot store not initialized")
	}
	return a.snapshotStr.Delete(tool, id)
}

// DiffConfigSnapshots returns a text diff between two snapshots
func (a *App) DiffConfigSnapshots(tool, id1, id2 string) (string, error) {
	if a.snapshotStr == nil {
		return "", fmt.Errorf("snapshot store not initialized")
	}
	return a.snapshotStr.Diff(tool, id1, id2)
}

// ClearAllSnapshots removes all config snapshots for all tools
func (a *App) ClearAllSnapshots() (int, error) {
	if a.snapshotStr == nil {
		return 0, fmt.Errorf("snapshot store not initialized")
	}
	return a.snapshotStr.ClearAll()
}

// ClearToolSnapshots removes all config snapshots for a specific tool
func (a *App) ClearToolSnapshots(tool string) (int, error) {
	if a.snapshotStr == nil {
		return 0, fmt.Errorf("snapshot store not initialized")
	}
	return a.snapshotStr.ClearTool(tool)
}

// ClearAllUserPrompts removes all user-created prompts (builtin prompts are unaffected)
func (a *App) ClearAllUserPrompts() (int, error) {
	if a.promptStr == nil {
		return 0, fmt.Errorf("prompt store not initialized")
	}
	return a.promptStr.ClearAllUser()
}

// ============================
// API Key Management (Phase H.3)
// ============================

// ListAllAPIKeys returns masked API keys found in tool configs
func (a *App) ListAllAPIKeys() ([]envmgr.KeyEntry, error) {
	tools := []string{"claude", "codex", "gemini", "picoclaw", "nullclaw"}
	return a.envMgr.ListAllKeys(tools)
}

// UpdateAPIKey updates an API key for a specific tool
func (a *App) UpdateAPIKey(tool, key, value string) error {
	return a.envMgr.UpdateKey(tool, key, value)
}

// ============================
// Tool Config File Methods
// ============================

// ReadToolConfig reads a tool's real config file from disk
func (a *App) ReadToolConfig(tool string) (*toolconfig.ToolConfigInfo, error) {
	return toolconfig.ReadConfig(tool)
}

// SaveToolConfig writes content to a tool's real config file.
// Before writing, it automatically takes an "auto-save" snapshot of the
// current on-disk content so users can always revert to the previous state.
func (a *App) SaveToolConfig(tool, content string) error {
	// Auto-snapshot the existing content before overwriting
	if a.snapshotStr != nil {
		if info, readErr := toolconfig.ReadConfig(tool); readErr == nil && info.Exists && info.Content != "" {
			_ = a.snapshotStr.Take(tool, "auto-save", info.Content)
		}
	}

	err := toolconfig.WriteConfig(tool, content)
	if err == nil && a.tracker != nil {
		_ = a.tracker.Record(analytics.Event{
			Tool: tool, Action: "config", Success: true,
		})
	}
	return err
}

// GetToolConfigPath returns the full path to a tool's config file
func (a *App) GetToolConfigPath(tool string) (string, error) {
	return toolconfig.GetConfigPath(tool)
}

// OpenToolConfigDir opens the config directory of a tool in the file explorer
func (a *App) OpenToolConfigDir(tool string) error {
	return toolconfig.OpenConfigDirectory(tool)
}

// GetAllToolConfigPaths returns the config file paths for all tools
func (a *App) GetAllToolConfigPaths() map[string]string {
	return toolconfig.GetAllConfigPaths()
}
