package mcp

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// Store manages MCP preset persistence
type Store struct {
	dir string
}

// NewStore creates a new MCP preset store, creating the storage directory if needed
func NewStore() (*Store, error) {
	dir, err := presetsDir()
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create mcp presets directory: %w", err)
	}
	return &Store{dir: dir}, nil
}

// presetsDir returns the directory where MCP presets are stored
func presetsDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
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

	return filepath.Join(base, "mcp-presets"), nil
}

// ListPresets returns all stored user presets (not including built-ins)
func (s *Store) ListPresets() ([]MCPPreset, error) {
	entries, err := os.ReadDir(s.dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read mcp presets directory: %w", err)
	}

	var presets []MCPPreset
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(s.dir, e.Name()))
		if err != nil {
			continue
		}
		var p MCPPreset
		if err := json.Unmarshal(data, &p); err != nil {
			continue
		}
		presets = append(presets, p)
	}
	return presets, nil
}

// SavePreset persists a preset to disk; generates an ID if empty
func (s *Store) SavePreset(p MCPPreset) error {
	if p.ID == "" {
		p.ID = fmt.Sprintf("user-%d", time.Now().UnixMilli())
	}
	if strings.ContainsAny(p.ID, `/\`) || strings.Contains(p.ID, "..") {
		return fmt.Errorf("invalid preset ID: %q", p.ID)
	}

	data, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal preset: %w", err)
	}

	path := filepath.Join(s.dir, p.ID+".json")
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write preset: %w", err)
	}
	return nil
}

// DeletePreset removes a preset by ID
func (s *Store) DeletePreset(id string) error {
	if strings.ContainsAny(id, `/\`) || strings.Contains(id, "..") {
		return fmt.Errorf("invalid preset ID: %q", id)
	}
	path := filepath.Join(s.dir, id+".json")
	if err := os.Remove(path); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("preset not found: %s", id)
		}
		return fmt.Errorf("failed to delete preset: %w", err)
	}
	return nil
}
