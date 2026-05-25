package main

import (
	"fmt"

	"lurus-switch/internal/analytics"
	"lurus-switch/internal/installer"
	"lurus-switch/internal/packager"
	"lurus-switch/internal/toolhealth"
	"lurus-switch/internal/toolmanifest"
	"lurus-switch/internal/toolruntime"
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
	// Activity bus — drives the live "what is Switch doing" panel. The
	// existing tool:install:progress events are kept for back-compat with
	// the per-tool progress bar in the Tools card.
	op := a.activityBus.Op("install-"+name, "安装 "+name, "Installing "+name)

	a.instMgr.SetProgressCallback(name, func(_, _ int64, pct int) {
		wailsRuntime.EventsEmit(a.ctx, "tool:install:progress", map[string]any{
			"tool": name, "percent": pct,
		})
		op.Progress("下载中…", "Downloading…", pct, 0, 0)
	})
	defer a.instMgr.SetProgressCallback(name, nil) // clean up after install

	result, err := a.instMgr.InstallTool(a.ctx, name)
	wailsRuntime.EventsEmit(a.ctx, "tool:install:done", map[string]any{
		"tool":    name,
		"success": err == nil && result != nil && result.Success,
	})
	if err != nil {
		op.Error(err.Error())
	} else if result != nil && !result.Success {
		op.Error(result.Message)
	} else {
		op.Done("已安装", "Installed")
	}

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
	return a.loadManifest(), nil
}

// InstallAllTools installs all CLI tools sequentially.
// Each tool emits "tool:install:progress" and "tool:install:done" events
// so the frontend can display live progress bars.
func (a *App) InstallAllTools() []installer.InstallResult {
	toolOrder := []string{"claude", "codex", "gemini", "picoclaw", "nullclaw", "zeroclaw", "openclaw"}
	parent := a.activityBus.Op("install-all", "安装所有 CLI 工具", "Installing all CLI tools")
	var results []installer.InstallResult
	for i, name := range toolOrder {
		parent.Progress("正在装 "+name, "Installing "+name, (i*100)/len(toolOrder), len(toolOrder), i+1)
		result, err := a.InstallTool(name)
		if err != nil {
			results = append(results, installer.InstallResult{
				Tool:    name,
				Success: false,
				Message: fmt.Sprintf("install failed: %v", err),
			})
			continue
		}
		results = append(results, *result)
	}
	parent.Done(fmt.Sprintf("%d/%d 安装完成", len(results), len(toolOrder)),
		fmt.Sprintf("%d/%d installed", len(results), len(toolOrder)))
	return results
}

// UpdateTool updates a specific CLI tool to the latest version
func (a *App) UpdateTool(name string) (*installer.InstallResult, error) {
	op := a.activityBus.Op("update-"+name, "更新 "+name, "Updating "+name)
	result, err := a.instMgr.UpdateTool(a.ctx, name)
	if err != nil {
		op.Error(err.Error())
	} else if result != nil && !result.Success {
		op.Error(result.Message)
	} else {
		op.Done("已更新", "Updated")
	}
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
	op := a.activityBus.Op("update-all", "更新所有工具", "Updating all tools")
	results := a.instMgr.UpdateAll(a.ctx)
	op.Done(fmt.Sprintf("%d 项已处理", len(results)), fmt.Sprintf("%d items processed", len(results)))
	return results
}

// UninstallTool uninstalls a specific CLI tool by name
func (a *App) UninstallTool(name string) (*installer.InstallResult, error) {
	op := a.activityBus.Op("uninstall-"+name, "卸载 "+name, "Uninstalling "+name)
	result, err := a.instMgr.UninstallTool(a.ctx, name)
	if err != nil {
		op.Error(err.Error())
	} else if result != nil && !result.Success {
		op.Error(result.Message)
	} else {
		op.Done("已卸载", "Uninstalled")
	}
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
	op := a.activityBus.Op("uninstall-all", "卸载所有工具", "Uninstalling all tools")
	results := a.instMgr.UninstallAll(a.ctx)
	op.Done(fmt.Sprintf("%d 项已处理", len(results)), fmt.Sprintf("%d items processed", len(results)))
	return results
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

// CheckBunInstalled checks if Bun is installed
func (a *App) CheckBunInstalled() bool {
	return packager.IsBunInstalled()
}

// CheckNodeInstalled checks if Node.js is installed
func (a *App) CheckNodeInstalled() bool {
	return packager.IsNodeInstalled()
}

// GetToolRuntimes returns a snapshot of every supported CLI's live
// state — endpoint, model, process status, reachability — for the
// "Runtime Status" panel on Home. Probes run concurrently with a 3s
// per-host timeout, so the call settles in ~3s in the worst case.
func (a *App) GetToolRuntimes() []toolruntime.ToolRuntime {
	// Collect running PIDs once and pass them in so each tool probe
	// doesn't re-shell-out to enumerate processes.
	runningPIDs := map[string]int{}
	if a.processMon != nil {
		if procs, err := a.processMon.ListCLIProcesses(a.ctx); err == nil {
			for _, p := range procs {
				if p.PID > 0 && runningPIDs[p.Tool] == 0 {
					runningPIDs[p.Tool] = p.PID
				}
			}
		}
	}
	gwPort := 0
	if a.gatewaySrv != nil {
		gwPort = a.gatewaySrv.GetConfig().Port
	}
	return toolruntime.ProbeAll(a.ctx, toolruntime.ProbeOptions{
		RunningPIDs: runningPIDs,
		GatewayPort: gwPort,
	})
}
