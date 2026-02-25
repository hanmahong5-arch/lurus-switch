package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"lurus-switch/internal/config"
	"lurus-switch/internal/validator"
)

// === NewApp Tests ===

func TestNewApp(t *testing.T) {
	app := NewApp()
	if app == nil {
		t.Fatal("NewApp should return non-nil app")
	}
	if app.validator == nil {
		t.Error("App should have validator initialized")
	}
	// Store may or may not be nil depending on environment
}

func TestNewApp_ValidatorInitialized(t *testing.T) {
	app := NewApp()
	if app.validator == nil {
		t.Error("Validator should be initialized")
	}
}

// === startup Tests ===

func TestApp_startup(t *testing.T) {
	app := NewApp()
	ctx := context.Background()
	app.startup(ctx)

	if app.ctx == nil {
		t.Error("Context should be set after startup")
	}
}

// === Test App with Mock Store ===

// createTestApp creates an app with a test store in a temp directory
func createTestApp(t *testing.T) (*App, string) {
	t.Helper()
	tmpDir := t.TempDir()

	// Set environment variable to use temp dir for config
	originalEnv := os.Getenv("LOCALAPPDATA")
	os.Setenv("LOCALAPPDATA", tmpDir)
	t.Cleanup(func() {
		os.Setenv("LOCALAPPDATA", originalEnv)
	})

	store, err := config.NewStore()
	if err != nil {
		t.Fatalf("Failed to create test store: %v", err)
	}

	app := &App{
		store:     store,
		validator: validator.NewValidator(),
	}

	return app, tmpDir
}

// createTestAppNilStore creates an app with nil store for testing error paths
func createTestAppNilStore() *App {
	return &App{
		store:     nil,
		validator: validator.NewValidator(),
	}
}

// ============================
// Claude Code Method Tests
// ============================

func TestApp_GetDefaultClaudeConfig(t *testing.T) {
	app := NewApp()
	cfg := app.GetDefaultClaudeConfig()

	if cfg == nil {
		t.Fatal("GetDefaultClaudeConfig should return non-nil config")
	}
	if cfg.Model == "" {
		t.Error("Default config should have model set")
	}
	if cfg.Model != "claude-sonnet-4-20250514" {
		t.Errorf("Expected default model 'claude-sonnet-4-20250514', got '%s'", cfg.Model)
	}
}

func TestApp_GetDefaultClaudeConfig_HasPermissions(t *testing.T) {
	app := NewApp()
	cfg := app.GetDefaultClaudeConfig()

	if cfg.Permissions.AllowBash != true {
		t.Error("Default should allow bash")
	}
	if cfg.Permissions.AllowRead != true {
		t.Error("Default should allow read")
	}
	if cfg.Permissions.AllowWrite != true {
		t.Error("Default should allow write")
	}
}

func TestApp_SaveClaudeConfig(t *testing.T) {
	app, _ := createTestApp(t)
	cfg := config.NewClaudeConfig()
	cfg.Model = "test-model"

	err := app.SaveClaudeConfig("test-config", cfg)
	if err != nil {
		t.Fatalf("SaveClaudeConfig failed: %v", err)
	}
}

func TestApp_SaveClaudeConfig_StoreNil(t *testing.T) {
	app := createTestAppNilStore()
	cfg := config.NewClaudeConfig()

	err := app.SaveClaudeConfig("test", cfg)
	if err == nil {
		t.Error("SaveClaudeConfig with nil store should return error")
	}
	if !strings.Contains(err.Error(), "not initialized") {
		t.Error("Error should mention store not initialized")
	}
}

func TestApp_LoadClaudeConfig(t *testing.T) {
	app, _ := createTestApp(t)
	cfg := config.NewClaudeConfig()
	cfg.Model = "saved-model"

	// Save first
	if err := app.SaveClaudeConfig("load-test", cfg); err != nil {
		t.Fatalf("Failed to save config: %v", err)
	}

	// Load
	loaded, err := app.LoadClaudeConfig("load-test")
	if err != nil {
		t.Fatalf("LoadClaudeConfig failed: %v", err)
	}

	if loaded.Model != "saved-model" {
		t.Errorf("Expected model 'saved-model', got '%s'", loaded.Model)
	}
}

func TestApp_LoadClaudeConfig_NotFound(t *testing.T) {
	app, _ := createTestApp(t)

	_, err := app.LoadClaudeConfig("nonexistent")
	if err == nil {
		t.Error("LoadClaudeConfig for nonexistent config should return error")
	}
}

