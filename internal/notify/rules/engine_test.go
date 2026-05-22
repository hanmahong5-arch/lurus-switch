package rules

import (
	"context"
	"sync"
	"testing"
	"time"

	"lurus-switch/internal/livesession"
	"lurus-switch/internal/notify"
)

// fakeSnap returns a fixed slice on each Snapshot() call. Tests mutate the
// slice between e.evaluate() calls to simulate watcher updates.
type fakeSnap struct {
	mu       sync.Mutex
	sessions []livesession.LiveSession
}

func (f *fakeSnap) Snapshot() []livesession.LiveSession {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]livesession.LiveSession, len(f.sessions))
	copy(out, f.sessions)
	return out
}

func (f *fakeSnap) set(s []livesession.LiveSession) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.sessions = s
}

// capturePub records every Publish call. The bus interface returns int;
// we always return 1 (success) since the engine ignores it.
type capturePub struct {
	mu  sync.Mutex
	got []notify.Event
}

func (c *capturePub) Publish(_ context.Context, ev notify.Event) int {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.got = append(c.got, ev)
	return 1
}

func (c *capturePub) events() []notify.Event {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := make([]notify.Event, len(c.got))
	copy(out, c.got)
	return out
}

// newTestEngine wires an engine with the fakes and a frozen clock.
func newTestEngine(t *testing.T, cfg Config) (*Engine, *fakeSnap, *capturePub, *time.Time) {
	t.Helper()
	snap := &fakeSnap{}
	pub := &capturePub{}
	e := NewEngine(snap, pub, cfg)
	clock := time.Date(2026, 5, 18, 12, 0, 0, 0, time.UTC)
	e.now = func() time.Time { return clock }
	return e, snap, pub, &clock
}

func session(path string, status string, pending *livesession.PendingTool, lastActivity time.Time) livesession.LiveSession {
	return livesession.LiveSession{
		SessionID:      "sid-" + path,
		Tool:           "claude",
		ProjectName:    "demo",
		TranscriptPath: path,
		Status:         status,
		PendingTool:    pending,
		LastActivity:   lastActivity,
	}
}

func TestEngine_PublishesStuckOnceAtThreshold(t *testing.T) {
	cfg := DefaultConfig()
	e, snap, pub, clock := newTestEngine(t, cfg)

	startedAt := clock.Add(-90 * time.Second) // already past StuckAfter (60s)
	snap.set([]livesession.LiveSession{
		session("/t/a.jsonl", "tool_call", &livesession.PendingTool{
			Name: "Bash", Preview: "sleep 90", StartedAt: startedAt,
		}, *clock),
	})

	// Multiple ticks while still pending — should only emit one warning.
	for i := 0; i < 3; i++ {
		e.evaluate()
	}

	stuckCount := 0
	for _, ev := range pub.events() {
		if ev.Kind == notify.KindToolStuck {
			stuckCount++
		}
	}
	if stuckCount != 1 {
		t.Fatalf("expected exactly 1 stuck event, got %d", stuckCount)
	}
	if got := pub.events()[0].Severity; got != notify.SeverityWarning {
		t.Errorf("first stuck event should be warning, got %s", got)
	}
}

func TestEngine_EscalatesStuckAtSecondThreshold(t *testing.T) {
	cfg := DefaultConfig()
	e, snap, pub, clock := newTestEngine(t, cfg)

	startedAt := clock.Add(-90 * time.Second)
	snap.set([]livesession.LiveSession{
		session("/t/a.jsonl", "tool_call", &livesession.PendingTool{
			Name: "Bash", Preview: "sleep 9999", StartedAt: startedAt,
		}, *clock),
	})

	// First tick — warning.
	e.evaluate()
	if n := countKind(pub.events(), notify.KindToolStuck); n != 1 {
		t.Fatalf("want 1 stuck after warning tick, got %d", n)
	}

	// Advance the clock past the escalation threshold (5 minutes) and tick
	// again. The pending tool is still the same instance.
	*clock = clock.Add(6 * time.Minute)
	e.evaluate()

	evs := pub.events()
	stuck := filterKind(evs, notify.KindToolStuck)
	if len(stuck) != 2 {
		t.Fatalf("want 2 stuck events after escalation, got %d", len(stuck))
	}
	if stuck[0].Severity != notify.SeverityWarning {
		t.Errorf("first must be warning, got %s", stuck[0].Severity)
	}
	if stuck[1].Severity != notify.SeverityError {
		t.Errorf("second must be error, got %s", stuck[1].Severity)
	}
}

