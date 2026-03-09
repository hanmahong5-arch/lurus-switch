package generator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"lurus-switch/internal/config"
)

func TestZeroClawGenerator_GenerateString_ValidConfig(t *testing.T) {
	gen := NewZeroClawGenerator()
	cfg := config.NewZeroClawConfig()
	cfg.Provider.APIKey = "sk-ant-test"

	s, err := gen.GenerateString(cfg)
	if err != nil {
		t.Fatalf("GenerateString failed: %v", err)
	}
	if s == "" {
		t.Error("GenerateString returned empty string")
	}
	if !strings.Contains(s, "[provider]") {
		t.Errorf("TOML output missing [provider] section: %s", s)
	}
	if !strings.Contains(s, "api_key") {
		t.Errorf("TOML output missing api_key field: %s", s)
	}
}

func TestZeroClawGenerator_Generate_WritesTomlFile(t *testing.T) {
	gen := NewZeroClawGenerator()
	cfg := config.NewZeroClawConfig()

	tmpDir := t.TempDir()
	outPath, err := gen.Generate(cfg, tmpDir)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	if filepath.Base(outPath) != "config.toml" {
		t.Errorf("expected output file name config.toml, got %s", filepath.Base(outPath))
	}

	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("failed to read generated file: %v", err)
	}
	if len(data) == 0 {
		t.Error("generated config.toml is empty")
	}
}

func TestZeroClawGenerator_Validate_InvalidPort(t *testing.T) {
	gen := NewZeroClawGenerator()
	cfg := config.NewZeroClawConfig()
	cfg.Gateway.Port = 99999 // out of range

	if err := gen.Validate(cfg); err == nil {
		t.Error("expected Validate to return error for out-of-range port")
	}
}

func TestZeroClawGenerator_Validate_ValidConfig(t *testing.T) {
	gen := NewZeroClawGenerator()
	cfg := config.NewZeroClawConfig()
	cfg.Provider.APIKey = "sk-ant-test"

	if err := gen.Validate(cfg); err != nil {
		t.Errorf("Validate returned unexpected error: %v", err)
	}
}