func TestApp_LoadClaudeConfig_StoreNil(t *testing.T) {
	app := createTestAppNilStore()

	_, err := app.LoadClaudeConfig("test")
	if err == nil {
		t.Error("LoadClaudeConfig with nil store should return error")
	}
	if !strings.Contains(err.Error(), "not initialized") {
		t.Error("Error should mention store not initialized")
	}
}

func TestApp_ListClaudeConfigs(t *testing.T) {
	app, _ := createTestApp(t)

	// Save some configs
	cfg := config.NewClaudeConfig()
	app.SaveClaudeConfig("list-test-1", cfg)
	app.SaveClaudeConfig("list-test-2", cfg)

	configs, err := app.ListClaudeConfigs()
	if err != nil {
		t.Fatalf("ListClaudeConfigs failed: %v", err)
	}

	if len(configs) < 2 {
		t.Errorf("Expected at least 2 configs, got %d", len(configs))
	}
}

func TestApp_ListClaudeConfigs_StoreNil(t *testing.T) {
	app := createTestAppNilStore()

	_, err := app.ListClaudeConfigs()
	if err == nil {
		t.Error("ListClaudeConfigs with nil store should return error")
	}
}

func TestApp_DeleteClaudeConfig(t *testing.T) {
	app, _ := createTestApp(t)

	// Save first
	cfg := config.NewClaudeConfig()
	if err := app.SaveClaudeConfig("delete-test", cfg); err != nil {
		t.Fatalf("Failed to save config: %v", err)
	}

	// Delete
	err := app.DeleteClaudeConfig("delete-test")
	if err != nil {
		t.Fatalf("DeleteClaudeConfig failed: %v", err)
	}

	// Verify deleted
	_, err = app.LoadClaudeConfig("delete-test")
	if err == nil {
		t.Error("Config should be deleted")
	}
}

func TestApp_DeleteClaudeConfig_StoreNil(t *testing.T) {
	app := createTestAppNilStore()

	err := app.DeleteClaudeConfig("test")
	if err == nil {
		t.Error("DeleteClaudeConfig with nil store should return error")
	}
}

func TestApp_ValidateClaudeConfig(t *testing.T) {
	app := NewApp()
	cfg := config.NewClaudeConfig()

	result := app.ValidateClaudeConfig(cfg)
	if result == nil {
		t.Fatal("ValidateClaudeConfig should return non-nil result")
	}
	if !result.Valid {
		t.Errorf("Valid config should be valid, got errors: %v", result.Errors)
	}
}

func TestApp_ValidateClaudeConfig_Invalid(t *testing.T) {
	app := NewApp()
	cfg := config.NewClaudeConfig()
	cfg.Model = ""

	result := app.ValidateClaudeConfig(cfg)
	if result.Valid {
		t.Error("Config with empty model should be invalid")
	}
}

func TestApp_GenerateClaudeConfig(t *testing.T) {
	app := NewApp()
	cfg := config.NewClaudeConfig()
	cfg.Model = "test-model"

	content, err := app.GenerateClaudeConfig(cfg)
	if err != nil {
		t.Fatalf("GenerateClaudeConfig failed: %v", err)
	}

	if !strings.Contains(content, "test-model") {
		t.Error("Generated content should contain model name")
	}
}

// ExportClaudeConfig requires Wails runtime, tested via integration tests

// ============================
// Codex Method Tests
// ============================

func TestApp_GetDefaultCodexConfig(t *testing.T) {
	app := NewApp()
	cfg := app.GetDefaultCodexConfig()

	if cfg == nil {
		t.Fatal("GetDefaultCodexConfig should return non-nil config")
	}
	if cfg.Model != "o4-mini" {
		t.Errorf("Expected default model 'o4-mini', got '%s'", cfg.Model)
	}
	if cfg.ApprovalMode != "suggest" {
		t.Errorf("Expected default approval mode 'suggest', got '%s'", cfg.ApprovalMode)
	}
}

func TestApp_SaveCodexConfig(t *testing.T) {
	app, _ := createTestApp(t)
	cfg := config.NewCodexConfig()
	cfg.Model = "codex-test-model"

	err := app.SaveCodexConfig("codex-test", cfg)
	if err != nil {
		t.Fatalf("SaveCodexConfig failed: %v", err)
	}
}

