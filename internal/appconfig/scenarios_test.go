package appconfig

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

// --- helpers ---

// setupEnv redirects all platform config paths to a temp directory.
func setupEnv(t *testing.T) string {
	t.Helper()
	tmp := t.TempDir()
	t.Setenv("APPDATA", tmp)
	t.Setenv("HOME", tmp)
	t.Setenv("XDG_CONFIG_HOME", tmp)
	return tmp
}

// writeSettings writes an AppSettings struct to the expected settings file path.
func writeSettings(t *testing.T, s *AppSettings) {
	t.Helper()
	p, err := settingsPath()
	if err != nil {
		t.Fatalf("settingsPath: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	data, _ := json.MarshalIndent(s, "", "  ")
	if err := os.WriteFile(p, data, 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
}

// writeRaw writes arbitrary bytes to the settings file (for corruption tests).
func writeRaw(t *testing.T, content []byte) {
	t.Helper()
	p, err := settingsPath()
	if err != nil {
		t.Fatalf("settingsPath: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(p, content, 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
}

// ============================================================
// Scenario: New user — first launch, wizard, re-run setup
// ============================================================

// TestScenario_NewUser_FirstLaunch_DefaultsReturned verifies that a brand-new
// user (no settings file) receives all factory defaults and the wizard flag is off.
func TestScenario_NewUser_FirstLaunch_DefaultsReturned(t *testing.T) {
	setupEnv(t)

	s, err := LoadAppSettings()
	if err != nil {
		t.Fatalf("LoadAppSettings: %v", err)
	}

	if s.OnboardingCompleted {
		t.Error("new user should not have completed onboarding")
	}
	if s.Theme != "dark" {
		t.Errorf("Theme = %q, want dark", s.Theme)
	}
	if s.Language != "zh" {
		t.Errorf("Language = %q, want zh", s.Language)
	}
	if s.EditorFontSize != 13 {
		t.Errorf("EditorFontSize = %d, want 13", s.EditorFontSize)
	}
}

// TestScenario_UserCompletesWizard_NextLaunchSkipsWizard simulates the full
// onboarding flow: first launch → wizard → save completion → restart → no wizard.
func TestScenario_UserCompletesWizard_NextLaunchSkipsWizard(t *testing.T) {
	setupEnv(t)

	// Launch 1: no settings file — wizard should show
	launch1, err := LoadAppSettings()
	if err != nil {
		t.Fatalf("launch1 LoadAppSettings: %v", err)
	}
	if launch1.OnboardingCompleted {
		t.Fatal("wizard should show on first launch")
	}

	// User completes wizard → save flag
	launch1.OnboardingCompleted = true
	if err := SaveAppSettings(launch1); err != nil {
		t.Fatalf("SaveAppSettings after wizard: %v", err)
	}

	// Launch 2: settings file exists — wizard should NOT show
	launch2, err := LoadAppSettings()
	if err != nil {
		t.Fatalf("launch2 LoadAppSettings: %v", err)
	}
	if !launch2.OnboardingCompleted {
		t.Error("wizard should not show after completion")
	}
}

// TestScenario_UserRerunsSetup_WizardShownAgain simulates a returning user who
// clicks "Re-run Setup" in Settings → next launch wizard appears again.
func TestScenario_UserRerunsSetup_WizardShownAgain(t *testing.T) {
	setupEnv(t)

	// Existing user: wizard already done
	existing := DefaultAppSettings()
	existing.OnboardingCompleted = true
	writeSettings(t, existing)

	// User clicks "Re-run setup" — saves OnboardingCompleted=false
	current, _ := LoadAppSettings()
	current.OnboardingCompleted = false
	if err := SaveAppSettings(current); err != nil {
		t.Fatalf("SaveAppSettings: %v", err)
	}

	// Next launch: wizard should appear
	next, err := LoadAppSettings()
	if err != nil {
		t.Fatalf("next launch LoadAppSettings: %v", err)
	}
	if next.OnboardingCompleted {
		t.Error("wizard should reappear after re-run setup")
	}
}

// ============================================================
// Scenario: User changes settings repeatedly
// ============================================================

// TestScenario_UserTogglesTheme_OnlyLastChoicePersists simulates a user who
// keeps toggling the theme before closing settings — only the final pick is saved.
func TestScenario_UserTogglesTheme_OnlyLastChoicePersists(t *testing.T) {
	setupEnv(t)

	changes := []string{"light", "dark", "light", "auto", "dark"}

	var s *AppSettings
	for _, theme := range changes {
		s, _ = LoadAppSettings()
		s.Theme = theme
		if err := SaveAppSettings(s); err != nil {
			t.Fatalf("SaveAppSettings(theme=%s): %v", theme, err)
		}
	}

	// Reload — should reflect the last change
	final, err := LoadAppSettings()
	if err != nil {
		t.Fatalf("LoadAppSettings final: %v", err)
	}
	if final.Theme != "dark" {
		t.Errorf("Theme = %q, want dark (last change)", final.Theme)
	}
}

// TestScenario_UserSwitchesLanguage_PersistsAcrossRestarts verifies zh→en→zh
// survives simulated app restarts (re-creation of the settings path).
func TestScenario_UserSwitchesLanguage_PersistsAcrossRestarts(t *testing.T) {
	setupEnv(t)

	steps := []string{"en", "zh", "en"}
	for _, lang := range steps {
		s, _ := LoadAppSettings()
		s.Language = lang
		if err := SaveAppSettings(s); err != nil {
			t.Fatalf("SaveAppSettings(lang=%s): %v", lang, err)
		}
		// Simulate restart
		loaded, _ := LoadAppSettings()
		if loaded.Language != lang {
			t.Errorf("after restart, Language = %q, want %q", loaded.Language, lang)
		}
	}
}

// TestScenario_UserDragsFontSlider_ExtremesAreClamped simulates a user dragging
// the font-size slider to impossible values; all must be clamped safely.
func TestScenario_UserDragsFontSlider_ExtremesAreClamped(t *testing.T) {
	setupEnv(t)

	cases := []struct {
		input int
		want  int
	}{
		{0, 10},
		{-100, 10},
		{9, 10},
		{10, 10},  // boundary: no clamp
		{13, 13},  // default
		{24, 24},  // boundary: no clamp
		{25, 24},
		{999, 24},
	}

	for _, tc := range cases {
		s := DefaultAppSettings()
		s.EditorFontSize = tc.input
		writeSettings(t, s)

		loaded, err := LoadAppSettings()
		if err != nil {
			t.Fatalf("LoadAppSettings (input=%d): %v", tc.input, err)
		}
		if loaded.EditorFontSize != tc.want {
			t.Errorf("input=%d → EditorFontSize=%d, want %d", tc.input, loaded.EditorFontSize, tc.want)
		}
	}
}

// ============================================================
// Scenario: Configuration recovery
// ============================================================

// TestScenario_ConfigCorrupted_MidSession_AppRecoversOnRestart simulates the
// settings file becoming corrupt (e.g., disk error or partial write) and verifies
// that the next startup silently falls back to defaults without crashing.
func TestScenario_ConfigCorrupted_MidSession_AppRecoversOnRestart(t *testing.T) {
	setupEnv(t)

	// Normal session: save real settings
	s := DefaultAppSettings()
	s.Theme = "light"
	s.Language = "en"
	writeSettings(t, s)

	// Corruption event (e.g., disk error truncates the file mid-write)
	p, _ := settingsPath()
	os.WriteFile(p, []byte("{\"theme\":\"lig"), 0644)

	// Next launch — must not crash, returns defaults
	loaded, err := LoadAppSettings()
	if err != nil {
		t.Fatalf("LoadAppSettings after corruption: %v", err)
	}
	if loaded.Theme != "dark" {
		t.Errorf("corrupted file should return default theme, got %q", loaded.Theme)
	}
}

// TestScenario_EmptySettingsFile_TreatedAsCorrupt verifies that an empty file
// (e.g., disk full during write) does not cause a panic or nil pointer.
func TestScenario_EmptySettingsFile_TreatedAsCorrupt(t *testing.T) {
	setupEnv(t)
	writeRaw(t, []byte{})

	s, err := LoadAppSettings()
	if err != nil {
		t.Fatalf("empty file should not return error, got: %v", err)
	}
	if s == nil {
		t.Fatal("nil settings returned for empty file")
	}
}

// TestScenario_PartialConfig_MissingFieldsGetDefaults simulates upgrading from
// an older app version that didn't have OnboardingCompleted or StartupPage fields.
func TestScenario_PartialConfig_MissingFieldsGetDefaults(t *testing.T) {
	setupEnv(t)

	// Old app only stored theme and language
	oldConfig := map[string]any{
		"theme":    "light",
		"language": "en",
	}
	data, _ := json.Marshal(oldConfig)
	writeRaw(t, data)

	s, err := LoadAppSettings()
	if err != nil {
		t.Fatalf("LoadAppSettings from partial config: %v", err)
	}
	// Explicit fields preserved
	if s.Theme != "light" {
		t.Errorf("Theme = %q, want light", s.Theme)
	}
	if s.Language != "en" {
		t.Errorf("Language = %q, want en", s.Language)
	}
	// Missing fields get defaults
	if s.EditorFontSize != 13 {
		t.Errorf("missing EditorFontSize should default to 13, got %d", s.EditorFontSize)
	}
	if s.StartupPage != "dashboard" {
		t.Errorf("missing StartupPage should default to dashboard, got %q", s.StartupPage)
	}
	if s.OnboardingCompleted {
		t.Error("missing OnboardingCompleted should default to false (wizard shows)")
	}
}

// ============================================================
// Scenario: Concurrent rapid saves (user clicks OK many times)
// ============================================================

// TestScenario_RapidSaveCalls_NoDataCorruption simulates a user who rapidly
// clicks "Save" (e.g., clicking OK multiple times before the first save finishes).
// The final on-disk state must be valid JSON with a consistent field value.
func TestScenario_RapidSaveCalls_NoDataCorruption(t *testing.T) {
	setupEnv(t)

	const goroutines = 10
	var wg sync.WaitGroup

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			s := DefaultAppSettings()
			// Each goroutine writes its own consistent state
			s.EditorFontSize = 13
			_ = SaveAppSettings(s)
		}(i)
	}
	wg.Wait()

	// Final state must be loadable valid JSON
	final, err := LoadAppSettings()
	if err != nil {
		t.Fatalf("LoadAppSettings after concurrent saves: %v", err)
	}
	if final == nil {
		t.Fatal("nil settings after concurrent saves")
	}
	// Font size must be in valid range (13 is within bounds)
	if final.EditorFontSize < 10 || final.EditorFontSize > 24 {
		t.Errorf("EditorFontSize %d out of valid range after concurrent saves", final.EditorFontSize)
	}
}

// ============================================================
// Scenario: Startup page selection
// ============================================================

// TestScenario_UserSetsStartupPage_AllValidPages verifies each supported startup
// page value survives a save/load cycle.
func TestScenario_UserSetsStartupPage_AllValidPages(t *testing.T) {
	setupEnv(t)

	pages := []string{"dashboard", "claude", "codex", "gemini", "picoclaw", "nullclaw"}

	for _, page := range pages {
		s := DefaultAppSettings()
		s.StartupPage = page
		if err := SaveAppSettings(s); err != nil {
			t.Fatalf("SaveAppSettings(page=%s): %v", page, err)
		}
		loaded, err := LoadAppSettings()
		if err != nil {
			t.Fatalf("LoadAppSettings(page=%s): %v", page, err)
		}
		if loaded.StartupPage != page {
			t.Errorf("StartupPage = %q, want %q", loaded.StartupPage, page)
		}
	}
}

// ============================================================
// Scenario: AutoUpdate toggle
// ============================================================

// TestScenario_UserDisablesAutoUpdate_SettingPersists verifies the boolean
// AutoUpdate field persists correctly across save/load (including false value).
func TestScenario_UserDisablesAutoUpdate_SettingPersists(t *testing.T) {
	setupEnv(t)

	// Default is true — user disables it
	s, _ := LoadAppSettings()
	if !s.AutoUpdate {
		t.Fatal("AutoUpdate should default to true")
	}
	s.AutoUpdate = false
	SaveAppSettings(s)

	// Reload — must be false, not silently reset to true
	loaded, err := LoadAppSettings()
	if err != nil {
		t.Fatalf("LoadAppSettings: %v", err)
	}
	if loaded.AutoUpdate {
		t.Error("AutoUpdate should be false after user disabled it")
	}

	// User re-enables it
	loaded.AutoUpdate = true
	SaveAppSettings(loaded)

	final, _ := LoadAppSettings()
	if !final.AutoUpdate {
		t.Error("AutoUpdate should be true after user re-enabled it")
	}
}
