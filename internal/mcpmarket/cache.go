package mcpmarket

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"
)

const (
	// cacheMaxAge is the maximum age of a registry cache before it is considered stale.
	cacheMaxAge = 12 * time.Hour
)

// cacheFile is the on-disk representation stored at
// %APPDATA%\lurus-switch\mcpmarket\cache.json.
type cacheFile struct {
	FetchedAt time.Time      `json:"fetched_at"`
	ETag      string         `json:"etag,omitempty"`
	Servers   []MarketServer `json:"servers"`
}

// cacheDir returns the platform-appropriate directory for the MCP market cache.
func cacheDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("mcpmarket: get home dir: %w", err)
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

	return filepath.Join(base, "mcpmarket"), nil
}

// loadCache reads the on-disk cache.  Returns empty values on any error or
// cache miss — callers should treat an empty/nil slice as "use builtins".
func loadCache() (servers []MarketServer, fetchedAt time.Time, etag string, err error) {
	dir, dirErr := cacheDir()
	if dirErr != nil {
		return nil, time.Time{}, "", dirErr
	}
	data, readErr := os.ReadFile(filepath.Join(dir, "cache.json"))
	if readErr != nil {
		if os.IsNotExist(readErr) {
			return nil, time.Time{}, "", nil
		}
		return nil, time.Time{}, "", fmt.Errorf("mcpmarket: read cache: %w", readErr)
	}
	var cf cacheFile
	if jsonErr := json.Unmarshal(data, &cf); jsonErr != nil {
		// Corrupt cache — return empty, not an error.
		return nil, time.Time{}, "", nil
	}
	return cf.Servers, cf.FetchedAt, cf.ETag, nil
}

// saveCache persists registry servers to disk.
func saveCache(servers []MarketServer, etag string) error {
	dir, err := cacheDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("mcpmarket: create cache dir: %w", err)
	}
	cf := cacheFile{FetchedAt: time.Now(), ETag: etag, Servers: servers}
	data, err := json.MarshalIndent(cf, "", "  ")
	if err != nil {
		return fmt.Errorf("mcpmarket: marshal cache: %w", err)
	}
	path := filepath.Join(dir, "cache.json")
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("mcpmarket: write cache: %w", err)
	}
	return nil
}
