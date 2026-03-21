//go:build windows

package sysenv

import (
	"fmt"
	"strings"

	"golang.org/x/sys/windows/registry"
)

const (
	autostartRegistryKey = `Software\Microsoft\Windows\CurrentVersion\Run`
	autostartValueName   = "LurusSwitch"
)

// EnableAutostart creates a registry Run key so the application starts on login.
// exePath is the full path to the executable. args are additional CLI flags.
func EnableAutostart(exePath string, args string) error {
	if exePath == "" {
		return fmt.Errorf("executable path must not be empty")
	}

	// Read current state for rollback.
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

	k, err := registry.OpenKey(registry.CURRENT_USER, autostartRegistryKey, registry.SET_VALUE)
	if err != nil {
		return fmt.Errorf("failed to open HKCU\\%s: %w", autostartRegistryKey, err)
	}
	defer k.Close()

	// Build command line: quoted exe path + optional args.
	cmdLine := fmt.Sprintf(`"%s"`, exePath)
	if args != "" {
		cmdLine += " " + args
	}

	if err := k.SetStringValue(autostartValueName, cmdLine); err != nil {
		return fmt.Errorf("failed to set autostart value: %w", err)
	}
	return nil
}

// DisableAutostart removes the registry Run key.
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

	k, err := registry.OpenKey(registry.CURRENT_USER, autostartRegistryKey, registry.SET_VALUE)
	if err != nil {
		return fmt.Errorf("failed to open HKCU\\%s: %w", autostartRegistryKey, err)
	}
	defer k.Close()

	if err := k.DeleteValue(autostartValueName); err != nil {
		if err == registry.ErrNotExist {
			return nil // already absent
		}
		return fmt.Errorf("failed to delete autostart value: %w", err)
	}
	return nil
}

// IsAutostartEnabled checks if the autostart registry entry exists.
func IsAutostartEnabled() (AutostartConfig, error) {
	k, err := registry.OpenKey(registry.CURRENT_USER, autostartRegistryKey, registry.QUERY_VALUE)
	if err != nil {
		return AutostartConfig{}, nil
	}
	defer k.Close()

	val, _, err := k.GetStringValue(autostartValueName)
	if err != nil {
		return AutostartConfig{}, nil
	}

	// Parse args from the command line value.
	// Format: "C:\path\to\exe.exe" --flag1 --flag2
	args := ""
	if val != "" {
		// Find closing quote of the exe path.
		if strings.HasPrefix(val, `"`) {
			if idx := strings.Index(val[1:], `"`); idx >= 0 {
				rest := strings.TrimSpace(val[idx+2:])
				if rest != "" {
					args = rest
				}
			}
		}
	}

	return AutostartConfig{
		Enabled: true,
		Args:    args,
	}, nil
}
