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

	// Extra holds every top-level field not modelled above (mcp, agent, plugin,
	// instructions, keybinds, …). It is populated on read and merged back on
	// write so that a Read→mutate→Write round-trip never drops the user's
	// unmanaged configuration. Known fields always win on conflict.
	Extra map[string]json.RawMessage `json:"-"`
}

// opencodeKnownFields lists the JSON keys handled by the structured fields of
// OpenCodeConfig. They are stripped from Extra so a round-trip does not emit a
// field twice.
var opencodeKnownFields = []string{"model", "small_model", "provider", "autoupdate", "shell"}

// UnmarshalJSON decodes the known fields and captures any remaining top-level
// keys into Extra for verbatim round-tripping.
func (c *OpenCodeConfig) UnmarshalJSON(data []byte) error {
	type alias OpenCodeConfig // alias drops the custom methods to avoid recursion
	var known alias
	if err := json.Unmarshal(data, &known); err != nil {
		return err
	}
	*c = OpenCodeConfig(known)

	var all map[string]json.RawMessage
	if err := json.Unmarshal(data, &all); err != nil {
		return err
	}
	for _, k := range opencodeKnownFields {
		delete(all, k)
	}
	if len(all) > 0 {
		c.Extra = all
	} else {
		c.Extra = nil
	}
	return nil
}

// MarshalJSON emits the known fields and overlays Extra, preserving unmanaged
// configuration. Known fields take precedence if a key appears in both.
func (c OpenCodeConfig) MarshalJSON() ([]byte, error) {
	type alias OpenCodeConfig // alias drops the custom methods and Extra (json:"-")
	knownData, err := json.Marshal(alias(c))
	if err != nil {
		return nil, err
	}
	if len(c.Extra) == 0 {
		return knownData, nil
	}

	merged := make(map[string]json.RawMessage)
	if err := json.Unmarshal(knownData, &merged); err != nil {
		return nil, err
	}
	for k, v := range c.Extra {
		if _, exists := merged[k]; exists {
			continue // known field wins
		}
		merged[k] = v
	}
	return json.Marshal(merged)
}

// OpenCodeProviderConfig holds per-provider settings that opencode supports.
type OpenCodeProviderConfig struct {
	// APIKey overrides the environment-variable-based key for this provider.
	// Prefer env vars for secrets; this field supports scenarios where the proxy
	// injects a synthetic key (e.g. for routing through the Switch relay).
	APIKey string `json:"api,omitempty"`

	// Name is an optional human-readable label for the provider entry.
	Name string `json:"name,omitempty"`

	// Extra preserves provider sub-fields Switch does not model (options,
	// models, npm, …) so a round-trip never drops them. Without this, mutating
	// one provider's APIKey would silently delete its custom base URL / model
	// list. Known fields win on conflict.
	Extra map[string]json.RawMessage `json:"-"`
}

// opencodeProviderKnownFields lists the JSON keys handled by the structured
// fields of OpenCodeProviderConfig.
var opencodeProviderKnownFields = []string{"api", "name"}

// UnmarshalJSON decodes the known provider fields and captures the rest in Extra.
func (p *OpenCodeProviderConfig) UnmarshalJSON(data []byte) error {
	type alias OpenCodeProviderConfig
	var known alias
	if err := json.Unmarshal(data, &known); err != nil {
		return err
	}
	*p = OpenCodeProviderConfig(known)

	var all map[string]json.RawMessage
	if err := json.Unmarshal(data, &all); err != nil {
		return err
	}
	for _, k := range opencodeProviderKnownFields {
		delete(all, k)
	}
	if len(all) > 0 {
		p.Extra = all
	} else {
		p.Extra = nil
	}
	return nil
}

// MarshalJSON emits the known provider fields and overlays Extra.
func (p OpenCodeProviderConfig) MarshalJSON() ([]byte, error) {
	type alias OpenCodeProviderConfig
	knownData, err := json.Marshal(alias(p))
	if err != nil {
		return nil, err
	}
	if len(p.Extra) == 0 {
		return knownData, nil
	}

	merged := make(map[string]json.RawMessage)
	if err := json.Unmarshal(knownData, &merged); err != nil {
		return nil, err
	}
	for k, v := range p.Extra {
		if _, exists := merged[k]; exists {
			continue // known field wins
		}
		merged[k] = v
	}
	return json.Marshal(merged)
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
