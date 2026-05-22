package livesession

import (
	"encoding/json"
	"testing"
	"time"

	"lurus-switch/internal/conversation"
)

// applyEvents must promote a session's status to "tool_call" when an
// unresolved tool_use is the latest event. This is the single most
// important signal in the live view ("Claude is busy"), so cover it
// directly rather than relying on integration tests.
func TestSessionState_PendingToolBlocksOnUnresolvedToolUse(t *testing.T) {
	s := newState("sid", "claude", "/tmp/proj", "/tmp/proj.jsonl")
	now := time.Date(2026, 5, 13, 12, 0, 0, 0, time.UTC)

	s.applyEvent(conversation.Event{
		Type:      conversation.EventToolUse,
		ToolName:  "Bash",
		ToolArgs:  json.RawMessage(`{"command":"sleep 60"}`),
		Timestamp: now,
		Raw:       json.RawMessage(`{"tool_use_id":"tu_1"}`),
	})

	if s.pending == nil {
		t.Fatal("expected pending tool after tool_use; got nil")
	}
	if s.pending.Name != "Bash" || s.pending.Preview != "sleep 60" {
		t.Errorf("pending tool preview wrong: %+v", s.pending)
	}
	if got := s.statusAt(now.Add(2 * time.Second)); got != "tool_call" {
		t.Errorf("status with pending tool = %q, want tool_call", got)
	}

	// Matching tool_result should clear pending and flip to "running".
	s.applyEvent(conversation.Event{
		Type:      conversation.EventToolResult,
		Content:   "exit 0",
		Timestamp: now.Add(3 * time.Second),
		Raw:       json.RawMessage(`{"tool_use_id":"tu_1"}`),
	})
	if s.pending != nil {
		t.Errorf("pending should clear after tool_result; got %+v", s.pending)
	}
	if got := s.statusAt(now.Add(5 * time.Second)); got != "running" {
		t.Errorf("status after result = %q, want running", got)
	}
}

func TestSessionState_TracksBashHistoryAndFiles(t *testing.T) {
	s := newState("sid", "claude", "/proj", "/proj.jsonl")
	ts := time.Now()

	s.applyEvent(conversation.Event{
		Type: conversation.EventToolUse, ToolName: "Bash",
		ToolArgs: json.RawMessage(`{"command":"ls -la"}`), Timestamp: ts,
	})
	s.applyEvent(conversation.Event{
		Type: conversation.EventToolUse, ToolName: "Read",
		ToolArgs: json.RawMessage(`{"file_path":"/proj/app.go"}`), Timestamp: ts.Add(time.Second),
	})
	s.applyEvent(conversation.Event{
		Type: conversation.EventToolUse, ToolName: "Edit",
		ToolArgs: json.RawMessage(`{"file_path":"/proj/app.go","old_string":"x","new_string":"y"}`), Timestamp: ts.Add(2 * time.Second),
	})

	if len(s.bashes) != 1 || s.bashes[0] != "ls -la" {
		t.Errorf("bash history: %+v", s.bashes)
	}
	touch, ok := s.files["/proj/app.go"]
	if !ok || touch.Count != 2 {
		t.Errorf("file should be touched twice: %+v", touch)
	}
	// Edit > Read promotion: kind should be "edit".
	if touch.Kind != "edit" {
		t.Errorf("kind promotion failed: %+v", touch)
	}
}

func TestEstimateCost_KnownAndUnknownModels(t *testing.T) {
	// 1M input + 1M output on sonnet → $3 + $15 = $18.
	got := estimateCost("claude-sonnet-4-6", 1_000_000, 1_000_000)
	if got < 17.99 || got > 18.01 {
		t.Errorf("sonnet cost = %.4f, want ~18.00", got)
	}
	// Opus is 5x sonnet — covers the prefix-match branch.
	got = estimateCost("claude-opus-4-7", 1_000_000, 1_000_000)
	if got < 89.99 || got > 90.01 {
		t.Errorf("opus cost = %.4f, want ~90.00", got)
	}
	// Unknown model must fall through to the sonnet-tier default rather
	// than returning 0 (which would silently hide cost in the UI).
	got = estimateCost("unknown-future-model-9000", 1_000_000, 0)
	if got < 2.99 || got > 3.01 {
		t.Errorf("unknown cost = %.4f, want sonnet fallback ~3.00", got)
	}
}

// Cache tokens are billed at 1.25× input / 0.10× input. Verify both
// rates are wired and that they sum into the session's running cost.
func TestEventCost_IncludesCacheFields(t *testing.T) {
	// 1M cache_create tokens on sonnet → 3.00 × 1.25 = $3.75
	cc := eventCost("claude-sonnet-4-6", 0, 0, 1_000_000, 0)
	if cc < 3.74 || cc > 3.76 {
		t.Errorf("cache_create cost = %.4f, want ~3.75", cc)
	}
	// 1M cache_read tokens on sonnet → 3.00 × 0.10 = $0.30
	cr := eventCost("claude-sonnet-4-6", 0, 0, 0, 1_000_000)
	if cr < 0.29 || cr > 0.31 {
		t.Errorf("cache_read cost = %.4f, want ~0.30", cr)
	}
	// All four streams summed correctly.
	total := eventCost("claude-sonnet-4-6", 100, 50, 200, 500)
	want := 100*3.0/1e6 + 50*15.0/1e6 + 200*3.75/1e6 + 500*0.30/1e6
	if total < want-1e-9 || total > want+1e-9 {
		t.Errorf("mixed cost = %v, want %v", total, want)
	}
}

