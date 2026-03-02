package main

import (
	"context"
	"fmt"
	goruntime "runtime"

	"sync"

	"lurus-switch/internal/analytics"
	"lurus-switch/internal/billing"
	"lurus-switch/internal/config"
	"lurus-switch/internal/docmgr"
	"lurus-switch/internal/envmgr"
	"lurus-switch/internal/installer"
	"lurus-switch/internal/mcp"
	"lurus-switch/internal/packager"
	"lurus-switch/internal/process"
	"lurus-switch/internal/promptlib"
	"lurus-switch/internal/proxy"
	"lurus-switch/internal/serverctl"
	"lurus-switch/internal/snapshot"
	"lurus-switch/internal/updater"
	"lurus-switch/internal/validator"
)

// AppVersion is the current version of Lurus Switch, set at build time via -ldflags
var AppVersion = "0.1.0"

// SystemInfo contains runtime information about the host system
type SystemInfo struct {
	AppVersion string `json:"appVersion"`
	GOOS       string `json:"goos"`
	GOARCH     string `json:"goarch"`
}

// App struct
type App struct {
	ctx           context.Context
	store         *config.Store
	validator     *validator.Validator
	instMgr       *installer.Manager
	proxyMgr      *proxy.ProxyManager
	selfUpdater   *updater.SelfUpdater
	npmChecker    *updater.NpmChecker
	billingMu     sync.Mutex
	billingClient *billing.Client

	// New components added in Phase A-I
	processMon   *process.Monitor
	snapshotStr  *snapshot.Store
	promptStr    *promptlib.Store
	mcpStr       *mcp.Store
	docMgr       *docmgr.Manager
	envMgr       *envmgr.Manager
	tracker      *analytics.Tracker

	// Embedded gateway server manager
	serverMgr *serverctl.Manager
}

// NewApp creates a new App application struct
func NewApp() *App {
	store, err := config.NewStore()
	if err != nil {
		fmt.Printf("Warning: failed to initialize config store: %v\n", err)
	}

	proxyMgr, _ := proxy.NewProxyManager()

	snapStr, err := snapshot.NewStore()
	if err != nil {
		fmt.Printf("Warning: failed to initialize snapshot store: %v\n", err)
	}

	promptStr, err := promptlib.NewStore()
	if err != nil {
		fmt.Printf("Warning: failed to initialize prompt store: %v\n", err)
	}

	mcpStr, err := mcp.NewStore()
	if err != nil {
		fmt.Printf("Warning: failed to initialize mcp store: %v\n", err)
	}

	tracker, err := analytics.NewTracker()
	if err != nil {
		fmt.Printf("Warning: failed to initialize analytics tracker: %v\n", err)
	}

	appDataDir := appDataBaseDir()
	svrMgr := serverctl.NewManager(appDataDir)

	return &App{
		store:       store,
		validator:   validator.NewValidator(),
		instMgr:     installer.NewManager(),
		proxyMgr:    proxyMgr,
		selfUpdater: updater.NewSelfUpdater(AppVersion),
		npmChecker:  updater.NewNpmChecker(),
		processMon:  process.NewMonitor(),
		snapshotStr: snapStr,
		promptStr:   promptStr,
		mcpStr:      mcpStr,
		docMgr:      docmgr.NewManager(),
		envMgr:      envmgr.NewManager(),
		tracker:     tracker,
		serverMgr:   svrMgr,
	}
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx

	// Auto-start gateway server if configured to do so.
	if a.serverMgr != nil {
		if cfg := a.serverMgr.GetConfig(); cfg.AutoStart {
			go func() {
				if err := a.serverMgr.Start(ctx); err != nil {
					fmt.Printf("Warning: auto-start gateway server failed: %v\n", err)
				}
			}()
		}
	}
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
