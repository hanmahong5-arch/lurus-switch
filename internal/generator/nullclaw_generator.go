package generator

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"lurus-switch/internal/config"
)

// NullClawGenerator generates NullClaw configuration files
type NullClawGenerator struct{}

// NewNullClawGenerator creates a new NullClaw generator
func NewNullClawGenerator() *NullClawGenerator {
	return &NullClawGenerator{}
}

// Generate creates the config.json file for NullClaw
func (g *NullClawGenerator) Generate(cfg *config.NullClawConfig, outputDir string) (string, error) {
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create output directory: %w", err)
	}

	outputPath := filepath.Join(outputDir, "config.json")

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(outputPath, data, 0644); err != nil {
		return "", fmt.Errorf("failed to write config.json: %w", err)
	}

	return outputPath, nil
}

// GenerateString generates the config.json content as a string
func (g *NullClawGenerator) GenerateString(cfg *config.NullClawConfig) (string, error) {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal config: %w", err)
	}
	return string(data), nil
}

// Validate validates the NullClaw configuration
func (g *NullClawGenerator) Validate(cfg *config.NullClawConfig) error {
	if len(cfg.ModelList) == 0 {
		return fmt.Errorf("model_list must contain at least one model")
	}

	for i, m := range cfg.ModelList {
		if m.Name == "" {
			return fmt.Errorf("model_list[%d].name is required", i)
		}
	}

	return nil
}