// Without cache fields a long Claude Code session was reading
// ~$0 on huge transcripts because almost everything was cached. Guard
// the regression: a 100k cache_read line must produce a non-zero cost.
func TestSessionState_AccumulatesCostAcrossEvents(t *testing.T) {
	s := newState("sid", "claude", "/p", "/p.jsonl")
	s.applyEvent(conversation.Event{
		Type: conversation.EventAssistant, Model: "claude-sonnet-4-6",
		InputTokens: 1_000, OutputTokens: 500,
		CacheCreationTokens: 80_000, CacheReadTokens: 0,
		Timestamp: time.Now(),
	})
	s.applyEvent(conversation.Event{
		Type: conversation.EventAssistant, Model: "claude-sonnet-4-6",
		InputTokens: 500, OutputTokens: 800,
		CacheCreationTokens: 0, CacheReadTokens: 80_000,
		Timestamp: time.Now().Add(time.Minute),
	})
	snap := s.snapshot(time.Now().Add(2 * time.Minute))
	if snap.CacheCreateTokens != 80_000 {
		t.Errorf("cache_create sum = %d, want 80000", snap.CacheCreateTokens)
	}
	if snap.CacheReadTokens != 80_000 {
		t.Errorf("cache_read sum = %d, want 80000", snap.CacheReadTokens)
	}
	if snap.EstimatedUSD <= 0 {
		t.Errorf("estimated cost should include cache; got %.6f", snap.EstimatedUSD)
	}
	// Expected: (1500*3 + 1300*15 + 80000*3.75 + 80000*0.30) / 1e6
	want := (1500*3.0 + 1300*15.0 + 80000*3.75 + 80000*0.30) / 1e6
	if snap.EstimatedUSD < want-0.0005 || snap.EstimatedUSD > want+0.0005 {
		t.Errorf("estimated cost = %.6f, want %.6f", snap.EstimatedUSD, want)
	}
}

// When a session toggles models mid-flight (opus→sonnet), historical
// tokens must NOT be re-priced at the latest rate. The previous code
// took the session's latest model and multiplied through the whole
// totals — wrong by a factor of 5 between opus and sonnet.
func TestSessionState_PerMessagePricingAcrossModels(t *testing.T) {
	s := newState("sid", "claude", "/p", "/p.jsonl")
	// Turn 1: opus, 1M output → $75.
	s.applyEvent(conversation.Event{
		Type: conversation.EventAssistant, Model: "claude-opus-4-7",
		OutputTokens: 1_000_000, Timestamp: time.Now(),
	})
	// Turn 2: sonnet, 1M output → $15.
	s.applyEvent(conversation.Event{
		Type: conversation.EventAssistant, Model: "claude-sonnet-4-6",
		OutputTokens: 1_000_000, Timestamp: time.Now().Add(time.Minute),
	})
	snap := s.snapshot(time.Now().Add(2 * time.Minute))
	// 75 + 15 = 90, NOT 30 (both as sonnet) or 150 (both as opus).
	if snap.EstimatedUSD < 89.99 || snap.EstimatedUSD > 90.01 {
		t.Errorf("per-message pricing failed: got %.4f, want ~90.00", snap.EstimatedUSD)
	}
	if len(snap.ModelsSeen) != 2 {
		t.Errorf("modelsSeen should record both: got %+v", snap.ModelsSeen)
	}
}

func TestSessionState_RecentRingBufferCaps(t *testing.T) {
	s := newState("sid", "claude", "/p", "/p.jsonl")
	for i := 0; i < maxRecentEvents*2; i++ {
		s.applyEvent(conversation.Event{
			Type: conversation.EventUser, Content: "msg", Timestamp: time.Now().Add(time.Duration(i) * time.Second),
		})
	}
	if len(s.recent) != maxRecentEvents {
		t.Errorf("recent should cap at %d; got %d", maxRecentEvents, len(s.recent))
	}
}

func TestSnapshot_SortsFilesByCount(t *testing.T) {
	s := newState("sid", "claude", "/p", "/p.jsonl")
	args := func(p string) json.RawMessage {
		b, _ := json.Marshal(map[string]string{"file_path": p})
		return b
	}
	s.applyEvent(conversation.Event{Type: conversation.EventToolUse, ToolName: "Read", ToolArgs: args("/a.go"), Timestamp: time.Now()})
	s.applyEvent(conversation.Event{Type: conversation.EventToolUse, ToolName: "Read", ToolArgs: args("/b.go"), Timestamp: time.Now()})
	s.applyEvent(conversation.Event{Type: conversation.EventToolUse, ToolName: "Read", ToolArgs: args("/b.go"), Timestamp: time.Now()})
	s.applyEvent(conversation.Event{Type: conversation.EventToolUse, ToolName: "Edit", ToolArgs: args("/c.go"), Timestamp: time.Now()})

	snap := s.snapshot(time.Now())
	if len(snap.FilesTouched) != 3 {
		t.Fatalf("expected 3 files, got %d", len(snap.FilesTouched))
	}
	if snap.FilesTouched[0].Path != "/b.go" {
		t.Errorf("hottest file first: %+v", snap.FilesTouched)
	}
}