func TestApp_SaveCodexConfig_StoreNil(t *testing.T) {
	app := createTestAppNilStore()
	cfg := config.NewCodexConfig()

	err := app.SaveCodexConfig("test", cfg)
	if err == nil {
		t.Error("SaveCodexConfig with nil store should return error")
	}
}

func TestApp_LoadCodexConfig(t *testing.T) {
	app, _ := createTestApp(t)
	cfg := config.NewCodexConfig()
	cfg.Model = "saved-codex-model"

	if err := app.SaveCodexConfig("codex-load-test", cfg); err != nil {
		t.Fatalf("Failed to save config: %v", err)
	}

	loaded, err := app.LoadCodexConfig("codex-load-test")
	if err != nil {
		t.Fatalf("LoadCodexConfig failed: %v", err)
	}

	if loaded.Model != "saved-codex-model" {
		t.Errorf("Expected model 'saved-codex-model', got '%s'", loaded.Model)
	}
}

func TestApp_LoadCodexConfig_StoreNil(t *testing.T) {
	app := createTestAppNilStore()

	_, err := app.LoadCodexConfig("test")
	if err == nil {
		t.Error("LoadCodexConfig with nil store should return error")
	}
}

func TestApp_ListCodexConfigs(t *testing.T) {
	app, _ := createTestApp(t)

	cfg := config.NewCodexConfig()
	app.SaveCodexConfig("codex-list-1", cfg)
	app.SaveCodexConfig("codex-list-2", cfg)

	configs, err := app.ListCodexConfigs()
	if err != nil {
		t.Fatalf("ListCodexConfigs failed: %v", err)
	}

	if len(configs) < 2 {
		t.Errorf("Expected at least 2 configs, got %d", len(configs))
	}
}

func TestApp_ListCodexConfigs_StoreNil(t *testing.T) {
	app := createTestAppNilStore()

	_, err := app.ListCodexConfigs()
	if err == nil {
		t.Error("ListCodexConfigs with nil store should return error")
	}
}

func TestApp_DeleteCodexConfig(t *testing.T) {
	app, _ := createTestApp(t)

	cfg := config.NewCodexConfig()
	if err := app.SaveCodexConfig("codex-delete-test", cfg); err != nil {
		t.Fatalf("Failed to save config: %v", err)
	}

	err := app.DeleteCodexConfig("codex-delete-test")
	if err != nil {
		t.Fatalf("DeleteCodexConfig failed: %v", err)
	}
}

func TestApp_DeleteCodexConfig_StoreNil(t *testing.T) {
	app := createTestAppNilStore()

	err := app.DeleteCodexConfig("test")
	if err == nil {
		t.Error("DeleteCodexConfig with nil store should return error")
	}
}

func TestApp_ValidateCodexConfig(t *testing.T) {
	app := NewApp()
	cfg := config.NewCodexConfig()

	result := app.ValidateCodexConfig(cfg)
	if result == nil {
		t.Fatal("ValidateCodexConfig should return non-nil result")
	}
	if !result.Valid {
		t.Errorf("Valid config should be valid, got errors: %v", result.Errors)
	}
}

func TestApp_ValidateCodexConfig_Invalid(t *testing.T) {
	app := NewApp()
	cfg := config.NewCodexConfig()
	cfg.Model = ""

	result := app.ValidateCodexConfig(cfg)
	if result.Valid {
		t.Error("Config with empty model should be invalid")
	}
}

func TestApp_GenerateCodexConfig(t *testing.T) {
	app := NewApp()
	cfg := config.NewCodexConfig()
	cfg.Model = "test-codex-model"

	content, err := app.GenerateCodexConfig(cfg)
	if err != nil {
		t.Fatalf("GenerateCodexConfig failed: %v", err)
	}

	if !strings.Contains(content, "test-codex-model") {
		t.Error("Generated content should contain model name")
	}
}

// ============================
// Gemini Method Tests
// ============================

func TestApp_GetDefaultGeminiConfig(t *testing.T) {
	app := NewApp()
	cfg := app.GetDefaultGeminiConfig()

	if cfg == nil {
		t.Fatal("GetDefaultGeminiConfig should return non-nil config")
	}
	if cfg.Model != "gemini-2.0-flash" {
		t.Errorf("Expected default model 'gemini-2.0-flash', got '%s'", cfg.Model)
	}
}

func TestApp_GetDefaultGeminiConfig_HasAuth(t *testing.T) {
	app := NewApp()
	cfg := app.GetDefaultGeminiConfig()

	if cfg.Auth.Type != "api_key" {
		t.Errorf("Expected default auth type 'api_key', got '%s'", cfg.Auth.Type)
	}
}

