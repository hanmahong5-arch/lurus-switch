package sysenv

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
)

// rollbackMaxAge defines the maximum age for rollback entries before cleanup.
const rollbackMaxAge = 30 * 24 * time.Hour

// rollbackDir returns the directory for storing rollback entries.
func rollbackDir() (string, error) {
	cfgDir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("failed to locate user config directory: %w", err)
	}
	return filepath.Join(cfgDir, "lurus-switch", "rollback"), nil
}

// RollbackDirPath returns the rollback directory path (for testing and diagnostics).
func RollbackDirPath() (string, error) {
	return rollbackDir()
}

// SaveRollback persists a rollback entry as a JSON file.
func SaveRollback(entry RollbackEntry) error {
	dir, err := rollbackDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create rollback directory %s: %w", dir, err)
	}

	if entry.ID == "" {
		entry.ID = uuid.New().String()
	}
	if entry.Timestamp.IsZero() {
		entry.Timestamp = time.Now()
	}

	data, err := json.MarshalIndent(entry, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal rollback entry: %w", err)
	}

	path := filepath.Join(dir, entry.ID+".json")
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write rollback file %s: %w", path, err)
	}
	return nil
}

// ListRollbacks returns all stored rollback entries, newest first.
func ListRollbacks() ([]RollbackEntry, error) {
	dir, err := rollbackDir()
	if err != nil {
		return nil, err
	}

	dirEntries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read rollback directory: %w", err)
	}

	var result []RollbackEntry
	for _, e := range dirEntries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			continue
		}
		var entry RollbackEntry
		if err := json.Unmarshal(data, &entry); err != nil {
			continue
		}
		result = append(result, entry)
	}

	// Sort newest first (simple insertion sort; rollback lists are small).
	for i := 1; i < len(result); i++ {
		for j := i; j > 0 && result[j].Timestamp.After(result[j-1].Timestamp); j-- {
			result[j], result[j-1] = result[j-1], result[j]
		}
	}
	return result, nil
}

// ApplyRollback reverses the operation recorded in the given rollback entry.
func ApplyRollback(id string) error {
	dir, err := rollbackDir()
	if err != nil {
		return err
	}

	path := filepath.Join(dir, id+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("rollback entry %q not found: %w", id, err)
	}

	var entry RollbackEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		return fmt.Errorf("failed to parse rollback entry: %w", err)
	}

	if err := applyRollbackAction(entry); err != nil {
		return err
	}

	// Remove the consumed rollback file.
	_ = os.Remove(path)
	return nil
}

// applyRollbackAction dispatches the rollback based on entry.Action.
func applyRollbackAction(entry RollbackEntry) error {
	switch entry.Action {
	case "path_add":
		// We added entry.NewValue to PATH; reverse = remove it.
		return RemoveFromUserPath(entry.NewValue)
	case "path_remove":
		// We removed entry.OldValue from PATH; reverse = add it back.
		return AddToUserPath(entry.OldValue)
	case "env_set":
		if entry.OldValue == "" {
			// Variable did not exist before; delete it.
			// entry.NewValue is "key=value"; extract key.
			key := entry.NewValue
			if idx := strings.Index(key, "="); idx >= 0 {
				key = key[:idx]
			}
			return DeleteEnvVar(key)
		}
		// Restore the old value. entry.NewValue is "key=value"; key is before first "=".
		key := entry.NewValue
		if idx := strings.Index(key, "="); idx >= 0 {
			key = key[:idx]
		}
		return SetEnvVar(key, entry.OldValue)
	case "env_delete":
		// We deleted a variable; restore old value.
		key := entry.NewValue
		return SetEnvVar(key, entry.OldValue)
	case "autostart_enable":
		return DisableAutostart()
	case "autostart_disable":
		// Re-enable with the args that were active before.
		exePath, _ := os.Executable()
		return EnableAutostart(exePath, entry.OldValue)
	default:
		return fmt.Errorf("unknown rollback action: %s", entry.Action)
	}
}

// CleanupOldRollbacks removes rollback entries older than 30 days.
func CleanupOldRollbacks() (int, error) {
	dir, err := rollbackDir()
	if err != nil {
		return 0, err
	}

	dirEntries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, fmt.Errorf("failed to read rollback directory: %w", err)
	}

	cutoff := time.Now().Add(-rollbackMaxAge)
	removed := 0
	for _, e := range dirEntries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			continue
		}
		var entry RollbackEntry
		if err := json.Unmarshal(data, &entry); err != nil {
			continue
		}
		if entry.Timestamp.Before(cutoff) {
			if err := os.Remove(filepath.Join(dir, e.Name())); err == nil {
				removed++
			}
		}
	}
	return removed, nil
}
