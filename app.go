package main

import (
	"context"
	"fmt"
	"log"
	"os"
	goruntime "runtime"
	"time"

	"sync/atomic"

	"lurus-switch/internal/hotkey"
	"lurus-switch/internal/packager"
	"lurus-switch/internal/toolmanifest"
	"lurus-switch/internal/tray"

	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// AppVersion is the current version of Lurus Switch, set at build time via -ldflags
var AppVersion = "0.1.0"

// SystemInfo contains runtime information about the host system
type SystemInfo struct {
	AppVersion string `json:"appVersion"`
	GOOS       string `json:"goos"`
	GOARCH     string `json:"goarch"`
}

// App struct — Wails-bound application.
// Service dependencies live in the embedded *services struct (services.go),
// keeping App focused on Wails lifecycle and manifest coordination.
type App struct {
	ctx context.Context

	// All service dependencies (config, installer, billing, etc.)
	*services

	// Tool download manifest (loaded in background at startup).
	// Accessed from both the background goroutine and the Wails UI thread,
	// so atomic.Pointer is used to avoid data races.
	manifest atomic.Pointer[toolmanifest.Manifest]

	// Desktop-only services: tray badge + global hotkeys.
	// Initialized in startup(), stopped in shutdown(). Nil-safe.
	trayMgr   *tray.Manager
	hotkeyMgr *hotkey.Manager
}

// NewApp creates a new App application struct
func NewApp() *App {
	svc, warnings := newServices(appDataBaseDir(), AppVersion)
	for _, w := range warnings {
		fmt.Printf("Warning: %s\n", w)
	}
	return &App{services: svc}
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx

	// Auto-start legacy gateway server if configured to do so.
	if a.serverMgr != nil {
		if cfg := a.serverMgr.GetConfig(); cfg.AutoStart {
			go safeGo("legacy-gateway-start", func() {
				if err := a.serverMgr.Start(ctx); err != nil {
					fmt.Printf("Warning: auto-start legacy gateway server failed: %v\n", err)
				}
			})
		}
	}

	// Auto-start the new local API gateway if configured.
	if a.gatewaySrv != nil {
		// Register crash recovery callback to notify the frontend.
		a.gatewaySrv.SetCrashCallback(func(attempt int, err error) {
			wailsRuntime.EventsEmit(ctx, "gateway:crash", map[string]any{
				"attempt": attempt,
				"error":   err.Error(),
			})
		})
		a.syncGatewayUpstream()
		if cfg := a.gatewaySrv.GetConfig(); cfg.AutoStart {
			go safeGo("gateway-start", func() {
				if err := a.gatewaySrv.Start(ctx); err != nil {
					fmt.Printf("Warning: auto-start gateway failed: %v\n", err)
				}
			})
		}
	}

	// Migrate legacy proxy settings to relay store (one-time, idempotent).
	a.migrateProxyToRelay()

	// Sync tool connection status from actual config files (non-blocking).
	go safeGo("sync-tool-status", func() { a.SyncToolConnectionStatus() })

	// Reconcile stale agent statuses from a previous unclean shutdown.
	if a.agentInstMgr != nil {
		go safeGo("agent-status-sync", func() { a.agentInstMgr.SyncStatuses() })
	}

	// Fetch tool download manifest in the background so it is ready before the
	// user reaches the install step. Does not block startup.
	go safeGo("refresh-manifest", func() { a.refreshManifest() })

	// Tray: surface quota + gateway status in the system tray.
	a.trayMgr = tray.New(a.trayQuotaSnapshot, a.trayGatewayStatus)
	a.trayMgr.Start(ctx)

	// Global hotkeys: quick-switch + show-window from anywhere.
	a.hotkeyMgr = hotkey.New(appDataBaseDir(), func(binding string) {
		wailsRuntime.WindowShow(ctx)
		wailsRuntime.EventsEmit(ctx, "hotkey:"+binding)
	})
	for _, e := range a.hotkeyMgr.Start(ctx) {
		log.Printf("hotkey registration failed: binding=%s shortcut=%s err=%v", e.Binding, e.Shortcut, e.Err)
	}
}

// trayQuotaSnapshot is the tray's quota-usage provider. Returns UsedPercent = -1
// when the billing client is unavailable so the tray can render an "unknown" tier.
func (a *App) trayQuotaSnapshot() tray.QuotaSnapshot {
	client, err := a.ensureBillingClient()
	if err != nil || client == nil {
		return tray.QuotaSnapshot{UsedPercent: -1}
	}
	ctx, cancel := context.WithTimeout(a.ctx, 5*time.Second)
	defer cancel()
	info, err := client.GetUserInfo(ctx)
	if err != nil || info == nil || info.Quota == 0 {
		return tray.QuotaSnapshot{UsedPercent: -1}
	}
	pct := float64(info.UsedQuota) / float64(info.Quota) * 100
	return tray.QuotaSnapshot{UsedPercent: pct}
}

// trayGatewayStatus is the tray's gateway-status provider.
func (a *App) trayGatewayStatus() tray.GatewayStatus {
	if a.gatewaySrv == nil {
		return tray.GatewayStatus{}
	}
	s := a.gatewaySrv.Status()
	return tray.GatewayStatus{Running: s.Running, Port: s.Port}
}

// shutdown is called when the Wails app is closing.
// It gracefully stops all running services and releases resources.
func (a *App) shutdown(ctx context.Context) {
	// Stop tray + hotkey first so menus/hotkeys stop firing callbacks
	// against a teardown-in-progress app.
	if a.trayMgr != nil {
		a.trayMgr.Stop()
	}
	if a.hotkeyMgr != nil {
		a.hotkeyMgr.Stop()
	}

	// Stop local API gateway (flushes metering buffer).
	if a.gatewaySrv != nil {
		if err := a.gatewaySrv.Stop(); err != nil {
			fmt.Fprintf(os.Stderr, "shutdown: gateway stop failed: %v\n", err)
		}
	}

	// Stop legacy gateway server.
	if a.serverMgr != nil {
		if err := a.serverMgr.Stop(); err != nil {
			fmt.Fprintf(os.Stderr, "shutdown: legacy server stop failed: %v\n", err)
		}
	}

	// Stop all running agent instances.
	if a.agentInstMgr != nil {
		for _, id := range a.agentInstMgr.RunningAgentIDs() {
			if err := a.agentInstMgr.Stop(id); err != nil {
				fmt.Fprintf(os.Stderr, "shutdown: agent %s stop failed: %v\n", id, err)
			}
		}
	}

	// Close database connection.
	if a.database != nil {
		if err := a.database.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "shutdown: database close failed: %v\n", err)
		}
	}
}

