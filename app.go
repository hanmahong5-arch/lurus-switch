package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"sync/atomic"

	"lurus-switch/internal/activity"
	"lurus-switch/internal/diagnostics"
	"lurus-switch/internal/hotkey"
	"lurus-switch/internal/livesession"
	"lurus-switch/internal/notify"
	"lurus-switch/internal/notify/rules"
	"lurus-switch/internal/toolmanifest"
	"lurus-switch/internal/tray"

	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// AppVersion is the current version of Lurus Switch, set at build time via -ldflags
var AppVersion = "0.1.0"

// App is the Wails-bound application surface. The struct stays minimal —
// service dependencies live in the embedded *services value (services.go),
// and Wails methods are split across bindings_*.go files by subsystem.
// Only lifecycle (startup/shutdown), shared pointers, and panic-safe
// goroutine helpers live here.
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

	// Live activity bus — emits Wails events so the UI's "what is Switch
	// doing right now" panel stays in sync with long-running operations.
	activityBus *activity.Bus

	// Live-session watcher: polls Claude/Codex/Gemini transcript JSONLs
	// and feeds the "Claude is doing X right now" view. Nil-safe — the
	// binding falls back to an empty slice when watcher hasn't started.
	liveWatcher *livesession.Watcher

	// Outbound notification bus + rules engine — surface long-running /
	// stuck / done events to Feishu (and future transports). Both nil
	// while disabled in user prefs; bindings_notify.go gates access.
	notifyBus    *notify.Bus
	notifyEngine *rules.Engine
}

// NewApp creates a new App application struct
func NewApp() *App {
	svc, warnings := newServices(appDataBaseDir(), AppVersion)
	for _, w := range warnings {
		fmt.Printf("Warning: %s\n", w)
	}
	return &App{services: svc, activityBus: activity.New()}
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	if a.activityBus != nil {
		a.activityBus.Bind(ctx)
	}
	diagnostics.Default.Mark("activity-bus")

	// Wire undo handlers for state-mutating Wails bindings. Has to run
	// after services are constructed (the journal lives there) but
	// before any user interaction can record entries.
	a.registerAuditUndoHandlers()
	diagnostics.Default.Mark("audit-undo-handlers")

	// White-label sidecar: if a signed whitelabel.json sits next to the
	// running exe, lock the app to the embedded Hub URL + EndUser mode.
	// Tampered sidecars cause a hard refusal — better to fail loudly than
	// to let an EndUser silently fall back to Personal mode and dial home
	// to hub.lurus.cn.
	a.applyWhiteLabelSidecar()
	diagnostics.Default.Mark("whitelabel-sidecar")

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
	diagnostics.Default.Mark("gateway-autostart")

	// Sync tool connection status from actual config files (non-blocking).
	go safeGo("sync-tool-status", func() { a.SyncToolConnectionStatus() })

	// Reconcile stale agent statuses from a previous unclean shutdown.
	// Failure is logged but never blocks startup — a stale "running"
	// status surfaces as a stuck UI badge, not a crash.
	if a.agentInstMgr != nil {
		go safeGo("agent-status-sync", func() {
			if err := a.agentInstMgr.SyncStatuses(); err != nil {
				log.Printf("agent: SyncStatuses on startup: %v", err)
			}
		})
	}

	// Fetch tool download manifest in the background so it is ready before the
	// user reaches the install step. Does not block startup.
	go safeGo("refresh-manifest", func() { a.refreshManifest() })

	// Conversation index: walk the CLI session directories so the
	// Conversations page is populated by the time the user navigates
	// there. mtime-driven incremental — cheap on subsequent boots.
	if a.conversationIndex != nil {
		go safeGo("conversation-reindex", func() { a.conversationIndex.Rebuild() })
	}

	// EndUser heartbeat: probe Hub liveness so revoked tokens evict within
	// minutes. No-op when no activation file is on disk; safe to start in
	// any mode (Personal/Reseller users have no activation, so the loop
	// just exits early on each tick).
	if a.redemptionStore != nil {
		a.restartHeartbeatLocked()
	}
	diagnostics.Default.Mark("heartbeat-init")

	// Live-session watcher: polls Claude/Codex/Gemini transcripts on disk
	// and pushes "livesession:update" events whenever state changes so
	// the Live Inspector page stays in sync without explicit polling.
	a.liveWatcher = livesession.New(func() {
		wailsRuntime.EventsEmit(ctx, "livesession:update")
	})
	a.liveWatcher.Start()
	diagnostics.Default.Mark("live-watcher")

	// Notify subsystem — opt-in remote push (Feishu first). Wired only
	// when user enabled it AND filled in a webhook URL, so the rules
	// engine isn't burning ticks on a no-op fan-out.
	a.startNotifySubsystem()
	diagnostics.Default.Mark("notify-subsystem")

	// Tray: surface quota + gateway status in the system tray.
	a.trayMgr = tray.New(a.trayQuotaSnapshot, a.trayGatewayStatus)
	a.trayMgr.SetRelayProvider(&appRelayProvider{app: a})
	a.trayMgr.Start(ctx)
	diagnostics.Default.Mark("tray-start")

	// Global hotkeys: quick-switch + show-window from anywhere.
	a.hotkeyMgr = hotkey.New(appDataBaseDir(), func(binding string) {
		wailsRuntime.WindowShow(ctx)
		wailsRuntime.EventsEmit(ctx, "hotkey:"+binding)
	})
	for _, e := range a.hotkeyMgr.Start(ctx) {
		log.Printf("hotkey registration failed: binding=%s shortcut=%s err=%v", e.Binding, e.Shortcut, e.Err)
	}
	diagnostics.Default.Mark("hotkey-start")
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
	if a.notifyEngine != nil {
		a.notifyEngine.Stop()
	}
	if a.liveWatcher != nil {
		a.liveWatcher.Stop()
	}
	if a.heartbeat != nil {
		a.heartbeat.Stop()
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
