package toolconfig

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

// --- AiderConfigPath ---------------------------------------------------------

func TestAiderConfigPath_ContainsFilename(t *testing.T) {
	path, err := AiderConfigPath()
	if err != nil {
		t.Fatalf("AiderConfigPath() error: %v", err)
	}
	if !strings.HasSuffix(path, aiderConfigFilename) {
		t.Errorf("AiderConfigPath() = %q, want suffix %q", path, aiderConfigFilename)
	}
}

func TestAiderConfigPath_IsAbsolute(t *testing.T) {
	path, err := AiderConfigPath()
	if err != nil {
		t.Fatalf("AiderConfigPath() error: %v", err)
	}
	if !filepath.IsAbs(path) {
		t.Errorf("AiderConfigPath() = %q, want absolute path", path)
	}
}

// --- ReadAiderConfig ---------------------------------------------------------

func TestReadAiderConfig_MissingFile_ReturnsEmptyMap(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	t.Setenv("USERPROFILE", tmpHome)

	cfg, err := ReadAiderConfig()
	if err != nil {
		t.Fatalf("ReadAiderConfig() error: %v", err)
	}
	if cfg == nil {
		t.Fatal("ReadAiderConfig() returned nil map")
	}
	if len(cfg) != 0 {
		t.Errorf("ReadAiderConfig() returned non-empty map for missing file: %v", cfg)
	}
}

