package appconfig

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultAppSettings(t *testing.T) {
	d := DefaultAppSettings()
	if d.Theme != "dark" {
		t.Errorf("Theme = %q, want dark", d.Theme)
	}
	if d.Language != "zh" {
		t.Errorf("Language = %q, want zh", d.Language)
	}
	if !d.AutoUpdate {
		t.Error("AutoUpdate should be true by default")
	}
	if d.EditorFontSize != 13 {
		t.Errorf("EditorFontSize = %d, want 13", d.EditorFontSize)
	}
	if d.StartupPage != "dashboard" {
		t.Errorf("StartupPage = %q, want dashboard", d.StartupPage)
	}
	if d.OnboardingCompleted {
		t.Error("OnboardingCompleted should be false by default")
	}
}

func TestLoadAppSettings_FileNotExist(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("APPDATA", tmp)
	t.Setenv("HOME", tmp)
	t.Setenv("XDG_CONFIG_HOME", tmp)

	s, err := LoadAppSettings()
	if err != nil {
		t.Fatalf("expected no error for missing file, got: %v", err)
	}
	if s.Theme != "dark" {
		t.Errorf("Theme = %q, want dark", s.Theme)
	}
	if s.EditorFontSize != 13 {
		t.Errorf("EditorFontSize = %d, want 13", s.EditorFontSize)
	}
}

func TestLoadAppSettings_ValidJSON(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("APPDATA", tmp)
	t.Setenv("HOME", tmp)
	t.Setenv("XDG_CONFIG_HOME", tmp)

	// Write a valid settings file
	dir := filepath.Join(tmp, "lurus-switch")
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	settings := AppSettings{
		Theme:               "light",
		Language:            "en",
		AutoUpdate:          false,
		EditorFontSize:      16,
		StartupPage:         "claude",
		OnboardingCompleted: true,
	}
	data, _ := json.Marshal(settings)
	os.WriteFile(filepath.Join(dir, "app-settings.json"), data, 0644)

	loaded, err := LoadAppSettings()
	if err != nil {
		t.Fatalf("LoadAppSettings error: %v", err)
	}
	if loaded.Theme != "light" {
		t.Errorf("Theme = %q, want light", loaded.Theme)
	}
	if loaded.Language != "en" {
		t.Errorf("Language = %q, want en", loaded.Language)
	}
	if loaded.AutoUpdate {
		t.Error("AutoUpdate should be false")
	}
	if loaded.EditorFontSize != 16 {
		t.Errorf("EditorFontSize = %d, want 16", loaded.EditorFontSize)
	}
	if loaded.StartupPage != "claude" {
		t.Errorf("StartupPage = %q, want claude", loaded.StartupPage)
	}
	if !loaded.OnboardingCompleted {
		t.Error("OnboardingCompleted should be true")
	}
}

func TestLoadAppSettings_CorruptJSON(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("APPDATA", tmp)
	t.Setenv("HOME", tmp)
	t.Setenv("XDG_CONFIG_HOME", tmp)

	dir := filepath.Join(tmp, "lurus-switch")
	os.MkdirAll(dir, 0755)
	os.WriteFile(filepath.Join(dir, "app-settings.json"), []byte("{corrupt json!!!"), 0644)

	// Should silently return defaults
	s, err := LoadAppSettings()
	if err != nil {
		t.Fatalf("expected no error for corrupt JSON, got: %v", err)
	}
	if s.Theme != "dark" {
		t.Errorf("corrupt file should return defaults, Theme = %q", s.Theme)
	}
}

func TestLoadAppSettings_FontSizeClampLow(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("APPDATA", tmp)
	t.Setenv("HOME", tmp)
	t.Setenv("XDG_CONFIG_HOME", tmp)

	dir := filepath.Join(tmp, "lurus-switch")
	os.MkdirAll(dir, 0755)
	settings := map[string]interface{}{
		"theme":          "dark",
		"language":       "zh",
		"autoUpdate":     true,
		"editorFontSize": 5, // below minimum
		"startupPage":    "dashboard",
	}
	data, _ := json.Marshal(settings)
	os.WriteFile(filepath.Join(dir, "app-settings.json"), data, 0644)

	s, err := LoadAppSettings()
	if err != nil {
		t.Fatalf("LoadAppSettings error: %v", err)
	}
	if s.EditorFontSize != 10 {
		t.Errorf("EditorFontSize should be clamped to 10, got %d", s.EditorFontSize)
	}
}

