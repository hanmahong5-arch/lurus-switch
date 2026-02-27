package installer

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// === Constants Tests ===

func TestConstants_NotEmpty(t *testing.T) {
	constants := map[string]string{
		"ClaudeNpmPackage":   ClaudeNpmPackage,
		"CodexNpmPackage":    CodexNpmPackage,
		"GeminiNpmPackage":   GeminiNpmPackage,
		"PicoClawPipPackage": PicoClawPipPackage,
		"NpmRegistryURL":     NpmRegistryURL,
		"ToolClaude":         ToolClaude,
		"ToolCodex":          ToolCodex,
		"ToolGemini":         ToolGemini,
		"ToolPicoClaw":       ToolPicoClaw,
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
	if len(mgr.installers) != 4 {
		t.Errorf("expected 4 installers, got %d", len(mgr.installers))
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

	expected := []string{ToolClaude, ToolCodex, ToolGemini, ToolPicoClaw}
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

// === PicoClaw Tests ===

func TestNewPicoClawInstaller(t *testing.T) {
	inst := NewPicoClawInstaller()
	if inst == nil {
		t.Fatal("NewPicoClawInstaller should return non-nil")
	}
}

func TestPicoClawInstaller_Detect_ReturnsStatus(t *testing.T) {
	inst := NewPicoClawInstaller()
	ctx := context.Background()

	status, err := inst.Detect(ctx)
	if err != nil {
		t.Fatalf("Detect should not error: %v", err)
	}
	if status == nil {
		t.Fatal("Detect should return non-nil status")
	}
	if status.Name != ToolPicoClaw {
		t.Errorf("expected name %q, got %q", ToolPicoClaw, status.Name)
	}
}

func TestPicoClawInstaller_ConfigureProxy(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	t.Setenv("USERPROFILE", tmpHome)

	inst := NewPicoClawInstaller()
	ctx := context.Background()

	err := inst.ConfigureProxy(ctx, "https://api.example.com/v1", "sk-test-key")
	if err != nil {
		t.Fatalf("ConfigureProxy error: %v", err)
	}
}

// === ConfigureProxy Content Verification Tests ===

func TestClaudeInstaller_ConfigureProxy_WritesCorrectContent(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	t.Setenv("USERPROFILE", tmpHome)

	rt := NewBunRuntime()
	inst := NewClaudeInstaller(rt)
	ctx := context.Background()

	endpoint := "https://my-proxy.example.com/v1"
	apiKey := "sk-ant-my-secret-key"
	if err := inst.ConfigureProxy(ctx, endpoint, apiKey); err != nil {
		t.Fatalf("ConfigureProxy error: %v", err)
	}

	// Read back the written file
	configPath := filepath.Join(tmpHome, ".claude", "settings.json")
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("read config: %v", err)
	}

	var settings map[string]interface{}
	if err := json.Unmarshal(data, &settings); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	env, ok := settings["env"].(map[string]interface{})
	if !ok {
		t.Fatal("missing env block in settings")
	}
	if env["ANTHROPIC_BASE_URL"] != endpoint {
		t.Errorf("ANTHROPIC_BASE_URL = %v", env["ANTHROPIC_BASE_URL"])
	}
	if env["ANTHROPIC_API_KEY"] != apiKey {
		t.Errorf("ANTHROPIC_API_KEY = %v", env["ANTHROPIC_API_KEY"])
	}
}

func TestPicoClawInstaller_ConfigureProxy_WritesCorrectContent(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	t.Setenv("USERPROFILE", tmpHome)

	inst := NewPicoClawInstaller()
	ctx := context.Background()

	endpoint := "https://proxy.example.com/v1"
	apiKey := "sk-test"
	if err := inst.ConfigureProxy(ctx, endpoint, apiKey); err != nil {
		t.Fatalf("ConfigureProxy error: %v", err)
	}

	configPath := filepath.Join(tmpHome, ".picoclaw", "config.json")
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("read config: %v", err)
	}

	var cfg map[string]interface{}
	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	modelList, ok := cfg["model_list"].([]interface{})
	if !ok {
		t.Fatal("missing model_list")
	}
	if len(modelList) != 1 {
		t.Fatalf("model_list length = %d, want 1", len(modelList))
	}

	entry, ok := modelList[0].(map[string]interface{})
	if !ok {
		t.Fatal("model_list[0] is not a map")
	}
	if entry["name"] != "code-switch" {
		t.Errorf("name = %v", entry["name"])
	}
	if entry["api_base"] != endpoint {
		t.Errorf("api_base = %v", entry["api_base"])
	}
	if entry["api_key"] != apiKey {
		t.Errorf("api_key = %v", entry["api_key"])
	}
	if entry["model_name"] != DefaultPicoClawModel {
		t.Errorf("model_name = %v, want %s", entry["model_name"], DefaultPicoClawModel)
	}
}

func TestPicoClawInstaller_ConfigureProxy_Upsert(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	t.Setenv("USERPROFILE", tmpHome)

	inst := NewPicoClawInstaller()
	ctx := context.Background()

	// First call creates the entry
	inst.ConfigureProxy(ctx, "https://old.example.com", "old-key")

	// Second call should update, not duplicate
	inst.ConfigureProxy(ctx, "https://new.example.com", "new-key")

	configPath := filepath.Join(tmpHome, ".picoclaw", "config.json")
	data, _ := os.ReadFile(configPath)

	var cfg map[string]interface{}
	json.Unmarshal(data, &cfg)

	modelList := cfg["model_list"].([]interface{})
	if len(modelList) != 1 {
		t.Errorf("upsert should keep 1 entry, got %d", len(modelList))
	}

	entry := modelList[0].(map[string]interface{})
	if entry["api_base"] != "https://new.example.com" {
		t.Errorf("api_base not updated: %v", entry["api_base"])
	}
}

func TestPicoClawInstaller_ConfigureProxy_PreservesExistingEntries(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	t.Setenv("USERPROFILE", tmpHome)

	// Pre-create config with existing model
	configDir := filepath.Join(tmpHome, ".picoclaw")
	os.MkdirAll(configDir, 0755)
	existing := `{"model_list":[{"name":"custom-model","api_base":"https://custom.com","api_key":"custom-key","model_name":"gpt-4"}]}`
	os.WriteFile(filepath.Join(configDir, "config.json"), []byte(existing), 0600)

	inst := NewPicoClawInstaller()
	ctx := context.Background()
	inst.ConfigureProxy(ctx, "https://proxy.com", "proxy-key")

	data, _ := os.ReadFile(filepath.Join(configDir, "config.json"))
	var cfg map[string]interface{}
	json.Unmarshal(data, &cfg)

	modelList := cfg["model_list"].([]interface{})
	if len(modelList) != 2 {
		t.Fatalf("should have 2 entries (custom + code-switch), got %d", len(modelList))
	}

	// Verify custom entry is preserved
	first := modelList[0].(map[string]interface{})
	if first["name"] != "custom-model" {
		t.Errorf("first entry name = %v, custom model should be preserved", first["name"])
	}
}

func TestPicoClawInstaller_ConfigureProxy_CorruptExistingFile(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	t.Setenv("USERPROFILE", tmpHome)

	// Pre-create corrupt config
	configDir := filepath.Join(tmpHome, ".picoclaw")
	os.MkdirAll(configDir, 0755)
	os.WriteFile(filepath.Join(configDir, "config.json"), []byte("{corrupt}}"), 0600)

	inst := NewPicoClawInstaller()
	ctx := context.Background()

	// Should not fail; should start fresh
	err := inst.ConfigureProxy(ctx, "https://api.com", "key")
	if err != nil {
		t.Fatalf("should handle corrupt file gracefully: %v", err)
	}

	data, _ := os.ReadFile(filepath.Join(configDir, "config.json"))
	var cfg map[string]interface{}
	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("written file should be valid JSON: %v", err)
	}
}

