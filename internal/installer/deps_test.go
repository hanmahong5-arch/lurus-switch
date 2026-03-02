package installer

import (
	"context"
	"testing"
)

func TestDepGraph_CoversAllTools(t *testing.T) {
	allTools := []string{ToolClaude, ToolCodex, ToolGemini, ToolPicoClaw, ToolNullClaw, ToolZeroClaw, ToolOpenClaw}
	for _, tool := range allTools {
		deps, ok := depGraph[tool]
		if !ok {
			t.Errorf("depGraph missing tool %q", tool)
			continue
		}
		if len(deps) == 0 {
			t.Errorf("depGraph[%q] has empty deps", tool)
		}
	}
}

func TestDepGraph_StandaloneToolsHaveNoDeps(t *testing.T) {
	standaloneTools := []string{ToolPicoClaw, ToolNullClaw, ToolZeroClaw}
	for _, tool := range standaloneTools {
		deps := depGraph[tool]
		if len(deps) != 1 || deps[0] != RuntimeNone {
			t.Errorf("depGraph[%q] = %v, want [RuntimeNone]", tool, deps)
		}
	}
}

func TestDepGraph_NpmToolsRequireNodeAndBun(t *testing.T) {
	npmTools := []string{ToolClaude, ToolCodex, ToolGemini, ToolOpenClaw}
	for _, tool := range npmTools {
		deps := depGraph[tool]
		if len(deps) != 2 {
			t.Errorf("depGraph[%q] has %d deps, want 2", tool, len(deps))
			continue
		}
		if deps[0] != RuntimeNodeJS {
			t.Errorf("depGraph[%q][0] = %q, want %q", tool, deps[0], RuntimeNodeJS)
		}
		if deps[1] != RuntimeBun {
			t.Errorf("depGraph[%q][1] = %q, want %q", tool, deps[1], RuntimeBun)
		}
	}
}

func TestGetToolDependencies_KnownTool(t *testing.T) {
	deps := GetToolDependencies(ToolClaude)
	if len(deps) == 0 {
		t.Error("GetToolDependencies(claude) should return deps")
	}
}

func TestGetToolDependencies_UnknownTool(t *testing.T) {
	deps := GetToolDependencies("nonexistent")
	if deps != nil {
		t.Errorf("GetToolDependencies(nonexistent) = %v, want nil", deps)
	}
}

func TestCheckDependencies_ReturnsDeduplicatedResults(t *testing.T) {
	mgr := NewManager()
	ctx := context.Background()

	result, err := mgr.CheckDependencies(ctx)
	if err != nil {
		t.Fatalf("CheckDependencies error: %v", err)
	}
	if result == nil {
		t.Fatal("CheckDependencies returned nil")
	}

	// Should have at least 3 runtime entries: nodejs, bun, standalone
	if len(result.Runtimes) < 3 {
		t.Errorf("expected at least 3 runtimes, got %d", len(result.Runtimes))
	}

	// Check deduplication: each runtime ID should appear at most once
	seen := make(map[string]bool)
	for _, rs := range result.Runtimes {
		if seen[rs.ID] {
			t.Errorf("duplicate runtime ID: %s", rs.ID)
		}
		seen[rs.ID] = true
	}
}

func TestCheckDependencies_StandaloneHasTools(t *testing.T) {
	mgr := NewManager()
	ctx := context.Background()

	result, err := mgr.CheckDependencies(ctx)
	if err != nil {
		t.Fatalf("CheckDependencies error: %v", err)
	}

	var standalone *RuntimeStatus
	for _, rs := range result.Runtimes {
		if rs.ID == string(RuntimeNone) {
			standalone = &rs
			break
		}
	}
	if standalone == nil {
		t.Fatal("missing standalone runtime entry")
	}
	if len(standalone.Tools) == 0 {
		t.Error("standalone should have tools")
	}
}

func TestInstallDependency_None(t *testing.T) {
	mgr := NewManager()
	ctx := context.Background()

	result, err := mgr.InstallDependency(ctx, string(RuntimeNone))
	if err != nil {
		t.Fatalf("InstallDependency(none) error: %v", err)
	}
	if !result.Success {
		t.Error("InstallDependency(none) should succeed")
	}
}

func TestInstallDependency_Unknown(t *testing.T) {
	mgr := NewManager()
	ctx := context.Background()

	result, err := mgr.InstallDependency(ctx, "unknown-runtime")
	if err != nil {
		t.Fatalf("InstallDependency(unknown) error: %v", err)
	}
	if result.Success {
		t.Error("InstallDependency(unknown) should not succeed")
	}
}

func TestDedup(t *testing.T) {
	tests := []struct {
		input    []string
		expected int
	}{
		{[]string{"a", "b", "a"}, 2},
		{[]string{"x"}, 1},
		{[]string{}, 0},
		{nil, 0},
		{[]string{"a", "a", "a"}, 1},
	}

	for _, tt := range tests {
		result := dedup(tt.input)
		if len(result) != tt.expected {
			t.Errorf("dedup(%v) = %d items, want %d", tt.input, len(result), tt.expected)
		}
	}
}
