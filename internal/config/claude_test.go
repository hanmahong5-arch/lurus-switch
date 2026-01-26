package config

import (
	"encoding/json"
	"testing"
)

func TestNewClaudeConfig(t *testing.T) {
	cfg := NewClaudeConfig()

	if cfg.Model == "" {
		t.Error("Model should have default value")
	}

	if cfg.MaxTokens <= 0 {
		t.Error("MaxTokens should have positive default value")
	}

	if cfg.Permissions.AllowBash != true {
		t.Error("AllowBash should be true by default")
	}

	if cfg.Sandbox.Type != "none" {
		t.Error("Sandbox type should be 'none' by default")
	}
}

func TestClaudeConfigJSON(t *testing.T) {
	cfg := NewClaudeConfig()
	cfg.CustomInstructions = "Test instructions"

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal config: %v", err)
	}

	var decoded ClaudeConfig
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal config: %v", err)
	}

	if decoded.CustomInstructions != cfg.CustomInstructions {
		t.Error("CustomInstructions mismatch after round-trip")
	}
}
