package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// === Constructor Tests ===

func TestNewStore(t *testing.T) {
	// Use temp dir to avoid polluting real config
	tmpDir := t.TempDir()
	t.Setenv("APPDATA", tmpDir)
	t.Setenv("HOME", tmpDir)
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	store, err := NewStore()
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}

	if store == nil {
		t.Fatal("NewStore should return non-nil store")
	}

	if store.configDir == "" {
		t.Error("configDir should not be empty")
	}
}

func TestNewStore_CreatesConfigDir(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("APPDATA", tmpDir)
	t.Setenv("HOME", tmpDir)
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	store, err := NewStore()
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}

	// Check that config directory was created
	if _, err := os.Stat(store.configDir); os.IsNotExist(err) {
		t.Error("Config directory should be created")
	}
}

func TestStore_GetConfigDir(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("APPDATA", tmpDir)
	t.Setenv("HOME", tmpDir)
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	store, err := NewStore()
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}

	configDir := store.GetConfigDir()
	if configDir == "" {
		t.Error("GetConfigDir should return non-empty path")
	}
}

// === Claude Config CRUD Tests ===

func TestStore_SaveClaudeConfig(t *testing.T) {
	store := createTestStore(t)
	cfg := NewClaudeConfig()
	cfg.CustomInstructions = "Test instructions"

	err := store.SaveClaudeConfig("test-config", cfg)
	if err != nil {
		t.Fatalf("Failed to save config: %v", err)
	}

	// Verify file exists
	configPath := filepath.Join(store.configDir, "claude", "test-config.json")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Error("Config file should be created")
	}
}

func TestStore_LoadClaudeConfig(t *testing.T) {
	store := createTestStore(t)
	original := NewClaudeConfig()
	original.CustomInstructions = "Test instructions"
	original.Model = "claude-3-opus"

	err := store.SaveClaudeConfig("test-config", original)
	if err != nil {
		t.Fatalf("Failed to save config: %v", err)
	}

	loaded, err := store.LoadClaudeConfig("test-config")
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	if loaded.CustomInstructions != original.CustomInstructions {
		t.Error("CustomInstructions mismatch")
	}
	if loaded.Model != original.Model {
		t.Error("Model mismatch")
	}
}

func TestStore_LoadClaudeConfig_NotFound(t *testing.T) {
	store := createTestStore(t)

	_, err := store.LoadClaudeConfig("nonexistent")
	if err == nil {
		t.Error("Expected error for nonexistent config")
	}
}

func TestStore_LoadClaudeConfig_InvalidJSON(t *testing.T) {
	store := createTestStore(t)

	// Create invalid JSON file
	toolDir := filepath.Join(store.configDir, "claude")
	if err := os.MkdirAll(toolDir, 0755); err != nil {
		t.Fatalf("Failed to create tool dir: %v", err)
	}
	configPath := filepath.Join(toolDir, "invalid.json")
	if err := os.WriteFile(configPath, []byte("not valid json"), 0644); err != nil {
		t.Fatalf("Failed to write invalid file: %v", err)
	}

	_, err := store.LoadClaudeConfig("invalid")
	if err == nil {
		t.Error("Expected error for invalid JSON")
	}
}

// === Codex Config CRUD Tests ===

func TestStore_SaveCodexConfig(t *testing.T) {
	store := createTestStore(t)
	cfg := NewCodexConfig()
	cfg.Model = "gpt-4"

	err := store.SaveCodexConfig("test-codex", cfg)
	if err != nil {
		t.Fatalf("Failed to save config: %v", err)
	}

	configPath := filepath.Join(store.configDir, "codex", "test-codex.json")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Error("Config file should be created")
	}
}

