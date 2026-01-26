package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

// Store handles configuration persistence
type Store struct {
	configDir string
}

// NewStore creates a new configuration store
func NewStore() (*Store, error) {
	configDir, err := getConfigDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get config directory: %w", err)
	}

	// Ensure the config directory exists
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create config directory: %w", err)
	}

	return &Store{configDir: configDir}, nil
}

// getConfigDir returns the platform-specific configuration directory
func getConfigDir() (string, error) {
	var baseDir string

	switch runtime.GOOS {
	case "windows":
		baseDir = os.Getenv("APPDATA")
		if baseDir == "" {
			baseDir = filepath.Join(os.Getenv("USERPROFILE"), "AppData", "Roaming")
		}
	case "darwin":
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		baseDir = filepath.Join(home, "Library", "Application Support")
	default: // Linux and others
		baseDir = os.Getenv("XDG_CONFIG_HOME")
		if baseDir == "" {
			home, err := os.UserHomeDir()
			if err != nil {
				return "", err
			}
			baseDir = filepath.Join(home, ".config")
		}
	}

	return filepath.Join(baseDir, "lurus-switch", "configs"), nil
}

// GetConfigDir returns the configuration directory path
func (s *Store) GetConfigDir() string {
	return s.configDir
}

// SaveClaudeConfig saves a Claude configuration to disk
func (s *Store) SaveClaudeConfig(name string, config *ClaudeConfig) error {
	return s.saveConfig("claude", name, config)
}

// LoadClaudeConfig loads a Claude configuration from disk
func (s *Store) LoadClaudeConfig(name string) (*ClaudeConfig, error) {
	var config ClaudeConfig
	if err := s.loadConfig("claude", name, &config); err != nil {
		return nil, err
	}
	return &config, nil
}

// SaveCodexConfig saves a Codex configuration to disk
func (s *Store) SaveCodexConfig(name string, config *CodexConfig) error {
	return s.saveConfig("codex", name, config)
}

// LoadCodexConfig loads a Codex configuration from disk
func (s *Store) LoadCodexConfig(name string) (*CodexConfig, error) {
	var config CodexConfig
	if err := s.loadConfig("codex", name, &config); err != nil {
		return nil, err
	}
	return &config, nil
}

// SaveGeminiConfig saves a Gemini configuration to disk
func (s *Store) SaveGeminiConfig(name string, config *GeminiConfig) error {
	return s.saveConfig("gemini", name, config)
}

// LoadGeminiConfig loads a Gemini configuration from disk
func (s *Store) LoadGeminiConfig(name string) (*GeminiConfig, error) {
	var config GeminiConfig
	if err := s.loadConfig("gemini", name, &config); err != nil {
		return nil, err
	}
	return &config, nil
}

// ListConfigs lists all saved configurations for a given tool
func (s *Store) ListConfigs(tool string) ([]string, error) {
	toolDir := filepath.Join(s.configDir, tool)
	if _, err := os.Stat(toolDir); os.IsNotExist(err) {
		return []string{}, nil
	}

	entries, err := os.ReadDir(toolDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}

	var names []string
	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".json" {
			name := entry.Name()[:len(entry.Name())-5] // Remove .json extension
			names = append(names, name)
		}
	}

	return names, nil
}

// DeleteConfig deletes a saved configuration
func (s *Store) DeleteConfig(tool, name string) error {
	configPath := filepath.Join(s.configDir, tool, name+".json")
	if err := os.Remove(configPath); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("config not found: %s/%s", tool, name)
		}
		return fmt.Errorf("failed to delete config: %w", err)
	}
	return nil
}

// saveConfig saves a configuration to disk
func (s *Store) saveConfig(tool, name string, config interface{}) error {
	toolDir := filepath.Join(s.configDir, tool)
	if err := os.MkdirAll(toolDir, 0755); err != nil {
		return fmt.Errorf("failed to create tool directory: %w", err)
	}

	configPath := filepath.Join(toolDir, name+".json")
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// loadConfig loads a configuration from disk
func (s *Store) loadConfig(tool, name string, config interface{}) error {
	configPath := filepath.Join(s.configDir, tool, name+".json")
	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("config not found: %s/%s", tool, name)
		}
		return fmt.Errorf("failed to read config file: %w", err)
	}

	if err := json.Unmarshal(data, config); err != nil {
		return fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return nil
}
