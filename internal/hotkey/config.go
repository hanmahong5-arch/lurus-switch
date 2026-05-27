package hotkey

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const configFileName = "hotkey.json"

// Bindings maps binding-key → shortcut string ("" means disabled).
type Bindings map[string]string

// DefaultBindings returns the factory-default shortcut bindings.
//
// "quickSwitch" shows the provider quick-switch overlay — lets the user
//               flip all tools to a different relay endpoint in one
//               keystroke without opening the full main window.
// "showWindow"  brings the main window to the foreground.
// "show-live"   brings the main window forward AND jumps to the Live
//               Sessions Inspector page in one keystroke.
//
// CommandOrControl is the Electron-style cross-platform modifier: it parses
// to Cmd on macOS and Ctrl on Windows/Linux, so the same default chord
// works everywhere without per-OS forks.
func DefaultBindings() Bindings {
	return Bindings{
		"quickSwitch": "Ctrl+Shift+P",
		"showWindow":  "Ctrl+Shift+W",
		"show-live":   "CommandOrControl+Shift+L",
	}
}

// configPath returns the full path to hotkey.json inside configDir.
func configPath(configDir string) string {
	return filepath.Join(configDir, configFileName)
}

// loadBindings reads Bindings from configDir/hotkey.json.
// Returns DefaultBindings if the file is missing.
// Returns an error (and DefaultBindings) if the file is corrupt.
func loadBindings(configDir string) (Bindings, error) {
	p := configPath(configDir)
	data, err := os.ReadFile(p)
	if err != nil {
		if os.IsNotExist(err) {
			return DefaultBindings(), nil
		}
		return DefaultBindings(), fmt.Errorf("hotkey: read config %q: %w", p, err)
	}

	var b Bindings
	if err := json.Unmarshal(data, &b); err != nil {
		fmt.Fprintf(os.Stderr, "hotkey: corrupt %s, falling back to defaults: %v\n", configFileName, err)
		return DefaultBindings(), fmt.Errorf("hotkey: parse config %q: %w", p, err)
	}

	// Back-fill any missing keys introduced by later versions.
	defaults := DefaultBindings()
	for k, v := range defaults {
		if _, exists := b[k]; !exists {
			b[k] = v
		}
	}

	return b, nil
}

// saveBindings persists Bindings to configDir/hotkey.json.
func saveBindings(configDir string, b Bindings) error {
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("hotkey: create config dir %q: %w", configDir, err)
	}

	data, err := json.MarshalIndent(b, "", "  ")
	if err != nil {
		return fmt.Errorf("hotkey: marshal config: %w", err)
	}

	p := configPath(configDir)
	if err := os.WriteFile(p, data, 0644); err != nil {
		return fmt.Errorf("hotkey: write config %q: %w", p, err)
	}
	return nil
}
