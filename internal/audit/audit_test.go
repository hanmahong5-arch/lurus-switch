package audit

import (
	"errors"
	"path/filepath"
	"testing"
	"time"

	"lurus-switch/internal/capability"
)

func newTestJournal(t *testing.T) *Journal {
	t.Helper()
	dir := t.TempDir()
	j, err := NewJournal(dir)
	if err != nil {
		t.Fatal(err)
	}
	return j
}

func TestRecord_Ok(t *testing.T) {
	j := newTestJournal(t)
	e := j.Record("channel.create", "ch-1", nil, map[string]string{"name": "ch-1"}, nil)
	if e.Outcome != "ok" {
		t.Errorf("outcome = %s", e.Outcome)
	}
	if e.Reversible {
		t.Error("no handler registered, should be Reversible=false")
	}
}

func TestRecord_DeniedOutcome(t *testing.T) {
	j := newTestJournal(t)
	denied := &capability.Error{Required: capability.CapChannelWrite, Principal: "agent:x"}
	e := j.Record("channel.create", "ch-1", nil, nil, denied)
	if e.Outcome != "denied" {
		t.Errorf("outcome = %s, want denied", e.Outcome)
	}
}

func TestRecord_ErrorOutcome(t *testing.T) {
	j := newTestJournal(t)
	e := j.Record("channel.create", "ch-1", nil, nil, errors.New("upstream 500"))
	if e.Outcome != "error" {
		t.Errorf("outcome = %s", e.Outcome)
	}
	if e.Error == "" {
		t.Error("expected Error field to be populated")
	}
}

func TestRegister_MakesReversible(t *testing.T) {
	j := newTestJournal(t)
	j.Register("channel.create", func(_ Entry) error { return nil })
	e := j.Record("channel.create", "ch-1", nil, nil, nil)
	if !e.Reversible {
		t.Error("expected Reversible=true after Register")
	}
}

func TestUndo_HappyPath(t *testing.T) {
	j := newTestJournal(t)
	called := false
	j.Register("channel.create", func(e Entry) error {
		called = true
		if e.Target != "ch-1" {
			t.Errorf("undo got Target=%s, want ch-1", e.Target)
		}
		return nil
	})
	e := j.Record("channel.create", "ch-1", nil, map[string]any{"id": 1}, nil)
	if err := j.Undo(e.ID); err != nil {
		t.Fatal(err)
	}
	if !called {
		t.Error("undo handler not invoked")
	}
	// Marker entry should be appended.
	list := j.List(10, Filter{})
	if list[0].Operation != "audit.undo" {
		t.Errorf("expected newest entry to be audit.undo, got %s", list[0].Operation)
	}
}

func TestUndo_TwiceFails(t *testing.T) {
	j := newTestJournal(t)
	j.Register("channel.create", func(_ Entry) error { return nil })
	e := j.Record("channel.create", "ch-1", nil, nil, nil)
	if err := j.Undo(e.ID); err != nil {
		t.Fatal(err)
	}
	if err := j.Undo(e.ID); err == nil {
		t.Error("expected error on second Undo")
	}
}

func TestUndo_NonReversibleFails(t *testing.T) {
	j := newTestJournal(t)
	e := j.Record("channel.create", "ch-1", nil, nil, nil) // no handler
	if err := j.Undo(e.ID); err == nil {
		t.Error("expected error for non-reversible op")
	}
}

func TestList_Filtered(t *testing.T) {
	j := newTestJournal(t)
	capability.SetCurrent(capability.NewToken("agent:sales", capability.CapNotifyUser))
	defer capability.SetCurrent(capability.AllToken("desktop-user"))

	j.Record("channel.create", "ch-1", nil, nil, nil)
	j.Record("channel.delete", "ch-2", nil, nil, nil)
	j.Record("user.create", "u-1", nil, nil, nil)

	got := j.List(10, Filter{Operation: "channel"})
	if len(got) != 2 {
		t.Errorf("expected 2 channel.* entries, got %d", len(got))
	}
}