func TestLoadAppSettings_FontSizeClampHigh(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("APPDATA", tmp)
	t.Setenv("HOME", tmp)
	t.Setenv("XDG_CONFIG_HOME", tmp)

	dir := filepath.Join(tmp, "lurus-switch")
	os.MkdirAll(dir, 0755)
	settings := map[string]interface{}{
		"theme":          "dark",
		"language":       "zh",
		"autoUpdate":     true,
		"editorFontSize": 99, // above maximum
		"startupPage":    "dashboard",
	}
	data, _ := json.Marshal(settings)
	os.WriteFile(filepath.Join(dir, "app-settings.json"), data, 0644)

	s, err := LoadAppSettings()
	if err != nil {
		t.Fatalf("LoadAppSettings error: %v", err)
	}
	if s.EditorFontSize != 24 {
		t.Errorf("EditorFontSize should be clamped to 24, got %d", s.EditorFontSize)
	}
}

func TestLoadAppSettings_FontSizeAtBoundary(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("APPDATA", tmp)
	t.Setenv("HOME", tmp)
	t.Setenv("XDG_CONFIG_HOME", tmp)

	dir := filepath.Join(tmp, "lurus-switch")
	os.MkdirAll(dir, 0755)
	settings := map[string]interface{}{
		"theme":          "dark",
		"language":       "zh",
		"autoUpdate":     true,
		"editorFontSize": 10, // exactly at minimum — should not clamp
		"startupPage":    "dashboard",
	}
	data, _ := json.Marshal(settings)
	os.WriteFile(filepath.Join(dir, "app-settings.json"), data, 0644)

	s, err := LoadAppSettings()
	if err != nil {
		t.Fatalf("LoadAppSettings error: %v", err)
	}
	if s.EditorFontSize != 10 {
		t.Errorf("EditorFontSize should stay 10, got %d", s.EditorFontSize)
	}
}

func TestSaveAppSettings_Nil(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("APPDATA", tmp)
	t.Setenv("HOME", tmp)
	t.Setenv("XDG_CONFIG_HOME", tmp)

	err := SaveAppSettings(nil)
	if err == nil {
		t.Error("expected error for nil settings, got nil")
	}
}

func TestSaveAppSettings_CreatesDirectory(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("APPDATA", tmp)
	t.Setenv("HOME", tmp)
	t.Setenv("XDG_CONFIG_HOME", tmp)

	s := DefaultAppSettings()
	s.Theme = "light"

	if err := SaveAppSettings(s); err != nil {
		t.Fatalf("SaveAppSettings error: %v", err)
	}

	// Verify file was created
	p, _ := settingsPath()
	if _, err := os.Stat(p); os.IsNotExist(err) {
		t.Error("settings file should have been created")
	}
}

func TestSaveAndLoadRoundTrip(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("APPDATA", tmp)
	t.Setenv("HOME", tmp)
	t.Setenv("XDG_CONFIG_HOME", tmp)

	original := &AppSettings{
		Theme:               "light",
		Language:            "en",
		AutoUpdate:          false,
		EditorFontSize:      18,
		StartupPage:         "gemini",
		OnboardingCompleted: true,
	}

	if err := SaveAppSettings(original); err != nil {
		t.Fatalf("SaveAppSettings error: %v", err)
	}

	loaded, err := LoadAppSettings()
	if err != nil {
		t.Fatalf("LoadAppSettings error: %v", err)
	}

	if loaded.Theme != original.Theme {
		t.Errorf("Theme = %q, want %q", loaded.Theme, original.Theme)
	}
	if loaded.Language != original.Language {
		t.Errorf("Language = %q, want %q", loaded.Language, original.Language)
	}
	if loaded.AutoUpdate != original.AutoUpdate {
		t.Errorf("AutoUpdate = %v, want %v", loaded.AutoUpdate, original.AutoUpdate)
	}
	if loaded.EditorFontSize != original.EditorFontSize {
		t.Errorf("EditorFontSize = %d, want %d", loaded.EditorFontSize, original.EditorFontSize)
	}
	if loaded.StartupPage != original.StartupPage {
		t.Errorf("StartupPage = %q, want %q", loaded.StartupPage, original.StartupPage)
	}
	if loaded.OnboardingCompleted != original.OnboardingCompleted {
		t.Errorf("OnboardingCompleted = %v, want %v", loaded.OnboardingCompleted, original.OnboardingCompleted)
	}
}

func TestSaveAppSettings_ValidJSON(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("APPDATA", tmp)
	t.Setenv("HOME", tmp)
	t.Setenv("XDG_CONFIG_HOME", tmp)

	s := &AppSettings{
		Theme:    "dark",
		Language: "zh",
	}
	if err := SaveAppSettings(s); err != nil {
		t.Fatalf("SaveAppSettings error: %v", err)
	}

	p, _ := settingsPath()
	data, err := os.ReadFile(p)
	if err != nil {
		t.Fatalf("ReadFile error: %v", err)
	}

	var decoded AppSettings
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("saved file is not valid JSON: %v", err)
	}
	if decoded.Theme != "dark" {
		t.Errorf("Theme = %q, want dark", decoded.Theme)
	}
}
