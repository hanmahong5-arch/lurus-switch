package generator

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"lurus-switch/internal/config"
)

// === Constructor Tests ===

func TestNewGeminiGenerator(t *testing.T) {
	gen := NewGeminiGenerator()
	if gen == nil {
		t.Error("NewGeminiGenerator should return non-nil generator")
	}
}

// === GenerateMarkdown Tests ===

func TestGeminiGeneratorGenerateMarkdown(t *testing.T) {
	gen := NewGeminiGenerator()
	cfg := config.NewGeminiConfig()
	cfg.Instructions.ProjectDescription = "Test project"
	cfg.Instructions.TechStack = "Go, React"
	cfg.Instructions.CustomRules = []string{"Rule 1", "Rule 2"}

	content := gen.GenerateMarkdown(cfg)

	if !strings.Contains(content, "# GEMINI.md") {
		t.Error("Generated content should contain GEMINI.md header")
	}

	if !strings.Contains(content, "Test project") {
		t.Error("Generated content should contain project description")
	}

	if !strings.Contains(content, "Go, React") {
		t.Error("Generated content should contain tech stack")
	}

	if !strings.Contains(content, "Rule 1") {
		t.Error("Generated content should contain custom rules")
	}
}

func TestGeminiGeneratorGenerateMarkdown_Header(t *testing.T) {
	gen := NewGeminiGenerator()
	cfg := config.NewGeminiConfig()

	content := gen.GenerateMarkdown(cfg)

	if !strings.Contains(content, "# GEMINI.md") {
		t.Error("Generated content should contain GEMINI.md header")
	}
	if !strings.Contains(content, "This file provides guidance") {
		t.Error("Generated content should contain description")
	}
}

func TestGeminiGeneratorGenerateMarkdown_Empty(t *testing.T) {
	gen := NewGeminiGenerator()
	cfg := &config.GeminiConfig{}

	content := gen.GenerateMarkdown(cfg)

	if !strings.Contains(content, "# GEMINI.md") {
		t.Error("Generated content should contain GEMINI.md header")
	}
	// Should not contain any section headers for empty config
	if strings.Contains(content, "## Project Description") {
		t.Error("Empty config should not have Project Description section")
	}
}

func TestGeminiGeneratorGenerateMarkdown_AllSections(t *testing.T) {
	gen := NewGeminiGenerator()
	cfg := &config.GeminiConfig{
		Instructions: config.GeminiInstructions{
			ProjectDescription: "My project",
			TechStack:          "Go, TypeScript",
			CodeStyle:          "Google style",
			CustomRules:        []string{"Rule 1", "Rule 2"},
			FileStructure:      "cmd/, internal/",
			TestingGuidelines:  "TDD approach",
		},
	}

	content := gen.GenerateMarkdown(cfg)

	expectedSections := []string{
		"## Project Description",
		"## Tech Stack",
		"## Code Style",
		"## Rules",
		"## File Structure",
		"## Testing",
	}

	for _, section := range expectedSections {
		if !strings.Contains(content, section) {
			t.Errorf("Generated content should contain '%s'", section)
		}
	}
}

func TestGeminiGeneratorGenerateMarkdown_CustomRulesFormat(t *testing.T) {
	gen := NewGeminiGenerator()
	cfg := config.NewGeminiConfig()
	cfg.Instructions.CustomRules = []string{"Rule A", "Rule B", "Rule C"}

	content := gen.GenerateMarkdown(cfg)

	// Rules should be formatted as bullet points
	if !strings.Contains(content, "- Rule A") {
		t.Error("Rules should be formatted as bullet points")
	}
	if !strings.Contains(content, "- Rule B") {
		t.Error("Rules should be formatted as bullet points")
	}
	if !strings.Contains(content, "- Rule C") {
		t.Error("Rules should be formatted as bullet points")
	}
}

func TestGeminiGeneratorGenerateMarkdown_FileStructureInCodeBlock(t *testing.T) {
	gen := NewGeminiGenerator()
	cfg := config.NewGeminiConfig()
	cfg.Instructions.FileStructure = "cmd/\ninternal/\npkg/"

	content := gen.GenerateMarkdown(cfg)

	// File structure should be in a code block
	if !strings.Contains(content, "```") {
		t.Error("File structure should be in a code block")
	}
}

// === Generate Tests ===

func TestGeminiGeneratorGenerate(t *testing.T) {
	gen := NewGeminiGenerator()
	cfg := config.NewGeminiConfig()
	cfg.Instructions.ProjectDescription = "Test"

	tmpDir, err := os.MkdirTemp("", "gemini-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	outputPath, err := gen.Generate(cfg, tmpDir)
	if err != nil {
		t.Fatalf("Failed to generate: %v", err)
	}

	if filepath.Base(outputPath) != "GEMINI.md" {
		t.Errorf("Expected GEMINI.md, got %s", filepath.Base(outputPath))
	}

	// Verify file exists
	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		t.Error("Output file was not created")
	}
}

