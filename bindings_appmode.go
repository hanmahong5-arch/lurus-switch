package main

import (
	"fmt"
	"strings"

	"lurus-switch/internal/appconfig"
)

// ============================
// AppMode (S-Xa.1) — three-mode dispatch surface for the frontend.
// ============================
//
// The frontend reads GetAppMode() on boot to decide whether to render the
// first-launch wizard, and calls SetAppMode() once the user picks a mode.
// IsModeLocked() guards the Settings-page mode switch UI in white-label
// EndUser builds, where the mode is pinned by the distributor.

// GetAppMode returns the current operating mode.
//
// Returns one of:
//   - "" (ModeUnset) — first launch, frontend should show mode picker
//   - "personal" — solo developer, talks to Lurus-operated Hub
//   - "reseller" — distributor running their own Hub deployment
//   - "enduser" — white-labeled client locked to a specific reseller Hub
func (a *App) GetAppMode() (string, error) {
	s, err := appconfig.LoadAppSettings()
	if err != nil {
		return "", fmt.Errorf("load app settings: %w", err)
	}
	return s.AppMode, nil
}

// SetAppMode changes the operating mode and persists it.
//
// Refuses transitions when the package is locked (EndUser white-label) — the
// frontend should call IsModeLocked() first to disable the UI control rather
// than relying on the error path.
func (a *App) SetAppMode(mode string) error {
	target := appconfig.AppMode(strings.TrimSpace(strings.ToLower(mode)))
	if !target.Valid() {
		return fmt.Errorf("invalid mode %q (allowed: personal, reseller, enduser)", mode)
	}

	s, err := appconfig.LoadAppSettings()
	if err != nil {
		return fmt.Errorf("load app settings: %w", err)
	}

	current := appconfig.AppMode(s.AppMode)
	if err := appconfig.CanTransition(current, target, appconfig.IsModeLocked(s)); err != nil {
		return err
	}

	s.AppMode = string(target)
	if err := appconfig.SaveAppSettings(s); err != nil {
		return fmt.Errorf("save app settings: %w", err)
	}
	return nil
}

// IsModeLocked reports whether the current mode is pinned by a white-label
// distribution and cannot be changed via the UI.
func (a *App) IsModeLocked() (bool, error) {
	s, err := appconfig.LoadAppSettings()
	if err != nil {
		return false, fmt.Errorf("load app settings: %w", err)
	}
	return appconfig.IsModeLocked(s), nil
}
