package process

import (
	"context"
	"testing"
)

// Scope note: the OS-side surface (ListCLIProcesses, KillProcess,
// LaunchTool, StopSession) shells out to `tasklist` / `ps` and spawns
// real child processes, so it isn't covered here — those are exercised
// from the Wails layer in `app_test.go` style integration tests. This
// file covers the pure in-memory pieces: the FIFO ring buffer, the
// tool→binary name table, and the Monitor constructor.

// TestNewMonitor_Initialized locks in the zero-state contract — a fresh
// Monitor must have an empty session map (not nil), so the first
// GetOutput / StopSession call returns a clean "not found" rather than a
// nil-map panic.
func TestNewMonitor_Initialized(t *testing.T) {
	m := NewMonitor()
	if m == nil {
		t.Fatal("NewMonitor returned nil")
	}
	if m.sessions == nil {
		t.Fatal("sessions map is nil")
	}
	if len(m.sessions) != 0 {
		t.Errorf("fresh monitor has %d sessions, want 0", len(m.sessions))
	}
	// GetOutput on an unknown session must error, not panic.
	if _, err := m.GetOutput("does-not-exist", 10); err == nil {
		t.Error("GetOutput on unknown session must error")
	}
	if err := m.StopSession("does-not-exist"); err == nil {
		t.Error("StopSession on unknown session must error")
	}
}

// TestResolveBinary_TableDriven captures the full tool→binary mapping —
// this is the only place renames are mediated, and a typo here means
// `LaunchTool` silently fails for that tool.
func TestResolveBinary_TableDriven(t *testing.T) {
	cases := []struct {
		tool    string
		want    string
		wantErr bool
	}{
		{"claude", "claude", false},
		{"codex", "codex", false},
		{"gemini", "gemini", false},
		{"picoclaw", "pclaw", false},
		{"nullclaw", "nclaw", false},
		{"unknown", "", true},
		{"", "", true},
	}
	for _, tc := range cases {
		t.Run(tc.tool, func(t *testing.T) {
			got, err := resolveBinary(tc.tool)
			if (err != nil) != tc.wantErr {
				t.Fatalf("resolveBinary(%q) err = %v, wantErr %v", tc.tool, err, tc.wantErr)
			}
			if got != tc.want {
				t.Errorf("resolveBinary(%q) = %q, want %q", tc.tool, got, tc.want)
			}
		})
	}
}

// TestKnownTools_BaselineShape pins the count and inverse mapping of the
// knownTools table — `listWindowsProcesses` and `listUnixProcesses` both
// loop this map, so an accidental drop would silently stop tracking that
// tool's running processes.
func TestKnownTools_BaselineShape(t *testing.T) {
	if len(knownTools) < 5 {
		t.Fatalf("knownTools len = %d, want >= 5", len(knownTools))
	}
	wantBinaries := []string{"claude", "codex", "gemini", "pclaw", "nclaw"}
	for _, b := range wantBinaries {
		if _, ok := knownTools[b]; !ok {
			t.Errorf("knownTools missing binary %q", b)
		}
	}
	// Tool IDs should be unique (one binary maps to one tool).
	seen := map[string]string{}
	for binary, tool := range knownTools {
		if prev, ok := seen[tool]; ok {
			t.Errorf("tool %q mapped from both %q and %q", tool, prev, binary)
		}
		seen[tool] = binary
	}
}

// TestRingBuffer_AppendGet exercises the FIFO eviction contract — the
// live-log panel scrolls these lines, so eviction order and the "tail N"
// retrieval are the load-bearing semantics.
func TestRingBuffer_AppendGet(t *testing.T) {
	rb := newRingBuffer(3)
	rb.append("a")
	rb.append("b")
	rb.append("c")
	rb.append("d") // should evict "a"

	all := rb.get(0)
	if len(all) != 3 {
		t.Fatalf("buffer len = %d, want 3", len(all))
	}
	if all[0] != "b" || all[2] != "d" {
		t.Errorf("eviction order wrong: %v", all)
	}

	last2 := rb.get(2)
	if len(last2) != 2 || last2[0] != "c" || last2[1] != "d" {
		t.Errorf("get(2) = %v, want [c d]", last2)
	}

	// get(>=len) returns full copy
	bigger := rb.get(99)
	if len(bigger) != 3 {
		t.Errorf("get(99) len = %d, want 3", len(bigger))
	}

	// returned slice must be a copy — mutating it must not corrupt buffer
	bigger[0] = "MUT"
	again := rb.get(0)
	if again[0] == "MUT" {
		t.Error("get returned aliased slice; buffer was corrupted")
	}
}

// TestMonitor_StopAll_CancelsLaunchedSessions verifies that StopAll cancels
// every active session. We inject sessions directly into the map (bypassing
// LaunchTool which shells out to real binaries) so the test is hermetic.
func TestMonitor_StopAll_CancelsLaunchedSessions(t *testing.T) {
	m := NewMonitor()

	// Inject two fake sessions, each with a real cancellable context.
	ctx1, cancel1 := context.WithCancel(context.Background())
	ctx2, cancel2 := context.WithCancel(context.Background())
	defer cancel1() // safety: ensure no leak even if StopAll fails
	defer cancel2()

	m.mu.Lock()
	m.sessions["sess-1"] = &session{cancel: cancel1, output: newRingBuffer(10)}
	m.sessions["sess-2"] = &session{cancel: cancel2, output: newRingBuffer(10)}
	m.mu.Unlock()

	m.StopAll()

	// Both contexts must be cancelled after StopAll.
	select {
	case <-ctx1.Done():
		// good
	default:
		t.Error("StopAll did not cancel session sess-1")
	}
	select {
	case <-ctx2.Done():
		// good
	default:
		t.Error("StopAll did not cancel session sess-2")
	}
}

// TestMonitor_StopAll_EmptyIsNoop verifies that StopAll on an empty Monitor
// does not panic.
func TestMonitor_StopAll_EmptyIsNoop(t *testing.T) {
	m := NewMonitor()
	m.StopAll() // must not panic
}
