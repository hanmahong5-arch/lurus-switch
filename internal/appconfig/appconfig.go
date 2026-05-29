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
	FeatureTourSeen     bool   `json:"featureTourSeen"`     // true after the post-setup feature tour has been shown at least once
	AppMode             string `json:"appMode"`             // "" | "personal" | "reseller" | "enduser" (legacy: "user"|"promoter" auto-migrated)
	UserLevel           string `json:"userLevel"`           // "beginner" | "regular" | "power"

	// LockedHubURL is set by white-label EndUser builds. When present and AppMode
	// is "enduser", the mode is pinned and the user cannot change it via UI.
	LockedHubURL string `json:"lockedHubUrl,omitempty"`

	// White-label branding fields. Populated by applyWhiteLabelSidecar at
	// startup from the signed sidecar. Frontend reads them via GetAppSettings
	// and applies them to the sidebar logo / accent color / support link.
	// All are empty for non-white-label installs.
	BrandName        string `json:"brandName,omitempty"`
	BrandLogoBase64  string `json:"brandLogoBase64,omitempty"`
	BrandPrimaryColor string `json:"brandPrimaryColor,omitempty"`
	BrandSupportContact string `json:"brandSupportContact,omitempty"`

	// Reseller mode configuration. Populated by ResellerSetupWizard (S-Xb.1)
	// after a Hub instance is provisioned and an admin token is obtained.
	// HubURL is also referenced by Personal mode (defaults to hub.lurus.cn)
	// and EndUser mode (always equals LockedHubURL).
	Reseller ResellerConfig `json:"reseller,omitempty"`

	// OIDC authentication settings (Zitadel).
	AuthClientID string `json:"authClientId,omitempty"`
	AuthIssuer   string `json:"authIssuer,omitempty"` // default: "https://auth.lurus.cn"

	// AuthPlatformURL overrides the platform-core base URL used to fetch
	// /api/v1/account/me + /api/v1/wallet after login. Distinct from
	// AuthIssuer (which points at Zitadel). Empty = use built-in default
	// (auth.DefaultPlatformBaseURL = https://identity.lurus.cn).
	AuthPlatformURL string `json:"authPlatformUrl,omitempty"`

	// Observability configures optional OpenTelemetry export of gateway
	// GenAI traffic (gen_ai.* traces + token/latency metrics). Default off;
	// when Enabled the gateway records one span + metric set per request to
	// the configured OTLP/HTTP endpoint via internal/obs.
	Observability ObservabilityConfig `json:"observability,omitempty"`
}

// ObservabilityConfig is the OpenTelemetry export setup. Off by default;
// only the OTLP/HTTP transport is supported for now (Protocol reserved for
// a future gRPC option).
type ObservabilityConfig struct {
	Enabled  bool              `json:"enabled"`
	Endpoint string            `json:"endpoint,omitempty"` // OTLP/HTTP, e.g. "http://localhost:4318" or "host:4318"
	Protocol string            `json:"protocol,omitempty"` // "http" (default); "grpc" reserved
	Headers  map[string]string `json:"headers,omitempty"`  // optional OTLP headers (e.g. auth)
}

// ResellerConfig holds the per-Reseller Hub deployment context.
type ResellerConfig struct {
	HubURL      string `json:"hubUrl,omitempty"`      // root URL, e.g. "https://hub.acme.example"
	AdminToken  string `json:"adminToken,omitempty"`  // Hub-issued root/admin access token
	TenantSlug  string `json:"tenantSlug,omitempty"`  // multi-tenant slug for V2 endpoints
	DisplayName string `json:"displayName,omitempty"` // shown in UI ("Acme Corp")
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

// DefaultAppSettings returns factory defaults. AppMode defaults to ModeUnset
// so the first-launch wizard prompts for selection; existing v0.1.0 users who
// already have a saved config will be auto-migrated by LoadAppSettings.
func DefaultAppSettings() *AppSettings {
	return &AppSettings{
		Theme:          "dark",
		Language:       "zh",
		AutoUpdate:     true,
		EditorFontSize: 13,
		StartupPage:    "home",
		AppMode:        string(ModeUnset),
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

	// Migrate + validate app mode: legacy "user"/"promoter" map to
	// personal/reseller respectively; unrecognized values fall back to unset
	// so the first-launch wizard prompts the user to pick.
	resolved, ok := normalizeMode(s.AppMode)
	s.AppMode = string(resolved)
	if !ok {
		fmt.Fprintf(os.Stderr, "warning: unknown appMode %q in settings, defaulting to %q\n", resolved, ModeUnset)
		s.AppMode = string(ModeUnset)
	}

	// Validate user level
	if s.UserLevel != "beginner" && s.UserLevel != "regular" && s.UserLevel != "power" {
		s.UserLevel = "beginner"
	}

	return s, nil
}

// SaveAppSettings persists app settings to disk. Mode is normalized via the
// same migration path as LoadAppSettings, so callers may pass legacy values.
// Once the EndUser lock is engaged (mode=enduser AND lockedHubUrl set), this
// function refuses to write back a different mode — protects against UI bugs
// or malicious deeplinks unlocking a white-label package.
func SaveAppSettings(s *AppSettings) error {
	if s == nil {
		return fmt.Errorf("settings must not be nil")
	}

	resolved, ok := normalizeMode(s.AppMode)
	if !ok {
		return fmt.Errorf("invalid appMode: %q", s.AppMode)
	}
	s.AppMode = string(resolved)

	// If a previous run wrote a locked EndUser config to disk, refuse changes
	// that would break the lock. Reading from disk first means we trust the
	// FS-side state, not whatever the caller passed in.
	if existing, err := LoadAppSettings(); err == nil && IsModeLocked(existing) {
		if AppMode(s.AppMode) != ModeEndUser {
			return fmt.Errorf("cannot change app mode while white-label lock is active")
		}
		// Preserve the locked URL — caller cannot blank it out.
		if s.LockedHubURL == "" {
			s.LockedHubURL = existing.LockedHubURL
		}
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