func TestStore_LoadCodexConfig(t *testing.T) {
	store := createTestStore(t)
	original := NewCodexConfig()
	original.Model = "gpt-4-turbo"
	original.ApprovalMode = "full-auto"

	err := store.SaveCodexConfig("test-codex", original)
	if err != nil {
		t.Fatalf("Failed to save config: %v", err)
	}

	loaded, err := store.LoadCodexConfig("test-codex")
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	if loaded.Model != original.Model {
		t.Error("Model mismatch")
	}
	if loaded.ApprovalMode != original.ApprovalMode {
		t.Error("ApprovalMode mismatch")
	}
}

func TestStore_LoadCodexConfig_NotFound(t *testing.T) {
	store := createTestStore(t)

	_, err := store.LoadCodexConfig("nonexistent")
	if err == nil {
		t.Error("Expected error for nonexistent config")
	}
}

// === Gemini Config CRUD Tests ===

func TestStore_SaveGeminiConfig(t *testing.T) {
	store := createTestStore(t)
	cfg := NewGeminiConfig()
	cfg.ProjectID = "test-project"

	err := store.SaveGeminiConfig("test-gemini", cfg)
	if err != nil {
		t.Fatalf("Failed to save config: %v", err)
	}

	configPath := filepath.Join(store.configDir, "gemini", "test-gemini.json")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Error("Config file should be created")
	}
}

func TestStore_LoadGeminiConfig(t *testing.T) {
	store := createTestStore(t)
	original := NewGeminiConfig()
	original.Model = "gemini-1.5-pro"
	original.ProjectID = "test-project"

	err := store.SaveGeminiConfig("test-gemini", original)
	if err != nil {
		t.Fatalf("Failed to save config: %v", err)
	}

	loaded, err := store.LoadGeminiConfig("test-gemini")
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	if loaded.Model != original.Model {
		t.Error("Model mismatch")
	}
	if loaded.ProjectID != original.ProjectID {
		t.Error("ProjectID mismatch")
	}
}

func TestStore_LoadGeminiConfig_NotFound(t *testing.T) {
	store := createTestStore(t)

	_, err := store.LoadGeminiConfig("nonexistent")
	if err == nil {
		t.Error("Expected error for nonexistent config")
	}
}

// === ListConfigs Tests ===

func TestStore_ListConfigs_Empty(t *testing.T) {
	store := createTestStore(t)

	configs, err := store.ListConfigs("claude")
	if err != nil {
		t.Fatalf("Failed to list configs: %v", err)
	}

	if len(configs) != 0 {
		t.Errorf("Expected 0 configs, got %d", len(configs))
	}
}

func TestStore_ListConfigs_MultipleFiles(t *testing.T) {
	store := createTestStore(t)

	// Save multiple configs
	for i, name := range []string{"config1", "config2", "config3"} {
		cfg := NewClaudeConfig()
		cfg.CustomInstructions = "Test " + name
		cfg.MaxTokens = i * 1000
		if err := store.SaveClaudeConfig(name, cfg); err != nil {
			t.Fatalf("Failed to save %s: %v", name, err)
		}
	}

	configs, err := store.ListConfigs("claude")
	if err != nil {
		t.Fatalf("Failed to list configs: %v", err)
	}

	if len(configs) != 3 {
		t.Errorf("Expected 3 configs, got %d", len(configs))
	}
}

func TestStore_ListConfigs_IgnoresNonJSON(t *testing.T) {
	store := createTestStore(t)

	// Save a valid config
	cfg := NewClaudeConfig()
	if err := store.SaveClaudeConfig("valid-config", cfg); err != nil {
		t.Fatalf("Failed to save config: %v", err)
	}

	// Create non-JSON file
	toolDir := filepath.Join(store.configDir, "claude")
	nonJSONPath := filepath.Join(toolDir, "notes.txt")
	if err := os.WriteFile(nonJSONPath, []byte("some notes"), 0644); err != nil {
		t.Fatalf("Failed to write non-JSON file: %v", err)
	}

	configs, err := store.ListConfigs("claude")
	if err != nil {
		t.Fatalf("Failed to list configs: %v", err)
	}

	if len(configs) != 1 {
		t.Errorf("Expected 1 config (ignoring .txt), got %d", len(configs))
	}
	if configs[0] != "valid-config" {
		t.Errorf("Expected 'valid-config', got '%s'", configs[0])
	}
}

