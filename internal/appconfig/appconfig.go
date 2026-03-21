package appconfig

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

// AppSettings holds application-level UI/UX preferences
type AppSettings struct {
	Theme               string `json:"theme"`               // "dark" | "light" | "auto"
	Language            string `json:"language"`             // "zh" | "en"
	AutoUpdate          bool   `json:"autoUpdate"`
	EditorFontSize      int    `json:"editorFontSize"`      // 10-24
	StartupPage         string `json:"startupPage"`         // "home" | "tools" | "gateway" etc.
	OnboardingCompleted bool   `json:"onboardingCompleted"` // true after setup wizard completes
	AppMode             string `json:"appMode"`             // "user" | "promoter"
	UserLevel           string `json:"userLevel"`           // "beginner" | "regular" | "power"
}

// settingsPath returns the path to app-settings.json
func settingsPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}

	var dir string
	switch runtime.GOOS {
	case "windows":
		appData := os.Getenv("APPDATA")
		if appData == "" {
			appData = filepath.Join(home, "AppData", "Roaming")
		}
		dir = filepath.Join(appData, "lurus-switch")
	case "darwin":
		dir = filepath.Join(home, "Library", "Application Support", "lurus-switch")
	default:
		dir = filepath.Join(home, ".lurus-switch")
	}

	return filepath.Join(dir, "app-settings.json"), nil
}

// DefaultAppSettings returns factory defaults
func DefaultAppSettings() *AppSettings {
	return &AppSettings{
		Theme:          "dark",
		Language:       "zh",
		AutoUpdate:     true,
		EditorFontSize: 13,
		StartupPage:    "home",
		AppMode:        "user",
		UserLevel:      "beginner",
	}
}

// LoadAppSettings reads app settings from disk; returns defaults if file is missing
func LoadAppSettings() (*AppSettings, error) {
	p, err := settingsPath()
	if err != nil {
		return DefaultAppSettings(), nil
	}

	data, err := os.ReadFile(p)
	if err != nil {
		if os.IsNotExist(err) {
			return DefaultAppSettings(), nil
		}
		return nil, fmt.Errorf("failed to read app settings: %w", err)
	}

	s := DefaultAppSettings()
	if err := json.Unmarshal(data, s); err != nil {
		fmt.Fprintf(os.Stderr, "warning: corrupt app-settings.json, using defaults: %v\n", err)
		return DefaultAppSettings(), nil
	}

	// Clamp editor font size to a safe range
	if s.EditorFontSize < 10 {
		s.EditorFontSize = 10
	} else if s.EditorFontSize > 24 {
		s.EditorFontSize = 24
	}

	// Validate app mode
	if s.AppMode != "user" && s.AppMode != "promoter" {
		s.AppMode = "user"
	}

	// Validate user level
	if s.UserLevel != "beginner" && s.UserLevel != "regular" && s.UserLevel != "power" {
		s.UserLevel = "beginner"
	}

	return s, nil
}

// SaveAppSettings persists app settings to disk
func SaveAppSettings(s *AppSettings) error {
	if s == nil {
		return fmt.Errorf("settings must not be nil")
	}

	p, err := settingsPath()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
		return fmt.Errorf("failed to create settings directory: %w", err)
	}

	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal app settings: %w", err)
	}

	if err := os.WriteFile(p, data, 0644); err != nil {
		return fmt.Errorf("failed to write app settings: %w", err)
	}

	return nil
}
