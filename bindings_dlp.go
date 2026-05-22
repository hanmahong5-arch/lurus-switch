package main

import (
	"lurus-switch/internal/capability"
	"lurus-switch/internal/dlp"
)

// DLP admin surface. The scanner instance lives on services and is
// shared with the gateway middleware (services.go wires it via
// SetDLPScanner), so policy changes made in the admin UI immediately
// apply to in-flight gateway traffic.

// dlpScannerOrNil returns the configured scanner, or nil when services
// failed to initialise (defensive — bindings should still no-op rather
// than panic).
func (a *App) dlpScannerOrNil() *dlp.Scanner {
	if a == nil || a.services == nil {
		return nil
	}
	return a.services.dlpScanner
}

// ScanText runs the active DLP pattern set against the given input and
// returns the Result struct. Used by the DLP admin UI for ad-hoc tests
// ("does this prompt trigger anything?") and by the gateway middleware
// on every inbound prompt.
func (a *App) ScanText(input string) dlp.Result {
	s := a.dlpScannerOrNil()
	if s == nil {
		return dlp.Result{}
	}
	res := s.Scan(input)
	// Recording with source="test" lets the admin distinguish ad-hoc
	// tests from real gateway traffic in the recent-hits view.
	s.RecordHits("test", "", res.Hits)
	return res
}

// ListDLPPatterns returns the active pattern table for the admin UI.
// Reading the pattern set requires no special cap — it's a config
// surface, not user content. Setting a policy does (CapOptionWrite).
func (a *App) ListDLPPatterns() []dlp.Pattern {
	s := a.dlpScannerOrNil()
	if s == nil {
		return nil
	}
	return s.Patterns()
}

// ListDLPHits returns the most-recent hits captured by the gateway
// middleware (and the test surface). The ring is bounded so this is
// safe to call on a slow polling interval from the admin UI.
func (a *App) ListDLPHits(limit int) []dlp.HitRecord {
	s := a.dlpScannerOrNil()
	if s == nil {
		return nil
	}
	return s.RecentHits(limit)
}

// GetDLPStats rolls up the recent-hits ring into counters for the
// dashboard tile (today: blocked / redacted / warned).
func (a *App) GetDLPStats() dlp.HitStats {
	s := a.dlpScannerOrNil()
	if s == nil {
		return dlp.HitStats{}
	}
	return s.Stats()
}

// SetDLPPolicy mutates the policy of an existing pattern. Returns true
// if a pattern with the given name was found.
func (a *App) SetDLPPolicy(name string, policy string) (bool, error) {
	if err := capability.RequireCurrent(capability.CapOptionWrite); err != nil {
		return false, err
	}
	s := a.dlpScannerOrNil()
	if s == nil {
		return false, nil
	}
	return s.SetPolicy(name, dlp.Policy(policy)), nil
}

// AddDLPPattern lets the admin register a custom regex (e.g. for
// internal customer IDs). Returns nil on success.
func (a *App) AddDLPPattern(p dlp.Pattern) error {
	if err := capability.RequireCurrent(capability.CapOptionWrite); err != nil {
		return err
	}
	s := a.dlpScannerOrNil()
	if s == nil {
		return nil
	}
	return s.Add(p)
}

// RemoveDLPPattern drops a pattern by name. Useful for retiring
// false-positive-prone defaults in a specific deployment.
func (a *App) RemoveDLPPattern(name string) (bool, error) {
	if err := capability.RequireCurrent(capability.CapOptionWrite); err != nil {
		return false, err
	}
	s := a.dlpScannerOrNil()
	if s == nil {
		return false, nil
	}
	return s.Remove(name), nil
}
