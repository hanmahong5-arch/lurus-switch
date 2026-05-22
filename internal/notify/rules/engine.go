// Package rules turns the livesession.Watcher's polling stream into
// discrete notification Events: "this tool has been stuck for a minute",
// "this session just went idle (Claude is waiting for you)", etc.
//
// Rules are stateful: they remember the last verdict per session so we
// don't fire the same alert every 5-second tick. The state map is small
// and bounded by the watcher's own stale-session reaping.
package rules

import (
	"context"
	"fmt"
	"sync"
	"time"

	"lurus-switch/internal/livesession"
	"lurus-switch/internal/notify"
)

// Snapshotter is the subset of livesession.Watcher this package needs.
// Defining the seam here lets tests pass a fake without touching the
// real polling goroutine.
type Snapshotter interface {
	Snapshot() []livesession.LiveSession
}

// Publisher is the subset of notify.Bus we publish through. Same seam
// reasoning as Snapshotter.
type Publisher interface {
	Publish(ctx context.Context, ev notify.Event) int
}

// Config tunes when each rule fires. Defaults are conservative — better
// to be quiet than to train the user to mute the bot.
type Config struct {
	// StuckAfter is how long a single tool_use can be pending before we
	// emit a "tool 卡住" warning. 60s is roughly "a slow bash build" — most
	// real long bashes (test suites) cross it but legit fast tools don't.
	StuckAfter time.Duration
	// StuckEscalate is the second threshold at which we escalate the
	// warning to severity=error. Catches truly hung tools (network
	// timeouts that never resolve, broken pipes, …).
	StuckEscalate time.Duration
	// IdleAfter is how long a session must be idle (no events) following
	// activity before we fire "任务完成 — Claude 在等你". Below this we
	// assume the user is just typing.
	IdleAfter time.Duration
	// NotifyStuck / NotifyDone are the per-rule toggles the user sets
	// in the Settings UI.
	NotifyStuck bool
	NotifyDone  bool
}

// DefaultConfig is the factory baseline. The Settings UI overlays the
// user's preferences on top.
func DefaultConfig() Config {
	return Config{
		StuckAfter:    60 * time.Second,
		StuckEscalate: 5 * time.Minute,
		IdleAfter:     5 * time.Minute,
		NotifyStuck:   true,
		NotifyDone:    true,
	}
}

// sessionMemory holds the last-emitted state per transcript path. Used
// to avoid duplicate alerts.
type sessionMemory struct {
	lastStuckLevel  int    // 0 = no alert, 1 = warning emitted, 2 = error emitted
	stuckPendingID  string // pendingTool identity (name+startedAt) the level was for
	lastStatus      string
	doneEmitted     bool   // true while we've already announced "Claude done"; reset on activity
}

// Engine drives the polling loop. Construct via NewEngine, kick off with
// Start, drain with Stop. Engine is safe to leave running with no
// transports registered — events just get dropped at the bus level.
type Engine struct {
	cfg       Config
	snap      Snapshotter
	pub       Publisher
	interval  time.Duration

	mu     sync.Mutex
	mem    map[string]*sessionMemory // key: transcriptPath

	stop chan struct{}
	tick *time.Ticker

	// now lets tests freeze the clock.
	now func() time.Time
}

// NewEngine wires a fresh engine. The polling interval is 5s; that's a
// 2-3× multiplier over the watcher's own 2s poll, which is fine — these
// rules are about durations measured in tens of seconds, not seconds.
func NewEngine(snap Snapshotter, pub Publisher, cfg Config) *Engine {
	return &Engine{
		cfg:      cfg,
		snap:     snap,
		pub:      pub,
		interval: 5 * time.Second,
		mem:      map[string]*sessionMemory{},
		now:      time.Now,
	}
}

// Start launches the goroutine. Idempotent.
func (e *Engine) Start() {
	e.mu.Lock()
	if e.tick != nil {
		e.mu.Unlock()
		return
	}
	e.stop = make(chan struct{})
	e.tick = time.NewTicker(e.interval)
	e.mu.Unlock()
	go e.loop()
}

// Stop releases the goroutine. Safe to call multiple times.
func (e *Engine) Stop() {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.tick == nil {
		return
	}
	e.tick.Stop()
	close(e.stop)
	e.tick = nil
}

// SetConfig swaps in a new config. Takes effect on the next tick.
func (e *Engine) SetConfig(cfg Config) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.cfg = cfg
}

func (e *Engine) loop() {
	e.evaluate()
	for {
		select {
		case <-e.stop:
			return
		case <-e.tick.C:
			e.evaluate()
		}
	}
}

// evaluate walks every visible session and decides whether each rule
// should fire. Exposed for tests via the indirection on tick.
func (e *Engine) evaluate() {
	sessions := e.snap.Snapshot()
	now := e.now()

	e.mu.Lock()
	cfg := e.cfg
	// Reap memory for sessions the watcher dropped — keeps the map
	// bounded across long-running app sessions.
	alive := map[string]bool{}
	for _, s := range sessions {
		alive[s.TranscriptPath] = true
	}
	for k := range e.mem {
		if !alive[k] {
			delete(e.mem, k)
		}
	}
	e.mu.Unlock()

	for _, s := range sessions {
		e.checkOne(s, cfg, now)
	}
}