func TestEngine_ResetsStuckOnNewPending(t *testing.T) {
	cfg := DefaultConfig()
	e, snap, pub, clock := newTestEngine(t, cfg)

	startedAt := clock.Add(-90 * time.Second)
	first := &livesession.PendingTool{Name: "Bash", Preview: "x", StartedAt: startedAt}
	snap.set([]livesession.LiveSession{session("/t/a.jsonl", "tool_call", first, *clock)})
	e.evaluate() // warning fires

	// New pending tool (different StartedAt) at same path — counter resets.
	second := &livesession.PendingTool{Name: "Read", Preview: "y", StartedAt: clock.Add(-65 * time.Second)}
	snap.set([]livesession.LiveSession{session("/t/a.jsonl", "tool_call", second, *clock)})
	e.evaluate() // should publish warning AGAIN because it's a new pending

	stuck := filterKind(pub.events(), notify.KindToolStuck)
	if len(stuck) != 2 {
		t.Fatalf("want 2 stuck events across two distinct pending tools, got %d", len(stuck))
	}
	if stuck[1].Tool != "claude" {
		t.Errorf("second event missing tool label, got %q", stuck[1].Tool)
	}
}

func TestEngine_PublishesDoneTransitionOnly(t *testing.T) {
	cfg := DefaultConfig()
	e, snap, pub, clock := newTestEngine(t, cfg)

	// Seed: active session (running). First tick records "was active".
	snap.set([]livesession.LiveSession{
		session("/t/done.jsonl", "running", nil, *clock),
	})
	e.evaluate()

	// Now session went idle and the last activity was 6 minutes ago.
	idleAt := clock.Add(-6 * time.Minute)
	snap.set([]livesession.LiveSession{
		session("/t/done.jsonl", "idle", nil, idleAt),
	})
	e.evaluate() // should fire one done

	// Tick again with same idle session — must NOT re-fire.
	e.evaluate()
	e.evaluate()

	done := filterKind(pub.events(), notify.KindSessionDone)
	if len(done) != 1 {
		t.Fatalf("expected exactly 1 done event across repeated idle ticks, got %d", len(done))
	}
	if done[0].Severity != notify.SeveritySuccess {
		t.Errorf("done should be success severity, got %s", done[0].Severity)
	}
}

func TestEngine_ReapsDroppedSessions(t *testing.T) {
	cfg := DefaultConfig()
	e, snap, _, clock := newTestEngine(t, cfg)

	snap.set([]livesession.LiveSession{
		session("/t/x.jsonl", "running", nil, *clock),
		session("/t/y.jsonl", "running", nil, *clock),
	})
	e.evaluate()

	e.mu.Lock()
	if len(e.mem) != 2 {
		e.mu.Unlock()
		t.Fatalf("expected 2 memory entries after first tick, got %d", len(e.mem))
	}
	e.mu.Unlock()

	// Watcher drops one session (e.g. transcript file rotated away).
	snap.set([]livesession.LiveSession{
		session("/t/x.jsonl", "running", nil, *clock),
	})
	e.evaluate()

	e.mu.Lock()
	defer e.mu.Unlock()
	if len(e.mem) != 1 {
		t.Fatalf("reap should leave 1 memory entry, got %d", len(e.mem))
	}
	if _, ok := e.mem["/t/y.jsonl"]; ok {
		t.Errorf("dropped session /t/y.jsonl should have been reaped")
	}
}

func countKind(evs []notify.Event, k notify.Kind) int {
	n := 0
	for _, ev := range evs {
		if ev.Kind == k {
			n++
		}
	}
	return n
}

func filterKind(evs []notify.Event, k notify.Kind) []notify.Event {
	out := []notify.Event{}
	for _, ev := range evs {
		if ev.Kind == k {
			out = append(out, ev)
		}
	}
	return out
}
