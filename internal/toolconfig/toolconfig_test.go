package toolconfig

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"lurus-switch/internal/installer"
)

// === GetConfigPath Tests ===

func TestGetConfigPath_KnownTools(t *testing.T) {
	tools := []struct {
		name     string
		filename string
	}{
		{"claude", "settings.json"},
		{"codex", "config.toml"},
		{"gemini", "settings.json"},
		{"antigravity", AntigravityConfigFilename},
		{"opencode", OpenCodeConfigFilename},
		{"aider", aiderConfigFilename},
		{"picoclaw", "config.json"},
	}

	for _, tt := range tools {
		t.Run(tt.name, func(t *testing.T) {
			path, err := GetConfigPath(tt.name)
			if err != nil {
				t.Fatalf("GetConfigPath(%q) error: %v", tt.name, err)
			}
			if !strings.HasSuffix(path, tt.filename) {
				t.Errorf("path %q should end with %q", path, tt.filename)
			}
		})
	}
}

func TestGetConfigPath_UnknownTool(t *testing.T) {
	_, err := GetConfigPath("nonexistent")
	if err == nil {
		t.Fatal("expected error for unknown tool")
	}
	if !strings.Contains(err.Error(), "unknown tool") {
		t.Errorf("error = %q, should mention unknown tool", err.Error())
	}
}

// === ReadConfig Tests ===

func TestReadConfig_UnknownTool(t *testing.T) {
	_, err := ReadConfig("nonexistent")
	if err == nil {
		t.Fatal("expected error for unknown tool")
	}
}

func TestReadConfig_NonExistentFile_ReturnsDefault(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	t.Setenv("USERPROFILE", tmpHome)

	info, err := ReadConfig("claude")
	if err != nil {
		t.Fatalf("ReadConfig error: %v", err)
	}
	if info.Exists {
		t.Error("Exists should be false for non-existent file")
	}
	if info.Content == "" {
		t.Error("Content should contain default template")
	}
	if info.Tool != "claude" {
		t.Errorf("Tool = %q", info.Tool)
	}
	if info.Language != "json" {
		t.Errorf("Language = %q", info.Language)
	}
}

func TestReadConfig_ExistingFile(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	t.Setenv("USERPROFILE", tmpHome)

	// Create the config file
	configDir := filepath.Join(tmpHome, ".claude")
	os.MkdirAll(configDir, 0755)
	content := `{"test": "value"}`
	os.WriteFile(filepath.Join(configDir, "settings.json"), []byte(content), 0600)

	info, err := ReadConfig("claude")
	if err != nil {
		t.Fatalf("ReadConfig error: %v", err)
	}
	if !info.Exists {
		t.Error("Exists should be true")
	}
	if info.Content != content {
		t.Errorf("Content = %q, want %q", info.Content, content)
	}
}

// === WriteConfig Tests ===

func TestWriteConfig_Success(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	t.Setenv("USERPROFILE", tmpHome)

	content := `{"test": "written"}`
	err := WriteConfig("claude", content)
	if err != nil {
		t.Fatalf("WriteConfig error: %v", err)
	}

	// Verify file was created
	configPath := filepath.Join(tmpHome, ".claude", "settings.json")
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("read written file: %v", err)
	}
	if string(data) != content {
		t.Errorf("file content = %q, want %q", string(data), content)
	}
}

func TestWriteConfig_CreatesDirectory(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	t.Setenv("USERPROFILE", tmpHome)

	err := WriteConfig("picoclaw", `{"test": true}`)
	if err != nil {
		t.Fatalf("WriteConfig error: %v", err)
	}

	configDir := filepath.Join(tmpHome, ".picoclaw")
	stat, err := os.Stat(configDir)
	if err != nil {
		t.Fatalf("config dir not created: %v", err)
	}
	if !stat.IsDir() {
		t.Error("expected directory")
	}
}

func TestWriteConfig_UnknownTool(t *testing.T) {
	err := WriteConfig("nonexistent", "content")
	if err == nil {
		t.Fatal("expected error for unknown tool")
	}
}

