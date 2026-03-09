package generator

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"

	"lurus-switch/internal/config"
)

// ZeroClawGenerator generates ZeroClaw configuration files (TOML)
type ZeroClawGenerator struct{}

// NewZeroClawGenerator creates a new ZeroClaw generator
func NewZeroClawGenerator() *ZeroClawGenerator {
	return &ZeroClawGenerator{}
}

// Generate writes config.toml to the given output directory
func (g *ZeroClawGenerator) Generate(cfg *config.ZeroClawConfig, outputDir string) (string, error) {
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create output directory: %w", err)
	}

	outputPath := filepath.Join(outputDir, "config.toml")
	content, err := g.GenerateString(cfg)
	if err != nil {
		return "", err
	}

	if err := os.WriteFile(outputPath, []byte(content), 0644); err != nil {
		return "", fmt.Errorf("failed to write config.toml: %w", err)
	}

	return outputPath, nil
}

// GenerateString returns the TOML config as a string
func (g *ZeroClawGenerator) GenerateString(cfg *config.ZeroClawConfig) (string, error) {
	var buf strings.Builder
	enc := toml.NewEncoder(&buf)
	if err := enc.Encode(cfg); err != nil {
		return "", fmt.Errorf("failed to encode ZeroClaw config as TOML: %w", err)
	}
	return buf.String(), nil
}

// Validate checks that the ZeroClaw configuration is valid
func (g *ZeroClawGenerator) Validate(cfg *config.ZeroClawConfig) error {
	if cfg.Provider.APIKey == "" {
		// Advisory only — ZeroClaw can be configured with env vars
		_ = "api_key is empty; ensure ANTHROPIC_API_KEY or equivalent is set"
	}
	if cfg.Gateway.Port < 1 || cfg.Gateway.Port > 65535 {
		return fmt.Errorf("gateway.port must be in range 1–65535, got %d", cfg.Gateway.Port)
	}
	return nil
}