func TestApp_SaveGeminiConfig(t *testing.T) {
	app, _ := createTestApp(t)
	cfg := config.NewGeminiConfig()
	cfg.Model = "gemini-test-model"

	err := app.SaveGeminiConfig("gemini-test", cfg)
	if err != nil {
		t.Fatalf("SaveGeminiConfig failed: %v", err)
	}
}

func TestApp_SaveGeminiConfig_StoreNil(t *testing.T) {
	app := createTestAppNilStore()
	cfg := config.NewGeminiConfig()

	err := app.SaveGeminiConfig("test", cfg)
	if err == nil {
		t.Error("SaveGeminiConfig with nil store should return error")
	}
}

func TestApp_LoadGeminiConfig(t *testing.T) {
	app, _ := createTestApp(t)
	cfg := config.NewGeminiConfig()
	cfg.Model = "saved-gemini-model"

	if err := app.SaveGeminiConfig("gemini-load-test", cfg); err != nil {
		t.Fatalf("Failed to save config: %v", err)
	}

	loaded, err := app.LoadGeminiConfig("gemini-load-test")
	if err != nil {
		t.Fatalf("LoadGeminiConfig failed: %v", err)
	}

	if loaded.Model != "saved-gemini-model" {
		t.Errorf("Expected model 'saved-gemini-model', got '%s'", loaded.Model)
	}
}

func TestApp_LoadGeminiConfig_StoreNil(t *testing.T) {
	app := createTestAppNilStore()

	_, err := app.LoadGeminiConfig("test")
	if err == nil {
		t.Error("LoadGeminiConfig with nil store should return error")
	}
}

func TestApp_ListGeminiConfigs(t *testing.T) {
	app, _ := createTestApp(t)

	cfg := config.NewGeminiConfig()
	app.SaveGeminiConfig("gemini-list-1", cfg)
	app.SaveGeminiConfig("gemini-list-2", cfg)

	configs, err := app.ListGeminiConfigs()
	if err != nil {
		t.Fatalf("ListGeminiConfigs failed: %v", err)
	}

	if len(configs) < 2 {
		t.Errorf("Expected at least 2 configs, got %d", len(configs))
	}
}

func TestApp_ListGeminiConfigs_StoreNil(t *testing.T) {
	app := createTestAppNilStore()

	_, err := app.ListGeminiConfigs()
	if err == nil {
		t.Error("ListGeminiConfigs with nil store should return error")
	}
}

func TestApp_DeleteGeminiConfig(t *testing.T) {
	app, _ := createTestApp(t)

	cfg := config.NewGeminiConfig()
	if err := app.SaveGeminiConfig("gemini-delete-test", cfg); err != nil {
		t.Fatalf("Failed to save config: %v", err)
	}

	err := app.DeleteGeminiConfig("gemini-delete-test")
	if err != nil {
		t.Fatalf("DeleteGeminiConfig failed: %v", err)
	}
}

func TestApp_DeleteGeminiConfig_StoreNil(t *testing.T) {
	app := createTestAppNilStore()

	err := app.DeleteGeminiConfig("test")
	if err == nil {
		t.Error("DeleteGeminiConfig with nil store should return error")
	}
}

func TestApp_ValidateGeminiConfig(t *testing.T) {
	app := NewApp()
	cfg := config.NewGeminiConfig()

	result := app.ValidateGeminiConfig(cfg)
	if result == nil {
		t.Fatal("ValidateGeminiConfig should return non-nil result")
	}
	if !result.Valid {
		t.Errorf("Valid config should be valid, got errors: %v", result.Errors)
	}
}

func TestApp_ValidateGeminiConfig_Invalid(t *testing.T) {
	app := NewApp()
	cfg := config.NewGeminiConfig()
	cfg.Model = ""

	result := app.ValidateGeminiConfig(cfg)
	if result.Valid {
		t.Error("Config with empty model should be invalid")
	}
}

func TestApp_GenerateGeminiConfig(t *testing.T) {
	app := NewApp()
	cfg := config.NewGeminiConfig()
	cfg.Instructions.ProjectDescription = "Test Project"

	content := app.GenerateGeminiConfig(cfg)

	if !strings.Contains(content, "# GEMINI.md") {
		t.Error("Generated content should contain GEMINI.md header")
	}
	if !strings.Contains(content, "Test Project") {
		t.Error("Generated content should contain project description")
	}
}

