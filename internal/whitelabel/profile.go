// Package whitelabel implements the Reseller-side packager that turns a
// stock Switch build into a branded EndUser distribution, plus the
// EndUser-side loader that verifies and applies the resulting sidecar at
// app startup.
//
// Design choices:
//
//   - **Sidecar over PE patching for the variable bits.** Brand name,
//     Hub URL, primary color, and logo data live in a JSON sidecar
//     (`whitelabel.json`) shipped next to the exe, signed with HMAC-SHA256.
//     This keeps the packager pure-Go and testable; the only PE-touching
//     step (icon replacement) is optional and degrades gracefully when
//     rcedit isn't present.
//   - **HMAC, not crypto signatures.** The threat model is "stop a
//     curious EndUser from rewiring the Hub URL", not "withstand a
//     determined adversary with the binary in hand". HMAC + a key
//     embedded at build time is the right cost/value point. Hub admin
//     can rotate the key; EndUser mode refuses to launch if the local
//     sidecar fails verification.
//   - **Idempotent build.** Re-running Build() with the same inputs
//     produces a byte-identical sidecar (deterministic JSON ordering)
//     and copies the base exe by-byte — so distributors can re-pack
//     after a Switch upgrade without changing the output's hash unless
//     the source actually changed.
package whitelabel

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

// SidecarVersion is the schema version for whitelabel.json. Bump on any
// breaking field change; the loader rejects sidecars with an unrecognized
// version so stale clients can't accept new payload semantics blindly.
const SidecarVersion = 1

// Profile captures the branding parameters a Reseller customizes per
// EndUser distribution. Pure data — no methods that mutate it.
type Profile struct {
	// Version pins the schema. Set automatically by Build().
	Version int `json:"version"`

	// Reseller-facing identifier. Surfaced in EndUser dashboard and used
	// as the default output filename ("<brand>-Switch-windows-amd64.exe").
	BrandName string `json:"brand_name"`

	// HubURL is the Hub the EndUser binary must talk to. Becomes the
	// LockedHubURL in AppSettings on first boot.
	HubURL string `json:"hub_url"`

	// TenantSlug scopes V2 endpoints when the Hub is multi-tenant. Empty
	// for single-tenant deployments.
	TenantSlug string `json:"tenant_slug,omitempty"`

	// PrimaryColor is the brand accent (CSS color, e.g. "#9333ea").
	// Frontend reads it via a CSS variable injected at boot.
	PrimaryColor string `json:"primary_color,omitempty"`

	// LogoBase64 is a PNG/SVG payload embedded as base64 so the sidecar
	// is a single self-contained file. Capped at 256KB by Build() to
	// keep the EndUser binary's footprint sane.
	LogoBase64 string `json:"logo_base64,omitempty"`

	// SupportContact is a mailto: or https: URL the EndUser dashboard
	// surfaces as "Contact your reseller". Optional.
	SupportContact string `json:"support_contact,omitempty"`

	// CreatedAt is when the sidecar was generated. Diagnostic only.
	CreatedAt time.Time `json:"created_at"`

	// HMAC is the signature over the canonical-JSON of all preceding
	// fields. Computed by Build(), verified by Loader.Load().
	HMAC string `json:"hmac"`
}

// MaxLogoBytes caps the embedded logo at 256KB after base64 decoding.
// Distributors who need bigger assets should host them externally and
// reference via SupportContact / a future LogoURL field.
const MaxLogoBytes = 256 * 1024

// Validate runs basic sanity checks. Called by Build() before signing
// and by Loader.Load() after verifying. Cheap, non-network.
func (p *Profile) Validate() error {
	if strings.TrimSpace(p.BrandName) == "" {
		return errors.New("brand_name is required")
	}
	hub := strings.TrimSpace(p.HubURL)
	if hub == "" {
		return errors.New("hub_url is required")
	}
	if !strings.HasPrefix(hub, "http://") && !strings.HasPrefix(hub, "https://") {
		return errors.New("hub_url must start with http:// or https://")
	}
	if p.PrimaryColor != "" && !looksLikeColor(p.PrimaryColor) {
		return fmt.Errorf("primary_color %q does not look like a CSS color", p.PrimaryColor)
	}
	return nil
}

// looksLikeColor approximates "is this a valid CSS color literal" with
// the cases the Reseller UI actually emits: #rrggbb, #rgb, rgb(...), or
// a known named color. Stricter than a regex, looser than a parser —
// the goal is to catch typos, not validate every CSS spec edge case.
func looksLikeColor(s string) bool {
	s = strings.TrimSpace(strings.ToLower(s))
	if strings.HasPrefix(s, "#") {
		hex := s[1:]
		if len(hex) != 3 && len(hex) != 6 && len(hex) != 8 {
			return false
		}
		for _, c := range hex {
			if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
				return false
			}
		}
		return true
	}
	if strings.HasPrefix(s, "rgb(") || strings.HasPrefix(s, "rgba(") || strings.HasPrefix(s, "hsl(") {
		return strings.HasSuffix(s, ")")
	}
	// Common named colors used in dashboards.
	named := []string{"black", "white", "red", "green", "blue", "purple", "orange", "yellow", "teal", "pink", "indigo", "emerald", "amber"}
	for _, n := range named {
		if s == n {
			return true
		}
	}
	return false
}

// Sign computes HMAC-SHA256 over the canonical JSON of p (excluding the
// HMAC field itself). The same input always produces the same signature
// — necessary for idempotent builds.
//
// Canonicalization: marshal a *copy* with HMAC cleared, using
// json.Marshal which produces deterministic field order for structs.
func (p *Profile) Sign(key []byte) (string, error) {
	canon, err := canonicalBytes(p)
	if err != nil {
		return "", err
	}
	mac := hmac.New(sha256.New, key)
	mac.Write(canon)
	return hex.EncodeToString(mac.Sum(nil)), nil
}

// Verify checks the embedded HMAC matches what Sign() would produce for
// the rest of the profile under key. Used by Loader.Load() and tests.
//
// Constant-time comparison via hmac.Equal — defends against a malicious
// EndUser who patches the binary to early-exit on the first mismatched
// byte hoping to brute-force the signature.
func (p *Profile) Verify(key []byte) error {
	if p.HMAC == "" {
		return errors.New("missing hmac")
	}
	expected, err := p.Sign(key)
	if err != nil {
		return err
	}
	if !hmac.Equal([]byte(p.HMAC), []byte(expected)) {
		return errors.New("hmac mismatch")
	}
	return nil
}

// canonicalBytes returns a deterministic JSON encoding of p with the
// HMAC field cleared. Sort isn't needed because json.Marshal already
// emits struct fields in declaration order.
func canonicalBytes(p *Profile) ([]byte, error) {
	cp := *p
	cp.HMAC = ""
	return json.Marshal(&cp)
}
