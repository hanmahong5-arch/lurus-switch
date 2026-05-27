// Package toolconfig manages on-disk configuration for CLI tools managed by Switch.
// This file handles opencode (https://github.com/sst/opencode), a Go-based terminal
// coding assistant that uses XDG Base Directory paths for its configuration.
//
// Configuration path resolution follows xdg-basedir@6 behaviour (the npm package
// opencode bundles at runtime):
//
//   - $XDG_CONFIG_HOME/opencode          — all platforms when XDG_CONFIG_HOME is set
//   - %LOCALAPPDATA%\xdg.config\opencode — Windows default (xdg-basedir@6 convention)
//   - $HOME/.config/opencode             — Unix/macOS/Windows fallback
//
// TODO: verify Windows path against the official opencode documentation if a
// platform-specific override is ever published by sst/opencode.
package toolconfig

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

const (
	// ToolOpenCode is the canonical tool name used throughout Switch.
	ToolOpenCode = "opencode"

	// OpenCodeBinaryName is the executable placed on PATH by opencode's installer.
	OpenCodeBinaryName = "opencode"

	// OpenCodeConfigFilename is the primary config file name inside the config directory.
	// opencode also accepts opencode.jsonc; the plain .json form is the writable target.
	OpenCodeConfigFilename = "opencode.json"
)

// opencodeConfigDir returns the OS-appropriate config directory for opencode.
//
// Resolution order:
//  1. $XDG_CONFIG_HOME/opencode   (set explicitly by the user on any OS)
//  2. %LOCALAPPDATA%\xdg.config\opencode  (Windows — matches xdg-basedir@6)
//  3. $HOME/.config/opencode      (Unix / macOS / Windows fallback)
//
// TODO: verify Windows path against sst/opencode official docs if they publish
// a platform-specific config path that differs from the XDG default.
func opencodeConfigDir() string {
	// Honour explicit XDG override on all platforms.
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "opencode")
	}

	// Windows: xdg-basedir@6 resolves to %LOCALAPPDATA%\xdg.config
	if runtime.GOOS == "windows" {
		if localApp := os.Getenv("LOCALAPPDATA"); localApp != "" {
			return filepath.Join(localApp, "xdg.config", "opencode")
		}
	}

	// Unix / macOS / Windows fallback: ~/.config/opencode
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "opencode")
}

// OpenCodeConfig represents the commonly-used fields in opencode.json.
//
// opencode supports a much richer schema (agents, MCP servers, plugins, etc.);
// only the fields relevant for Switch-managed proxy routing are structured here.
// All other fields are preserved verbatim via the Extra map when round-tripping.
//
// TODO: extend with mcp, agent, and plugin fields once Switch requires them.
type OpenCodeConfig struct {
	// Model is the default model in "provider/model" format, e.g. "anthropic/claude-sonnet-4".
	Model string `json:"model,omitempty"`

	// SmallModel is used for lightweight tasks such as title generation.
	SmallModel string `json:"small_model,omitempty"`

	// Provider contains per-provider settings (API key overrides, base URLs).
	// Keys are provider IDs (e.g. "anthropic", "openai").
	Provider map[string]OpenCodeProviderConfig `json:"provider,omitempty"`

	// AutoUpdate controls automatic version updates.
	// Valid values: true, false, or the string "notify".
	AutoUpdate any `json:"autoupdate,omitempty"`

	// Shell overrides the default shell used for terminal and bash tool execution.
	Shell string `json:"shell,omitempty"`
}

// OpenCodeProviderConfig holds per-provider settings that opencode supports.
type OpenCodeProviderConfig struct {
	// APIKey overrides the environment-variable-based key for this provider.
	// Prefer env vars for secrets; this field supports scenarios where the proxy
	// injects a synthetic key (e.g. for routing through the Switch relay).
	APIKey string `json:"api,omitempty"`

	// Name is an optional human-readable label for the provider entry.
	Name string `json:"name,omitempty"`
}

// ReadOpenCodeConfig reads and parses the opencode.json config file.
// Returns a zero-value OpenCodeConfig if the file does not exist.
func ReadOpenCodeConfig() (*OpenCodeConfig, error) {
	configPath := filepath.Join(opencodeConfigDir(), OpenCodeConfigFilename)
	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return &OpenCodeConfig{}, nil
		}
		return nil, fmt.Errorf("failed to read opencode config %s: %w", configPath, err)
	}

	var cfg OpenCodeConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse opencode config %s: %w", configPath, err)
	}
	return &cfg, nil
}

// WriteOpenCodeConfig marshals cfg to JSON and writes it to the opencode config
// file, creating the config directory if it does not exist.
func WriteOpenCodeConfig(cfg *OpenCodeConfig) error {
	if cfg == nil {
		return fmt.Errorf("cfg must not be nil")
	}

	configDir := opencodeConfigDir()
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create opencode config directory %s: %w", configDir, err)
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal opencode config: %w", err)
	}

	configPath := filepath.Join(configDir, OpenCodeConfigFilename)
	if err := os.WriteFile(configPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write opencode config %s: %w", configPath, err)
	}
	return nil
}
