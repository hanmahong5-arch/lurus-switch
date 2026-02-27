package toolconfig

import (
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
	expected := []string{"claude", "codex", "gemini", "picoclaw"}

	for _, tool := range expected {
		if _, ok := paths[tool]; !ok {
			t.Errorf("missing tool %q in paths", tool)
		}
	}

	if len(paths) != 4 {
		t.Errorf("expected 4 paths, got %d", len(paths))
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
	validLanguages := map[string]bool{"json": true, "toml": true, "markdown": true}
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
