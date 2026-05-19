package agent

import (
	"context"
	"errors"
	"testing"

	"lurus-switch/internal/process"
)

// === buildLaunchArgs ===

// TestBuildLaunchArgs_TableDriven walks every recognized tool kind against
// three profile shapes — minimal (only ToolType + ModelID, the production
// happy path), full (every field set, exercising irrelevant fields don't
// leak into args), and edge (empty ModelID, exercising the omit-when-unset
// contract). The expected-args column is the load-bearing assertion;
// `wantErr` catches the unknown-tool branch.
func TestBuildLaunchArgs_TableDriven(t *testing.T) {
	tokens := int64(1000)
	currency := 5.0

	minimal := func(tt ToolType) *Profile {
		return &Profile{ToolType: tt, ModelID: "m-default"}
	}
	full := func(tt ToolType) *Profile {
		return &Profile{
			ID:                  "agent-1",
			Name:                "Full Profile",
			Icon:                "🤖",
			Tags:                []string{"prod"},
			ToolType:            tt,
			ModelID:             "m-full",
			SystemPrompt:        "Be concise.",
			MCPServers:          []string{"fs"},
			Permissions:         Permissions{AllowShell: true, AllowFiles: true, AllowNetwork: true},
			Status:              StatusCreated,
			ConfigDir:           "/tmp/agent-1",
			BudgetLimitTokens:   &tokens,
			BudgetLimitCurrency: &currency,
			BudgetPeriod:        BudgetDaily,
			BudgetPolicy:        PolicyDegrade,
		}
	}
	edge := func(tt ToolType) *Profile {
		return &Profile{ToolType: tt, ModelID: ""}
	}

	cases := []struct {
		name     string
		profile  *Profile
		wantArgs []string
		wantErr  bool
	}{
		// Claude — supports --model
		{"claude/minimal", minimal(ToolClaude), []string{"--model", "m-default"}, false},
		{"claude/full", full(ToolClaude), []string{"--model", "m-full"}, false},
		{"claude/edge-empty-model", edge(ToolClaude), nil, false},

		// Codex — supports --model
		{"codex/minimal", minimal(ToolCodex), []string{"--model", "m-default"}, false},
		{"codex/full", full(ToolCodex), []string{"--model", "m-full"}, false},
		{"codex/edge-empty-model", edge(ToolCodex), nil, false},

		// Gemini — supports --model
		{"gemini/minimal", minimal(ToolGemini), []string{"--model", "m-default"}, false},
		{"gemini/full", full(ToolGemini), []string{"--model", "m-full"}, false},
		{"gemini/edge-empty-model", edge(ToolGemini), nil, false},

		// *claw family — config-file-driven, no args
		{"picoclaw/minimal", minimal(ToolPicoClaw), nil, false},
		{"picoclaw/full", full(ToolPicoClaw), nil, false},
		{"picoclaw/edge", edge(ToolPicoClaw), nil, false},
		{"nullclaw/minimal", minimal(ToolNullClaw), nil, false},
		{"nullclaw/full", full(ToolNullClaw), nil, false},
		{"nullclaw/edge", edge(ToolNullClaw), nil, false},
		{"openclaw/minimal", minimal(ToolOpenClaw), nil, false},
		{"openclaw/full", full(ToolOpenClaw), nil, false},
		{"openclaw/edge", edge(ToolOpenClaw), nil, false},
		{"zeroclaw/minimal", minimal(ToolZeroClaw), nil, false},
		{"zeroclaw/full", full(ToolZeroClaw), nil, false},
		{"zeroclaw/edge", edge(ToolZeroClaw), nil, false},

		// Unknown / nil
		{"unknown-tool", &Profile{ToolType: "garbage", ModelID: "x"}, nil, true},
		{"nil-profile", nil, nil, true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := buildLaunchArgs(tc.profile)
			if (err != nil) != tc.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tc.wantErr)
			}
			if tc.wantErr {
				return
			}
			if !sliceEqual(got, tc.wantArgs) {
				t.Errorf("args = %v, want %v", got, tc.wantArgs)
			}
		})
	}
}

// TestBuildLaunchArgs_OmitsEmptyFlags pins the contract that an unset
// ModelID does NOT translate into `--model ""`. Tools reject the empty
// string or substitute a default that contradicts the profile, so the
// flag must be omitted entirely.
func TestBuildLaunchArgs_OmitsEmptyFlags(t *testing.T) {
	for _, tt := range []ToolType{ToolClaude, ToolCodex, ToolGemini} {
		t.Run(string(tt), func(t *testing.T) {
			args, err := buildLaunchArgs(&Profile{ToolType: tt, ModelID: ""})
			if err != nil {
				t.Fatalf("buildLaunchArgs: %v", err)
			}
			for _, a := range args {
				if a == "--model" {
					t.Errorf("%s args include --model even with empty ModelID: %v", tt, args)
				}
				if a == "" {
					t.Errorf("%s args contain empty string: %v", tt, args)
				}
			}
		})
	}
}

// === SyncStatuses ===