func TestStats_Aggregates(t *testing.T) {
	j := newTestJournal(t)
	j.Record("channel.create", "ch-1", nil, nil, nil)
	j.Record("channel.create", "ch-2", nil, nil, nil)
	j.Record("user.delete", "u-1", nil, nil, errors.New("boom"))

	s := j.Stats()
	if s.Total != 3 {
		t.Errorf("Total=%d, want 3", s.Total)
	}
	if s.OK != 2 || s.Error != 1 {
		t.Errorf("OK/Error counts wrong: %+v", s)
	}
	if s.ByOperation["channel.create"] != 2 {
		t.Errorf("channel.create count = %d", s.ByOperation["channel.create"])
	}
}

func TestStatsWindow_FailRateZeroTotal(t *testing.T) {
	j := newTestJournal(t)
	// FailRate must NOT be NaN/Inf when Total is zero — UI relies on a
	// clean numeric 0 to render an em-dash chip.
	s := j.StatsWindow(time.Now().Add(time.Hour))
	if s.Total != 0 {
		t.Fatalf("Total=%d, want 0", s.Total)
	}
	if s.FailRate != 0 {
		t.Errorf("FailRate=%v, want 0", s.FailRate)
	}
}

func TestStatsWindow_FailRateMixed(t *testing.T) {
	j := newTestJournal(t)
	// 2 ok + 1 denied + 1 error → fail rate = 2/4 = 0.5.
	j.Record("channel.create", "ch-1", nil, nil, nil)
	j.Record("channel.create", "ch-2", nil, nil, nil)
	j.Record("channel.create", "ch-3", nil, nil, &capability.Error{Required: capability.CapChannelWrite, Principal: "x"})
	j.Record("channel.delete", "ch-4", nil, nil, errors.New("upstream 500"))

	s := j.Stats()
	if s.Total != 4 {
		t.Fatalf("Total=%d, want 4", s.Total)
	}
	if s.FailRate != 0.5 {
		t.Errorf("FailRate=%v, want 0.5", s.FailRate)
	}
	if s.ByOperationPrefix["channel"] != 4 {
		t.Errorf("ByOperationPrefix[channel]=%d, want 4", s.ByOperationPrefix["channel"])
	}
}

func TestStatsWindow_LastHour(t *testing.T) {
	j := newTestJournal(t)
	j.Record("channel.create", "old", nil, nil, nil)
	// Backdate by hand — the hot ring exposes Timestamp on the Entry.
	j.mu.Lock()
	j.hot[0].Timestamp = time.Now().Add(-2 * time.Hour)
	j.mu.Unlock()
	j.Record("channel.create", "new", nil, nil, nil)

	since := time.Now().Add(-1 * time.Hour)
	s := j.StatsWindow(since)
	if s.Total != 1 {
		t.Errorf("expected 1 entry inside last hour, got %d", s.Total)
	}
	if s.WindowStart == nil || !s.WindowStart.Equal(since) {
		t.Errorf("WindowStart not propagated: %v", s.WindowStart)
	}
}

func TestOpPrefix(t *testing.T) {
	cases := []struct{ in, want string }{
		{"channel.create", "channel"},
		{"audit.undo", "audit"},
		{"snapshot", "snapshot"},
		{"", "unknown"},
		{"a.b.c", "a"},
	}
	for _, c := range cases {
		if got := opPrefix(c.in); got != c.want {
			t.Errorf("opPrefix(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestColdStorage_Persists(t *testing.T) {
	dir := t.TempDir()
	// First journal session
	j, _ := NewJournal(dir)
	j.Record("channel.create", "ch-1", nil, map[string]string{"name": "first"}, nil)

	// Hydrate from disk
	j2, _ := NewJournal(dir)
	list := j2.List(10, Filter{})
	if len(list) == 0 {
		t.Fatal("expected hydrated entries")
	}
	if list[0].Target != "ch-1" {
		t.Errorf("hydrated target = %s, want ch-1", list[0].Target)
	}

	// Verify file actually exists
	matches, _ := filepath.Glob(filepath.Join(dir, "audit", "*.ndjson"))
	if len(matches) == 0 {
		t.Error("expected at least one cold-storage NDJSON file")
	}
}
