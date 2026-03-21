package main

import (
	"context"
	"fmt"
	"os"
	goruntime "runtime"

	"sync/atomic"

	"lurus-switch/internal/packager"
	"lurus-switch/internal/toolmanifest"

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

	// Fetch tool download manifest in the background so it is ready before the
	// user reaches the install step. Does not block startup.
	go safeGo("refresh-manifest", func() { a.refreshManifest() })
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
