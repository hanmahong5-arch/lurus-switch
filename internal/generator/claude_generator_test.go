package generator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"lurus-switch/internal/config"
)

func TestClaudeGeneratorGenerateString(t *testing.T) {
	gen := NewClaudeGenerator()
	cfg := config.NewClaudeConfig()
	cfg.Model = "claude-sonnet-4-20250514"
	cfg.CustomInstructions = "Be helpful"

	content, err := gen.GenerateString(cfg)
	if err != nil {
		t.Fatalf("Failed to generate string: %v", err)
	}

	if !strings.Contains(content, "claude-sonnet-4-20250514") {
		t.Error("Generated content should contain model name")
	}

	if !strings.Contains(content, "Be helpful") {
		t.Error("Generated content should contain custom instructions")
	}
}

func TestClaudeGeneratorGenerate(t *testing.T) {
	gen := NewClaudeGenerator()
	cfg := config.NewClaudeConfig()

	tmpDir, err := os.MkdirTemp("", "claude-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	outputPath, err := gen.Generate(cfg, tmpDir)
	if err != nil {
		t.Fatalf("Failed to generate: %v", err)
	}

	if filepath.Base(outputPath) != "settings.json" {
		t.Errorf("Expected settings.json, got %s", filepath.Base(outputPath))
	}

	// Verify file exists
	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		t.Error("Output file was not created")
	}
}

func TestClaudeGeneratorValidate(t *testing.T) {
	gen := NewClaudeGenerator()

	// Valid config
	validCfg := config.NewClaudeConfig()
	if err := gen.Validate(validCfg); err != nil {
		t.Errorf("Valid config should not return error: %v", err)
	}

	// Invalid config - empty model
	invalidCfg := &config.ClaudeConfig{}
	if err := gen.Validate(invalidCfg); err == nil {
		t.Error("Empty model should return error")
	}

	// Invalid config - negative max tokens
	invalidCfg2 := config.NewClaudeConfig()
	invalidCfg2.MaxTokens = -1
	if err := gen.Validate(invalidCfg2); err == nil {
		t.Error("Negative maxTokens should return error")
	}
}
