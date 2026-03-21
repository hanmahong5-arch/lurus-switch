//go:build windows

package sysenv

import (
	"fmt"

	"golang.org/x/sys/windows/registry"
)

// SetEnvVar sets a user environment variable in the Windows registry.
// Saves a rollback entry with the previous value before modifying.
func SetEnvVar(key, value string) error {
	if key == "" {
		return fmt.Errorf("environment variable key must not be empty")
	}

	// Read old value for rollback.
	oldValue, _ := GetEnvVar(key)

	if err := SaveRollback(RollbackEntry{
		Action:   "env_set",
		OldValue: oldValue,
		NewValue: key + "=" + value,
	}); err != nil {
		return fmt.Errorf("failed to save rollback: %w", err)
	}

	k, err := registry.OpenKey(registry.CURRENT_USER, envRegistryKey, registry.SET_VALUE)
	if err != nil {
		return fmt.Errorf("failed to open HKCU\\%s for writing: %w", envRegistryKey, err)
	}
	defer k.Close()

	if err := k.SetStringValue(key, value); err != nil {
		return fmt.Errorf("failed to set HKCU\\%s\\%s: %w", envRegistryKey, key, err)
	}
	broadcastSettingChange()
	return nil
}

// GetEnvVar reads a user environment variable from the Windows registry.
func GetEnvVar(key string) (string, error) {
	if key == "" {
		return "", fmt.Errorf("environment variable key must not be empty")
	}

	k, err := registry.OpenKey(registry.CURRENT_USER, envRegistryKey, registry.QUERY_VALUE)
	if err != nil {
		return "", fmt.Errorf("failed to open HKCU\\%s: %w", envRegistryKey, err)
	}
	defer k.Close()

	val, _, err := k.GetStringValue(key)
	if err != nil {
		if err == registry.ErrNotExist {
			return "", nil
		}
		return "", fmt.Errorf("failed to read HKCU\\%s\\%s: %w", envRegistryKey, key, err)
	}
	return val, nil
}

// DeleteEnvVar removes a user environment variable from the Windows registry.
// Saves a rollback entry with the previous value before deleting.
func DeleteEnvVar(key string) error {
	if key == "" {
		return fmt.Errorf("environment variable key must not be empty")
	}

	oldValue, _ := GetEnvVar(key)

	if err := SaveRollback(RollbackEntry{
		Action:   "env_delete",
		OldValue: oldValue,
		NewValue: key,
	}); err != nil {
		return fmt.Errorf("failed to save rollback: %w", err)
	}

	k, err := registry.OpenKey(registry.CURRENT_USER, envRegistryKey, registry.SET_VALUE)
	if err != nil {
		return fmt.Errorf("failed to open HKCU\\%s for writing: %w", envRegistryKey, err)
	}
	defer k.Close()

	if err := k.DeleteValue(key); err != nil {
		if err == registry.ErrNotExist {
			return nil // already absent
		}
		return fmt.Errorf("failed to delete HKCU\\%s\\%s: %w", envRegistryKey, key, err)
	}
	broadcastSettingChange()
	return nil
}
