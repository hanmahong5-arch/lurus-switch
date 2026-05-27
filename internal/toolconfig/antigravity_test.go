package toolconfig

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// === antigravityConfigDir Tests ===

func TestAntigravityConfigDir_NonEmpty(t *testing.T) {
	dir := antigravityConfigDir()
	if dir == "" {
		t.Fatal("antigravityConfigDir() returned empty string")
	}
}

func TestAntigravityConfigDir_ContainsAntigravity(t *testing.T) {
	dir := antigravityConfigDir()
	lower := strings.ToLower(dir)
	if !strings.Contains(lower, "antigravity") {
		t.Errorf("antigravityConfigDir() = %q, expected to contain 'antigravity'", dir)
	}
}

func TestAntigravityConfigDir_WindowsUsesLocalAppData(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows-only test")
	}
	localAppData := os.Getenv("LOCALAPPDATA")
	if localAppData == "" {
		t.Skip("LOCALAPPDATA not set")
	}
	dir := antigravityConfigDir()
	if !strings.HasPrefix(dir, localAppData) {
		t.Errorf("on Windows, antigravityConfigDir() = %q should be under LOCALAPPDATA=%q", dir, localAppData)
	}
}

// === ReadAntigravityConfig Tests ===

func TestReadAntigravityConfig_FileNotExist_ReturnsEmpty(t *testing.T) {
	// Point LOCALAPPDATA / HOME to a temp dir so no real config is read.
	tmp := t.TempDir()
	t.Setenv("LOCALAPPDATA", tmp)
	t.Setenv("HOME", tmp)
	t.Setenv("XDG_CONFIG_HOME", tmp)

	cfg, err := ReadAntigravityConfig()
	if err != nil {
		t.Fatalf("ReadAntigravityConfig() with no file should not error, got: %v", err)
	}
	if cfg == nil {
		t.Fatal("expected non-nil config for missing file")
	}
	// Zero-value struct expected
	if cfg.APIKey != "" {
		t.Errorf("APIKey = %q, want empty", cfg.APIKey)
	}
}

