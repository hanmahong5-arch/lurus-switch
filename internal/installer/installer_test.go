package installer

import (
	"context"
	"testing"
)

// === Constants Tests ===

func TestConstants_NotEmpty(t *testing.T) {
	constants := map[string]string{
		"ClaudeNpmPackage": ClaudeNpmPackage,
		"CodexNpmPackage":  CodexNpmPackage,
		"GeminiNpmPackage": GeminiNpmPackage,
		"NpmRegistryURL":   NpmRegistryURL,
		"ToolClaude":       ToolClaude,
		"ToolCodex":        ToolCodex,
		"ToolGemini":       ToolGemini,
	}

	for name, val := range constants {
		if val == "" {
			t.Errorf("constant %s should not be empty", name)
		}
	}
}

func TestDefaultInstallTimeout_Positive(t *testing.T) {
	if DefaultInstallTimeout <= 0 {
		t.Errorf("DefaultInstallTimeout should be positive, got %d", DefaultInstallTimeout)
	}
}

// === Version Extraction Tests ===

func TestExtractVersion_Valid(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"claude v1.2.3", "1.2.3"},
		{"1.0.0", "1.0.0"},
		{"version 10.20.30-beta", "10.20.30"},
		{"v0.1.0 (stable)", "0.1.0"},
		{"@anthropic-ai/claude-code@1.0.41", "1.0.41"},
	}

	for _, tt := range tests {
		result := extractVersion(tt.input)
		if result != tt.expected {
			t.Errorf("extractVersion(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestExtractVersion_NoVersion(t *testing.T) {
	result := extractVersion("no version here")
	if result != "unknown" {
		t.Errorf("extractVersion with no version = %q, want %q", result, "unknown")
	}
}

func TestExtractVersion_Empty(t *testing.T) {
	result := extractVersion("")
	if result != "unknown" {
		t.Errorf("extractVersion empty = %q, want %q", result, "unknown")
	}
}

// === Manager Tests ===

func TestNewManager(t *testing.T) {
	mgr := NewManager()
	if mgr == nil {
		t.Fatal("NewManager should return non-nil manager")
	}
	if len(mgr.installers) != 3 {
		t.Errorf("expected 3 installers, got %d", len(mgr.installers))
	}
	if mgr.runtime == nil {
		t.Error("manager runtime should not be nil")
	}
}

func TestManager_DetectAll_ReturnsAllTools(t *testing.T) {
	mgr := NewManager()
	ctx := context.Background()

	results, err := mgr.DetectAll(ctx)
	if err != nil {
		t.Fatalf("DetectAll error: %v", err)
	}

	expected := []string{ToolClaude, ToolCodex, ToolGemini}
	for _, name := range expected {
		status, ok := results[name]
		if !ok {
			t.Errorf("expected tool %q in results", name)
			continue
		}
		if status.Name != name {
			t.Errorf("tool %q has name %q", name, status.Name)
		}
	}
}

func TestManager_InstallTool_UnknownTool(t *testing.T) {
	mgr := NewManager()
	ctx := context.Background()

	_, err := mgr.InstallTool(ctx, "nonexistent")
	if err == nil {
		t.Error("expected error for unknown tool")
	}
}

func TestManager_UpdateTool_UnknownTool(t *testing.T) {
	mgr := NewManager()
	ctx := context.Background()

	_, err := mgr.UpdateTool(ctx, "nonexistent")
	if err == nil {
		t.Error("expected error for unknown tool")
	}
}

// === BunRuntime Tests ===

func TestNewBunRuntime(t *testing.T) {
	rt := NewBunRuntime()
	if rt == nil {
		t.Fatal("NewBunRuntime should return non-nil")
	}
	if rt.bunPath != "" {
		t.Error("bunPath should be empty initially")
	}
}

func TestBunRuntime_GetPath_InitiallyEmpty(t *testing.T) {
	rt := NewBunRuntime()
	if rt.GetPath() != "" {
		t.Error("GetPath should be empty before FindBun is called")
	}
}

// === ToolStatus/InstallResult Struct Tests ===

func TestToolStatus_Fields(t *testing.T) {
	status := &ToolStatus{
		Name:            "claude",
		Installed:       true,
		Version:         "1.0.0",
		LatestVersion:   "1.1.0",
		UpdateAvailable: true,
		Path:            "/usr/local/bin/claude",
	}

	if status.Name != "claude" {
		t.Errorf("expected name 'claude', got %q", status.Name)
	}
	if !status.Installed {
		t.Error("expected Installed=true")
	}
	if !status.UpdateAvailable {
		t.Error("expected UpdateAvailable=true")
	}
}

func TestInstallResult_Fields(t *testing.T) {
	result := &InstallResult{
		Tool:    "codex",
		Success: true,
		Version: "0.1.0",
		Message: "installed successfully",
	}

	if result.Tool != "codex" {
		t.Errorf("expected tool 'codex', got %q", result.Tool)
	}
	if !result.Success {
		t.Error("expected Success=true")
	}
}

// === Individual Installer Constructor Tests ===

func TestNewClaudeInstaller(t *testing.T) {
	rt := NewBunRuntime()
	inst := NewClaudeInstaller(rt)
	if inst == nil {
		t.Fatal("NewClaudeInstaller should return non-nil")
	}
}

func TestNewCodexInstaller(t *testing.T) {
	rt := NewBunRuntime()
	inst := NewCodexInstaller(rt)
	if inst == nil {
		t.Fatal("NewCodexInstaller should return non-nil")
	}
}

func TestNewGeminiInstaller(t *testing.T) {
	rt := NewBunRuntime()
	inst := NewGeminiInstaller(rt)
	if inst == nil {
		t.Fatal("NewGeminiInstaller should return non-nil")
	}
}

// === Detect Tests (pure local, no install) ===

func TestClaudeInstaller_Detect_ReturnsStatus(t *testing.T) {
	rt := NewBunRuntime()
	inst := NewClaudeInstaller(rt)
	ctx := context.Background()

	status, err := inst.Detect(ctx)
	if err != nil {
		t.Fatalf("Detect should not error: %v", err)
	}
	if status == nil {
		t.Fatal("Detect should return non-nil status")
	}
	if status.Name != ToolClaude {
		t.Errorf("expected name %q, got %q", ToolClaude, status.Name)
	}
	// installed may be true or false depending on the environment
}

func TestCodexInstaller_Detect_ReturnsStatus(t *testing.T) {
	rt := NewBunRuntime()
	inst := NewCodexInstaller(rt)
	ctx := context.Background()

	status, err := inst.Detect(ctx)
	if err != nil {
		t.Fatalf("Detect should not error: %v", err)
	}
	if status == nil {
		t.Fatal("Detect should return non-nil status")
	}
	if status.Name != ToolCodex {
		t.Errorf("expected name %q, got %q", ToolCodex, status.Name)
	}
}

func TestGeminiInstaller_Detect_ReturnsStatus(t *testing.T) {
	rt := NewBunRuntime()
	inst := NewGeminiInstaller(rt)
	ctx := context.Background()

	status, err := inst.Detect(ctx)
	if err != nil {
		t.Fatalf("Detect should not error: %v", err)
	}
	if status == nil {
		t.Fatal("Detect should return non-nil status")
	}
	if status.Name != ToolGemini {
		t.Errorf("expected name %q, got %q", ToolGemini, status.Name)
	}
}

// === ConfigureProxy Tests (writes to temp dirs) ===

func TestClaudeInstaller_ConfigureProxy(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	t.Setenv("USERPROFILE", tmpHome)

	rt := NewBunRuntime()
	inst := NewClaudeInstaller(rt)
	ctx := context.Background()

	err := inst.ConfigureProxy(ctx, "https://api.example.com/v1", "sk-test-key")
	if err != nil {
		t.Fatalf("ConfigureProxy error: %v", err)
	}
}

func TestCodexInstaller_ConfigureProxy(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	t.Setenv("USERPROFILE", tmpHome)

	rt := NewBunRuntime()
	inst := NewCodexInstaller(rt)
	ctx := context.Background()

	err := inst.ConfigureProxy(ctx, "https://api.example.com/v1", "sk-test-key")
	if err != nil {
		t.Fatalf("ConfigureProxy error: %v", err)
	}
}

func TestGeminiInstaller_ConfigureProxy(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	t.Setenv("USERPROFILE", tmpHome)

	rt := NewBunRuntime()
	inst := NewGeminiInstaller(rt)
	ctx := context.Background()

	err := inst.ConfigureProxy(ctx, "https://api.example.com/v1", "test-key")
	if err != nil {
		t.Fatalf("ConfigureProxy error: %v", err)
	}
}
