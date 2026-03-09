package generator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"lurus-switch/internal/config"
)

func TestOpenClawGenerator_GenerateString_ValidConfig(t *testing.T) {
	gen := NewOpenClawGenerator()
	cfg := config.NewOpenClawConfig()
	cfg.Provider.APIKey = "sk-ant-test"

	s, err := gen.GenerateString(cfg)
	if err != nil {
		t.Fatalf("GenerateString failed: %v", err)
	}
	if s == "" {
		t.Error("GenerateString returned empty string")
	}
	if !strings.Contains(s, `"gateway"`) {
		t.Errorf("JSON output missing gateway section: %s", s)
	}
	if !strings.Contains(s, `"provider"`) {
		t.Errorf("JSON output missing provider section: %s", s)
	}
}

func TestOpenClawGenerator_Generate_WritesJsonFile(t *testing.T) {
	gen := NewOpenClawGenerator()
	cfg := config.NewOpenClawConfig()

	tmpDir := t.TempDir()
	outPath, err := gen.Generate(cfg, tmpDir)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	if filepath.Base(outPath) != "openclaw.json" {
		t.Errorf("expected output file name openclaw.json, got %s", filepath.Base(outPath))
	}

	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("failed to read generated file: %v", err)
	}
	if len(data) == 0 {
		t.Error("generated openclaw.json is empty")
	}
}

func TestOpenClawGenerator_Validate_InvalidPort(t *testing.T) {
	gen := NewOpenClawGenerator()
	cfg := config.NewOpenClawConfig()
	cfg.Gateway.Port = 0 // invalid

	if err := gen.Validate(cfg); err == nil {
		t.Error("expected Validate to return error for port=0")
	}
}

func TestOpenClawGenerator_Validate_InvalidProviderType(t *testing.T) {
	gen := NewOpenClawGenerator()
	cfg := config.NewOpenClawConfig()
	cfg.Provider.Type = "unknown-provider"

	if err := gen.Validate(cfg); err == nil {
		t.Error("expected Validate to return error for invalid provider type")
	}
}

func TestOpenClawGenerator_Validate_ValidConfig(t *testing.T) {
	gen := NewOpenClawGenerator()
	cfg := config.NewOpenClawConfig()

	if err := gen.Validate(cfg); err != nil {
		t.Errorf("Validate returned unexpected error: %v", err)
	}
}
