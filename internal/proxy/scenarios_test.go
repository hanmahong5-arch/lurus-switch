package proxy

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

// setupProxyEnv redirects platform config path to a temp dir.
func setupProxyEnv(t *testing.T) string {
	t.Helper()
	tmp := t.TempDir()
	t.Setenv("APPDATA", tmp)
	t.Setenv("HOME", tmp)
	t.Setenv("XDG_CONFIG_HOME", tmp)
	return tmp
}

// newManager creates a ProxyManager using the current env (must call setupProxyEnv first).
func newManager(t *testing.T) *ProxyManager {
	t.Helper()
	pm, err := NewProxyManager()
	if err != nil {
		t.Fatalf("NewProxyManager: %v", err)
	}
	return pm
}

// ============================================================
// Scenario: User configures proxy for the first time
// ============================================================

// TestScenario_UserConfiguresProxy_AppRestart_SettingsPersist simulates the
// common onboarding flow: configure proxy endpoint + key → close app → reopen →
// settings still present.
func TestScenario_UserConfiguresProxy_AppRestart_SettingsPersist(t *testing.T) {
	setupProxyEnv(t)

	// Session 1: user enters proxy settings
	pm1 := newManager(t)
	if err := pm1.SaveSettings(&ProxySettings{
		APIEndpoint: "https://api.lurus.cn/v1",
		APIKey:      "sk-abcdef123456",
		TenantSlug:  "my-org",
	}); err != nil {
		t.Fatalf("SaveSettings: %v", err)
	}

	// Session 2: app restarts (new manager instance reads from disk)
	pm2 := newManager(t)
	loaded := pm2.GetSettings()

	if loaded.APIEndpoint != "https://api.lurus.cn/v1" {
		t.Errorf("APIEndpoint = %q after restart", loaded.APIEndpoint)
	}
	if loaded.APIKey != "sk-abcdef123456" {
		t.Errorf("APIKey = %q after restart", loaded.APIKey)
	}
	if loaded.TenantSlug != "my-org" {
		t.Errorf("TenantSlug = %q after restart", loaded.TenantSlug)
	}
}

// ============================================================
// Scenario: User rotates API key
// ============================================================

// TestScenario_UserRotatesApiKey_OldKeyOverwritten verifies that saving a new
// API key completely replaces the old one — no residual data from previous key.
func TestScenario_UserRotatesApiKey_OldKeyOverwritten(t *testing.T) {
	setupProxyEnv(t)

	pm := newManager(t)

	// First key
	pm.SaveSettings(&ProxySettings{APIEndpoint: "https://api.example.com", APIKey: "old-key-111"})
	// Key rotation
	pm.SaveSettings(&ProxySettings{APIEndpoint: "https://api.example.com", APIKey: "new-key-999"})

	current := pm.GetSettings()
	if current.APIKey != "new-key-999" {
		t.Errorf("APIKey after rotation = %q, want new-key-999", current.APIKey)
	}
	if current.APIKey == "old-key-111" {
		t.Error("old API key was not overwritten")
	}
}

// ============================================================
// Scenario: User clears proxy settings
// ============================================================

// TestScenario_UserClearsProxy_AllFieldsEmpty verifies that saving an empty
// ProxySettings struct wipes all previously stored values — supporting the use
// case where a user disconnects their account.
func TestScenario_UserClearsProxy_AllFieldsEmpty(t *testing.T) {
	setupProxyEnv(t)

	pm := newManager(t)

	// Configure first
	pm.SaveSettings(&ProxySettings{
		APIEndpoint: "https://api.example.com",
		APIKey:      "sk-test",
		UserToken:   "token-xyz",
	})

	// User clicks "Disconnect Account" → saves empty settings
	pm.SaveSettings(&ProxySettings{})

	cleared := pm.GetSettings()
	if cleared.APIEndpoint != "" {
		t.Errorf("APIEndpoint should be empty after clear, got %q", cleared.APIEndpoint)
	}
	if cleared.APIKey != "" {
		t.Errorf("APIKey should be empty after clear, got %q", cleared.APIKey)
	}
	if cleared.UserToken != "" {
		t.Errorf("UserToken should be empty after clear, got %q", cleared.UserToken)
	}

	// Cleared state persists across restart
	pm2 := newManager(t)
	reloaded := pm2.GetSettings()
	if reloaded.APIEndpoint != "" {
		t.Errorf("cleared APIEndpoint should persist as empty, got %q", reloaded.APIEndpoint)
	}
}