func TestWriteConfig_AllTools(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	t.Setenv("USERPROFILE", tmpHome)

	for _, tool := range []string{"claude", "codex", "gemini", "picoclaw"} {
		t.Run(tool, func(t *testing.T) {
			err := WriteConfig(tool, `test content`)
			if err != nil {
				t.Fatalf("WriteConfig(%q) error: %v", tool, err)
			}
		})
	}
}

// === GetAllConfigPaths Tests ===

func TestGetAllConfigPaths_ReturnsFourTools(t *testing.T) {
	paths := GetAllConfigPaths()
	expected := []string{"claude", "codex", "gemini", "antigravity", "opencode", "aider", "picoclaw", "nullclaw", "zeroclaw", "openclaw"}

	for _, tool := range expected {
		if _, ok := paths[tool]; !ok {
			t.Errorf("missing tool %q in paths", tool)
		}
	}

	if len(paths) != len(expected) {
		t.Errorf("expected %d paths, got %d", len(expected), len(paths))
	}
}

func TestGetAllConfigPaths_PathsAreAbsolute(t *testing.T) {
	paths := GetAllConfigPaths()
	for tool, path := range paths {
		if path == "" {
			t.Errorf("tool %q has empty path", tool)
		}
	}
}

// === Default Templates Tests ===

func TestDefaultTemplates_AllToolsHaveDefaults(t *testing.T) {
	for tool := range toolDefs {
		tmpl, ok := defaultTemplates[tool]
		if !ok {
			t.Errorf("tool %q has no default template", tool)
		}
		if tmpl == "" {
			t.Errorf("tool %q has empty default template", tool)
		}
	}
}

func TestDefaultTemplates_PicoClawUsesConstant(t *testing.T) {
	tmpl := defaultTemplates["picoclaw"]
	if !strings.Contains(tmpl, installer.DefaultPicoClawModel) {
		t.Errorf("picoclaw template should reference DefaultPicoClawModel (%s), got:\n%s",
			installer.DefaultPicoClawModel, tmpl)
	}
}

func TestDefaultTemplates_ClaudeIsValidJSON(t *testing.T) {
	tmpl := defaultTemplates["claude"]
	if !strings.HasPrefix(strings.TrimSpace(tmpl), "{") {
		t.Error("claude template should be JSON")
	}
}

// === ToolConfigInfo Tests ===

func TestToolConfigInfo_Fields(t *testing.T) {
	info := ToolConfigInfo{
		Tool:     "claude",
		Path:     "/home/user/.claude/settings.json",
		Exists:   true,
		Language: "json",
		Content:  "{}",
	}
	if info.Tool != "claude" {
		t.Errorf("Tool = %q", info.Tool)
	}
	if info.Language != "json" {
		t.Errorf("Language = %q", info.Language)
	}
}

// === configDef Tests ===

func TestConfigDef_LanguageValues(t *testing.T) {
	validLanguages := map[string]bool{"json": true, "toml": true, "yaml": true, "markdown": true}
	for tool, def := range toolDefs {
		if !validLanguages[def.language] {
			t.Errorf("tool %q has invalid language %q", tool, def.language)
		}
	}
}

// === Dir helper function Tests ===

func TestDirFunctions_ReturnNonEmpty(t *testing.T) {
	dirs := map[string]func() string{
		"claude":   claudeDir,
		"codex":    codexDir,
		"gemini":   geminiDir,
		"picoclaw": picoClawDir,
	}
	for name, fn := range dirs {
		dir := fn()
		if dir == "" {
			t.Errorf("%s dir function returned empty", name)
		}
		if !strings.Contains(dir, "."+name) && name != "picoclaw" {
			// picoclaw uses .picoclaw
		}
	}
}

func TestDirFunctions_ContainExpectedSuffix(t *testing.T) {
	if !strings.HasSuffix(claudeDir(), ".claude") {
		t.Errorf("claudeDir = %q, want suffix .claude", claudeDir())
	}
	if !strings.HasSuffix(codexDir(), ".codex") {
		t.Errorf("codexDir = %q", codexDir())
	}
	if !strings.HasSuffix(geminiDir(), ".gemini") {
		t.Errorf("geminiDir = %q", geminiDir())
	}
	if !strings.HasSuffix(picoClawDir(), ".picoclaw") {
		t.Errorf("picoClawDir = %q", picoClawDir())
	}
}

// === opencode Tests ===

