//go:build !windows

package sysenv

import (
	"fmt"
	"os"
	"strings"
)

// envExportMarker identifies lines managed by this package in shell profiles.
const envExportMarker = "# lurus-switch:env"

// SetEnvVar adds an export line to the shell profile for the given key/value.
// Saves a rollback entry with the previous value.
func SetEnvVar(key, value string) error {
	if key == "" {
		return fmt.Errorf("environment variable key must not be empty")
	}

	oldValue := os.Getenv(key)

	if err := SaveRollback(RollbackEntry{
		Action:   "env_set",
		OldValue: oldValue,
		NewValue: key + "=" + value,
	}); err != nil {
		return fmt.Errorf("failed to save rollback: %w", err)
	}

	profile, err := shellProfile()
	if err != nil {
		return err
	}

	// Remove any existing line for this key first.
	_ = removeMatchingLines(profile, func(line string) bool {
		return strings.HasPrefix(strings.TrimSpace(line), "export "+key+"=") &&
			strings.Contains(line, envExportMarker)
	})

	line := fmt.Sprintf("export %s=%q %s", key, value, envExportMarker)
	return appendLineToFile(profile, line)
}

// GetEnvVar reads an environment variable from the current process environment.
// On non-Windows systems, user environment variables are not centrally stored;
// the best we can do is check the running process.
func GetEnvVar(key string) (string, error) {
	if key == "" {
		return "", fmt.Errorf("environment variable key must not be empty")
	}
	return os.Getenv(key), nil
}

// DeleteEnvVar removes the export line for the given key from the shell profile.
// Saves a rollback entry with the previous value.
func DeleteEnvVar(key string) error {
	if key == "" {
		return fmt.Errorf("environment variable key must not be empty")
	}

	oldValue := os.Getenv(key)

	if err := SaveRollback(RollbackEntry{
		Action:   "env_delete",
		OldValue: oldValue,
		NewValue: key,
	}); err != nil {
		return fmt.Errorf("failed to save rollback: %w", err)
	}

	profile, err := shellProfile()
	if err != nil {
		return err
	}

	return removeMatchingLines(profile, func(line string) bool {
		return strings.HasPrefix(strings.TrimSpace(line), "export "+key+"=") &&
			strings.Contains(line, envExportMarker)
	})
}
