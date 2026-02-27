package generator

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"lurus-switch/internal/config"
)

// PicoClawGenerator generates PicoClaw configuration files
type PicoClawGenerator struct{}

// NewPicoClawGenerator creates a new PicoClaw generator
func NewPicoClawGenerator() *PicoClawGenerator {
	return &PicoClawGenerator{}
}

// Generate creates the config.json file for PicoClaw
func (g *PicoClawGenerator) Generate(cfg *config.PicoClawConfig, outputDir string) (string, error) {
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
func (g *PicoClawGenerator) GenerateString(cfg *config.PicoClawConfig) (string, error) {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal config: %w", err)
	}
	return string(data), nil
}

// Validate validates the PicoClaw configuration
func (g *PicoClawGenerator) Validate(cfg *config.PicoClawConfig) error {
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