func TestStore_ListConfigs_DifferentTools(t *testing.T) {
	store := createTestStore(t)

	// Save configs for different tools
	if err := store.SaveClaudeConfig("claude-config", NewClaudeConfig()); err != nil {
		t.Fatalf("Failed to save claude config: %v", err)
	}
	if err := store.SaveCodexConfig("codex-config", NewCodexConfig()); err != nil {
		t.Fatalf("Failed to save codex config: %v", err)
	}
	if err := store.SaveGeminiConfig("gemini-config", NewGeminiConfig()); err != nil {
		t.Fatalf("Failed to save gemini config: %v", err)
	}

	// List each tool's configs
	claudeConfigs, _ := store.ListConfigs("claude")
	codexConfigs, _ := store.ListConfigs("codex")
	geminiConfigs, _ := store.ListConfigs("gemini")

	if len(claudeConfigs) != 1 {
		t.Error("Expected 1 claude config")
	}
	if len(codexConfigs) != 1 {
		t.Error("Expected 1 codex config")
	}
	if len(geminiConfigs) != 1 {
		t.Error("Expected 1 gemini config")
	}
}

// === DeleteConfig Tests ===

func TestStore_DeleteConfig(t *testing.T) {
	store := createTestStore(t)

	// Save then delete
	cfg := NewClaudeConfig()
	if err := store.SaveClaudeConfig("to-delete", cfg); err != nil {
		t.Fatalf("Failed to save config: %v", err)
	}

	err := store.DeleteConfig("claude", "to-delete")
	if err != nil {
		t.Fatalf("Failed to delete config: %v", err)
	}

	// Verify file is gone
	configPath := filepath.Join(store.configDir, "claude", "to-delete.json")
	if _, err := os.Stat(configPath); !os.IsNotExist(err) {
		t.Error("Config file should be deleted")
	}
}

func TestStore_DeleteConfig_NotFound(t *testing.T) {
	store := createTestStore(t)

	err := store.DeleteConfig("claude", "nonexistent")
	if err == nil {
		t.Error("Expected error for nonexistent config")
	}
}

// === Cross-Platform Config Directory Tests ===

func TestGetConfigDir_Windows(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Skipping Windows-specific test")
	}

	tmpDir := t.TempDir()
	t.Setenv("APPDATA", tmpDir)

	configDir, err := getConfigDir()
	if err != nil {
		t.Fatalf("Failed to get config dir: %v", err)
	}

	expectedPrefix := tmpDir
	if !hasPrefix(configDir, expectedPrefix) {
		t.Errorf("Expected config dir to start with '%s', got '%s'", expectedPrefix, configDir)
	}

	if !hasSuffix(configDir, filepath.Join("lurus-switch", "configs")) {
		t.Error("Config dir should end with lurus-switch/configs")
	}
}

func TestGetConfigDir_Darwin(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("Skipping macOS-specific test")
	}

	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	configDir, err := getConfigDir()
	if err != nil {
		t.Fatalf("Failed to get config dir: %v", err)
	}

	expectedPart := "Library/Application Support"
	if !containsPath(configDir, expectedPart) {
		t.Errorf("Expected config dir to contain '%s', got '%s'", expectedPart, configDir)
	}
}

func TestGetConfigDir_Linux(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Skipping Linux-specific test")
	}

	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("HOME", tmpDir)

	configDir, err := getConfigDir()
	if err != nil {
		t.Fatalf("Failed to get config dir: %v", err)
	}

	expectedPart := ".config"
	if !containsPath(configDir, expectedPart) {
		t.Errorf("Expected config dir to contain '%s', got '%s'", expectedPart, configDir)
	}
}

