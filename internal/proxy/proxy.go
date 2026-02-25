package proxy

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

// ProxySettings holds the NewAPI proxy configuration
type ProxySettings struct {
	APIEndpoint     string `json:"apiEndpoint"`
	APIKey          string `json:"apiKey"`
	RegistrationURL string `json:"registrationUrl,omitempty"`
}

// ProxyManager handles loading and saving proxy settings
type ProxyManager struct {
	configPath string
	settings   *ProxySettings
}

// NewProxyManager creates a new ProxyManager and loads existing settings if present
func NewProxyManager() (*ProxyManager, error) {
	configPath, err := getProxyConfigPath()
	if err != nil {
		return nil, fmt.Errorf("failed to determine proxy config path: %w", err)
	}

	pm := &ProxyManager{
		configPath: configPath,
		settings:   &ProxySettings{},
	}

	// Load existing settings (ignore error if file doesn't exist)
	if data, err := os.ReadFile(configPath); err == nil {
		json.Unmarshal(data, pm.settings)
	}

	return pm, nil
}

// GetSettings returns the current proxy settings
func (pm *ProxyManager) GetSettings() *ProxySettings {
	return pm.settings
}

// SaveSettings persists proxy settings to disk
func (pm *ProxyManager) SaveSettings(s *ProxySettings) error {
	pm.settings = s

	dir := filepath.Dir(pm.configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create proxy config directory: %w", err)
	}

	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal proxy settings: %w", err)
	}

	if err := os.WriteFile(pm.configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write proxy settings: %w", err)
	}

	return nil
}

// getProxyConfigPath returns the platform-specific path for proxy.json
func getProxyConfigPath() (string, error) {
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
	default:
		baseDir = os.Getenv("XDG_CONFIG_HOME")
		if baseDir == "" {
			home, err := os.UserHomeDir()
			if err != nil {
				return "", err
			}
			baseDir = filepath.Join(home, ".config")
		}
	}

	return filepath.Join(baseDir, "lurus-switch", "configs", "proxy.json"), nil
}
