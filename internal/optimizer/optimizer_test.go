package optimizer

import (
	"testing"

	"lurus-switch/internal/installer"
	"lurus-switch/internal/toolhealth"
)

func TestAnalyze_NilDeps(t *testing.T) {
	result := Analyze(nil)
	if result == nil {
		t.Fatal("expected non-nil result for nil deps")
	}
	if len(result.Optimizations) != 0 {
		t.Errorf("expected 0 optimizations for nil deps, got %d", len(result.Optimizations))
	}
	if result.FixableCount != 0 {
		t.Errorf("expected fixableCount=0, got %d", result.FixableCount)
	}
}

func TestAnalyze_EmptyDeps(t *testing.T) {
	d := &Deps{
		ToolStatuses:  map[string]*installer.ToolStatus{},
		HealthResults: map[string]*toolhealth.HealthResult{},
	}
	result := Analyze(d)
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	// With empty statuses and no gateway, the only possible items are
	// system checks (git). We cannot predict git presence, so just verify structure.
	if result.TotalCount != len(result.Optimizations) {
		t.Errorf("totalCount mismatch: field=%d, len=%d", result.TotalCount, len(result.Optimizations))
	}
}

func TestAnalyze_MissingRuntime(t *testing.T) {
	d := &Deps{
		ToolStatuses:  map[string]*installer.ToolStatus{},
		HealthResults: map[string]*toolhealth.HealthResult{},
		DepCheck: &installer.DepCheckResult{
			Runtimes: []installer.RuntimeStatus{
				{ID: "bun", Name: "Bun", Required: true, Installed: false},
				{ID: "nodejs", Name: "Node.js", Required: true, Installed: false},
			},
		},
	}
	result := Analyze(d)

	found := map[string]bool{}
	for _, opt := range result.Optimizations {
		if opt.Action == "install-runtime" {
			found[opt.Target] = true
			if !opt.AutoFixable {
				t.Errorf("expected install-runtime for %s to be autoFixable", opt.Target)
			}
			if opt.Priority != 1 {
				t.Errorf("expected priority=1 for missing runtime %s, got %d", opt.Target, opt.Priority)
			}
			if opt.Category != "runtime" {
				t.Errorf("expected category=runtime for %s, got %s", opt.Target, opt.Category)
			}
		}
	}
	if !found["bun"] {
		t.Error("expected install-runtime optimization for bun")
	}
	if !found["nodejs"] {
		t.Error("expected install-runtime optimization for nodejs")
	}
}

func TestAnalyze_GatewayNotRunning(t *testing.T) {
	d := &Deps{
		ToolStatuses: map[string]*installer.ToolStatus{
			"claude": {Installed: true, Version: "1.0"},
		},
		HealthResults:  map[string]*toolhealth.HealthResult{},
		GatewayRunning: false,
	}
	result := Analyze(d)

	found := false
	for _, opt := range result.Optimizations {
		if opt.Action == "start-gateway" {
			found = true
			if opt.Priority != 1 {
				t.Errorf("expected priority=1 for start-gateway, got %d", opt.Priority)
			}
			if !opt.AutoFixable {
				t.Error("expected start-gateway to be autoFixable")
			}
		}
	}
	if !found {
		t.Error("expected start-gateway optimization when gateway not running")
	}
}

func TestAnalyze_ToolsNotConnected(t *testing.T) {
	d := &Deps{
		ToolStatuses: map[string]*installer.ToolStatus{
			"claude": {Installed: true},
			"codex":  {Installed: true},
		},
		HealthResults:  map[string]*toolhealth.HealthResult{},
		GatewayRunning: true,
		GatewayURL:     "http://localhost:19090",
		InstalledCount: 2,
		BoundCount:     0,
	}
	result := Analyze(d)

	found := false
	for _, opt := range result.Optimizations {
		if opt.Action == "connect-gateway" {
			found = true
			if opt.Priority != 1 {
				t.Errorf("expected priority=1 for connect-gateway, got %d", opt.Priority)
			}
		}
	}
	if !found {
		t.Error("expected connect-gateway optimization when tools not bound")
	}
}

