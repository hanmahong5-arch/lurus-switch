//go:build !windows

package sysenv

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const pathSeparator = ":"

// shellProfile returns the path to the user's primary shell profile file.
// Prefers .zshrc on macOS, .bashrc on Linux, falling back to .profile.
func shellProfile() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to determine home directory: %w", err)
	}

	candidates := []string{
		filepath.Join(home, ".zshrc"),
		filepath.Join(home, ".bashrc"),
		filepath.Join(home, ".profile"),
	}

	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			return c, nil
		}
	}
	// Default to .profile if none exist.
	return filepath.Join(home, ".profile"), nil
}

// pathExportMarker is used to identify lines managed by this package.
const pathExportMarker = "# lurus-switch:path"

// GetUserPath returns the current user's PATH parsed into entries.
func GetUserPath() ([]PathEntry, error) {
	raw := os.Getenv("PATH")
	return ParsePathEntries(raw, pathSeparator), nil
}

// AddToUserPath appends a directory to the user's PATH via shell profile.
func AddToUserPath(dir string) error {
	if dir == "" {
		return fmt.Errorf("directory must not be empty")
	}

	// Check if already in current PATH.
	for _, existing := range strings.Split(os.Getenv("PATH"), pathSeparator) {
		if strings.TrimRight(existing, "/") == strings.TrimRight(dir, "/") {
			return nil
		}
	}

	oldPath := os.Getenv("PATH")

	if err := SaveRollback(RollbackEntry{
		Action:   "path_add",
		OldValue: oldPath,
		NewValue: dir,
	}); err != nil {
		return fmt.Errorf("failed to save rollback: %w", err)
	}

	profile, err := shellProfile()
	if err != nil {
		return err
	}

	line := fmt.Sprintf("export PATH=\"$PATH:%s\" %s", dir, pathExportMarker)
	return appendLineToFile(profile, line)
}

// RemoveFromUserPath removes a directory from the user's PATH in the shell profile.
func RemoveFromUserPath(dir string) error {
	if dir == "" {
		return fmt.Errorf("directory must not be empty")
	}

	if err := SaveRollback(RollbackEntry{
		Action:   "path_remove",
		OldValue: dir,
		NewValue: os.Getenv("PATH"),
	}); err != nil {
		return fmt.Errorf("failed to save rollback: %w", err)
	}

	profile, err := shellProfile()
	if err != nil {
		return err
	}

	// Remove lines that export this exact directory with our marker.
	return removeMatchingLines(profile, func(line string) bool {
		return strings.Contains(line, dir) && strings.Contains(line, pathExportMarker)
	})
}

// appendLineToFile appends a line to a file, creating it if necessary.
func appendLineToFile(path, line string) error {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open %s: %w", path, err)
	}
	defer f.Close()

	if _, err := fmt.Fprintf(f, "\n%s\n", line); err != nil {
		return fmt.Errorf("failed to write to %s: %w", path, err)
	}
	return nil
}

// removeMatchingLines removes all lines from a file for which match() returns true.
func removeMatchingLines(path string, match func(string) bool) error {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("failed to read %s: %w", path, err)
	}

	lines := strings.Split(string(data), "\n")
	var kept []string
	for _, l := range lines {
		if !match(l) {
			kept = append(kept, l)
		}
	}

	return os.WriteFile(path, []byte(strings.Join(kept, "\n")), 0644)
}