// ============================================================
// Scenario: User switches tenants
// ============================================================

// TestScenario_UserSwitchesTenants_SlugAndTokenUpdated simulates a user who
// works with multiple Lurus tenants and switches between them.
func TestScenario_UserSwitchesTenants_SlugAndTokenUpdated(t *testing.T) {
	setupProxyEnv(t)

	pm := newManager(t)

	tenants := []struct {
		slug  string
		token string
	}{
		{"corp-alpha", "token-alpha-001"},
		{"corp-beta", "token-beta-002"},
		{"corp-alpha", "token-alpha-refreshed"},
	}

	for _, tenant := range tenants {
		pm.SaveSettings(&ProxySettings{
			APIEndpoint: "https://api.lurus.cn/v1",
			TenantSlug:  tenant.slug,
			UserToken:   tenant.token,
		})
		s := pm.GetSettings()
		if s.TenantSlug != tenant.slug {
			t.Errorf("TenantSlug = %q, want %q", s.TenantSlug, tenant.slug)
		}
		if s.UserToken != tenant.token {
			t.Errorf("UserToken = %q, want %q", s.UserToken, tenant.token)
		}
	}
}

// ============================================================
// Scenario: Corrupt proxy.json on disk
// ============================================================

// TestScenario_CorruptProxyFile_AppStartsWithEmptySettings verifies that if
// the proxy config file on disk is corrupt (e.g., interrupted write), the app
// starts cleanly with empty settings rather than crashing.
func TestScenario_CorruptProxyFile_AppStartsWithEmptySettings(t *testing.T) {
	tmp := setupProxyEnv(t)

	// Pre-create a corrupt config file
	configDir := filepath.Join(tmp, "lurus-switch", "configs")
	os.MkdirAll(configDir, 0755)
	os.WriteFile(filepath.Join(configDir, "proxy.json"), []byte("{corrupt!!!}}}"), 0644)

	pm, err := NewProxyManager()
	if err != nil {
		t.Fatalf("NewProxyManager should not error on corrupt file: %v", err)
	}

	s := pm.GetSettings()
	if s.APIEndpoint != "" || s.APIKey != "" {
		t.Errorf("corrupt file should result in empty settings, got endpoint=%q key=%q",
			s.APIEndpoint, s.APIKey)
	}
}

// TestScenario_EmptyProxyFile_AppStartsWithEmptySettings handles the edge case
// of an empty proxy.json (zero-byte file from a failed write).
func TestScenario_EmptyProxyFile_AppStartsWithEmptySettings(t *testing.T) {
	tmp := setupProxyEnv(t)

	configDir := filepath.Join(tmp, "lurus-switch", "configs")
	os.MkdirAll(configDir, 0755)
	os.WriteFile(filepath.Join(configDir, "proxy.json"), []byte{}, 0644)

	pm, err := NewProxyManager()
	if err != nil {
		t.Fatalf("NewProxyManager should not error on empty file: %v", err)
	}

	s := pm.GetSettings()
	if s == nil {
		t.Fatal("GetSettings should not return nil")
	}
}

// ============================================================
// Scenario: GetSettings returns a defensive copy
// ============================================================

