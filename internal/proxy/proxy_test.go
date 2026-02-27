package proxy

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestNewProxyManager(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("APPDATA", tmpDir)
	t.Setenv("HOME", tmpDir)
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	pm, err := NewProxyManager()
	if err != nil {
		t.Fatalf("NewProxyManager error: %v", err)
	}
	if pm == nil {
		t.Fatal("NewProxyManager should return non-nil")
	}
}

func TestProxyManager_GetSettings_Default(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("APPDATA", tmpDir)
	t.Setenv("HOME", tmpDir)
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	pm, err := NewProxyManager()
	if err != nil {
		t.Fatalf("NewProxyManager error: %v", err)
	}

	settings := pm.GetSettings()
	if settings == nil {
		t.Fatal("GetSettings should return non-nil")
	}
	if settings.APIEndpoint != "" {
		t.Errorf("default APIEndpoint should be empty, got %q", settings.APIEndpoint)
	}
	if settings.APIKey != "" {
		t.Errorf("default APIKey should be empty, got %q", settings.APIKey)
	}
}

func TestProxyManager_SaveAndLoad(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("APPDATA", tmpDir)
	t.Setenv("HOME", tmpDir)
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	pm, err := NewProxyManager()
	if err != nil {
		t.Fatalf("NewProxyManager error: %v", err)
	}

	// Save
	settings := &ProxySettings{
		APIEndpoint:     "https://api.example.com/v1",
		APIKey:          "sk-test-123",
		RegistrationURL: "https://register.example.com",
	}

	if err := pm.SaveSettings(settings); err != nil {
		t.Fatalf("SaveSettings error: %v", err)
	}

	// Load with a new manager to verify persistence
	pm2, err := NewProxyManager()
	if err != nil {
		t.Fatalf("NewProxyManager (reload) error: %v", err)
	}

	loaded := pm2.GetSettings()
	if loaded.APIEndpoint != settings.APIEndpoint {
		t.Errorf("APIEndpoint = %q, want %q", loaded.APIEndpoint, settings.APIEndpoint)
	}
	if loaded.APIKey != settings.APIKey {
		t.Errorf("APIKey = %q, want %q", loaded.APIKey, settings.APIKey)
	}
	if loaded.RegistrationURL != settings.RegistrationURL {
		t.Errorf("RegistrationURL = %q, want %q", loaded.RegistrationURL, settings.RegistrationURL)
	}
}

func TestProxyManager_SaveSettings_CreatesDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("APPDATA", tmpDir)
	t.Setenv("HOME", tmpDir)
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	pm, err := NewProxyManager()
	if err != nil {
		t.Fatalf("NewProxyManager error: %v", err)
	}

	settings := &ProxySettings{
		APIEndpoint: "https://api.test.com",
		APIKey:      "test-key",
	}

	if err := pm.SaveSettings(settings); err != nil {
		t.Fatalf("SaveSettings error: %v", err)
	}

	// Verify the file exists and is valid JSON
	data, err := os.ReadFile(pm.configPath)
	if err != nil {
		t.Fatalf("Failed to read saved config: %v", err)
	}

	var loaded ProxySettings
	if err := json.Unmarshal(data, &loaded); err != nil {
		t.Fatalf("Saved file is not valid JSON: %v", err)
	}

	if loaded.APIEndpoint != "https://api.test.com" {
		t.Errorf("loaded APIEndpoint = %q, want %q", loaded.APIEndpoint, "https://api.test.com")
	}
}

func TestProxyManager_LoadsExistingConfig(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("APPDATA", tmpDir)
	t.Setenv("HOME", tmpDir)
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	// Pre-create the config file
	configDir := filepath.Join(tmpDir, "lurus-switch", "configs")
	os.MkdirAll(configDir, 0755)

	preExisting := ProxySettings{
		APIEndpoint: "https://pre-existing.com",
		APIKey:      "pre-key",
	}
	data, _ := json.Marshal(preExisting)
	os.WriteFile(filepath.Join(configDir, "proxy.json"), data, 0644)

	pm, err := NewProxyManager()
	if err != nil {
		t.Fatalf("NewProxyManager error: %v", err)
	}

	settings := pm.GetSettings()
	if settings.APIEndpoint != "https://pre-existing.com" {
		t.Errorf("should load pre-existing APIEndpoint, got %q", settings.APIEndpoint)
	}
	if settings.APIKey != "pre-key" {
		t.Errorf("should load pre-existing APIKey, got %q", settings.APIKey)
	}
}

