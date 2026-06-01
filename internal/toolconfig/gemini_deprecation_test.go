package toolconfig

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// === GeminiDeprecation metadata Tests ===

func TestGeminiDeprecation_IsDeprecated(t *testing.T) {
	dep := GeminiDeprecation{}
	if !dep.IsDeprecated() {
		t.Error("IsDeprecated() should return true")
	}
}

func TestGeminiDeprecation_DeprecatedAfter_CorrectDate(t *testing.T) {
	dep := GeminiDeprecation{}
	eol := dep.DeprecatedAfter()
	want := time.Date(2026, 6, 18, 0, 0, 0, 0, time.UTC)
	if !eol.Equal(want) {
		t.Errorf("DeprecatedAfter() = %v, want %v", eol, want)
	}
}

func TestGeminiDeprecation_DeprecatedAfter_IsFuture(t *testing.T) {
	dep := GeminiDeprecation{}
	eol := dep.DeprecatedAfter()
	// At time of implementation (2026-05-27), EOL date is still in the future.
	// This test documents the invariant without hard-coding today's date.
	now := time.Date(2026, 5, 27, 0, 0, 0, 0, time.UTC)
	if !eol.After(now) {
		t.Errorf("DeprecatedAfter() = %v should be after implementation date %v", eol, now)
	}
}

func TestGeminiDeprecation_MigrateTo_IsAntigravity(t *testing.T) {
	dep := GeminiDeprecation{}
	if dep.MigrateTo() != ToolAntigravity {
		t.Errorf("MigrateTo() = %q, want %q", dep.MigrateTo(), ToolAntigravity)
	}
}

func TestDefaultGeminiDeprecation_IsDeprecated(t *testing.T) {
	if !DefaultGeminiDeprecation.IsDeprecated() {
		t.Error("DefaultGeminiDeprecation.IsDeprecated() should return true")
	}
}

// === BuildMigrationPlan Tests ===

// geminiTestConfig is a helper that writes a Gemini settings.json under tmp
// and sets HOME / LOCALAPPDATA so antigravityConfigDir() and geminiDir() use tmp.
func geminiTestConfig(t *testing.T, content string) string {
	t.Helper()
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	t.Setenv("USERPROFILE", tmp)
	t.Setenv("LOCALAPPDATA", tmp)
	t.Setenv("XDG_CONFIG_HOME", tmp)

	gemDir := filepath.Join(tmp, ".gemini")
	if err := os.MkdirAll(gemDir, 0755); err != nil {
		t.Fatalf("MkdirAll gemini dir: %v", err)
	}
	settingsPath := filepath.Join(gemDir, "settings.json")
	if err := os.WriteFile(settingsPath, []byte(content), 0600); err != nil {
		t.Fatalf("WriteFile gemini settings: %v", err)
	}
	return tmp
}

func TestBuildMigrationPlan_NoGeminiConfig_ReturnsEmpty(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	t.Setenv("USERPROFILE", tmp)
	t.Setenv("LOCALAPPDATA", tmp)
	t.Setenv("XDG_CONFIG_HOME", tmp)

	plan, err := BuildMigrationPlan(context.Background())
	if err != nil {
		t.Fatalf("BuildMigrationPlan error: %v", err)
	}
	if plan == nil {
		t.Fatal("expected non-nil plan")
	}
	if len(plan.Fields) != 0 {
		t.Errorf("expected 0 fields for missing config, got %d", len(plan.Fields))
	}
	if len(plan.Warnings) == 0 {
		t.Error("expected at least one warning for missing Gemini config")
	}
	if plan.Proposed == nil {
		t.Error("Proposed should not be nil")
	}
}

func TestBuildMigrationPlan_APIKey_Mapped(t *testing.T) {
	geminiTestConfig(t, `{"apiKey": "my-api-key-xyz"}`)

	plan, err := BuildMigrationPlan(context.Background())
	if err != nil {
		t.Fatalf("BuildMigrationPlan error: %v", err)
	}

	if plan.Proposed.APIKey != "my-api-key-xyz" {
		t.Errorf("Proposed.APIKey = %q, want %q", plan.Proposed.APIKey, "my-api-key-xyz")
	}

	found := false
	for _, f := range plan.Fields {
		if f.GeminiField == "apiKey" {
			found = true
			if f.AntigravityField != "apiKey" {
				t.Errorf("apiKey AntigravityField = %q, want apiKey", f.AntigravityField)
			}
			if f.Value != "my-api-key-xyz" {
				t.Errorf("apiKey Value = %q", f.Value)
			}
		}
	}
	if !found {
		t.Error("apiKey field missing from migration plan")
	}
}