// ============================
// Utility Method Tests
// ============================

func TestApp_GetConfigDir(t *testing.T) {
	app, _ := createTestApp(t)

	dir := app.GetConfigDir()
	if dir == "" {
		t.Error("GetConfigDir should return non-empty string")
	}
}

func TestApp_GetConfigDir_StoreNil(t *testing.T) {
	app := createTestAppNilStore()

	dir := app.GetConfigDir()
	if dir != "" {
		t.Error("GetConfigDir with nil store should return empty string")
	}
}

func TestApp_OpenConfigDir_StoreNil(t *testing.T) {
	app := createTestAppNilStore()

	err := app.OpenConfigDir()
	if err == nil {
		t.Error("OpenConfigDir with nil store should return error")
	}
}

func TestApp_CheckBunInstalled(t *testing.T) {
	app := NewApp()

	// Just verify it doesn't panic and returns a boolean
	result := app.CheckBunInstalled()
	_ = result // Result depends on system, just verify no panic
}

func TestApp_CheckNodeInstalled(t *testing.T) {
	app := NewApp()

	// Just verify it doesn't panic and returns a boolean
	result := app.CheckNodeInstalled()
	_ = result // Result depends on system, just verify no panic
}

// ============================
// openDirectory Tests
// ============================

func TestOpenDirectory_CreatesDir(t *testing.T) {
	tmpDir := t.TempDir()
	newDir := filepath.Join(tmpDir, "new-directory")

	// Directory doesn't exist yet
	if _, err := os.Stat(newDir); !os.IsNotExist(err) {
		t.Fatal("Directory should not exist before test")
	}

	// openDirectory should create it (we don't check if explorer opens)
	err := openDirectory(newDir)
	if err != nil {
		// On CI without display, this might fail, but directory should be created
		t.Logf("openDirectory returned error (expected in headless environment): %v", err)
	}

	// Verify directory was created
	if _, err := os.Stat(newDir); os.IsNotExist(err) {
		t.Error("openDirectory should create the directory")
	}
}

// ============================
// Cross-config Tests
// ============================

func TestApp_AllGetDefaultConfigs_NotNil(t *testing.T) {
	app := NewApp()

	if app.GetDefaultClaudeConfig() == nil {
		t.Error("GetDefaultClaudeConfig should not return nil")
	}
	if app.GetDefaultCodexConfig() == nil {
		t.Error("GetDefaultCodexConfig should not return nil")
	}
	if app.GetDefaultGeminiConfig() == nil {
		t.Error("GetDefaultGeminiConfig should not return nil")
	}
}

func TestApp_AllValidateConfigs_ReturnResults(t *testing.T) {
	app := NewApp()

	claudeResult := app.ValidateClaudeConfig(config.NewClaudeConfig())
	codexResult := app.ValidateCodexConfig(config.NewCodexConfig())
	geminiResult := app.ValidateGeminiConfig(config.NewGeminiConfig())

	if claudeResult == nil {
		t.Error("ValidateClaudeConfig should return non-nil result")
	}
	if codexResult == nil {
		t.Error("ValidateCodexConfig should return non-nil result")
	}
	if geminiResult == nil {
		t.Error("ValidateGeminiConfig should return non-nil result")
	}
}

func TestApp_AllGenerateConfigs_ReturnContent(t *testing.T) {
	app := NewApp()

	claudeContent, err := app.GenerateClaudeConfig(config.NewClaudeConfig())
	if err != nil {
		t.Errorf("GenerateClaudeConfig failed: %v", err)
	}
	if claudeContent == "" {
		t.Error("GenerateClaudeConfig should return non-empty content")
	}

	codexContent, err := app.GenerateCodexConfig(config.NewCodexConfig())
	if err != nil {
		t.Errorf("GenerateCodexConfig failed: %v", err)
	}
	if codexContent == "" {
		t.Error("GenerateCodexConfig should return non-empty content")
	}

	geminiContent := app.GenerateGeminiConfig(config.NewGeminiConfig())
	if geminiContent == "" {
		t.Error("GenerateGeminiConfig should return non-empty content")
	}
}

// ============================
// Integration Tests
// ============================