func TestProxySettings_Fields(t *testing.T) {
	s := ProxySettings{
		APIEndpoint:     "https://api.test.com",
		APIKey:          "key123",
		RegistrationURL: "https://reg.test.com",
	}

	if s.APIEndpoint != "https://api.test.com" {
		t.Errorf("APIEndpoint = %q", s.APIEndpoint)
	}
	if s.APIKey != "key123" {
		t.Errorf("APIKey = %q", s.APIKey)
	}
	if s.RegistrationURL != "https://reg.test.com" {
		t.Errorf("RegistrationURL = %q", s.RegistrationURL)
	}
}

func TestProxyManager_SaveAndLoad_WithNewFields(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("APPDATA", tmpDir)
	t.Setenv("HOME", tmpDir)
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	pm, err := NewProxyManager()
	if err != nil {
		t.Fatalf("NewProxyManager error: %v", err)
	}

	settings := &ProxySettings{
		APIEndpoint: "https://api.example.com/v1",
		APIKey:      "sk-test",
		TenantSlug:  "my-tenant",
		UserToken:   "jwt-token-abc123",
	}
	if err := pm.SaveSettings(settings); err != nil {
		t.Fatalf("SaveSettings error: %v", err)
	}

	pm2, err := NewProxyManager()
	if err != nil {
		t.Fatalf("reload error: %v", err)
	}
	loaded := pm2.GetSettings()
	if loaded.TenantSlug != "my-tenant" {
		t.Errorf("TenantSlug = %q, want my-tenant", loaded.TenantSlug)
	}
	if loaded.UserToken != "jwt-token-abc123" {
		t.Errorf("UserToken = %q", loaded.UserToken)
	}
}

func TestProxyManager_ConcurrentAccess(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("APPDATA", tmpDir)
	t.Setenv("HOME", tmpDir)
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	pm, err := NewProxyManager()
	if err != nil {
		t.Fatalf("NewProxyManager error: %v", err)
	}

	done := make(chan bool, 20)

	// 10 concurrent writers
	for i := 0; i < 10; i++ {
		go func(i int) {
			s := &ProxySettings{
				APIEndpoint: fmt.Sprintf("https://api%d.example.com", i),
				APIKey:      fmt.Sprintf("key-%d", i),
			}
			pm.SaveSettings(s)
			done <- true
		}(i)
	}

	// 10 concurrent readers
	for i := 0; i < 10; i++ {
		go func() {
			s := pm.GetSettings()
			_ = s.APIEndpoint // should not panic
			done <- true
		}()
	}

	for i := 0; i < 20; i++ {
		<-done
	}
}

func TestProxyManager_GetSettingsReturnsCopy(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("APPDATA", tmpDir)
	t.Setenv("HOME", tmpDir)
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	pm, err := NewProxyManager()
	if err != nil {
		t.Fatalf("NewProxyManager error: %v", err)
	}

	pm.SaveSettings(&ProxySettings{APIEndpoint: "original"})

	// Get a copy and modify it
	copy := pm.GetSettings()
	copy.APIEndpoint = "modified"

	// Original should be unchanged
	original := pm.GetSettings()
	if original.APIEndpoint != "original" {
		t.Errorf("modifying copy affected original: got %q", original.APIEndpoint)
	}
}

func TestProxyManager_CorruptJSON(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("APPDATA", tmpDir)
	t.Setenv("HOME", tmpDir)
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	// Create corrupt config file
	configDir := filepath.Join(tmpDir, "lurus-switch", "configs")
	os.MkdirAll(configDir, 0755)
	os.WriteFile(filepath.Join(configDir, "proxy.json"), []byte("{invalid json}}}"), 0644)

	pm, err := NewProxyManager()
	if err != nil {
		t.Fatalf("should not error on corrupt file: %v", err)
	}

	// Should fall back to empty defaults
	settings := pm.GetSettings()
	if settings.APIEndpoint != "" {
		t.Errorf("corrupt file should reset to empty, got %q", settings.APIEndpoint)
	}
}

func TestProxySettings_JSONRoundTrip(t *testing.T) {
	original := &ProxySettings{
		APIEndpoint:     "https://api.example.com/v1",
		APIKey:          "sk-test-roundtrip",
		RegistrationURL: "https://register.example.com",
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("JSON marshal error: %v", err)
	}

	var decoded ProxySettings
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("JSON unmarshal error: %v", err)
	}

	if decoded.APIEndpoint != original.APIEndpoint {
		t.Errorf("APIEndpoint = %q, want %q", decoded.APIEndpoint, original.APIEndpoint)
	}
	if decoded.APIKey != original.APIKey {
		t.Errorf("APIKey = %q, want %q", decoded.APIKey, original.APIKey)
	}
	if decoded.RegistrationURL != original.RegistrationURL {
		t.Errorf("RegistrationURL = %q, want %q", decoded.RegistrationURL, original.RegistrationURL)
	}
}