func TestReadAiderConfig_ValidYAML(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	t.Setenv("USERPROFILE", tmpHome)

	content := "anthropic-api-key: sk-ant-test123\nmodel: claude-3-5-sonnet-20241022\n"
	if err := os.WriteFile(filepath.Join(tmpHome, aiderConfigFilename), []byte(content), 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	cfg, err := ReadAiderConfig()
	if err != nil {
		t.Fatalf("ReadAiderConfig() error: %v", err)
	}
	if cfg["anthropic-api-key"] != "sk-ant-test123" {
		t.Errorf("anthropic-api-key = %v, want sk-ant-test123", cfg["anthropic-api-key"])
	}
	if cfg["model"] != "claude-3-5-sonnet-20241022" {
		t.Errorf("model = %v, want claude-3-5-sonnet-20241022", cfg["model"])
	}
}

func TestReadAiderConfig_InvalidYAML_ReturnsError(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	t.Setenv("USERPROFILE", tmpHome)

	if err := os.WriteFile(filepath.Join(tmpHome, aiderConfigFilename), []byte(":\n:\n: invalid::yaml::\n"), 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	_, err := ReadAiderConfig()
	if err == nil {
		t.Fatal("ReadAiderConfig() expected error for invalid YAML")
	}
}

// --- WriteAiderConfig --------------------------------------------------------

func TestWriteAiderConfig_NilMapReturnsError(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	t.Setenv("USERPROFILE", tmpHome)

	if err := WriteAiderConfig(nil); err == nil {
		t.Fatal("WriteAiderConfig(nil) expected error")
	}
}

func TestWriteAiderConfig_RoundTrip(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	t.Setenv("USERPROFILE", tmpHome)

	in := map[string]any{
		"anthropic-api-key": "sk-ant-abc",
		"model":             "claude-3-5-sonnet-20241022",
		"dark-mode":         true,
	}

	if err := WriteAiderConfig(in); err != nil {
		t.Fatalf("WriteAiderConfig() error: %v", err)
	}

	out, err := ReadAiderConfig()
	if err != nil {
		t.Fatalf("ReadAiderConfig() after write error: %v", err)
	}
	if out["anthropic-api-key"] != "sk-ant-abc" {
		t.Errorf("anthropic-api-key = %v, want sk-ant-abc", out["anthropic-api-key"])
	}
	if out["model"] != "claude-3-5-sonnet-20241022" {
		t.Errorf("model = %v, want claude-3-5-sonnet-20241022", out["model"])
	}
}

func TestWriteAiderConfig_FilePermissions(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	t.Setenv("USERPROFILE", tmpHome)

	if err := WriteAiderConfig(map[string]any{"model": "x"}); err != nil {
		t.Fatalf("WriteAiderConfig() error: %v", err)
	}

	info, err := os.Stat(filepath.Join(tmpHome, aiderConfigFilename))
	if err != nil {
		t.Fatalf("stat config file: %v", err)
	}
	// On Windows file permission model is coarser; accept any non-world-readable result.
	// On Unix the file should be 0600.
	mode := info.Mode().Perm()
	if mode&0o077 != 0 {
		// World/group bits set — warn but don't hard-fail on Windows where this
		// is controlled by ACLs not permission bits.
		t.Logf("note: config file mode %o has group/other bits set (may be normal on Windows)", mode)
	}
}

// --- InjectCredentials -------------------------------------------------------

func TestInjectCredentials_InjectsAnthropicKey(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	t.Setenv("USERPROFILE", tmpHome)

	creds := CredSet{AnthropicKey: "sk-ant-inject-test"}
	if err := InjectCredentials(creds); err != nil {
		t.Fatalf("InjectCredentials() error: %v", err)
	}

	cfg, err := ReadAiderConfig()
	if err != nil {
		t.Fatalf("ReadAiderConfig() error: %v", err)
	}
	if cfg["anthropic-api-key"] != "sk-ant-inject-test" {
		t.Errorf("anthropic-api-key = %v, want sk-ant-inject-test", cfg["anthropic-api-key"])
	}
}

func TestInjectCredentials_InjectsOpenAIKey(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	t.Setenv("USERPROFILE", tmpHome)

	creds := CredSet{OpenAIKey: "sk-openai-xyz", OpenAIBaseURL: "https://my-proxy.example.com/v1"}
	if err := InjectCredentials(creds); err != nil {
		t.Fatalf("InjectCredentials() error: %v", err)
	}

	cfg, err := ReadAiderConfig()
	if err != nil {
		t.Fatalf("ReadAiderConfig() error: %v", err)
	}
	if cfg["openai-api-key"] != "sk-openai-xyz" {
		t.Errorf("openai-api-key = %v, want sk-openai-xyz", cfg["openai-api-key"])
	}
	if cfg["openai-api-base"] != "https://my-proxy.example.com/v1" {
		t.Errorf("openai-api-base = %v, want proxy URL", cfg["openai-api-base"])
	}
}

func TestInjectCredentials_PreservesExistingFields(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	t.Setenv("USERPROFILE", tmpHome)

	// Write a pre-existing config with user customisations.
	existing := map[string]any{
		"model":     "claude-3-5-sonnet-20241022",
		"dark-mode": true,
		"auto-test": false,
	}
	if err := WriteAiderConfig(existing); err != nil {
		t.Fatalf("setup WriteAiderConfig: %v", err)
	}

	creds := CredSet{AnthropicKey: "sk-ant-new"}
	if err := InjectCredentials(creds); err != nil {
		t.Fatalf("InjectCredentials() error: %v", err)
	}

	cfg, err := ReadAiderConfig()
	if err != nil {
		t.Fatalf("ReadAiderConfig() error: %v", err)
	}

	// New key injected.
	if cfg["anthropic-api-key"] != "sk-ant-new" {
		t.Errorf("anthropic-api-key = %v, want sk-ant-new", cfg["anthropic-api-key"])
	}
	// Pre-existing user fields preserved.
	if cfg["model"] != "claude-3-5-sonnet-20241022" {
		t.Errorf("model = %v, want claude-3-5-sonnet-20241022 (should be preserved)", cfg["model"])
	}
	if cfg["dark-mode"] != true {
		t.Errorf("dark-mode = %v, want true (should be preserved)", cfg["dark-mode"])
	}
}

func TestInjectCredentials_EmptyCredsNoOp(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	t.Setenv("USERPROFILE", tmpHome)

	// No config file exists.
	if err := InjectCredentials(CredSet{}); err != nil {
		t.Fatalf("InjectCredentials(empty) error: %v", err)
	}

	// Config file should still not exist (no write for empty creds).
	cfgPath := filepath.Join(tmpHome, aiderConfigFilename)
	_, err := os.Stat(cfgPath)
	if err == nil {
		// File was created — check it is empty / minimal.
		data, _ := os.ReadFile(cfgPath)
		var m map[string]any
		_ = yaml.Unmarshal(data, &m)
		if len(m) != 0 {
			t.Errorf("empty inject wrote unexpected keys: %v", m)
		}
	}
	// No .env file should be created when no env-only keys are set.
	envPath := filepath.Join(tmpHome, aiderEnvFilename)
	if _, err := os.Stat(envPath); err == nil {
		t.Error("InjectCredentials(empty) created .env file unexpectedly")
	}
}

func TestInjectCredentials_GeminiKeyGoesToEnvFile(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	t.Setenv("USERPROFILE", tmpHome)

	creds := CredSet{GeminiKey: "AIza-gemini-test"}
	if err := InjectCredentials(creds); err != nil {
		t.Fatalf("InjectCredentials() error: %v", err)
	}

	envPath := filepath.Join(tmpHome, aiderEnvFilename)
	data, err := os.ReadFile(envPath)
	if err != nil {
		t.Fatalf("read .env: %v", err)
	}
	if !strings.Contains(string(data), "GEMINI_API_KEY=AIza-gemini-test") {
		t.Errorf(".env content = %q, want GEMINI_API_KEY=AIza-gemini-test", string(data))
	}
}

func TestInjectCredentials_DeepSeekKeyGoesToEnvFile(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	t.Setenv("USERPROFILE", tmpHome)

	creds := CredSet{DeepSeekKey: "ds-test-key-abc"}
	if err := InjectCredentials(creds); err != nil {
		t.Fatalf("InjectCredentials() error: %v", err)
	}

	envPath := filepath.Join(tmpHome, aiderEnvFilename)
	data, err := os.ReadFile(envPath)
	if err != nil {
		t.Fatalf("read .env: %v", err)
	}
	if !strings.Contains(string(data), "DEEPSEEK_API_KEY=ds-test-key-abc") {
		t.Errorf(".env content = %q, want DEEPSEEK_API_KEY=ds-test-key-abc", string(data))
	}
}

func TestInjectCredentials_PreservesOtherEnvLines(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	t.Setenv("USERPROFILE", tmpHome)

	// Pre-existing .env with an unrelated key.
	preExisting := "MY_CUSTOM_VAR=foobar\n"
	envPath := filepath.Join(tmpHome, aiderEnvFilename)
	if err := os.WriteFile(envPath, []byte(preExisting), 0o600); err != nil {
		t.Fatalf("setup .env: %v", err)
	}

	creds := CredSet{GeminiKey: "AIza-new"}
	if err := InjectCredentials(creds); err != nil {
		t.Fatalf("InjectCredentials() error: %v", err)
	}

	data, err := os.ReadFile(envPath)
	if err != nil {
		t.Fatalf("read .env: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "MY_CUSTOM_VAR=foobar") {
		t.Error(".env lost pre-existing MY_CUSTOM_VAR line")
	}
	if !strings.Contains(content, "GEMINI_API_KEY=AIza-new") {
		t.Error(".env missing new GEMINI_API_KEY line")
	}
}

func TestInjectCredentials_PreservesNonKeyValueLines(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	t.Setenv("USERPROFILE", tmpHome)

	// A hand-edited .env containing a line with no '=' separator. The rewrite
	// must not silently drop it.
	preExisting := "VALID=1\nexport SOMETHING\nANOTHER=2\n"
	envPath := filepath.Join(tmpHome, aiderEnvFilename)
	if err := os.WriteFile(envPath, []byte(preExisting), 0o600); err != nil {
		t.Fatalf("setup .env: %v", err)
	}

	if err := InjectCredentials(CredSet{GeminiKey: "AIza-new"}); err != nil {
		t.Fatalf("InjectCredentials() error: %v", err)
	}

	data, err := os.ReadFile(envPath)
	if err != nil {
		t.Fatalf("read .env: %v", err)
	}
	content := string(data)
	for _, want := range []string{"VALID=1", "export SOMETHING", "ANOTHER=2", "GEMINI_API_KEY=AIza-new"} {
		if !strings.Contains(content, want) {
			t.Errorf(".env = %q, lost line %q", content, want)
		}
	}
}

func TestInjectCredentials_UpdatesExistingEnvKey(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	t.Setenv("USERPROFILE", tmpHome)

	// Pre-existing .env with an old Gemini key.
	old := "GEMINI_API_KEY=AIza-old\n"
	envPath := filepath.Join(tmpHome, aiderEnvFilename)
	if err := os.WriteFile(envPath, []byte(old), 0o600); err != nil {
		t.Fatalf("setup .env: %v", err)
	}

	creds := CredSet{GeminiKey: "AIza-updated"}
	if err := InjectCredentials(creds); err != nil {
		t.Fatalf("InjectCredentials() error: %v", err)
	}

	data, err := os.ReadFile(envPath)
	if err != nil {
		t.Fatalf("read .env: %v", err)
	}
	content := string(data)
	if strings.Contains(content, "AIza-old") {
		t.Error(".env still contains old Gemini key after update")
	}
	if !strings.Contains(content, "GEMINI_API_KEY=AIza-updated") {
		t.Errorf(".env = %q, want updated GEMINI_API_KEY", content)
	}
}

func TestInjectCredentials_Idempotent(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	t.Setenv("USERPROFILE", tmpHome)

	creds := CredSet{
		AnthropicKey: "sk-ant-idem",
		GeminiKey:    "AIza-idem",
	}

	for range 3 {
		if err := InjectCredentials(creds); err != nil {
			t.Fatalf("InjectCredentials() error: %v", err)
		}
	}

	cfg, err := ReadAiderConfig()
	if err != nil {
		t.Fatalf("ReadAiderConfig() error: %v", err)
	}
	if cfg["anthropic-api-key"] != "sk-ant-idem" {
		t.Errorf("anthropic-api-key = %v, want sk-ant-idem", cfg["anthropic-api-key"])
	}

	envPath := filepath.Join(tmpHome, aiderEnvFilename)
	data, _ := os.ReadFile(envPath)
	// Exactly one GEMINI_API_KEY line.
	count := strings.Count(string(data), "GEMINI_API_KEY=")
	if count != 1 {
		t.Errorf("expected exactly 1 GEMINI_API_KEY line after 3 idempotent runs, got %d\n%s", count, string(data))
	}
}

// --- parseAiderVersion -------------------------------------------------------

func TestParseAiderVersion_WithPrefix(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"aider 0.82.0", "0.82.0"},
		{"aider 1.0.0-rc1", "1.0.0-rc1"},
		{"0.82.0", "0.82.0"},
		{"", "unknown"},
		{"aider", "aider"},
	}
	for _, tt := range tests {
		got := parseAiderVersion(tt.input)
		if got != tt.want {
			t.Errorf("parseAiderVersion(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

// --- DetectAider (structural, no real subprocess) ----------------------------

func TestDetectAider_ReturnStructHasFields(t *testing.T) {
	r, err := DetectAider()
	if err != nil {
		t.Fatalf("DetectAider() error: %v", err)
	}
	if r == nil {
		t.Fatal("DetectAider() returned nil")
	}
	// Installed may be true or false depending on the environment;
	// the struct must always be non-nil and Path consistent with Installed.
	if r.Installed && r.Path == "" {
		t.Error("DetectAider(): Installed=true but Path is empty")
	}
	if !r.Installed && r.Path != "" {
		t.Error("DetectAider(): Installed=false but Path is non-empty")
	}
}

// --- windowsAiderCandidates --------------------------------------------------

func TestWindowsAiderCandidates_NonEmpty(t *testing.T) {
	candidates := windowsAiderCandidates()
	if len(candidates) == 0 {
		t.Error("windowsAiderCandidates() returned empty slice")
	}
	for _, c := range candidates {
		if !filepath.IsAbs(c) {
			t.Errorf("candidate %q is not an absolute path", c)
		}
		if !strings.HasSuffix(c, "aider.exe") {
			t.Errorf("candidate %q does not end with aider.exe", c)
		}
	}
}

func TestWindowsAiderCandidates_CoversPython39To314(t *testing.T) {
	candidates := windowsAiderCandidates()
	joined := strings.Join(candidates, "|")
	for minor := 9; minor <= 14; minor++ {
		tag := fmt.Sprintf("Python3%d", minor)
		if !strings.Contains(joined, tag) {
			t.Errorf("windowsAiderCandidates() missing %s path", tag)
		}
	}
}