func TestBuildMigrationPlan_ModelName_Mapped(t *testing.T) {
	geminiTestConfig(t, `{"model": {"name": "gemini-2.5-pro"}}`)

	plan, err := BuildMigrationPlan(context.Background())
	if err != nil {
		t.Fatalf("BuildMigrationPlan error: %v", err)
	}

	if plan.Proposed.Model.Name != "gemini-2.5-pro" {
		t.Errorf("Proposed.Model.Name = %q, want gemini-2.5-pro", plan.Proposed.Model.Name)
	}

	found := false
	for _, f := range plan.Fields {
		if f.GeminiField == "model.name" {
			found = true
			if f.AntigravityField != "model.name" {
				t.Errorf("model.name AntigravityField = %q", f.AntigravityField)
			}
		}
	}
	if !found {
		t.Error("model.name field missing from migration plan")
	}
}

func TestBuildMigrationPlan_AllBasicFields_Mapped(t *testing.T) {
	raw := map[string]interface{}{
		"apiKey":      "key-abc",
		"apiEndpoint": "https://myproxy.example.com/gemini",
		"proxy":       "http://localhost:3128",
		"model":       map[string]interface{}{"name": "gemini-2.5-flash"},
		"general":     map[string]interface{}{"defaultApprovalMode": "auto"},
	}
	data, _ := json.Marshal(raw)
	geminiTestConfig(t, string(data))

	plan, err := BuildMigrationPlan(context.Background())
	if err != nil {
		t.Fatalf("BuildMigrationPlan error: %v", err)
	}

	// All 5 basic fields should be present
	if len(plan.Fields) < 5 {
		t.Errorf("expected at least 5 field migrations, got %d", len(plan.Fields))
	}

	if plan.Proposed.APIKey != "key-abc" {
		t.Errorf("Proposed.APIKey = %q", plan.Proposed.APIKey)
	}
	if plan.Proposed.APIEndpoint != "https://myproxy.example.com/gemini" {
		t.Errorf("Proposed.APIEndpoint = %q", plan.Proposed.APIEndpoint)
	}
	if plan.Proposed.Proxy != "http://localhost:3128" {
		t.Errorf("Proposed.Proxy = %q", plan.Proposed.Proxy)
	}
	if plan.Proposed.Model.Name != "gemini-2.5-flash" {
		t.Errorf("Proposed.Model.Name = %q", plan.Proposed.Model.Name)
	}
	if plan.Proposed.General.DefaultApprovalMode != "auto" {
		t.Errorf("Proposed.General.DefaultApprovalMode = %q", plan.Proposed.General.DefaultApprovalMode)
	}
}

func TestBuildMigrationPlan_EmptyGeminiConfig_NoFields(t *testing.T) {
	geminiTestConfig(t, `{}`)

	plan, err := BuildMigrationPlan(context.Background())
	if err != nil {
		t.Fatalf("BuildMigrationPlan error: %v", err)
	}
	if len(plan.Fields) != 0 {
		t.Errorf("expected 0 fields for empty config, got %d: %+v", len(plan.Fields), plan.Fields)
	}
}

func TestBuildMigrationPlan_InvalidJSON_ReturnsError(t *testing.T) {
	geminiTestConfig(t, `{not valid json`)

	_, err := BuildMigrationPlan(context.Background())
	if err == nil {
		t.Fatal("expected error for invalid JSON Gemini config")
	}
}

func TestBuildMigrationPlan_SourceTargetPaths_Set(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	t.Setenv("USERPROFILE", tmp)
	t.Setenv("LOCALAPPDATA", tmp)
	t.Setenv("XDG_CONFIG_HOME", tmp)

	plan, err := BuildMigrationPlan(context.Background())
	if err != nil {
		t.Fatalf("BuildMigrationPlan error: %v", err)
	}

	if plan.SourcePath == "" {
		t.Error("SourcePath should not be empty")
	}
	if plan.TargetPath == "" {
		t.Error("TargetPath should not be empty")
	}
	// Source must reference gemini, target must reference antigravity
	if !strings.Contains(strings.ToLower(plan.SourcePath), ".gemini") &&
		!strings.Contains(strings.ToLower(plan.SourcePath), "gemini") {
		t.Errorf("SourcePath %q should reference gemini directory", plan.SourcePath)
	}
	if !strings.Contains(strings.ToLower(plan.TargetPath), "antigravity") {
		t.Errorf("TargetPath %q should reference antigravity directory", plan.TargetPath)
	}
}