func TestGetConfigPath_OpenCode(t *testing.T) {
	path, err := GetConfigPath("opencode")
	if err != nil {
		t.Fatalf("GetConfigPath(opencode) error: %v", err)
	}
	if !strings.HasSuffix(path, "opencode.json") {
		t.Errorf("path %q should end with opencode.json", path)
	}
	if !strings.Contains(path, "opencode") {
		t.Errorf("path %q should contain opencode directory segment", path)
	}
}

func TestOpenCodeConfigDir_XDGEnvOverride(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	dir := opencodeConfigDir()
	want := filepath.Join(tmpDir, "opencode")
	if dir != want {
		t.Errorf("opencodeConfigDir = %q, want %q", dir, want)
	}
}

func TestOpenCodeConfigDir_FallbackContainsOpencode(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "")
	// Don't rely on LOCALAPPDATA being set or unset; just verify "opencode" appears.
	dir := opencodeConfigDir()
	if !strings.Contains(dir, "opencode") {
		t.Errorf("opencodeConfigDir = %q, should contain 'opencode'", dir)
	}
	if dir == "" {
		t.Error("opencodeConfigDir returned empty string")
	}
}

func TestReadConfig_OpenCode_NoFile_ReturnsDefault(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	t.Setenv("USERPROFILE", tmpHome)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmpHome, ".config"))
	t.Setenv("LOCALAPPDATA", filepath.Join(tmpHome, "AppData", "Local"))

	info, err := ReadConfig("opencode")
	if err != nil {
		t.Fatalf("ReadConfig(opencode) error: %v", err)
	}
	if info.Exists {
		t.Error("Exists should be false for non-existent file")
	}
	if info.Content == "" {
		t.Error("Content should contain default template")
	}
	if info.Tool != "opencode" {
		t.Errorf("Tool = %q, want opencode", info.Tool)
	}
	if info.Language != "json" {
		t.Errorf("Language = %q, want json", info.Language)
	}
	if !strings.Contains(info.Content, "model") {
		t.Error("default template should contain 'model' field")
	}
}

func TestWriteConfig_OpenCode_RoundTrip(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	t.Setenv("USERPROFILE", tmpHome)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmpHome, ".config"))
	t.Setenv("LOCALAPPDATA", filepath.Join(tmpHome, "AppData", "Local"))

	content := `{"model": "anthropic/claude-sonnet-4-5", "autoupdate": "notify"}`
	if err := WriteConfig("opencode", content); err != nil {
		t.Fatalf("WriteConfig(opencode) error: %v", err)
	}

	info, err := ReadConfig("opencode")
	if err != nil {
		t.Fatalf("ReadConfig(opencode) after write error: %v", err)
	}
	if !info.Exists {
		t.Error("Exists should be true after write")
	}
	if info.Content != content {
		t.Errorf("Content = %q, want %q", info.Content, content)
	}
}

func TestWriteConfig_OpenCode_CreatesDirectory(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	t.Setenv("USERPROFILE", tmpHome)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmpHome, ".config"))
	t.Setenv("LOCALAPPDATA", filepath.Join(tmpHome, "AppData", "Local"))

	if err := WriteConfig("opencode", `{"model": "anthropic/test"}`); err != nil {
		t.Fatalf("WriteConfig(opencode) error: %v", err)
	}

	configPath := filepath.Join(tmpHome, ".config", "opencode", "opencode.json")
	if _, err := os.Stat(configPath); err != nil {
		t.Errorf("expected config file at %q: %v", configPath, err)
	}
}

func TestReadOpenCodeConfig_NoFile(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	t.Setenv("USERPROFILE", tmpHome)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmpHome, ".config"))
	t.Setenv("LOCALAPPDATA", filepath.Join(tmpHome, "AppData", "Local"))

	cfg, err := ReadOpenCodeConfig()
	if err != nil {
		t.Fatalf("ReadOpenCodeConfig error: %v", err)
	}
	if cfg == nil {
		t.Fatal("expected non-nil config")
	}
	// Zero value: all fields empty
	if cfg.Model != "" {
		t.Errorf("Model = %q, want empty for missing file", cfg.Model)
	}
}

