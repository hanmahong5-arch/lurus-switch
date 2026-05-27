package rulesmarket

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"
)

// cacheFile holds a list of templates fetched from remote manifests, plus
// the timestamp of the last successful fetch.
type cacheFile struct {
	FetchedAt time.Time      `json:"fetched_at"`
	Templates []RuleTemplate `json:"templates"`
}

// cacheDir returns the platform-appropriate directory for the rules market cache.
func cacheDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("rulesmarket: get home dir: %w", err)
	}

	var base string
	switch runtime.GOOS {
	case "windows":
		appData := os.Getenv("APPDATA")
		if appData == "" {
			appData = filepath.Join(home, "AppData", "Roaming")
		}
		base = filepath.Join(appData, "lurus-switch")
	case "darwin":
		base = filepath.Join(home, "Library", "Application Support", "lurus-switch")
	default:
		base = filepath.Join(home, ".lurus-switch")
	}

	return filepath.Join(base, "rulesmarket"), nil
}

// loadCache reads the on-disk cache; returns an empty slice on any error.
func loadCache() ([]RuleTemplate, time.Time, error) {
	dir, err := cacheDir()
	if err != nil {
		return nil, time.Time{}, err
	}
	path := filepath.Join(dir, "cache.json")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, time.Time{}, nil
		}
		return nil, time.Time{}, fmt.Errorf("rulesmarket: read cache: %w", err)
	}
	var cf cacheFile
	if err := json.Unmarshal(data, &cf); err != nil {
		// Corrupt cache — treat as empty
		return nil, time.Time{}, nil
	}
	return cf.Templates, cf.FetchedAt, nil
}

// saveCache persists the remote templates to disk.
func saveCache(templates []RuleTemplate) error {
	dir, err := cacheDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("rulesmarket: create cache dir: %w", err)
	}
	cf := cacheFile{FetchedAt: time.Now(), Templates: templates}
	data, err := json.MarshalIndent(cf, "", "  ")
	if err != nil {
		return fmt.Errorf("rulesmarket: marshal cache: %w", err)
	}
	path := filepath.Join(dir, "cache.json")
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("rulesmarket: write cache: %w", err)
	}
	return nil
}