func TestGetConfigDir_XDGConfigHome(t *testing.T) {
	if runtime.GOOS == "windows" || runtime.GOOS == "darwin" {
		t.Skip("XDG_CONFIG_HOME only applies to Linux")
	}

	tmpDir := t.TempDir()
	customXDG := filepath.Join(tmpDir, "custom-config")
	t.Setenv("XDG_CONFIG_HOME", customXDG)

	configDir, err := getConfigDir()
	if err != nil {
		t.Fatalf("Failed to get config dir: %v", err)
	}

	if !hasPrefix(configDir, customXDG) {
		t.Errorf("Expected config dir to use XDG_CONFIG_HOME '%s', got '%s'", customXDG, configDir)
	}
}

// === saveConfig/loadConfig Internal Tests ===

func TestSaveConfig_CreatesToolDir(t *testing.T) {
	store := createTestStore(t)
	cfg := NewClaudeConfig()

	err := store.saveConfig("new-tool", "config1", cfg)
	if err != nil {
		t.Fatalf("Failed to save config: %v", err)
	}

	toolDir := filepath.Join(store.configDir, "new-tool")
	if _, err := os.Stat(toolDir); os.IsNotExist(err) {
		t.Error("Tool directory should be created")
	}
}

func TestSaveConfig_JSONIndented(t *testing.T) {
	store := createTestStore(t)
	cfg := NewClaudeConfig()
	cfg.CustomInstructions = "Test"

	err := store.saveConfig("claude", "test", cfg)
	if err != nil {
		t.Fatalf("Failed to save config: %v", err)
	}

	// Read the file and check it's indented
	configPath := filepath.Join(store.configDir, "claude", "test.json")
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("Failed to read config file: %v", err)
	}

	// Indented JSON should have newlines
	if !containsPath(string(data), "\n") {
		t.Error("JSON should be indented (contain newlines)")
	}
}

func TestLoadConfig_PartialData(t *testing.T) {
	store := createTestStore(t)

	// Create config with only some fields
	partialJSON := `{"model": "custom-model"}`
	toolDir := filepath.Join(store.configDir, "claude")
	if err := os.MkdirAll(toolDir, 0755); err != nil {
		t.Fatalf("Failed to create tool dir: %v", err)
	}
	configPath := filepath.Join(toolDir, "partial.json")
	if err := os.WriteFile(configPath, []byte(partialJSON), 0644); err != nil {
		t.Fatalf("Failed to write partial config: %v", err)
	}

	var cfg ClaudeConfig
	err := store.loadConfig("claude", "partial", &cfg)
	if err != nil {
		t.Fatalf("Failed to load partial config: %v", err)
	}

	if cfg.Model != "custom-model" {
		t.Error("Model should be loaded")
	}
	// Other fields should be zero values
	if cfg.MaxTokens != 0 {
		t.Error("MaxTokens should be zero (not in JSON)")
	}
}

// === Edge Cases ===

func TestStore_SaveConfig_SpecialCharactersInName(t *testing.T) {
	store := createTestStore(t)
	cfg := NewClaudeConfig()

	// Names with special characters (valid for filenames)
	validNames := []string{
		"config-with-dashes",
		"config_with_underscores",
		"config.with.dots",
		"Config123",
	}

	for _, name := range validNames {
		t.Run(name, func(t *testing.T) {
			err := store.SaveClaudeConfig(name, cfg)
			if err != nil {
				t.Errorf("Failed to save config with name '%s': %v", name, err)
			}

			loaded, err := store.LoadClaudeConfig(name)
			if err != nil {
				t.Errorf("Failed to load config with name '%s': %v", name, err)
			}
			if loaded == nil {
				t.Error("Loaded config should not be nil")
			}
		})
	}
}

