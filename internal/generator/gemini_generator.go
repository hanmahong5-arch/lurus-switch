package generator

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"lurus-switch/internal/config"
)

// GeminiGenerator generates Gemini CLI configuration files
type GeminiGenerator struct{}

// NewGeminiGenerator creates a new Gemini generator
func NewGeminiGenerator() *GeminiGenerator {
	return &GeminiGenerator{}
}

// Generate creates the GEMINI.md file for Gemini CLI
func (g *GeminiGenerator) Generate(cfg *config.GeminiConfig, outputDir string) (string, error) {
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create output directory: %w", err)
	}

	outputPath := filepath.Join(outputDir, "GEMINI.md")
	content := g.GenerateMarkdown(cfg)

	if err := os.WriteFile(outputPath, []byte(content), 0644); err != nil {
		return "", fmt.Errorf("failed to write GEMINI.md: %w", err)
	}

	return outputPath, nil
}

// GenerateMarkdown generates the GEMINI.md content
func (g *GeminiGenerator) GenerateMarkdown(cfg *config.GeminiConfig) string {
	var sb strings.Builder

	sb.WriteString("# GEMINI.md\n\n")
	sb.WriteString("This file provides guidance to Gemini CLI when working with this repository.\n\n")

	// Project description
	if cfg.Instructions.ProjectDescription != "" {
		sb.WriteString("## Project Description\n\n")
		sb.WriteString(cfg.Instructions.ProjectDescription)
		sb.WriteString("\n\n")
	}

	// Tech stack
	if cfg.Instructions.TechStack != "" {
		sb.WriteString("## Tech Stack\n\n")
		sb.WriteString(cfg.Instructions.TechStack)
		sb.WriteString("\n\n")
	}

	// Code style guidelines
	if cfg.Instructions.CodeStyle != "" {
		sb.WriteString("## Code Style\n\n")
		sb.WriteString(cfg.Instructions.CodeStyle)
		sb.WriteString("\n\n")
	}

	// Custom rules
	if len(cfg.Instructions.CustomRules) > 0 {
		sb.WriteString("## Rules\n\n")
		for _, rule := range cfg.Instructions.CustomRules {
			sb.WriteString("- ")
			sb.WriteString(rule)
			sb.WriteString("\n")
		}
		sb.WriteString("\n")
	}

	// File structure
	if cfg.Instructions.FileStructure != "" {
		sb.WriteString("## File Structure\n\n")
		sb.WriteString("```\n")
		sb.WriteString(cfg.Instructions.FileStructure)
		sb.WriteString("\n```\n\n")
	}

	// Testing guidelines
	if cfg.Instructions.TestingGuidelines != "" {
		sb.WriteString("## Testing\n\n")
		sb.WriteString(cfg.Instructions.TestingGuidelines)
		sb.WriteString("\n\n")
	}

	return sb.String()
}

// GenerateConfigJSON generates a JSON config file for Gemini CLI settings
func (g *GeminiGenerator) GenerateConfigJSON(cfg *config.GeminiConfig, outputDir string) (string, error) {
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create output directory: %w", err)
	}

	// Create a settings struct without sensitive data for the JSON config
	settings := map[string]interface{}{
		"model": cfg.Model,
		"auth": map[string]interface{}{
			"type": cfg.Auth.Type,
		},
		"behavior": map[string]interface{}{
			"sandbox":     cfg.Behavior.Sandbox,
			"yoloMode":    cfg.Behavior.YoloMode,
			"maxFileSize": cfg.Behavior.MaxFileSize,
		},
		"display": map[string]interface{}{
			"theme":           cfg.Display.Theme,
			"syntaxHighlight": cfg.Display.SyntaxHighlight,
			"markdownRender":  cfg.Display.MarkdownRender,
		},
	}

	if len(cfg.Behavior.AutoApprove) > 0 {
		settings["behavior"].(map[string]interface{})["autoApprove"] = cfg.Behavior.AutoApprove
	}

	if len(cfg.Behavior.AllowedExtensions) > 0 {
		settings["behavior"].(map[string]interface{})["allowedExtensions"] = cfg.Behavior.AllowedExtensions
	}

	outputPath := filepath.Join(outputDir, "gemini-settings.json")
	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal settings: %w", err)
	}

	if err := os.WriteFile(outputPath, data, 0644); err != nil {
		return "", fmt.Errorf("failed to write gemini-settings.json: %w", err)
	}

	return outputPath, nil
}

// GenerateAll generates all configuration files for Gemini CLI
func (g *GeminiGenerator) GenerateAll(cfg *config.GeminiConfig, outputDir string) ([]string, error) {
	var files []string

	// Generate GEMINI.md
	mdPath, err := g.Generate(cfg, outputDir)
	if err != nil {
		return nil, err
	}
	files = append(files, mdPath)

	// Generate JSON settings
	jsonPath, err := g.GenerateConfigJSON(cfg, outputDir)
	if err != nil {
		return files, err
	}
	files = append(files, jsonPath)

	return files, nil
}

// Validate validates the Gemini configuration
func (g *GeminiGenerator) Validate(cfg *config.GeminiConfig) error {
	if cfg.Model == "" {
		return fmt.Errorf("model is required")
	}

	// Validate auth type
	validAuthTypes := map[string]bool{"api_key": true, "oauth": true, "adc": true}
	if !validAuthTypes[cfg.Auth.Type] {
		return fmt.Errorf("invalid auth type: %s (must be 'api_key', 'oauth', or 'adc')", cfg.Auth.Type)
	}

	// Validate display theme
	validThemes := map[string]bool{"dark": true, "light": true, "auto": true}
	if !validThemes[cfg.Display.Theme] {
		return fmt.Errorf("invalid theme: %s (must be 'dark', 'light', or 'auto')", cfg.Display.Theme)
	}

	// Validate max file size
	if cfg.Behavior.MaxFileSize < 0 {
		return fmt.Errorf("maxFileSize must be positive")
	}

	return nil
}
