package main

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"lurus-switch/internal/notify"
	"lurus-switch/internal/notify/feishu"
	"lurus-switch/internal/notify/rules"
	"lurus-switch/internal/notify/slack"
	"lurus-switch/internal/notify/store"
	"lurus-switch/internal/notify/telegram"
)

// ============================
// Notify Bindings
// ============================
//
// The notify subsystem (internal/notify) pushes events to remote surfaces
// like Feishu when long-running tools get stuck or sessions complete.
// These bindings expose: read current prefs, save+rewire prefs, fire a
// synthetic event for connection testing, and surface the recent-ring
// for the Settings UI's "what got pushed lately" panel.
//
// The subsystem is opt-in (cfg.Enabled=false by default); when disabled
// the bus and engine are nil and TestNotify returns an explanatory error
// instead of silently dropping the test event.

// notifyRebuildMu serialises startup wiring + SaveNotifyConfig + TestNotify
// so a save mid-test doesn't swap the bus pointer under TestNotify's feet.
var notifyRebuildMu sync.Mutex

// startNotifySubsystem reads notify.json from appDataBaseDir and, if the
// user enabled the feature, builds the Bus + transports + rules Engine.
// Safe to call from startup goroutines — guarded by notifyRebuildMu.
func (a *App) startNotifySubsystem() {
	notifyRebuildMu.Lock()
	defer notifyRebuildMu.Unlock()

	cfg, err := store.Load(appDataBaseDir())
	if err != nil {
		log.Printf("notify: load config failed (using defaults): %v", err)
		cfg = store.DefaultAppConfig()
	}
	a.rebuildNotifyLocked(cfg)
}

// rebuildNotifyLocked tears down any previous bus + engine and rebuilds
// them from cfg. Caller holds notifyRebuildMu.
func (a *App) rebuildNotifyLocked(cfg store.AppConfig) {
	// Tear down the previous engine before swapping pointers — otherwise
	// the old loop keeps publishing into a stale bus that no transport is
	// listening to.
	if a.notifyEngine != nil {
		a.notifyEngine.Stop()
		a.notifyEngine = nil
	}

	if !cfg.Enabled {
		// Disabled → drop the bus too so GetRecentNotifications returns
		// nothing and TestNotify fails fast with a clear message.
		a.notifyBus = nil
		return
	}

	bus := notify.NewBus()
	if cfg.Feishu.WebhookURL != "" {
		bus.Register(feishu.New(cfg.Feishu))
	}
	if cfg.Telegram.BotToken != "" && cfg.Telegram.ChatID != "" {
		bus.Register(telegram.New(cfg.Telegram))
	}
	if cfg.Slack.WebhookURL != "" {
		bus.Register(slack.New(cfg.Slack))
	}
	a.notifyBus = bus

	// Only spin the rules engine when the watcher exists — otherwise the
	// engine would tick against a nil Snapshotter and panic on the first
	// loop iteration. Watcher is nil before startup() completes, which
	// matters only in tests since this function runs after watcher.Start().
	if a.liveWatcher != nil {
		eng := rules.NewEngine(a.liveWatcher, bus, cfg.Rules.ToRulesConfig())
		eng.Start()
		a.notifyEngine = eng
	}
}

// GetNotifyConfig returns the persisted notify preferences. On read error
// returns defaults so the UI form never blocks on a stale or missing file.
func (a *App) GetNotifyConfig() store.AppConfig {
	cfg, err := store.Load(appDataBaseDir())
	if err != nil {
		log.Printf("notify: GetNotifyConfig load failed (returning defaults): %v", err)
		return store.DefaultAppConfig()
	}
	return cfg
}

// SaveNotifyConfig validates → persists → rewires the bus + engine. Any
// validation failure returns before touching disk so the user keeps their
// last good config.
func (a *App) SaveNotifyConfig(cfg store.AppConfig) error {
	// Validate each transport's config when the feature is on and that
	// transport has been touched. An empty block while Enabled=true is
	// allowed — it lets the user toggle Enabled on as a precursor to
	// filling in credentials, and lets them run any subset of transports.
	if cfg.Enabled {
		if cfg.Feishu.WebhookURL != "" {
			if err := cfg.Feishu.Validate(); err != nil {
				return err
			}
		}
		// Either Telegram field being set means the user is configuring it;
		// Validate then surfaces "the other field is required".
		if cfg.Telegram.BotToken != "" || cfg.Telegram.ChatID != "" {
			if err := cfg.Telegram.Validate(); err != nil {
				return err
			}
		}
		if cfg.Slack.WebhookURL != "" {
			if err := cfg.Slack.Validate(); err != nil {
				return err
			}
		}
	}
	if err := store.Save(appDataBaseDir(), cfg); err != nil {
		return fmt.Errorf("save notify config: %w", err)
	}
	notifyRebuildMu.Lock()
	defer notifyRebuildMu.Unlock()
	a.rebuildNotifyLocked(cfg)
	return nil
}

// TestNotify publishes a synthetic KindTest event so the user can verify
// their webhook is wired up before relying on it for real alerts. Returns
// an error describing what went wrong (no bus / no transport / transport
// delivery failure) so the UI can render a useful inline message.
func (a *App) TestNotify() error {
	notifyRebuildMu.Lock()
	bus := a.notifyBus
	notifyRebuildMu.Unlock()

	if bus == nil {
		return fmt.Errorf("notify 子系统未启用 — 请先在设置中开启并填写 Webhook URL")
	}
	if len(bus.TransportNames()) == 0 {
		return fmt.Errorf("没有已注册的推送渠道 — 请填写 Feishu Webhook URL 后再试")
	}

	// Tracer captures the first delivery error so we can surface it as the
	// binding's return value. Buffered channel size = #transports to avoid
	// blocking the tracer's send when there's no reader yet.
	transports := bus.TransportNames()
	errCh := make(chan error, len(transports))
	bus.SetTracer(func(_ string, _ notify.Event, err error) {
		errCh <- err
	})
	defer bus.SetTracer(nil)

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	bus.Publish(ctx, notify.Event{
		ID:       fmt.Sprintf("test:%d", time.Now().UnixNano()),
		Time:     time.Now(),
		Kind:     notify.KindTest,
		Severity: notify.SeverityInfo,
		Title:    "Switch 测试通知",
		Body:     "如果你在群里看到这张卡片,说明 Webhook 已正确接通 🎉",
		Project:  "lurus-switch",
	})

	// Drain whatever the tracer recorded for each transport. Any single
	// non-nil error is enough to bubble up — the user wants a concrete
	// fix-this signal, not a multi-error envelope.
	for i := 0; i < len(transports); i++ {
		select {
		case err := <-errCh:
			if err != nil {
				return err
			}
		case <-ctx.Done():
			return fmt.Errorf("等待推送结果超时")
		}
	}
	return nil
}

// GetRecentNotifications returns the bus's recent-events ring buffer
// (newest last). Empty when notify is disabled or nothing's fired yet.
func (a *App) GetRecentNotifications() []notify.Event {
	notifyRebuildMu.Lock()
	bus := a.notifyBus
	notifyRebuildMu.Unlock()
	if bus == nil {
		return []notify.Event{}
	}
	return bus.Recent()
}
