// Package sysenv provides cross-platform system environment automation:
// PATH modification, environment variable management, autostart configuration,
// git detection, and rollback state persistence.
package sysenv

import (
	"os"
	"strings"
	"time"
)

// PathEntry describes a single directory in the user's PATH.
type PathEntry struct {
	Dir    string `json:"dir"`
	Exists bool   `json:"exists"`
}

// EnvVar represents a user environment variable.
type EnvVar struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// AutostartConfig describes the autostart state.
type AutostartConfig struct {
	Enabled bool   `json:"enabled"`
	Args    string `json:"args"` // e.g. "--minimized"
}

// RollbackEntry records a single reversible system change.
type RollbackEntry struct {
	ID        string    `json:"id"`
	Action    string    `json:"action"`   // "path_add", "path_remove", "env_set", "env_delete", "autostart_enable", "autostart_disable"
	OldValue  string    `json:"oldValue"` // previous state; empty string means "did not exist"
	NewValue  string    `json:"newValue"` // new state applied
	Timestamp time.Time `json:"timestamp"`
}

// GitInfo contains detected git installation details.
type GitInfo struct {
	Installed bool   `json:"installed"`
	Version   string `json:"version"`
	UserName  string `json:"userName"`
	UserEmail string `json:"userEmail"`
}

// SystemEnvironment is a composite snapshot of all environment state,
// used by the Wails binding to return everything in a single call.
type SystemEnvironment struct {
	PathEntries []PathEntry     `json:"pathEntries"`
	Autostart   AutostartConfig `json:"autostart"`
	Git         *GitInfo        `json:"git"`
}

// ParsePathEntries splits a PATH string by the given separator and checks
// whether each directory exists on disk.
func ParsePathEntries(raw string, sep string) []PathEntry {
	if raw == "" {
		return nil
	}
	dirs := strings.Split(raw, sep)
	entries := make([]PathEntry, 0, len(dirs))
	for _, d := range dirs {
		d = strings.TrimSpace(d)
		if d == "" {
			continue
		}
		_, err := os.Stat(d)
		entries = append(entries, PathEntry{
			Dir:    d,
			Exists: err == nil,
		})
	}
	return entries
}
