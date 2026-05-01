package whitelabel

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// Loader reads + verifies a sidecar at app startup. When found and valid,
// the calling layer (services bootstrap) writes the embedded HubURL and
// brand metadata into AppSettings as the LockedHubURL — engaging the
// EndUser mode lock from there on.
type Loader struct {
	HMACKey []byte
}

// FindSidecarPath returns the expected location of whitelabel.json based
// on the running executable's directory. Returns "" if os.Executable()
// fails (extremely unlikely outside hostile sandboxes).
func FindSidecarPath() string {
	exe, err := os.Executable()
	if err != nil {
		return ""
	}
	return filepath.Join(filepath.Dir(exe), SidecarFilename)
}

// Load reads + parses + verifies the sidecar at path. Returns:
//
//   - (profile, nil)        on a valid, signed sidecar.
//   - (nil, ErrNoSidecar)   when the file simply isn't there — caller
//     treats this as "not a white-label build, nothing to lock".
//   - (nil, err)            for any other problem (corruption, bad
//     HMAC, schema-version mismatch). EndUser mode must NOT proceed
//     in this case — refusing to launch is the correct response to a
//     tampered sidecar.
func (l *Loader) Load(path string) (*Profile, error) {
	if len(l.HMACKey) == 0 {
		return nil, errors.New("loader: hmac key not configured")
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, ErrNoSidecar
		}
		return nil, fmt.Errorf("read sidecar: %w", err)
	}
	var p Profile
	if err := json.Unmarshal(raw, &p); err != nil {
		return nil, fmt.Errorf("parse sidecar: %w", err)
	}
	if p.Version == 0 {
		return nil, errors.New("sidecar: missing version")
	}
	if p.Version > SidecarVersion {
		return nil, fmt.Errorf("sidecar: version %d unsupported (this build understands up to %d)",
			p.Version, SidecarVersion)
	}
	if err := p.Verify(l.HMACKey); err != nil {
		return nil, fmt.Errorf("sidecar verification failed: %w", err)
	}
	if err := p.Validate(); err != nil {
		return nil, fmt.Errorf("sidecar payload invalid: %w", err)
	}
	return &p, nil
}

// ErrNoSidecar is the "this isn't a white-label build" signal — distinct
// from "sidecar present but bad" so callers can decide whether to treat
// the absence as a hard error or a no-op.
var ErrNoSidecar = errors.New("whitelabel: sidecar not present")

// IsNoSidecar narrows an error for the bindings layer.
func IsNoSidecar(err error) bool {
	return errors.Is(err, ErrNoSidecar)
}