func TestAnalyze_ToolNotInstalled(t *testing.T) {
	d := &Deps{
		ToolStatuses: map[string]*installer.ToolStatus{
			"claude": {Installed: false},
		},
		HealthResults: map[string]*toolhealth.HealthResult{},
	}
	result := Analyze(d)

	found := false
	for _, opt := range result.Optimizations {
		if opt.Action == "install-tool" && opt.Target == "claude" {
			found = true
			if opt.Priority != 3 {
				t.Errorf("expected priority=3 for install-tool, got %d", opt.Priority)
			}
			if !opt.AutoFixable {
				t.Error("expected install-tool to be autoFixable")
			}
		}
	}
	if !found {
		t.Error("expected install-tool optimization for claude")
	}
}

func TestAnalyze_ToolUpdateAvailable(t *testing.T) {
	d := &Deps{
		ToolStatuses: map[string]*installer.ToolStatus{
			"codex": {Installed: true, Version: "1.0", UpdateAvailable: true},
		},
		HealthResults: map[string]*toolhealth.HealthResult{},
	}
	result := Analyze(d)

	found := false
	for _, opt := range result.Optimizations {
		if opt.Action == "update-tool" && opt.Target == "codex" {
			found = true
			if opt.Priority != 2 {
				t.Errorf("expected priority=2 for update-tool, got %d", opt.Priority)
			}
			if !opt.AutoFixable {
				t.Error("expected update-tool to be autoFixable")
			}
		}
	}
	if !found {
		t.Error("expected update-tool optimization for codex")
	}
}

func TestAnalyze_ConfigHealthRed(t *testing.T) {
	d := &Deps{
		ToolStatuses: map[string]*installer.ToolStatus{
			"claude": {Installed: true, Version: "1.0"},
		},
		HealthResults: map[string]*toolhealth.HealthResult{
			"claude": {Tool: "claude", Status: toolhealth.StatusRed, Issues: []string{"invalid JSON"}},
		},
	}
	result := Analyze(d)

	found := false
	for _, opt := range result.Optimizations {
		if opt.Action == "fix-config" && opt.Target == "claude" {
			found = true
			if opt.Priority != 1 {
				t.Errorf("expected priority=1 for red config, got %d", opt.Priority)
			}
		}
	}
	if !found {
		t.Error("expected fix-config optimization for claude with red status")
	}
}

func TestAnalyze_ConfigHealthYellow(t *testing.T) {
	d := &Deps{
		ToolStatuses: map[string]*installer.ToolStatus{
			"gemini": {Installed: true, Version: "1.0"},
		},
		HealthResults: map[string]*toolhealth.HealthResult{
			"gemini": {Tool: "gemini", Status: toolhealth.StatusYellow, Issues: []string{"model.name missing"}},
		},
	}
	result := Analyze(d)

	found := false
	for _, opt := range result.Optimizations {
		if opt.Action == "fix-config" && opt.Target == "gemini" {
			found = true
			if opt.Priority != 2 {
				t.Errorf("expected priority=2 for yellow config, got %d", opt.Priority)
			}
		}
	}
	if !found {
		t.Error("expected fix-config optimization for gemini with yellow status")
	}
}

func TestAnalyze_ConfigHealthGreen_NoOptimization(t *testing.T) {
	d := &Deps{
		ToolStatuses: map[string]*installer.ToolStatus{
			"claude": {Installed: true, Version: "1.0"},
		},
		HealthResults: map[string]*toolhealth.HealthResult{
			"claude": {Tool: "claude", Status: toolhealth.StatusGreen},
		},
		GatewayRunning: true,
		InstalledCount: 1,
		BoundCount:     1,
	}
	result := Analyze(d)

	for _, opt := range result.Optimizations {
		if opt.Action == "fix-config" && opt.Target == "claude" {
			t.Error("expected no fix-config optimization for green-status claude")
		}
	}
}

