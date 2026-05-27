// Package toolconfig manages on-disk configuration for CLI tools managed by Switch.
// This file handles the Antigravity CLI (binary: agy), the successor to Gemini CLI,
// announced at Google I/O 2026-05-19. Gemini CLI EOL: 2026-06-18.
//
// TODO: config path %LOCALAPPDATA%\Antigravity\config.json is inferred from typical
// Google CLI conventions. Verify against official Antigravity documentation once
// published at https://developers.google.com/antigravity.
package toolconfig

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

const (
	// ToolAntigravity is the canonical tool name used throughout Switch.
	ToolAntigravity = "antigravity"

	// AntigravityBinaryName is the executable name for the Antigravity CLI.
	AntigravityBinaryName = "agy"

	// AntigravityConfigFilename is the config file name inside the config directory.
	// TODO: verify against official docs once published.
	AntigravityConfigFilename = "config.json"
)

// antigravityConfigDir returns the OS-appropriate config directory for Antigravity CLI.
//
// Windows: %LOCALAPPDATA%\Antigravity
// macOS:   ~/Library/Application Support/Antigravity
// Linux:   $XDG_CONFIG_HOME/antigravity (fallback: ~/.config/antigravity)
//
// TODO: verify path against official Antigravity documentation once published.
func antigravityConfigDir() string {
	switch runtime.GOOS {
	case "windows":
		base := os.Getenv("LOCALAPPDATA")
		if base == "" {
			home, _ := os.UserHomeDir()
			base = filepath.Join(home, "AppData", "Local")
		}
		return filepath.Join(base, "Antigravity")
	case "darwin":
		home, _ := os.UserHomeDir()
		return filepath.Join(home, "Library", "Application Support", "Antigravity")
	default:
		xdgCfg := os.Getenv("XDG_CONFIG_HOME")
		if xdgCfg == "" {
			home, _ := os.UserHomeDir()
			xdgCfg = filepath.Join(home, ".config")
		}
		return filepath.Join(xdgCfg, "antigravity")
	}
}

// AntigravityConfig represents the structured contents of Antigravity CLI's config.json.
// Fields are based on Gemini CLI's schema plus Antigravity-specific additions.
//
// TODO: update struct once official schema is published.
type AntigravityConfig struct {
	// APIKey is the Gemini API key used for authentication.
	APIKey string `json:"apiKey,omitempty"`

	// APIEndpoint overrides the default API endpoint (e.g. for proxy routing).
	APIEndpoint string `json:"apiEndpoint,omitempty"`

	// Model specifies the default model to use.
	Model AntigravityModelConfig `json:"model,omitempty"`

	// General holds general behavior settings.
	General AntigravityGeneralConfig `json:"general,omitempty"`

	// Proxy holds HTTP proxy configuration.
	Proxy string `json:"proxy,omitempty"`
}

// AntigravityModelConfig holds model selection settings.
type AntigravityModelConfig struct {
	Name string `json:"name,omitempty"`
}

// AntigravityGeneralConfig holds general behavior settings.
type AntigravityGeneralConfig struct {
	DefaultApprovalMode string `json:"defaultApprovalMode,omitempty"`
}

// ReadAntigravityConfig reads and parses the Antigravity CLI config file.
// Returns a zero-value AntigravityConfig if the file does not exist.
func ReadAntigravityConfig() (*AntigravityConfig, error) {
	configPath := filepath.Join(antigravityConfigDir(), AntigravityConfigFilename)
	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return &AntigravityConfig{}, nil
		}
		return nil, fmt.Errorf("failed to read antigravity config %s: %w", configPath, err)
	}

	var cfg AntigravityConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse antigravity config %s: %w", configPath, err)
	}
	return &cfg, nil
}

// WriteAntigravityConfig marshals cfg to JSON and writes it to the Antigravity config file,
// creating the config directory if it does not exist.
func WriteAntigravityConfig(cfg *AntigravityConfig) error {
	if cfg == nil {
		return fmt.Errorf("cfg must not be nil")
	}

	configDir := antigravityConfigDir()
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create antigravity config directory %s: %w", configDir, err)
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal antigravity config: %w", err)
	}

	configPath := filepath.Join(configDir, AntigravityConfigFilename)
	if err := os.WriteFile(configPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write antigravity config %s: %w", configPath, err)
	}
	return nil
}
