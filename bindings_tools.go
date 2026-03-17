package main

import (
	"fmt"

	"lurus-switch/internal/analytics"
	"lurus-switch/internal/installer"
	"lurus-switch/internal/toolhealth"
	"lurus-switch/internal/toolmanifest"
	"lurus-switch/internal/updater"

	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// ============================
// Tool Installation Methods
// ============================

// DetectAllTools checks installation status of all CLI tools
func (a *App) DetectAllTools() (map[string]*installer.ToolStatus, error) {
	return a.instMgr.DetectAll(a.ctx)
}

// InstallTool installs a specific CLI tool by name.
// Emits "tool:install:progress" events ({tool, percent}) during download
// and a "tool:install:done" event ({tool, success}) on completion.
func (a *App) InstallTool(name string) (*installer.InstallResult, error) {
	// Attach progress callback so the frontend can show a live progress bar.
	a.instMgr.SetProgressCallback(name, func(_, _ int64, pct int) {
		wailsRuntime.EventsEmit(a.ctx, "tool:install:progress", map[string]any{
			"tool": name, "percent": pct,
		})
	})
	defer a.instMgr.SetProgressCallback(name, nil) // clean up after install

	result, err := a.instMgr.InstallTool(a.ctx, name)
	wailsRuntime.EventsEmit(a.ctx, "tool:install:done", map[string]any{
		"tool":    name,
		"success": err == nil && result != nil && result.Success,
	})

	if a.tracker != nil && err == nil {
		if tErr := a.tracker.Record(analytics.Event{
			Tool: name, Action: "install", Success: result != nil && result.Success,
		}); tErr != nil {
			fmt.Printf("analytics tracking failed for install %s: %v\n", name, tErr)
		}
	}
	return result, err
}

// GetToolDownloadManifest returns the current tool download manifest.
// Falls back to the compile-time builtin if the background fetch has not yet completed.
func (a *App) GetToolDownloadManifest() (*toolmanifest.Manifest, error) {
	if a.manifest != nil {
		return a.manifest, nil
	}
	return toolmanifest.Builtin(), nil
}

// InstallAllTools installs all CLI tools sequentially
func (a *App) InstallAllTools() []installer.InstallResult {
	return a.instMgr.InstallAll(a.ctx)
}

// UpdateTool updates a specific CLI tool to the latest version
func (a *App) UpdateTool(name string) (*installer.InstallResult, error) {
	result, err := a.instMgr.UpdateTool(a.ctx, name)
	if a.tracker != nil && err == nil {
		if tErr := a.tracker.Record(analytics.Event{
			Tool: name, Action: "update", Success: result != nil && result.Success,
		}); tErr != nil {
			fmt.Printf("analytics tracking failed for update %s: %v\n", name, tErr)
		}
	}
	return result, err
}

// UpdateAllTools updates all CLI tools to the latest versions
func (a *App) UpdateAllTools() []installer.InstallResult {
	return a.instMgr.UpdateAll(a.ctx)
}

// UninstallTool uninstalls a specific CLI tool by name
func (a *App) UninstallTool(name string) (*installer.InstallResult, error) {
	result, err := a.instMgr.UninstallTool(a.ctx, name)
	if a.tracker != nil && err == nil {
		if tErr := a.tracker.Record(analytics.Event{
			Tool: name, Action: "uninstall", Success: result != nil && result.Success,
		}); tErr != nil {
			fmt.Printf("analytics tracking failed for uninstall %s: %v\n", name, tErr)
		}
	}
	return result, err
}

// UninstallAllTools uninstalls all CLI tools
func (a *App) UninstallAllTools() []installer.InstallResult {
	return a.instMgr.UninstallAll(a.ctx)
}

// ============================
// Tool Health Check Methods
// ============================

// CheckToolHealth performs a configuration health check on a single tool
func (a *App) CheckToolHealth(tool string) *toolhealth.HealthResult {
	return toolhealth.CheckTool(tool)
}

// CheckAllToolHealth performs health checks on all known tools
func (a *App) CheckAllToolHealth() map[string]*toolhealth.HealthResult {
	return toolhealth.CheckAll()
}

// ============================
// Update Check Methods
// ============================

// CheckAllUpdates checks for updates on all installed CLI tools.
// It first tries to fetch latest versions from the Lurus cloud endpoint (single request),
// falling back to individual npm registry checks when the cloud endpoint is unavailable.
func (a *App) CheckAllUpdates() map[string]*updater.UpdateInfo {
	statuses, _ := a.instMgr.DetectAll(a.ctx)
	toolVersions := make(map[string]string)
	for name, status := range statuses {
		if status.Installed {
			toolVersions[name] = status.Version
		}
	}

	// Try cloud endpoint first when proxy is configured
	if a.proxyMgr != nil {
		settings := a.proxyMgr.GetSettings()
		if settings.APIEndpoint != "" {
			cloudChecker := updater.NewCloudVersionChecker(settings.APIEndpoint)
			cloudVersions, err := cloudChecker.FetchAllVersions(a.ctx)
			if err == nil && len(cloudVersions) > 0 {
				results := make(map[string]*updater.UpdateInfo)
				for toolName, current := range toolVersions {
					latest, ok := cloudVersions[toolName]
					if !ok || latest == "" {
						latest = "unknown"
					}
					results[toolName] = &updater.UpdateInfo{
						Name:            toolName,
						CurrentVersion:  current,
						LatestVersion:   latest,
						UpdateAvailable: latest != "unknown" && current != "" && latest != current && updater.IsNewerVersion(latest, current),
					}
				}
				return results
			}
		}
	}

	// Fallback to individual npm registry checks
	return a.npmChecker.CheckAllTools(toolVersions)
}

// CheckSelfUpdate checks if a newer version of Switch is available
func (a *App) CheckSelfUpdate() (*updater.UpdateInfo, error) {
	return a.selfUpdater.CheckUpdate()
}

// ApplySelfUpdate downloads and applies the latest Switch update
func (a *App) ApplySelfUpdate() error {
	return a.selfUpdater.ApplyUpdate()
}

// GetAppVersion returns the current Switch version string
func (a *App) GetAppVersion() string {
	return AppVersion
}

// ============================
// Dependency Management Methods
// ============================

// CheckDependencies returns the full runtime dependency tree status
func (a *App) CheckDependencies() (*installer.DepCheckResult, error) {
	return a.instMgr.CheckDependencies(a.ctx)
}

// InstallDependency installs a single runtime dependency by ID (e.g. "nodejs", "bun")
func (a *App) InstallDependency(runtimeID string) (*installer.DepInstallResult, error) {
	result, err := a.instMgr.InstallDependency(a.ctx, runtimeID)
	if a.tracker != nil && err == nil {
		if tErr := a.tracker.Record(analytics.Event{
			Tool: "runtime:" + runtimeID, Action: "install", Success: result != nil && result.Success,
		}); tErr != nil {
			fmt.Printf("analytics tracking failed for dependency install %s: %v\n", runtimeID, tErr)
		}
	}
	return result, err
}

// ============================
// Bun Runtime Methods
// ============================

// InstallBun installs the Bun runtime and returns its path
func (a *App) InstallBun() (string, error) {
	return a.instMgr.GetRuntime().InstallBun(a.ctx)
}