func (e *Engine) checkOne(s livesession.LiveSession, cfg Config, now time.Time) {
	e.mu.Lock()
	mem, ok := e.mem[s.TranscriptPath]
	if !ok {
		mem = &sessionMemory{}
		e.mem[s.TranscriptPath] = mem
	}
	prevStatus := mem.lastStatus
	mem.lastStatus = s.Status
	e.mu.Unlock()

	// Tool-stuck rule. Fires when a pending tool_use has been running
	// past StuckAfter. Escalates to error severity at StuckEscalate.
	if cfg.NotifyStuck && s.PendingTool != nil {
		elapsed := now.Sub(s.PendingTool.StartedAt)
		pendingID := s.PendingTool.Name + "@" + s.PendingTool.StartedAt.Format(time.RFC3339Nano)

		e.mu.Lock()
		// New pending tool → reset stuck memory.
		if mem.stuckPendingID != pendingID {
			mem.stuckPendingID = pendingID
			mem.lastStuckLevel = 0
		}
		level := mem.lastStuckLevel
		e.mu.Unlock()

		// Determine target level given the elapsed duration.
		target := 0
		switch {
		case elapsed >= cfg.StuckEscalate:
			target = 2
		case elapsed >= cfg.StuckAfter:
			target = 1
		}
		if target > level {
			e.publishStuck(s, elapsed, target)
			e.mu.Lock()
			mem.lastStuckLevel = target
			e.mu.Unlock()
		}
	} else if s.PendingTool == nil {
		// Pending cleared → reset for the next round.
		e.mu.Lock()
		mem.lastStuckLevel = 0
		mem.stuckPendingID = ""
		e.mu.Unlock()
	}

	// Session-done rule. Fires once when a session that was previously
	// active (running / tool_call) crosses into idle for IdleAfter.
	// Resets the moment it goes back to active so a resumed session can
	// re-fire on its next pause.
	if cfg.NotifyDone {
		wasActive := prevStatus == string(statusRunning) || prevStatus == string(statusToolCall)
		idleEnough := now.Sub(s.LastActivity) >= cfg.IdleAfter
		e.mu.Lock()
		if wasActive && idleEnough && !mem.doneEmitted {
			mem.doneEmitted = true
			e.mu.Unlock()
			e.publishDone(s, now)
		} else {
			if s.Status == string(statusRunning) || s.Status == string(statusToolCall) {
				mem.doneEmitted = false
			}
			e.mu.Unlock()
		}
	}
}

// Status constants mirroring the watcher's classification. Kept as a
// local type so a typo in the string is a compile error, not silent
// always-false.
type statusKind string

const (
	statusRunning  statusKind = "running"
	statusToolCall statusKind = "tool_call"
)

func (e *Engine) publishStuck(s livesession.LiveSession, elapsed time.Duration, level int) {
	sev := notify.SeverityWarning
	prefix := "工具调用偏长"
	if level >= 2 {
		sev = notify.SeverityError
		prefix = "工具可能卡住"
	}
	body := fmt.Sprintf("**%s** 已运行 %s\n```\n%s\n```",
		s.PendingTool.Name,
		fmtDuration(elapsed),
		truncate(s.PendingTool.Preview, 200),
	)
	e.pub.Publish(context.Background(), notify.Event{
		ID:       fmt.Sprintf("stuck:%s:%d:%s", s.TranscriptPath, level, s.PendingTool.StartedAt.Format(time.RFC3339)),
		Time:     e.now(),
		Kind:     notify.KindToolStuck,
		Severity: sev,
		Title:    fmt.Sprintf("%s · %s · %s", s.ProjectName, prefix, s.PendingTool.Name),
		Body:     body,
		Project:  s.ProjectName,
		Tool:     s.Tool,
	})
}

func (e *Engine) publishDone(s livesession.LiveSession, now time.Time) {
	body := fmt.Sprintf("项目 **%s** 静默已 %s · 累计 $%.2f · 共 %d 条消息",
		s.ProjectName,
		fmtDuration(now.Sub(s.LastActivity)),
		s.EstimatedUSD,
		s.MessageCount,
	)
	e.pub.Publish(context.Background(), notify.Event{
		ID:       fmt.Sprintf("done:%s:%d", s.TranscriptPath, s.LastActivity.Unix()),
		Time:     e.now(),
		Kind:     notify.KindSessionDone,
		Severity: notify.SeveritySuccess,
		Title:    fmt.Sprintf("%s · 任务完成,等你处理", s.ProjectName),
		Body:     body,
		Project:  s.ProjectName,
		Tool:     s.Tool,
	})
}

func fmtDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%d 秒", int(d.Seconds()))
	}
	if d < time.Hour {
		m := int(d.Minutes())
		s := int(d.Seconds()) - m*60
		if s == 0 {
			return fmt.Sprintf("%d 分", m)
		}
		return fmt.Sprintf("%d 分 %d 秒", m, s)
	}
	h := int(d.Hours())
	m := int(d.Minutes()) - h*60
	return fmt.Sprintf("%d 小时 %d 分", h, m)
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "…"
}