func TestGeminiGeneratorGenerate_CreatesDir(t *testing.T) {
	gen := NewGeminiGenerator()
	cfg := config.NewGeminiConfig()

	tmpDir := t.TempDir()
	outputDir := filepath.Join(tmpDir, "nested", "dir")

	_, err := gen.Generate(cfg, outputDir)
	if err != nil {
		t.Fatalf("Failed to generate: %v", err)
	}

	if _, err := os.Stat(outputDir); os.IsNotExist(err) {
		t.Error("Output directory should be created")
	}
}

func TestGeminiGeneratorGenerate_FileContent(t *testing.T) {
	gen := NewGeminiGenerator()
	cfg := config.NewGeminiConfig()
	cfg.Instructions.ProjectDescription = "Test Project Description"

	tmpDir := t.TempDir()
	outputPath, err := gen.Generate(cfg, tmpDir)
	if err != nil {
		t.Fatalf("Failed to generate: %v", err)
	}

	content, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("Failed to read output file: %v", err)
	}

	if !strings.Contains(string(content), "Test Project Description") {
		t.Error("Output file should contain project description")
	}
}

// === GenerateConfigJSON Tests ===

func TestGeminiGeneratorGenerateConfigJSON(t *testing.T) {
	gen := NewGeminiGenerator()
	cfg := config.NewGeminiConfig()
	cfg.Model = "gemini-1.5-pro"

	tmpDir := t.TempDir()
	outputPath, err := gen.GenerateConfigJSON(cfg, tmpDir)
	if err != nil {
		t.Fatalf("Failed to generate JSON: %v", err)
	}

	if filepath.Base(outputPath) != "gemini-settings.json" {
		t.Errorf("Expected gemini-settings.json, got %s", filepath.Base(outputPath))
	}

	// Verify file exists
	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		t.Error("Output file was not created")
	}
}

func TestGeminiGeneratorGenerateConfigJSON_Content(t *testing.T) {
	gen := NewGeminiGenerator()
	cfg := config.NewGeminiConfig()
	cfg.Model = "gemini-2.0-pro"
	cfg.Auth.Type = "oauth"
	cfg.Behavior.Sandbox = true
	cfg.Display.Theme = "dark"

	tmpDir := t.TempDir()
	outputPath, err := gen.GenerateConfigJSON(cfg, tmpDir)
	if err != nil {
		t.Fatalf("Failed to generate JSON: %v", err)
	}

	content, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("Failed to read output file: %v", err)
	}

	// Parse JSON to verify structure
	var settings map[string]interface{}
	if err := json.Unmarshal(content, &settings); err != nil {
		t.Fatalf("Output should be valid JSON: %v", err)
	}

	if settings["model"] != "gemini-2.0-pro" {
		t.Error("JSON should contain correct model")
	}
}

func TestGeminiGeneratorGenerateConfigJSON_AutoApprove(t *testing.T) {
	gen := NewGeminiGenerator()
	cfg := config.NewGeminiConfig()
	cfg.Behavior.AutoApprove = []string{"file_read", "file_write"}

	tmpDir := t.TempDir()
	outputPath, err := gen.GenerateConfigJSON(cfg, tmpDir)
	if err != nil {
		t.Fatalf("Failed to generate JSON: %v", err)
	}

	content, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("Failed to read output file: %v", err)
	}

	if !strings.Contains(string(content), "autoApprove") {
		t.Error("JSON should contain autoApprove when set")
	}
}

func TestGeminiGeneratorGenerateConfigJSON_AllowedExtensions(t *testing.T) {
	gen := NewGeminiGenerator()
	cfg := config.NewGeminiConfig()
	cfg.Behavior.AllowedExtensions = []string{".go", ".ts", ".py"}

	tmpDir := t.TempDir()
	outputPath, err := gen.GenerateConfigJSON(cfg, tmpDir)
	if err != nil {
		t.Fatalf("Failed to generate JSON: %v", err)
	}

	content, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("Failed to read output file: %v", err)
	}

	if !strings.Contains(string(content), "allowedExtensions") {
		t.Error("JSON should contain allowedExtensions when set")
	}
}

// === GenerateAll Tests ===

func TestGeminiGeneratorGenerateAll(t *testing.T) {
	gen := NewGeminiGenerator()
	cfg := config.NewGeminiConfig()
	cfg.Instructions.ProjectDescription = "Test"

	tmpDir := t.TempDir()
	files, err := gen.GenerateAll(cfg, tmpDir)
	if err != nil {
		t.Fatalf("Failed to generate all: %v", err)
	}

	if len(files) != 2 {
		t.Errorf("Expected 2 files, got %d", len(files))
	}

	// Check both files exist
	mdPath := filepath.Join(tmpDir, "GEMINI.md")
	jsonPath := filepath.Join(tmpDir, "gemini-settings.json")

	if _, err := os.Stat(mdPath); os.IsNotExist(err) {
		t.Error("GEMINI.md should be created")
	}
	if _, err := os.Stat(jsonPath); os.IsNotExist(err) {
		t.Error("gemini-settings.json should be created")
	}
}