// TestScenario_MutatingReturnedSettings_DoesNotAffectStored verifies that
// modifying the struct returned by GetSettings does not change the stored state.
// This guards against accidental mutation bugs in the dashboard code.
func TestScenario_MutatingReturnedSettings_DoesNotAffectStored(t *testing.T) {
	setupProxyEnv(t)

	pm := newManager(t)
	pm.SaveSettings(&ProxySettings{APIEndpoint: "https://original.com", APIKey: "orig-key"})

	// Dashboard code gets settings and "accidentally" mutates it
	copy1 := pm.GetSettings()
	copy1.APIEndpoint = "https://mutated.com"
	copy1.APIKey = "mutated-key"

	// The stored value must remain unchanged
	copy2 := pm.GetSettings()
	if copy2.APIEndpoint != "https://original.com" {
		t.Errorf("stored APIEndpoint mutated to %q", copy2.APIEndpoint)
	}
	if copy2.APIKey != "orig-key" {
		t.Errorf("stored APIKey mutated to %q", copy2.APIKey)
	}
}

// ============================================================
// Scenario: Concurrent readers and writers — no panic or race
// ============================================================

// TestScenario_ConcurrentReadersWhileWriting_NoPanic simulates the dashboard
// and settings page both accessing the ProxyManager simultaneously — a common
// pattern in the Wails app where multiple goroutines serve frontend calls.
func TestScenario_ConcurrentReadersWhileWriting_NoPanic(t *testing.T) {
	setupProxyEnv(t)

	pm := newManager(t)

	var wg sync.WaitGroup
	const readers = 8
	const writers = 4

	// Concurrent writers
	for i := 0; i < writers; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			_ = pm.SaveSettings(&ProxySettings{
				APIEndpoint: "https://api.example.com",
				APIKey:      "key",
			})
		}(i)
	}

	// Concurrent readers
	for i := 0; i < readers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			s := pm.GetSettings()
			_ = s.APIEndpoint // must not panic
		}()
	}

	wg.Wait()
}

// ============================================================
// Scenario: User's proxy.json does not yet exist (no config dir)
// ============================================================

// TestScenario_NoConfigDir_FirstSave_CreatesDirectoryAndFile verifies that
// SaveSettings automatically creates the config directory hierarchy when it
// doesn't yet exist — a fresh install scenario.
func TestScenario_NoConfigDir_FirstSave_CreatesDirectoryAndFile(t *testing.T) {
	tmp := setupProxyEnv(t)

	// Confirm config dir doesn't exist yet
	configDir := filepath.Join(tmp, "lurus-switch", "configs")
	if _, err := os.Stat(configDir); !os.IsNotExist(err) {
		t.Skip("config dir already exists, skipping creation test")
	}

	pm := newManager(t)
	if err := pm.SaveSettings(&ProxySettings{APIEndpoint: "https://new.com"}); err != nil {
		t.Fatalf("SaveSettings should create dir: %v", err)
	}

	// Verify directory and file were created
	if _, err := os.Stat(configDir); os.IsNotExist(err) {
		t.Error("config directory should have been created")
	}

	// File must contain valid JSON
	data, err := os.ReadFile(pm.configPath)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	var parsed ProxySettings
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Errorf("saved file is not valid JSON: %v", err)
	}
}

// ============================================================
// Scenario: User enters only partial settings (missing optional fields)
// ============================================================

// TestScenario_UserSavesPartialSettings_OptionalFieldsEmpty verifies that
// optional fields (RegistrationURL, TenantSlug, UserToken) can be omitted
// without corrupting required fields.
func TestScenario_UserSavesPartialSettings_OptionalFieldsEmpty(t *testing.T) {
	setupProxyEnv(t)

	pm := newManager(t)

	// User only fills required fields
	pm.SaveSettings(&ProxySettings{
		APIEndpoint: "https://api.lurus.cn/v1",
		APIKey:      "sk-minimal",
	})

	loaded := pm.GetSettings()
	if loaded.APIEndpoint != "https://api.lurus.cn/v1" {
		t.Errorf("APIEndpoint = %q", loaded.APIEndpoint)
	}
	if loaded.RegistrationURL != "" {
		t.Errorf("RegistrationURL should be empty, got %q", loaded.RegistrationURL)
	}
	if loaded.TenantSlug != "" {
		t.Errorf("TenantSlug should be empty, got %q", loaded.TenantSlug)
	}
	if loaded.UserToken != "" {
		t.Errorf("UserToken should be empty, got %q", loaded.UserToken)
	}
}
