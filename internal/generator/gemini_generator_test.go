package generator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"lurus-switch/internal/config"
)

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
