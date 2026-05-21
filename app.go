package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	goruntime "runtime"
	"time"

	"sync/atomic"

	"lurus-switch/internal/activity"
	"lurus-switch/internal/bashguard"
	"lurus-switch/internal/budget"
	"lurus-switch/internal/diagnostics"
	"lurus-switch/internal/hotkey"
	"lurus-switch/internal/livesession"
	"lurus-switch/internal/notify"
	"lurus-switch/internal/notify/rules"
	"lurus-switch/internal/packager"
	"lurus-switch/internal/repoaudit"
	"lurus-switch/internal/toolmanifest"
	"lurus-switch/internal/toolruntime"
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

// AuditRepo scans a project directory for AI-CLI config overrides that
// could indicate prompt-injection or credential-exfiltration vectors.
// Powers the "Repo Trust Audit" UI; the user reviews findings before
// launching any CLI inside the repo.
func (a *App) AuditRepo(path string) (*repoaudit.AuditReport, error) {
	if path == "" {
		return nil, fmt.Errorf("path is required")
	}
	return repoaudit.Audit(path)
}

// PickRepoAndAudit opens the native directory picker and immediately
// runs the audit on the chosen directory. Returns nil (no error) if the
// user cancels — the UI distinguishes "no result" from "error" by the
// presence of the report.
func (a *App) PickRepoAndAudit() (*repoaudit.AuditReport, error) {
	dir, err := wailsRuntime.OpenDirectoryDialog(a.ctx, wailsRuntime.OpenDialogOptions{
		Title: "Pick a project directory to audit",
	})
	if err != nil {
		return nil, err
	}
	if dir == "" {
		return nil, nil
	}
	return repoaudit.Audit(dir)
}

// QuarantineFile renames a file flagged by AuditRepo so the AI CLI no
// longer reads it. Returns the new path so the UI can show the user
// where the file went and how to restore it later.
func (a *App) QuarantineFile(path string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("path is required")
	}
	return repoaudit.Quarantine(path)
}

// ─── Budget Wall ────────────────────────────────────────────────────

// BudgetGetConfig returns the current spend-wall configuration.
func (a *App) BudgetGetConfig() budget.Config {
	if a.budgetGuard == nil {
		return budget.DefaultConfig()
	}
	return a.budgetGuard.GetConfig()
}

// BudgetSetConfig persists new limits and applies them immediately.
func (a *App) BudgetSetConfig(c budget.Config) error {
	if a.budgetGuard == nil {
		return fmt.Errorf("budget guard not initialised — gateway disabled?")
	}
	return a.budgetGuard.SetConfig(c)
}

// BudgetGetStatus returns current usage vs. limit for the UI gauge.
func (a *App) BudgetGetStatus() budget.Status {
	if a.budgetGuard == nil {
		return budget.Status{}
	}
	return a.budgetGuard.Status()
}

// BudgetResetSession zeroes the in-process session counter so the user
// can keep working after intentionally hitting the cap.
func (a *App) BudgetResetSession() {
	if a.budgetGuard == nil {
		return
	}
	a.budgetGuard.ResetSession()
}

// ─── Bash-Guard ─────────────────────────────────────────────────────

// BashGuardListRules returns the deny-list rules. UI uses this to show
// what Switch is willing to block.
func (a *App) BashGuardListRules() []*bashguard.Rule {
	return bashguard.DefaultRules()
}

// BashGuardTestCommand evaluates a command without installing/running
// any hook — pure preview so users can test their workflows without
// touching the live CLI integration.
func (a *App) BashGuardTestCommand(cmd string) (*bashguard.MatchResult, error) {
	eng, err := bashguard.NewEngine(bashguard.DefaultRules())
	if err != nil {
		return nil, err
	}
	r := eng.Evaluate(cmd)
	return &r, nil
}

// BashGuardClaudeStatus reports whether the PreToolUse hook is wired
// into the user's claude settings and (if so) what the hook command is.
func (a *App) BashGuardClaudeStatus() bashguard.HookInstallStatus {
	return bashguard.CheckClaudeHook(claudeSettingsPath())
}

// BashGuardInstallClaude wires the PreToolUse hook to call this very
// executable with --bashguard, so subsequent Claude Code shell tool
// invocations route through Switch's deny-list.
func (a *App) BashGuardInstallClaude() error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	hookCmd := fmt.Sprintf("%q --bashguard", exe)
	return bashguard.InstallClaudeHook(claudeSettingsPath(), hookCmd)
}

// BashGuardUninstallClaude removes only our hook entry, preserving any
// user-managed PreToolUse hooks.
func (a *App) BashGuardUninstallClaude() error {
	return bashguard.UninstallClaudeHook(claudeSettingsPath())
}

// BashGuardRecentBlocks returns the tail of the audit log so the UI
// can show what got blocked recently.
func (a *App) BashGuardRecentBlocks(max int) ([]bashguard.BlockEntry, error) {
	logPath := filepath.Join(appDataBaseDir(), "bashguard-blocks.jsonl")
	return bashguard.ReadRecentBlocks(logPath, max)
}

func claudeSettingsPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".claude", "settings.json")
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

// CheckBunInstalled checks if Bun is installed
func (a *App) CheckBunInstalled() bool {
	return packager.IsBunInstalled()
}

// CheckNodeInstalled checks if Node.js is installed
func (a *App) CheckNodeInstalled() bool {
	return packager.IsNodeInstalled()
}
