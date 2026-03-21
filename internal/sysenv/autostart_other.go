//go:build !windows

package sysenv

import (
	"fmt"
	"os"
	"path/filepath"
	goruntime "runtime"
	"strings"
)

const (
	launchAgentLabel = "cn.lurus.switch"
	xdgAutostartName = "lurus-switch.desktop"
)

// EnableAutostart configures the application to start on login.
// macOS: creates a LaunchAgent plist.
// Linux: creates an XDG autostart .desktop file.
func EnableAutostart(exePath string, args string) error {
	if exePath == "" {
		return fmt.Errorf("executable path must not be empty")
	}

	current, _ := IsAutostartEnabled()
	oldArgs := ""
	if current.Enabled {
		oldArgs = current.Args
	}

	if err := SaveRollback(RollbackEntry{
		Action:   "autostart_enable",
		OldValue: oldArgs,
		NewValue: args,
	}); err != nil {
		return fmt.Errorf("failed to save rollback: %w", err)
	}

	switch goruntime.GOOS {
	case "darwin":
		return enableMacOSAutostart(exePath, args)
	default:
		return enableLinuxAutostart(exePath, args)
	}
}

// DisableAutostart removes the autostart configuration.
func DisableAutostart() error {
	current, _ := IsAutostartEnabled()
	oldArgs := ""
	if current.Enabled {
		oldArgs = current.Args
	}

	if err := SaveRollback(RollbackEntry{
		Action:   "autostart_disable",
		OldValue: oldArgs,
		NewValue: "",
	}); err != nil {
		return fmt.Errorf("failed to save rollback: %w", err)
	}

	switch goruntime.GOOS {
	case "darwin":
		return disableMacOSAutostart()
	default:
		return disableLinuxAutostart()
	}
}

// IsAutostartEnabled checks if an autostart entry exists.
func IsAutostartEnabled() (AutostartConfig, error) {
	switch goruntime.GOOS {
	case "darwin":
		return checkMacOSAutostart()
	default:
		return checkLinuxAutostart()
	}
}

// --- macOS LaunchAgent ---

func launchAgentPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, "Library", "LaunchAgents", launchAgentLabel+".plist"), nil
}

func enableMacOSAutostart(exePath, args string) error {
	path, err := launchAgentPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("failed to create LaunchAgents directory: %w", err)
	}

	// Build program arguments array.
	progArgs := fmt.Sprintf("    <string>%s</string>", exePath)
	if args != "" {
		for _, a := range strings.Fields(args) {
			progArgs += fmt.Sprintf("\n    <string>%s</string>", a)
		}
	}

	plist := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>Label</key>
  <string>%s</string>
  <key>ProgramArguments</key>
  <array>
%s
  </array>
  <key>RunAtLoad</key>
  <true/>
</dict>
</plist>
`, launchAgentLabel, progArgs)

	return os.WriteFile(path, []byte(plist), 0644)
}

func disableMacOSAutostart() error {
	path, err := launchAgentPath()
	if err != nil {
		return err
	}
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove LaunchAgent: %w", err)
	}
	return nil
}

func checkMacOSAutostart() (AutostartConfig, error) {
	path, err := launchAgentPath()
	if err != nil {
		return AutostartConfig{}, nil
	}
	if _, err := os.Stat(path); err != nil {
		return AutostartConfig{}, nil
	}
	// A full plist parse is overkill; presence of the file means enabled.
	return AutostartConfig{Enabled: true}, nil
}

// --- Linux XDG autostart ---

func xdgAutostartPath() (string, error) {
	cfgDir := os.Getenv("XDG_CONFIG_HOME")
	if cfgDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		cfgDir = filepath.Join(home, ".config")
	}
	return filepath.Join(cfgDir, "autostart", xdgAutostartName), nil
}

func enableLinuxAutostart(exePath, args string) error {
	path, err := xdgAutostartPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("failed to create autostart directory: %w", err)
	}

	execLine := exePath
	if args != "" {
		execLine += " " + args
	}

	desktop := fmt.Sprintf(`[Desktop Entry]
Type=Application
Name=Lurus Switch
Exec=%s
X-GNOME-Autostart-enabled=true
`, execLine)

	return os.WriteFile(path, []byte(desktop), 0644)
}

func disableLinuxAutostart() error {
	path, err := xdgAutostartPath()
	if err != nil {
		return err
	}
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove autostart desktop file: %w", err)
	}
	return nil
}

func checkLinuxAutostart() (AutostartConfig, error) {
	path, err := xdgAutostartPath()
	if err != nil {
		return AutostartConfig{}, nil
	}
	if _, err := os.Stat(path); err != nil {
		return AutostartConfig{}, nil
	}
	return AutostartConfig{Enabled: true}, nil
}