func TestReadAntigravityConfig_ValidFile(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("LOCALAPPDATA", tmp)
	t.Setenv("HOME", tmp)
	t.Setenv("XDG_CONFIG_HOME", tmp)

	configDir := antigravityConfigDir()
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

	want := AntigravityConfig{
		APIKey:      "test-key-123",
		APIEndpoint: "https://example.com/v1",
		Model:       AntigravityModelConfig{Name: "gemini-2.5-pro"},
		General:     AntigravityGeneralConfig{DefaultApprovalMode: "auto"},
	}
	data, _ := json.MarshalIndent(want, "", "  ")
	configPath := filepath.Join(configDir, AntigravityConfigFilename)
	if err := os.WriteFile(configPath, data, 0600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	got, err := ReadAntigravityConfig()
	if err != nil {
		t.Fatalf("ReadAntigravityConfig() error: %v", err)
	}
	if got.APIKey != want.APIKey {
		t.Errorf("APIKey = %q, want %q", got.APIKey, want.APIKey)
	}
	if got.APIEndpoint != want.APIEndpoint {
		t.Errorf("APIEndpoint = %q, want %q", got.APIEndpoint, want.APIEndpoint)
	}
	if got.Model.Name != want.Model.Name {
		t.Errorf("Model.Name = %q, want %q", got.Model.Name, want.Model.Name)
	}
	if got.General.DefaultApprovalMode != want.General.DefaultApprovalMode {
		t.Errorf("General.DefaultApprovalMode = %q, want %q", got.General.DefaultApprovalMode, want.General.DefaultApprovalMode)
	}
}

func TestReadAntigravityConfig_InvalidJSON_ReturnsError(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("LOCALAPPDATA", tmp)
	t.Setenv("HOME", tmp)
	t.Setenv("XDG_CONFIG_HOME", tmp)

	configDir := antigravityConfigDir()
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	configPath := filepath.Join(configDir, AntigravityConfigFilename)
	if err := os.WriteFile(configPath, []byte("{invalid json"), 0600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	_, err := ReadAntigravityConfig()
	if err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}
}

// === WriteAntigravityConfig Tests ===

func TestWriteAntigravityConfig_NilReturnsError(t *testing.T) {
	err := WriteAntigravityConfig(nil)
	if err == nil {
		t.Fatal("expected error for nil config")
	}
}

func TestWriteAntigravityConfig_CreatesDirectory(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("LOCALAPPDATA", tmp)
	t.Setenv("HOME", tmp)
	t.Setenv("XDG_CONFIG_HOME", tmp)

	cfg := &AntigravityConfig{APIKey: "write-test"}
	if err := WriteAntigravityConfig(cfg); err != nil {
		t.Fatalf("WriteAntigravityConfig error: %v", err)
	}

	configDir := antigravityConfigDir()
	stat, err := os.Stat(configDir)
	if err != nil {
		t.Fatalf("config directory not created: %v", err)
	}
	if !stat.IsDir() {
		t.Error("expected a directory")
	}
}

func TestWriteAntigravityConfig_RoundTrip(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("LOCALAPPDATA", tmp)
	t.Setenv("HOME", tmp)
	t.Setenv("XDG_CONFIG_HOME", tmp)

	want := &AntigravityConfig{
		APIKey:      "round-trip-key",
		APIEndpoint: "https://proxy.example.com",
		Model:       AntigravityModelConfig{Name: "gemini-2.5-flash"},
		Proxy:       "http://localhost:8080",
	}

	if err := WriteAntigravityConfig(want); err != nil {
		t.Fatalf("WriteAntigravityConfig error: %v", err)
	}

	got, err := ReadAntigravityConfig()
	if err != nil {
		t.Fatalf("ReadAntigravityConfig error: %v", err)
	}
	if got.APIKey != want.APIKey {
		t.Errorf("APIKey round-trip: got %q, want %q", got.APIKey, want.APIKey)
	}
	if got.APIEndpoint != want.APIEndpoint {
		t.Errorf("APIEndpoint round-trip: got %q, want %q", got.APIEndpoint, want.APIEndpoint)
	}
	if got.Model.Name != want.Model.Name {
		t.Errorf("Model.Name round-trip: got %q, want %q", got.Model.Name, want.Model.Name)
	}
	if got.Proxy != want.Proxy {
		t.Errorf("Proxy round-trip: got %q, want %q", got.Proxy, want.Proxy)
	}
}

// === toolDefs integration Tests ===

func TestToolDefs_ContainsAntigravity(t *testing.T) {
	if _, ok := toolDefs[ToolAntigravity]; !ok {
		t.Errorf("toolDefs missing %q entry", ToolAntigravity)
	}
}

func TestGetConfigPath_Antigravity(t *testing.T) {
	path, err := GetConfigPath(ToolAntigravity)
	if err != nil {
		t.Fatalf("GetConfigPath(%q) error: %v", ToolAntigravity, err)
	}
	if !strings.HasSuffix(path, AntigravityConfigFilename) {
		t.Errorf("path %q should end with %q", path, AntigravityConfigFilename)
	}
}

func TestDefaultTemplates_ContainsAntigravity(t *testing.T) {
	tmpl, ok := defaultTemplates[ToolAntigravity]
	if !ok {
		t.Errorf("defaultTemplates missing %q entry", ToolAntigravity)
	}
	if tmpl == "" {
		t.Errorf("defaultTemplates[%q] is empty", ToolAntigravity)
	}
	// Template must be valid JSON
	var raw map[string]interface{}
	if err := json.Unmarshal([]byte(tmpl), &raw); err != nil {
		t.Errorf("defaultTemplates[%q] is not valid JSON: %v", ToolAntigravity, err)
	}
}

func TestGetAllConfigPaths_IncludesAntigravity(t *testing.T) {
	paths := GetAllConfigPaths()
	if _, ok := paths[ToolAntigravity]; !ok {
		t.Errorf("GetAllConfigPaths() missing %q", ToolAntigravity)
	}
}

func TestReadConfig_Antigravity_NonExistentFile_ReturnsDefault(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("LOCALAPPDATA", tmp)
	t.Setenv("HOME", tmp)
	t.Setenv("XDG_CONFIG_HOME", tmp)

	info, err := ReadConfig(ToolAntigravity)
	if err != nil {
		t.Fatalf("ReadConfig(%q) error: %v", ToolAntigravity, err)
	}
	if info.Exists {
		t.Error("Exists should be false for non-existent file")
	}
	if info.Content == "" {
		t.Error("Content should contain default template")
	}
	if info.Tool != ToolAntigravity {
		t.Errorf("Tool = %q, want %q", info.Tool, ToolAntigravity)
	}
	if info.Language != "json" {
		t.Errorf("Language = %q, want json", info.Language)
	}
}

func TestWriteConfig_Antigravity_Success(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("LOCALAPPDATA", tmp)
	t.Setenv("HOME", tmp)
	t.Setenv("XDG_CONFIG_HOME", tmp)

	content := `{"apiKey": "via-write-config"}`
	if err := WriteConfig(ToolAntigravity, content); err != nil {
		t.Fatalf("WriteConfig(%q) error: %v", ToolAntigravity, err)
	}

	configPath := filepath.Join(antigravityConfigDir(), AntigravityConfigFilename)
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("read written file: %v", err)
	}
	if string(data) != content {
		t.Errorf("file content = %q, want %q", string(data), content)
	}
}