func TestGeminiGeneratorGenerateAll_ReturnsCorrectPaths(t *testing.T) {
	gen := NewGeminiGenerator()
	cfg := config.NewGeminiConfig()

	tmpDir := t.TempDir()
	files, err := gen.GenerateAll(cfg, tmpDir)
	if err != nil {
		t.Fatalf("Failed to generate all: %v", err)
	}

	// First file should be GEMINI.md
	if filepath.Base(files[0]) != "GEMINI.md" {
		t.Errorf("First file should be GEMINI.md, got %s", filepath.Base(files[0]))
	}
	// Second file should be gemini-settings.json
	if filepath.Base(files[1]) != "gemini-settings.json" {
		t.Errorf("Second file should be gemini-settings.json, got %s", filepath.Base(files[1]))
	}
}

// === Validate Tests ===

func TestGeminiGeneratorValidate(t *testing.T) {
	gen := NewGeminiGenerator()

	// Valid config
	validCfg := config.NewGeminiConfig()
	if err := gen.Validate(validCfg); err != nil {
		t.Errorf("Valid config should not return error: %v", err)
	}

	// Invalid auth type
	invalidCfg := config.NewGeminiConfig()
	invalidCfg.Auth.Type = "invalid"
	if err := gen.Validate(invalidCfg); err == nil {
		t.Error("Invalid auth type should return error")
	}

	// Invalid theme
	invalidCfg2 := config.NewGeminiConfig()
	invalidCfg2.Display.Theme = "invalid"
	if err := gen.Validate(invalidCfg2); err == nil {
		t.Error("Invalid theme should return error")
	}
}

func TestGeminiGeneratorValidate_EmptyModel(t *testing.T) {
	gen := NewGeminiGenerator()
	cfg := config.NewGeminiConfig()
	cfg.Model = ""

	err := gen.Validate(cfg)
	if err == nil {
		t.Error("Empty model should return error")
	}
	if !strings.Contains(err.Error(), "model is required") {
		t.Error("Error should mention model is required")
	}
}

func TestGeminiGeneratorValidate_AllAuthTypes(t *testing.T) {
	gen := NewGeminiGenerator()

	validTypes := []string{"api_key", "oauth", "adc"}
	for _, authType := range validTypes {
		t.Run(authType, func(t *testing.T) {
			cfg := config.NewGeminiConfig()
			cfg.Auth.Type = authType

			if err := gen.Validate(cfg); err != nil {
				t.Errorf("Auth type '%s' should be valid: %v", authType, err)
			}
		})
	}
}

func TestGeminiGeneratorValidate_InvalidAuthType(t *testing.T) {
	gen := NewGeminiGenerator()
	cfg := config.NewGeminiConfig()
	cfg.Auth.Type = "basic"

	err := gen.Validate(cfg)
	if err == nil {
		t.Error("Invalid auth type should return error")
	}
	if !strings.Contains(err.Error(), "invalid auth type") {
		t.Error("Error should mention invalid auth type")
	}
}

func TestGeminiGeneratorValidate_AllThemes(t *testing.T) {
	gen := NewGeminiGenerator()

	validThemes := []string{"dark", "light", "auto"}
	for _, theme := range validThemes {
		t.Run(theme, func(t *testing.T) {
			cfg := config.NewGeminiConfig()
			cfg.Display.Theme = theme

			if err := gen.Validate(cfg); err != nil {
				t.Errorf("Theme '%s' should be valid: %v", theme, err)
			}
		})
	}
}

func TestGeminiGeneratorValidate_InvalidTheme(t *testing.T) {
	gen := NewGeminiGenerator()
	cfg := config.NewGeminiConfig()
	cfg.Display.Theme = "neon"

	err := gen.Validate(cfg)
	if err == nil {
		t.Error("Invalid theme should return error")
	}
	if !strings.Contains(err.Error(), "invalid theme") {
		t.Error("Error should mention invalid theme")
	}
}

func TestGeminiGeneratorValidate_NegativeMaxFileSize(t *testing.T) {
	gen := NewGeminiGenerator()
	cfg := config.NewGeminiConfig()
	cfg.Behavior.MaxFileSize = -1

	err := gen.Validate(cfg)
	if err == nil {
		t.Error("Negative maxFileSize should return error")
	}
	if !strings.Contains(err.Error(), "maxFileSize must be positive") {
		t.Error("Error should mention maxFileSize must be positive")
	}
}

func TestGeminiGeneratorValidate_ZeroMaxFileSize(t *testing.T) {
	gen := NewGeminiGenerator()
	cfg := config.NewGeminiConfig()
	cfg.Behavior.MaxFileSize = 0

	// Zero is valid (no limit)
	if err := gen.Validate(cfg); err != nil {
		t.Errorf("Zero maxFileSize should be valid: %v", err)
	}
}
