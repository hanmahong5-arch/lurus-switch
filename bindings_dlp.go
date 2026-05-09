package main

import (
	"sync"

	"lurus-switch/internal/capability"
	"lurus-switch/internal/dlp"
)

// dlpScanner is a process-wide DLP scanner. Lazily initialized; the
// services struct doesn't own it so that DLP can be added/removed as
// a Switch capability without touching every Wails binding signature.
var (
	dlpOnce    sync.Once
	dlpScanner *dlp.Scanner
)

func getDLPScanner() *dlp.Scanner {
	dlpOnce.Do(func() {
		dlpScanner = dlp.NewScanner()
	})
	return dlpScanner
}

// ScanText runs the active DLP pattern set against the given input and
// returns the Result struct. Used by the DLP admin UI for ad-hoc tests
// ("does this prompt trigger anything?") and — eventually — by the
// gateway request middleware on every inbound prompt.
func (a *App) ScanText(input string) dlp.Result {
	return getDLPScanner().Scan(input)
}

// ListDLPPatterns returns the active pattern table for the admin UI.
// Reading the pattern set requires no special cap — it's a config
// surface, not user content. Setting a policy does (CapOptionWrite).
func (a *App) ListDLPPatterns() []dlp.Pattern {
	return getDLPScanner().Patterns()
}

// SetDLPPolicy mutates the policy of an existing pattern. Returns true
// if a pattern with the given name was found.
func (a *App) SetDLPPolicy(name string, policy string) (bool, error) {
	if err := capability.RequireCurrent(capability.CapOptionWrite); err != nil {
		return false, err
	}
	return getDLPScanner().SetPolicy(name, dlp.Policy(policy)), nil
}

// AddDLPPattern lets the admin register a custom regex (e.g. for
// internal customer IDs). Returns nil on success.
func (a *App) AddDLPPattern(p dlp.Pattern) error {
	if err := capability.RequireCurrent(capability.CapOptionWrite); err != nil {
		return err
	}
	return getDLPScanner().Add(p)
}

// RemoveDLPPattern drops a pattern by name. Useful for retiring
// false-positive-prone defaults in a specific deployment.
func (a *App) RemoveDLPPattern(name string) (bool, error) {
	if err := capability.RequireCurrent(capability.CapOptionWrite); err != nil {
		return false, err
	}
	return getDLPScanner().Remove(name), nil
}
