package healthscore

import (
	"testing"

	"lurus-switch/internal/installer"
	"lurus-switch/internal/toolhealth"
)

func TestCompute_EmptyDeps(t *testing.T) {
	d := &Deps{
		ToolStatuses:  map[string]*installer.ToolStatus{},
		HealthResults: map[string]*toolhealth.HealthResult{},
	}
	report := Compute(d)
	if report == nil {
		t.Fatal("expected non-nil report")
	}
	if report.MaxScore != 100 {
		t.Errorf("expected MaxScore=100, got %d", report.MaxScore)
	}
	if len(report.Categories) != 5 {
		t.Errorf("expected 5 categories, got %d", len(report.Categories))
	}
}

func TestCompute_AllGreen(t *testing.T) {
	d := &Deps{
		ToolStatuses: map[string]*installer.ToolStatus{
			"claude": {Installed: true, Version: "1.0"},
			"codex":  {Installed: true, Version: "1.0"},
		},
		HealthResults: map[string]*toolhealth.HealthResult{
			"claude": {Tool: "claude", Status: toolhealth.StatusGreen},
			"codex":  {Tool: "codex", Status: toolhealth.StatusGreen},
		},
		DepCheck: &installer.DepCheckResult{
			Runtimes: []installer.RuntimeStatus{
				{ID: "bun", Name: "Bun", Installed: true, Required: true},
			},
		},
		GatewayRunning: true,
		GatewayURL:     "http://localhost:19090",
		AllToolsBound:  true,
		InstalledCount: 2,
		BoundCount:     2,
	}
	report := Compute(d)
	if report.TotalScore < 70 {
		t.Errorf("expected high score for healthy env, got %d", report.TotalScore)
	}
}

func TestCompute_NoToolsInstalled(t *testing.T) {
	d := &Deps{
		ToolStatuses: map[string]*installer.ToolStatus{
			"claude": {Installed: false},
		},
		HealthResults: map[string]*toolhealth.HealthResult{},
	}
	report := Compute(d)
	if report.TotalScore > 30 {
		t.Errorf("expected low score when no tools installed, got %d", report.TotalScore)
	}
	// Should have suggestion to install claude
	found := false
	for _, s := range report.Suggestions {
		if s.Action == "install-tool" && s.Target == "claude" {
			found = true
		}
	}
	if !found {
		t.Error("expected install-tool suggestion for claude")
	}
}

func TestCompute_GatewayNotRunning(t *testing.T) {
	d := &Deps{
		ToolStatuses: map[string]*installer.ToolStatus{
			"claude": {Installed: true, Version: "1.0"},
		},
		HealthResults: map[string]*toolhealth.HealthResult{
			"claude": {Tool: "claude", Status: toolhealth.StatusGreen},
		},
		GatewayRunning: false,
	}
	report := Compute(d)
	// Gateway category should be 0
	for _, cat := range report.Categories {
		if cat.Category == "gateway" && cat.Score != 0 {
			t.Errorf("expected gateway score=0 when not running, got %d", cat.Score)
		}
	}
	// Should have start-gateway suggestion
	found := false
	for _, s := range report.Suggestions {
		if s.Action == "start-gateway" {
			found = true
		}
	}
	if !found {
		t.Error("expected start-gateway suggestion")
	}
}
