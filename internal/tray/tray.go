// Package tray manages the system tray icon, menu, and badge for Lurus Switch.
//
// Systray library: github.com/energye/systray (maintained fork of getlantern/systray).
//
// Threading contract:
//   - Windows: systray.Run blocks the calling goroutine; we start it in its own
//     OS-thread-locked goroutine via runtime.LockOSThread.
//   - macOS: systray.Run must run on the main thread. Callers must invoke
//     Manager.Start before any other UI framework locks the main thread, OR use
//     the RunWithExternalLoop / Register API with Wails' main-thread callback.
//     For Wails v2, pass the tray onReady into Wails' OnStartup and call
//     systray.Register from there if OS == darwin.
//
// Icon switching: NOT implemented — all badge tiers use the same icon.
// Badge state is reflected in the tooltip string only (see badge.go).
// To add multi-color icons: embed separate .ico files per tier and call
// systray.SetIcon in updateBadge.
package tray

import (
	"context"
	"runtime"
	"sync"
	"time"

	"github.com/energye/systray"
	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

const (
	// refreshInterval controls how often quota/gateway state is polled.
	refreshInterval = 10 * time.Second
)

// QuotaSnapshot captures usage for badge coloring.
type QuotaSnapshot struct {
	// UsedPercent is 0-100; negative means unknown.
	UsedPercent float64
	// BalanceText is a human-readable balance string, e.g. "¥42.50".
	BalanceText string
}

// GatewayStatus captures gateway health.
type GatewayStatus struct {
	Running bool
	Port    int
}

// Manager owns the systray lifecycle.
// All exported methods are nil-safe — calling them on a nil Manager is a no-op.
type Manager struct {
	quotaProvider   func() QuotaSnapshot
	gatewayProvider func() GatewayStatus

	ctx    context.Context
	cancel context.CancelFunc
	once   sync.Once
	stopWg sync.WaitGroup

	// mutable state (protected by mu)
	mu       sync.Mutex
	lastQuota QuotaSnapshot
	lastGW    GatewayStatus
}

// New builds a tray Manager.
// quotaProvider and gatewayProvider are called periodically on a background
// goroutine to refresh badge state. Either may be nil.
func New(quotaProvider func() QuotaSnapshot, gatewayProvider func() GatewayStatus) *Manager {
	return &Manager{
		quotaProvider:   quotaProvider,
		gatewayProvider: gatewayProvider,
		lastQuota:       QuotaSnapshot{UsedPercent: -1},
	}
}

// Start runs the tray event loop. It returns immediately; the tray runs on a
// dedicated goroutine. ctx must be the Wails app context so events can be
// emitted back to the frontend.
//
// On macOS, systray.Run requires the OS main thread. For Wails v2 + macOS,
// integrate via Wails OnStartup callback using systray.Register instead of
// calling Start directly. Start is correct and sufficient on Windows.
func (m *Manager) Start(ctx context.Context) {
	if m == nil {
		return
	}
	m.once.Do(func() {
		m.ctx, m.cancel = context.WithCancel(ctx)
		m.stopWg.Add(1)
		go func() {
			defer m.stopWg.Done()
			// LockOSThread keeps systray's Windows message pump on a stable OS thread.
			runtime.LockOSThread()
			systray.Run(m.onReady, m.onExit)
		}()
	})
}

// Stop terminates the tray. Idempotent.
func (m *Manager) Stop() {
	if m == nil || m.cancel == nil {
		return
	}
	m.cancel()
	systray.Quit()
	m.stopWg.Wait()
}

// onReady is called by systray once the tray icon is initialised.
// Runs on the systray goroutine — safe to call systray APIs here.
func (m *Manager) onReady() {
	// Set initial icon and tooltip.
	if iconICO != nil {
		systray.SetIcon(iconICO)
	}
	q0, gw0 := m.snapshot()
	systray.SetTooltip(buildTooltip(q0, gw0))

	// Build menu.
	mShow := systray.AddMenuItem("Show Window", "Show the main window")
	mHide := systray.AddMenuItem("Hide to Tray", "Minimise to system tray")
	systray.AddSeparator()
	mSwitchProvider := systray.AddMenuItem("Switch Provider...", "Open provider command palette")
	mGatewayToggle := systray.AddMenuItem("Toggle Gateway", "Start or stop the local gateway")
	systray.AddSeparator()
	mOpenConfig := systray.AddMenuItem("Open Config Folder", "Open app data directory")
	systray.AddSeparator()
	mQuit := systray.AddMenuItem("Quit", "Exit Lurus Switch")

	// Background: poll state and handle menu clicks.
	go m.loop(mShow, mHide, mSwitchProvider, mGatewayToggle, mOpenConfig, mQuit)
}

// onExit is called by systray when the tray is torn down.
func (m *Manager) onExit() {
	if m.cancel != nil {
		m.cancel()
	}
}

// loop handles menu clicks and periodic badge updates.
// Runs on a plain goroutine (not the OS-locked one).
func (m *Manager) loop(
	mShow, mHide, mSwitchProvider, mGatewayToggle, mOpenConfig, mQuit *systray.MenuItem,
) {
	ticker := time.NewTicker(refreshInterval)
	defer ticker.Stop()

	// Initial badge update.
	m.refreshState()
	m.updateTooltip()

	mShow.Click(func() {
		if m.ctx != nil {
			wailsRuntime.WindowShow(m.ctx)
		}
	})

	mHide.Click(func() {
		if m.ctx != nil {
			wailsRuntime.WindowHide(m.ctx)
		}
	})

	mSwitchProvider.Click(func() {
		if m.ctx != nil {
			wailsRuntime.EventsEmit(m.ctx, "tray:switch-provider", nil)
		}
	})

	mGatewayToggle.Click(func() {
		if m.ctx != nil {
			wailsRuntime.EventsEmit(m.ctx, "tray:gateway-toggle", nil)
		}
	})

	mOpenConfig.Click(func() {
		if m.ctx != nil {
			wailsRuntime.EventsEmit(m.ctx, "tray:open-config-dir", nil)
		}
	})

	mQuit.Click(func() {
		if m.ctx != nil {
			wailsRuntime.Quit(m.ctx)
		}
	})

	for {
		select {
		case <-m.ctx.Done():
			return
		case <-ticker.C:
			m.refreshState()
			m.updateTooltip()
		}
	}
}

// refreshState pulls fresh quota and gateway data from the providers.
func (m *Manager) refreshState() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.quotaProvider != nil {
		m.lastQuota = m.quotaProvider()
	}
	if m.gatewayProvider != nil {
		m.lastGW = m.gatewayProvider()
	}
}

// snapshot returns a thread-safe copy of the last known state.
func (m *Manager) snapshot() (QuotaSnapshot, GatewayStatus) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.lastQuota, m.lastGW
}

// updateTooltip applies the current badge state to the tray tooltip.
func (m *Manager) updateTooltip() {
	q, gw := m.snapshot()
	systray.SetTooltip(buildTooltip(q, gw))
}