func TestStore_ComplexConfig_RoundTrip(t *testing.T) {
	store := createTestStore(t)

	// Create a complex config with all fields
	cfg := &ClaudeConfig{
		Model:              "claude-3-opus",
		CustomInstructions: "Complex instructions with\nnewlines and \"quotes\"",
		APIKey:             "sk-ant-test123",
		MaxTokens:          16384,
		Permissions: ClaudePermissions{
			AllowBash:          true,
			AllowRead:          true,
			AllowWrite:         false,
			AllowWebFetch:      true,
			TrustedDirectories: []string{"/home/user", "/tmp"},
			AllowedBashCommands: []string{"git*", "bun*"},
			DeniedBashCommands: []string{"rm -rf*"},
		},
		MCPServers: map[string]MCPServer{
			"fs": {
				Command: "mcp-fs",
				Args:    []string{"--root", "/"},
				Env:     map[string]string{"DEBUG": "true"},
			},
		},
		Sandbox: ClaudeSandbox{
			Enabled:     true,
			Type:        "docker",
			DockerImage: "ubuntu:22.04",
			Mounts: []SandboxMount{
				{Source: "/home", Destination: "/home", ReadOnly: false},
			},
		},
		Advanced: ClaudeAdvanced{
			Verbose:              true,
			DisableTelemetry:     true,
			APIEndpoint:          "https://custom.api.anthropic.com",
			Timeout:              600,
			ExperimentalFeatures: true,
		},
	}

	err := store.SaveClaudeConfig("complex", cfg)
	if err != nil {
		t.Fatalf("Failed to save complex config: %v", err)
	}

	loaded, err := store.LoadClaudeConfig("complex")
	if err != nil {
		t.Fatalf("Failed to load complex config: %v", err)
	}

	// Verify all fields
	if loaded.Model != cfg.Model {
		t.Error("Model mismatch")
	}
	if loaded.CustomInstructions != cfg.CustomInstructions {
		t.Error("CustomInstructions mismatch")
	}
	if loaded.Permissions.AllowWebFetch != cfg.Permissions.AllowWebFetch {
		t.Error("AllowWebFetch mismatch")
	}
	if len(loaded.MCPServers) != 1 {
		t.Error("MCPServers length mismatch")
	}
	if loaded.Sandbox.DockerImage != cfg.Sandbox.DockerImage {
		t.Error("DockerImage mismatch")
	}
	if loaded.Advanced.Timeout != cfg.Advanced.Timeout {
		t.Error("Timeout mismatch")
	}
}

// === Test Helpers ===

func createTestStore(t *testing.T) *Store {
	t.Helper()

	tmpDir := t.TempDir()
	configDir := filepath.Join(tmpDir, "lurus-switch", "configs")

	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("Failed to create config dir: %v", err)
	}

	return &Store{configDir: configDir}
}

func hasPrefix(s, prefix string) bool {
	return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}

func hasSuffix(s, suffix string) bool {
	return len(s) >= len(suffix) && s[len(s)-len(suffix):] == suffix
}

func containsPath(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// === Benchmark Tests ===

func BenchmarkStore_SaveClaudeConfig(b *testing.B) {
	tmpDir := b.TempDir()
	configDir := filepath.Join(tmpDir, "lurus-switch", "configs")
	os.MkdirAll(configDir, 0755)
	store := &Store{configDir: configDir}

	cfg := NewClaudeConfig()
	cfg.CustomInstructions = "Benchmark test instructions"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		store.SaveClaudeConfig("benchmark-config", cfg)
	}
}

func BenchmarkStore_LoadClaudeConfig(b *testing.B) {
	tmpDir := b.TempDir()
	configDir := filepath.Join(tmpDir, "lurus-switch", "configs")
	os.MkdirAll(configDir, 0755)
	store := &Store{configDir: configDir}

	cfg := NewClaudeConfig()
	store.SaveClaudeConfig("benchmark-config", cfg)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		store.LoadClaudeConfig("benchmark-config")
	}
}

func BenchmarkClaudeConfig_JSONMarshal(b *testing.B) {
	cfg := NewClaudeConfig()
	cfg.CustomInstructions = "Benchmark test"
	cfg.MCPServers = map[string]MCPServer{
		"fs": {Command: "mcp-fs", Args: []string{"--root", "/"}},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		json.Marshal(cfg)
	}
}