// safeGo wraps a function with panic recovery so goroutines never crash the app.
func safeGo(label string, fn func()) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Fprintf(os.Stderr, "PANIC [%s]: %v\n", label, r)
		}
	}()
	fn()
}

// refreshManifest fetches the latest tool manifest from the configured API endpoint,
// falling back to a stale cache and then the compile-time builtin on failure.
func (a *App) refreshManifest() {
	apiBase := ""
	if a.proxyMgr != nil {
		apiBase = a.proxyMgr.GetSettings().APIEndpoint
	}
	mf, err := toolmanifest.Fetch(a.ctx, apiBase, appDataBaseDir())
	if err != nil {
		mf = toolmanifest.Builtin()
	}
	a.manifest.Store(mf)
	if a.instMgr != nil {
		a.instMgr.SetManifest(mf)
	}
}

// loadManifest returns the current manifest, falling back to the compile-time builtin.
func (a *App) loadManifest() *toolmanifest.Manifest {
	if mf := a.manifest.Load(); mf != nil {
		return mf
	}
	return toolmanifest.Builtin()
}

// GetSystemInfo returns basic system information
func (a *App) GetSystemInfo() *SystemInfo {
	return &SystemInfo{
		AppVersion: AppVersion,
		GOOS:       goruntime.GOOS,
		GOARCH:     goruntime.GOARCH,
	}
}

// GetConfigDir returns the configuration directory path
func (a *App) GetConfigDir() string {
	if a.store == nil {
		return ""
	}
	return a.store.GetConfigDir()
}

// OpenConfigDir opens the configuration directory in the file explorer
func (a *App) OpenConfigDir() error {
	if a.store == nil {
		return fmt.Errorf("config store not initialized")
	}
	dir := a.store.GetConfigDir()
	return openDirectory(dir)
}

// CheckBunInstalled checks if Bun is installed
func (a *App) CheckBunInstalled() bool {
	return packager.IsBunInstalled()
}

// CheckNodeInstalled checks if Node.js is installed
func (a *App) CheckNodeInstalled() bool {
	return packager.IsNodeInstalled()
}
