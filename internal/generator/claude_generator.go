package generator

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"lurus-switch/internal/config"
)

// ClaudeGenerator generates Claude Code configuration files
type ClaudeGenerator struct{}

// NewClaudeGenerator creates a new Claude generator
func NewClaudeGenerator() *ClaudeGenerator {
	return &ClaudeGenerator{}
}

// Generate creates the settings.json file for Claude Code
func (g *ClaudeGenerator) Generate(cfg *config.ClaudeConfig, outputDir string) (string, error) {
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create output directory: %w", err)
	}

	outputPath := filepath.Join(outputDir, "settings.json")

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(outputPath, data, 0644); err != nil {
		return "", fmt.Errorf("failed to write settings.json: %w", err)
	}

	return outputPath, nil
}

// GenerateString generates the settings.json content as a string
func (g *ClaudeGenerator) GenerateString(cfg *config.ClaudeConfig) (string, error) {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal config: %w", err)
	}
	return string(data), nil
}

// GenerateCLAUDEMD generates the CLAUDE.md file content
func (g *ClaudeGenerator) GenerateCLAUDEMD(cfg *config.ClaudeConfig) string {
	md := "# CLAUDE.md\n\n"
	md += "This file provides guidance to Claude Code when working with this repository.\n\n"

	if cfg.CustomInstructions != "" {
		md += "## Instructions\n\n"
		md += cfg.CustomInstructions + "\n\n"
	}

	if cfg.Model != "" {
		md += "## Preferred Model\n\n"
		md += fmt.Sprintf("Use the `%s` model for this project.\n\n", cfg.Model)
	}

	if len(cfg.Permissions.TrustedDirectories) > 0 {
		md += "## Trusted Directories\n\n"
		for _, dir := range cfg.Permissions.TrustedDirectories {
			md += fmt.Sprintf("- `%s`\n", dir)
		}
		md += "\n"
	}

	return md
}

// Validate validates the Claude configuration
func (g *ClaudeGenerator) Validate(cfg *config.ClaudeConfig) error {
	if cfg.Model == "" {
		return fmt.Errorf("model is required")
	}

	// Validate API key format if provided
	if cfg.APIKey != "" && len(cfg.APIKey) < 10 {
		return fmt.Errorf("invalid API key format")
	}

	// Validate max tokens
	if cfg.MaxTokens < 0 {
		return fmt.Errorf("maxTokens must be positive")
	}

	// Validate sandbox settings
	if cfg.Sandbox.Enabled {
		validTypes := map[string]bool{"docker": true, "wsl": true, "none": true}
		if !validTypes[cfg.Sandbox.Type] {
			return fmt.Errorf("invalid sandbox type: %s", cfg.Sandbox.Type)
		}
	}

	return nil
}
