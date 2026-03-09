package generator

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"lurus-switch/internal/config"
)

// OpenClawGenerator generates OpenClaw configuration files (JSON)
type OpenClawGenerator struct{}

// NewOpenClawGenerator creates a new OpenClaw generator
func NewOpenClawGenerator() *OpenClawGenerator {
	return &OpenClawGenerator{}
}

// Generate writes openclaw.json to the given output directory
func (g *OpenClawGenerator) Generate(cfg *config.OpenClawConfig, outputDir string) (string, error) {
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create output directory: %w", err)
	}

	outputPath := filepath.Join(outputDir, "openclaw.json")
	content, err := g.GenerateString(cfg)
	if err != nil {
		return "", err
	}

	if err := os.WriteFile(outputPath, []byte(content), 0644); err != nil {
		return "", fmt.Errorf("failed to write openclaw.json: %w", err)
	}

	return outputPath, nil
}

// GenerateString returns the JSON config as a string
func (g *OpenClawGenerator) GenerateString(cfg *config.OpenClawConfig) (string, error) {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal OpenClaw config: %w", err)
	}
	return string(data), nil
}

// Validate checks that the OpenClaw configuration is valid
func (g *OpenClawGenerator) Validate(cfg *config.OpenClawConfig) error {
	if cfg.Gateway.Port < 1 || cfg.Gateway.Port > 65535 {
		return fmt.Errorf("gateway.port must be in range 1–65535, got %d", cfg.Gateway.Port)
	}

	validProviders := []string{"anthropic", "openai", "custom", ""}
	found := false
	for _, v := range validProviders {
		if cfg.Provider.Type == v {
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("provider.type %q is not a recognised value (expected: anthropic, openai, custom)", cfg.Provider.Type)
	}

	return nil
}
