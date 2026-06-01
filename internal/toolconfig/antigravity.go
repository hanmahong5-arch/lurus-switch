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

	"lurus-switch/internal/configapply"
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

	// Extra holds every top-level field not modelled above. It is populated on
	// read and merged back on write so that a Read→mutate→Write round-trip never
	// drops unmanaged configuration. Known fields always win on conflict.
	Extra map[string]json.RawMessage `json:"-"`
}

// antigravityKnownFields lists the JSON keys handled by the structured fields
// of AntigravityConfig. They are stripped from Extra to avoid double-emitting.
var antigravityKnownFields = []string{"apiKey", "apiEndpoint", "model", "general", "proxy"}

// UnmarshalJSON decodes the known fields and captures any remaining top-level
// keys into Extra for verbatim round-tripping.
func (c *AntigravityConfig) UnmarshalJSON(data []byte) error {
	type alias AntigravityConfig // alias drops the custom methods to avoid recursion
	var known alias
	if err := json.Unmarshal(data, &known); err != nil {
		return err
	}
	*c = AntigravityConfig(known)

	var all map[string]json.RawMessage
	if err := json.Unmarshal(data, &all); err != nil {
		return err
	}
	for _, k := range antigravityKnownFields {
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
func (c AntigravityConfig) MarshalJSON() ([]byte, error) {
	type alias AntigravityConfig // alias drops the custom methods and Extra (json:"-")
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

// MergeAntigravityExtra reads the existing on-disk Antigravity config (if any)
// and copies unknown keys into dst.Extra so they survive the subsequent write.
// Known fields in dst are never overwritten — the caller's values win.
// This must be called before WriteAntigravityConfig when the write would otherwise
// clobber pre-existing user configuration.
func MergeAntigravityExtra(dst *AntigravityConfig) error {
	if dst == nil {
		return fmt.Errorf("dst must not be nil")
	}
	existing, err := ReadAntigravityConfig()
	if err != nil {
		// Non-fatal: if we cannot read the existing file, skip the merge.
		return nil
	}
	if len(existing.Extra) == 0 {
		return nil
	}
	if dst.Extra == nil {
		dst.Extra = make(map[string]json.RawMessage)
	}
	for k, v := range existing.Extra {
		if _, exists := dst.Extra[k]; !exists {
			dst.Extra[k] = v
		}
	}
	return nil
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
	if err := configapply.WriteAtomic(configPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write antigravity config %s: %w", configPath, err)
	}
	return nil
}