func TestAnalyze_FixableCount(t *testing.T) {
	d := &Deps{
		ToolStatuses: map[string]*installer.ToolStatus{
			"claude": {Installed: false},
			"codex":  {Installed: true, Version: "1.0", UpdateAvailable: true},
		},
		HealthResults:  map[string]*toolhealth.HealthResult{},
		GatewayRunning: false,
	}
	result := Analyze(d)

	// We expect at least: install-tool-claude (fixable), update-tool-codex (fixable),
	// start-gateway (fixable). Possibly install-git (NOT fixable).
	// Count only auto-fixable items.
	expectedMin := 3
	if result.FixableCount < expectedMin {
		t.Errorf("expected at least %d fixable optimizations, got %d", expectedMin, result.FixableCount)
	}
	if result.FixableCount > result.TotalCount {
		t.Errorf("fixableCount (%d) should not exceed totalCount (%d)", result.FixableCount, result.TotalCount)
	}
}

func TestAnalyze_SortByPriority(t *testing.T) {
	d := &Deps{
		ToolStatuses: map[string]*installer.ToolStatus{
			"claude": {Installed: false},
		},
		HealthResults:  map[string]*toolhealth.HealthResult{},
		GatewayRunning: false,
		DepCheck: &installer.DepCheckResult{
			Runtimes: []installer.RuntimeStatus{
				{ID: "bun", Name: "Bun", Required: true, Installed: false},
			},
		},
	}
	result := Analyze(d)

	if len(result.Optimizations) < 2 {
		t.Skipf("need at least 2 optimizations to test sort, got %d", len(result.Optimizations))
	}

	for i := 1; i < len(result.Optimizations); i++ {
		prev := result.Optimizations[i-1]
		curr := result.Optimizations[i]
		if prev.Priority > curr.Priority {
			t.Errorf("optimizations not sorted by priority: [%d].Priority=%d > [%d].Priority=%d",
				i-1, prev.Priority, i, curr.Priority)
		}
	}
}

func TestAnalyze_UninstalledToolConfigIgnored(t *testing.T) {
	// Config issues for uninstalled tools should not produce optimizations.
	d := &Deps{
		ToolStatuses: map[string]*installer.ToolStatus{
			"claude": {Installed: false},
		},
		HealthResults: map[string]*toolhealth.HealthResult{
			"claude": {Tool: "claude", Status: toolhealth.StatusRed, Issues: []string{"broken"}},
		},
	}
	result := Analyze(d)

	for _, opt := range result.Optimizations {
		if opt.Action == "fix-config" && opt.Target == "claude" {
			t.Error("should not suggest fix-config for uninstalled tool")
		}
	}
}

func TestAnalyze_NoConnectWhenGatewayDown(t *testing.T) {
	d := &Deps{
		ToolStatuses: map[string]*installer.ToolStatus{
			"claude": {Installed: true},
		},
		HealthResults:  map[string]*toolhealth.HealthResult{},
		GatewayRunning: false,
		InstalledCount: 1,
		BoundCount:     0,
	}
	result := Analyze(d)

	for _, opt := range result.Optimizations {
		if opt.Action == "connect-gateway" {
			t.Error("should not suggest connect-gateway when gateway is not running")
		}
	}
}

func TestAnalyze_AllBound_NoConnectSuggestion(t *testing.T) {
	d := &Deps{
		ToolStatuses: map[string]*installer.ToolStatus{
			"claude": {Installed: true},
		},
		HealthResults:  map[string]*toolhealth.HealthResult{},
		GatewayRunning: true,
		InstalledCount: 1,
		BoundCount:     1,
	}
	result := Analyze(d)

	for _, opt := range result.Optimizations {
		if opt.Action == "connect-gateway" {
			t.Error("should not suggest connect-gateway when all tools are bound")
		}
	}
}