func TestApp_SaveLoadRoundTrip_Claude(t *testing.T) {
	app, _ := createTestApp(t)

	original := config.NewClaudeConfig()
	original.Model = "roundtrip-model"
	original.MaxTokens = 4096
	original.Permissions.AllowBash = false
	original.CustomInstructions = "Test instructions"

	// Save
	if err := app.SaveClaudeConfig("roundtrip", original); err != nil {
		t.Fatalf("Failed to save: %v", err)
	}

	// Load
	loaded, err := app.LoadClaudeConfig("roundtrip")
	if err != nil {
		t.Fatalf("Failed to load: %v", err)
	}

	// Verify
	if loaded.Model != original.Model {
		t.Error("Model mismatch")
	}
	if loaded.MaxTokens != original.MaxTokens {
		t.Error("MaxTokens mismatch")
	}
	if loaded.Permissions.AllowBash != original.Permissions.AllowBash {
		t.Error("AllowBash mismatch")
	}
	if loaded.CustomInstructions != original.CustomInstructions {
		t.Error("CustomInstructions mismatch")
	}
}

func TestApp_SaveLoadRoundTrip_Codex(t *testing.T) {
	app, _ := createTestApp(t)

	original := config.NewCodexConfig()
	original.Model = "codex-roundtrip"
	original.ApprovalMode = "full-auto"
	original.Provider.Type = "azure"
	original.Provider.AzureDeployment = "my-deployment"

	// Save
	if err := app.SaveCodexConfig("roundtrip", original); err != nil {
		t.Fatalf("Failed to save: %v", err)
	}

	// Load
	loaded, err := app.LoadCodexConfig("roundtrip")
	if err != nil {
		t.Fatalf("Failed to load: %v", err)
	}

	// Verify
	if loaded.Model != original.Model {
		t.Error("Model mismatch")
	}
	if loaded.ApprovalMode != original.ApprovalMode {
		t.Error("ApprovalMode mismatch")
	}
	if loaded.Provider.Type != original.Provider.Type {
		t.Error("Provider.Type mismatch")
	}
	if loaded.Provider.AzureDeployment != original.Provider.AzureDeployment {
		t.Error("Provider.AzureDeployment mismatch")
	}
}

func TestApp_SaveLoadRoundTrip_Gemini(t *testing.T) {
	app, _ := createTestApp(t)

	original := config.NewGeminiConfig()
	original.Model = "gemini-roundtrip"
	original.Auth.Type = "oauth"
	original.Instructions.ProjectDescription = "Test project"
	original.Instructions.CustomRules = []string{"Rule 1", "Rule 2"}

	// Save
	if err := app.SaveGeminiConfig("roundtrip", original); err != nil {
		t.Fatalf("Failed to save: %v", err)
	}

	// Load
	loaded, err := app.LoadGeminiConfig("roundtrip")
	if err != nil {
		t.Fatalf("Failed to load: %v", err)
	}

	// Verify
	if loaded.Model != original.Model {
		t.Error("Model mismatch")
	}
	if loaded.Auth.Type != original.Auth.Type {
		t.Error("Auth.Type mismatch")
	}
	if loaded.Instructions.ProjectDescription != original.Instructions.ProjectDescription {
		t.Error("ProjectDescription mismatch")
	}
	if len(loaded.Instructions.CustomRules) != len(original.Instructions.CustomRules) {
		t.Error("CustomRules length mismatch")
	}
}

func TestApp_MultipleConfigs_Isolation(t *testing.T) {
	app, _ := createTestApp(t)

	// Save different configs for each tool
	claudeCfg := config.NewClaudeConfig()
	claudeCfg.Model = "claude-specific"
	app.SaveClaudeConfig("isolation-test", claudeCfg)

	codexCfg := config.NewCodexConfig()
	codexCfg.Model = "codex-specific"
	app.SaveCodexConfig("isolation-test", codexCfg)

	geminiCfg := config.NewGeminiConfig()
	geminiCfg.Model = "gemini-specific"
	app.SaveGeminiConfig("isolation-test", geminiCfg)

	// Load and verify each maintains its own data
	loadedClaude, _ := app.LoadClaudeConfig("isolation-test")
	loadedCodex, _ := app.LoadCodexConfig("isolation-test")
	loadedGemini, _ := app.LoadGeminiConfig("isolation-test")

	if loadedClaude.Model != "claude-specific" {
		t.Error("Claude config has wrong model")
	}
	if loadedCodex.Model != "codex-specific" {
		t.Error("Codex config has wrong model")
	}
	if loadedGemini.Model != "gemini-specific" {
		t.Error("Gemini config has wrong model")
	}
}
