package appconfig

import (
	"fmt"
	"strings"
)

// AppMode is the operating mode of the Switch desktop client.
//
// Four modes are supported (see ADR-020 + Enterprise extension):
//   - ModePersonal   — single-user CLI manager, talks to Lurus-operated Hub.
//   - ModeReseller   — operator console for distributors: deploy private
//     Hub, manage channels/tokens/redemptions, export white-labeled
//     EndUser builds.
//   - ModeEndUser    — locked-down client distributed by a reseller;
//     activates via redemption code, talks only to the embedded reseller
//     Hub URL.
//   - ModeEnterprise — internal-tool deployment for traditional companies.
//     SSO-bound users, cost-center accounting, DLP middleware on the
//     gateway, employee dashboards. No outbound sale or white-label.
type AppMode string

const (
	ModePersonal   AppMode = "personal"
	ModeReseller   AppMode = "reseller"
	ModeEndUser    AppMode = "enduser"
	ModeEnterprise AppMode = "enterprise"

	// ModeUnset means the user hasn't picked a mode yet — first-launch wizard
	// is required before reaching any mode-gated UI.
	ModeUnset AppMode = ""
)

// Valid reports whether m is a recognized mode value (excluding ModeUnset).
func (m AppMode) Valid() bool {
	switch m {
	case ModePersonal, ModeReseller, ModeEndUser, ModeEnterprise:
		return true
	default:
		return false
	}
}

// String returns the canonical lowercase form for persistence.
func (m AppMode) String() string {
	return string(m)
}

// migrateLegacyMode maps the v0.1.0 two-state values ("user"/"promoter") to the
// tri-state values introduced in this release. Anything else (including the
// empty string) is returned unchanged — the caller decides whether to treat
// that as "unset, prompt for selection" or "invalid, fall back to personal".
func migrateLegacyMode(raw string) AppMode {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "user":
		return ModePersonal
	case "promoter":
		return ModeReseller
	default:
		return AppMode(raw)
	}
}

// normalizeMode applies legacy migration and validation. Returns the resolved
// mode plus a bool indicating whether the input was a valid recognized value
// (after migration). An invalid input is reported but coerced to ModePersonal
// so the app can keep starting.
func normalizeMode(raw string) (AppMode, bool) {
	m := migrateLegacyMode(raw)
	if !m.Valid() && m != ModeUnset {
		return ModePersonal, false
	}
	return m, true
}

// CanTransition reports whether the user is allowed to switch from the current
// mode to next. Constraints:
//   - EndUser mode is one-way when locked (a white-labeled build pinned to a
//     reseller Hub) — the caller must check IsModeLocked before allowing it.
//   - Any unlocked mode can transition to any other unlocked mode.
func CanTransition(current, next AppMode, locked bool) error {
	if !next.Valid() {
		return fmt.Errorf("invalid target mode: %q", next)
	}
	if locked && next != current {
		return fmt.Errorf("mode is locked by white-label package; cannot switch to %q", next)
	}
	return nil
}

// IsModeLocked reports whether the on-disk settings indicate the mode has been
// pinned by a white-label distribution and must not be user-changed. The lock
// is in effect when running in EndUser mode with a non-empty LockedHubURL.
func IsModeLocked(s *AppSettings) bool {
	if s == nil {
		return false
	}
	return AppMode(s.AppMode) == ModeEndUser && strings.TrimSpace(s.LockedHubURL) != ""
}
