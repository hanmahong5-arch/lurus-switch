package generator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"lurus-switch/internal/config"
)

func TestCodexGeneratorGenerateString(t *testing.T) {
	gen := NewCodexGenerator()
	cfg := config.NewCodexConfig()
	cfg.Model = "o4-mini"
	cfg.ApprovalMode = "suggest"

	content, err := gen.GenerateString(cfg)
	if err != nil {
		t.Fatalf("Failed to generate string: %v", err)
	}

	if !strings.Contains(content, "o4-mini") {
		t.Error("Generated content should contain model name")
	}

	if !strings.Contains(content, "suggest") {
		t.Error("Generated content should contain approval mode")
	}
}

func TestCodexGeneratorGenerate(t *testing.T) {
	gen := NewCodexGenerator()
	cfg := config.NewCodexConfig()

	tmpDir, err := os.MkdirTemp("", "codex-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	outputPath, err := gen.Generate(cfg, tmpDir)
	if err != nil {
		t.Fatalf("Failed to generate: %v", err)
	}

	if filepath.Base(outputPath) != "config.toml" {
		t.Errorf("Expected config.toml, got %s", filepath.Base(outputPath))
	}

	// Verify file exists
	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		t.Error("Output file was not created")
	}
}

func TestCodexGeneratorValidate(t *testing.T) {
	gen := NewCodexGenerator()

	// Valid config
	validCfg := config.NewCodexConfig()
	if err := gen.Validate(validCfg); err != nil {
		t.Errorf("Valid config should not return error: %v", err)
	}

	// Invalid approval mode
	invalidCfg := config.NewCodexConfig()
	invalidCfg.ApprovalMode = "invalid"
	if err := gen.Validate(invalidCfg); err == nil {
		t.Error("Invalid approval mode should return error")
	}
}