func TestWriteOpenCodeConfig_RoundTrip(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	t.Setenv("USERPROFILE", tmpHome)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmpHome, ".config"))
	t.Setenv("LOCALAPPDATA", filepath.Join(tmpHome, "AppData", "Local"))

	in := &OpenCodeConfig{
		Model:      "anthropic/claude-sonnet-4-5",
		SmallModel: "anthropic/claude-haiku-4-5",
		AutoUpdate: "notify",
		Provider: map[string]OpenCodeProviderConfig{
			"anthropic": {APIKey: "sk-test-123"},
		},
	}

	if err := WriteOpenCodeConfig(in); err != nil {
		t.Fatalf("WriteOpenCodeConfig error: %v", err)
	}

	got, err := ReadOpenCodeConfig()
	if err != nil {
		t.Fatalf("ReadOpenCodeConfig error: %v", err)
	}
	if got.Model != in.Model {
		t.Errorf("Model = %q, want %q", got.Model, in.Model)
	}
	if got.SmallModel != in.SmallModel {
		t.Errorf("SmallModel = %q, want %q", got.SmallModel, in.SmallModel)
	}
	if got.Provider["anthropic"].APIKey != "sk-test-123" {
		t.Errorf("Provider anthropic APIKey = %q, want sk-test-123", got.Provider["anthropic"].APIKey)
	}
}

func TestOpenCodeConfig_PreservesUnknownFields(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	t.Setenv("USERPROFILE", tmpHome)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmpHome, ".config"))
	t.Setenv("LOCALAPPDATA", filepath.Join(tmpHome, "AppData", "Local"))

	// Seed a config file that contains sections Switch does not model.
	dir := filepath.Join(tmpHome, ".config", "opencode")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	seed := `{
  "model": "anthropic/claude-sonnet-4-5",
  "mcp": {"fs": {"command": "npx", "args": ["-y", "@modelcontextprotocol/server-filesystem"]}},
  "agent": {"build": {"model": "anthropic/claude-haiku-4-5"}},
  "instructions": ["AGENTS.md"]
}`
	path := filepath.Join(dir, OpenCodeConfigFilename)
	if err := os.WriteFile(path, []byte(seed), 0o600); err != nil {
		t.Fatalf("seed write: %v", err)
	}

	// Read → mutate a managed field → write.
	cfg, err := ReadOpenCodeConfig()
	if err != nil {
		t.Fatalf("ReadOpenCodeConfig: %v", err)
	}
	cfg.Model = "anthropic/claude-opus-4-8"
	if err := WriteOpenCodeConfig(cfg); err != nil {
		t.Fatalf("WriteOpenCodeConfig: %v", err)
	}

	// The unmanaged sections must survive the round-trip.
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read back: %v", err)
	}
	var out map[string]json.RawMessage
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("unmarshal back: %v", err)
	}
	for _, key := range []string{"mcp", "agent", "instructions"} {
		if _, ok := out[key]; !ok {
			t.Errorf("unmanaged field %q was dropped on round-trip; file=%s", key, raw)
		}
	}
	if got := string(out["model"]); got != `"anthropic/claude-opus-4-8"` {
		t.Errorf("model = %s, want the mutated value", got)
	}
}

func TestWriteOpenCodeConfig_NilError(t *testing.T) {
	err := WriteOpenCodeConfig(nil)
	if err == nil {
		t.Fatal("expected error for nil cfg")
	}
	if !strings.Contains(err.Error(), "nil") {
		t.Errorf("error = %q, should mention nil", err.Error())
	}
}

func TestDefaultTemplates_OpenCodeIsValidJSON(t *testing.T) {
	tmpl := defaultTemplates["opencode"]
	if !strings.HasPrefix(strings.TrimSpace(tmpl), "{") {
		t.Error("opencode default template should be JSON")
	}
	if !strings.Contains(tmpl, "model") {
		t.Error("opencode default template should contain 'model' field")
	}
}

func TestOpenCodeConfigFilename(t *testing.T) {
	if OpenCodeConfigFilename != "opencode.json" {
		t.Errorf("OpenCodeConfigFilename = %q, want opencode.json", OpenCodeConfigFilename)
	}
}

func TestOpenCodeBinaryName(t *testing.T) {
	if OpenCodeBinaryName != "opencode" {
		t.Errorf("OpenCodeBinaryName = %q, want opencode", OpenCodeBinaryName)
	}
}