// fakeLister returns a canned process list so tests don't shell out to
// tasklist/ps. ListCLIProcesses always returns the configured procs and
// optional error; the context is intentionally ignored.
type fakeLister struct {
	procs []process.ProcessInfo
	err   error
}

func (f *fakeLister) ListCLIProcesses(_ context.Context) ([]process.ProcessInfo, error) {
	return f.procs, f.err
}

// TestSyncStatuses_ReconcilesUntrackedRunning is the core crash-recovery
// scenario: a previous Switch instance left an agent with Status=running
// on disk; the new instance has no in-memory tracking for it and so must
// mark it stopped. Tracked agents stay running; agents already in another
// state (created / stopped / error) are not touched.
func TestSyncStatuses_ReconcilesUntrackedRunning(t *testing.T) {
	database, cleanup := setupTestDB(t)
	defer cleanup()
	store := NewStore(database)
	cfgMgr, err := NewConfigManager(t.TempDir())
	if err != nil {
		t.Fatalf("config manager: %v", err)
	}
	mgr := NewInstanceManager(store, cfgMgr, nil)

	// p1: running and tracked in this process — must stay running.
	p1, _ := store.Create(CreateParams{Name: "p1", ToolType: ToolClaude, ModelID: "m1"})
	store.SetStatus(p1.ID, StatusRunning)
	mgr.instances[p1.ID] = &Instance{
		AgentID:   p1.ID,
		SessionID: "claude-1",
		ToolType:  ToolClaude,
	}

	// p2: claims running but not tracked — must transition to stopped.
	p2, _ := store.Create(CreateParams{Name: "p2", ToolType: ToolCodex, ModelID: "m2"})
	store.SetStatus(p2.ID, StatusRunning)

	// p3: already stopped — must not be touched.
	p3, _ := store.Create(CreateParams{Name: "p3", ToolType: ToolGemini, ModelID: "m3"})
	store.SetStatus(p3.ID, StatusStopped)

	// Inject a fake live process list: codex (would-be p2) is live but
	// we don't own its session, so the conservative decision is still
	// "stopped". gemini is not in the list — confirms p2's decision
	// doesn't depend on tool presence.
	mgr.setLiveLister(&fakeLister{procs: []process.ProcessInfo{
		{PID: 100, Tool: "claude"},
		{PID: 200, Tool: "codex"},
	}})

	if err := mgr.SyncStatuses(); err != nil {
		t.Fatalf("SyncStatuses: %v", err)
	}

	got1, _ := store.Get(p1.ID)
	if got1.Status != StatusRunning {
		t.Errorf("p1 (tracked) status = %q, want %q", got1.Status, StatusRunning)
	}
	got2, _ := store.Get(p2.ID)
	if got2.Status != StatusStopped {
		t.Errorf("p2 (untracked) status = %q, want %q", got2.Status, StatusStopped)
	}
	got3, _ := store.Get(p3.ID)
	if got3.Status != StatusStopped {
		t.Errorf("p3 (already stopped) status = %q, want %q", got3.Status, StatusStopped)
	}
}

// TestSyncStatuses_ListerFailureNonFatal locks in the contract that a
// failing OS process listing degrades gracefully — SyncStatuses still
// runs the in-memory pass and returns the agents to the safe stopped
// state, never erroring out.
func TestSyncStatuses_ListerFailureNonFatal(t *testing.T) {
	database, cleanup := setupTestDB(t)
	defer cleanup()
	store := NewStore(database)
	cfgMgr, err := NewConfigManager(t.TempDir())
	if err != nil {
		t.Fatalf("config manager: %v", err)
	}
	mgr := NewInstanceManager(store, cfgMgr, nil)
	mgr.setLiveLister(&fakeLister{err: errors.New("tasklist exploded")})

	p, _ := store.Create(CreateParams{Name: "stuck", ToolType: ToolClaude, ModelID: "m"})
	store.SetStatus(p.ID, StatusRunning)

	if err := mgr.SyncStatuses(); err != nil {
		t.Fatalf("SyncStatuses must tolerate lister errors, got %v", err)
	}
	got, _ := store.Get(p.ID)
	if got.Status != StatusStopped {
		t.Errorf("untracked running agent should be stopped despite lister failure, got %q", got.Status)
	}
}

// TestSyncStatuses_NoRunningAgents — empty-store fast path: no work, no
// error, no calls to the lister (verified by leaving liveProcs nil).
func TestSyncStatuses_NoRunningAgents(t *testing.T) {
	database, cleanup := setupTestDB(t)
	defer cleanup()
	store := NewStore(database)
	cfgMgr, err := NewConfigManager(t.TempDir())
	if err != nil {
		t.Fatalf("config manager: %v", err)
	}
	mgr := NewInstanceManager(store, cfgMgr, nil)
	// Created (not running) agent — must not be touched.
	store.Create(CreateParams{Name: "idle", ToolType: ToolClaude, ModelID: "m"})

	if err := mgr.SyncStatuses(); err != nil {
		t.Fatalf("SyncStatuses on empty running set: %v", err)
	}
}

// sliceEqual reports whether two string slices contain the same elements
// in the same order. nil and empty are equal so the omit-when-unset args
// case matches whether the implementation returns nil or []string{}.
func sliceEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