// === File Permission Tests ===

func TestPicoClawInstaller_ConfigureProxy_FilePermissions(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("file permissions not enforced on Windows")
	}

	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	t.Setenv("USERPROFILE", tmpHome)

	inst := NewPicoClawInstaller()
	ctx := context.Background()
	inst.ConfigureProxy(ctx, "https://api.com", "key")

	configPath := filepath.Join(tmpHome, ".picoclaw", "config.json")
	info, err := os.Stat(configPath)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}

	perm := info.Mode().Perm()
	if perm != 0600 {
		t.Errorf("file permissions = %o, want 0600", perm)
	}
}

// === DefaultPicoClawModel Constant Tests ===

func TestDefaultPicoClawModel_NotEmpty(t *testing.T) {
	if DefaultPicoClawModel == "" {
		t.Error("DefaultPicoClawModel should not be empty")
	}
}

// === Manager Batch Operation Tests ===

func TestManager_InstallAll_ReturnsResults(t *testing.T) {
	mgr := NewManager()
	ctx := context.Background()

	// InstallAll will likely fail (no python/bun in test env) but should return results
	results := mgr.InstallAll(ctx)
	if len(results) != 4 {
		t.Errorf("expected 4 results from InstallAll, got %d", len(results))
	}

	// Each result should have a tool name
	for _, r := range results {
		if r.Tool == "" {
			t.Error("result should have non-empty Tool")
		}
	}
}

func TestManager_UpdateAll_ReturnsResults(t *testing.T) {
	mgr := NewManager()
	ctx := context.Background()

	results := mgr.UpdateAll(ctx)
	if len(results) != 4 {
		t.Errorf("expected 4 results from UpdateAll, got %d", len(results))
	}
}

func TestManager_ConfigureAllProxy_SkipsUninstalled(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	t.Setenv("USERPROFILE", tmpHome)

	mgr := NewManager()
	ctx := context.Background()

	// On a clean system, all tools are uninstalled, so no errors should occur
	errs := mgr.ConfigureAllProxy(ctx, "https://api.com", "key")
	// We can't guarantee all tools are uninstalled, but we verify it doesn't crash
	_ = errs
}

func TestManager_GetRuntime_NotNil(t *testing.T) {
	mgr := NewManager()
	if mgr.GetRuntime() == nil {
		t.Error("GetRuntime should not return nil")
	}
}