func TestBuildMigrationPlan_LongProxy_MarkedForReview(t *testing.T) {
	longProxy := "http://" + strings.Repeat("x", 300) + ".example.com"
	raw := map[string]interface{}{"proxy": longProxy}
	data, _ := json.Marshal(raw)
	geminiTestConfig(t, string(data))

	plan, err := BuildMigrationPlan(context.Background())
	if err != nil {
		t.Fatalf("BuildMigrationPlan error: %v", err)
	}

	for _, f := range plan.Fields {
		if f.GeminiField == "proxy" {
			if !f.NeedsManualReview {
				t.Error("long proxy value should be marked NeedsManualReview")
			}
			return
		}
	}
	t.Error("proxy field not found in migration plan")
}

// === AntigravityConfig Extra round-trip Tests (R9) ===

// TestAntigravityConfig_UnknownKeysSurviveRoundTrip proves that a pre-existing
// unknown key in the on-disk antigravity config is preserved after a migration
// write (option b: Extra map mirrors OpenCodeConfig pattern).
func TestAntigravityConfig_UnknownKeysSurviveRoundTrip(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("LOCALAPPDATA", tmp)
	t.Setenv("HOME", tmp)
	t.Setenv("XDG_CONFIG_HOME", tmp)

	// Write a config that contains an unknown field "theme" that AntigravityConfig does not model.
	existing := `{"apiKey":"old-key","theme":"dark","debugMode":true}`
	configDir := antigravityConfigDir()
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	configPath := filepath.Join(configDir, AntigravityConfigFilename)
	if err := os.WriteFile(configPath, []byte(existing), 0600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	// Read, mutate a known field, write back.
	cfg, err := ReadAntigravityConfig()
	if err != nil {
		t.Fatalf("ReadAntigravityConfig: %v", err)
	}
	cfg.APIKey = "new-key"
	if err := WriteAntigravityConfig(cfg); err != nil {
		t.Fatalf("WriteAntigravityConfig: %v", err)
	}

	// Verify: unknown keys must survive.
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("Unmarshal written file: %v", err)
	}
	if _, ok := raw["theme"]; !ok {
		t.Error("unknown key 'theme' was silently dropped after WriteAntigravityConfig (Extra round-trip broken)")
	}
	if _, ok := raw["debugMode"]; !ok {
		t.Error("unknown key 'debugMode' was silently dropped after WriteAntigravityConfig (Extra round-trip broken)")
	}
	// Known field must be updated.
	cfg2, err := ReadAntigravityConfig()
	if err != nil {
		t.Fatalf("ReadAntigravityConfig after write: %v", err)
	}
	if cfg2.APIKey != "new-key" {
		t.Errorf("APIKey = %q after write, want 'new-key'", cfg2.APIKey)
	}
}

// TestApplyMigration_PreservesExtraKeys proves that BuildMigrationPlan produces
// a Proposed config whose Extra contains unknown keys from the existing antigravity
// config when MergeAntigravityExtra is used before writing.
func TestMergeAntigravityExtra_PreservesUnknownKeys(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("LOCALAPPDATA", tmp)
	t.Setenv("HOME", tmp)
	t.Setenv("XDG_CONFIG_HOME", tmp)

	// Write an existing antigravity config with unknown keys.
	configDir := antigravityConfigDir()
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	existing := `{"apiKey":"old-key","customPlugin":"my-plugin","legacyFlag":42}`
	configPath := filepath.Join(configDir, AntigravityConfigFilename)
	if err := os.WriteFile(configPath, []byte(existing), 0600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	// Simulate what ApplyGeminiMigration does: build plan, merge extra, write.
	proposed := &AntigravityConfig{APIKey: "migrated-key"}
	if err := MergeAntigravityExtra(proposed); err != nil {
		t.Fatalf("MergeAntigravityExtra: %v", err)
	}
	if err := WriteAntigravityConfig(proposed); err != nil {
		t.Fatalf("WriteAntigravityConfig: %v", err)
	}

	// Verify the written file still contains the unknown keys.
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if _, ok := raw["customPlugin"]; !ok {
		t.Error("unknown key 'customPlugin' was lost after MergeAntigravityExtra + WriteAntigravityConfig")
	}
	if _, ok := raw["legacyFlag"]; !ok {
		t.Error("unknown key 'legacyFlag' was lost after MergeAntigravityExtra + WriteAntigravityConfig")
	}
	// Known field from migration must win.
	if _, ok := raw["apiKey"]; !ok {
		t.Error("known key 'apiKey' missing after write")
	}
	cfg, err := ReadAntigravityConfig()
	if err != nil {
		t.Fatalf("ReadAntigravityConfig: %v", err)
	}
	if cfg.APIKey != "migrated-key" {
		t.Errorf("APIKey = %q, want 'migrated-key'", cfg.APIKey)
	}
}
