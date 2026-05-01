package main

import (
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"lurus-switch/internal/appconfig"
	"lurus-switch/internal/whitelabel"
)

// ============================
// White-label packager (S-Xc.1 + S-Xc.2) — Reseller-facing build pipeline.
// ============================
//
// The Reseller workflow:
//
//   1. PackagerPage collects brand inputs (name, hub URL, color, logo).
//   2. ResolveBaseBinaryPath() points at the running Switch.exe; the
//      Reseller can also override to a fresher build.
//   3. BuildWhiteLabelPackage() runs the packer, returning paths +
//      hashes. UI surfaces the output dir and a "open in explorer" link.
//
// The HMAC key would normally come from the Hub admin endpoint
// (/api/v2/admin/whitelabel/hmac-key — TBD). Until that ships, we derive
// a stable per-Hub key from the saved AdminToken so each Reseller's
// builds are still cross-verifiable on the EndUser side. The deferred
// migration is straightforward: swap the key source, no API change.

// WhiteLabelInputs is the JSON payload accepted from the frontend. Mirrors
// whitelabel.Profile but uses the fields the UI actually surfaces — keep
// the binding boundary stable even when the underlying Profile grows.
type WhiteLabelInputs struct {
	BrandName      string `json:"brandName"`
	HubURL         string `json:"hubUrl"`
	TenantSlug     string `json:"tenantSlug,omitempty"`
	PrimaryColor   string `json:"primaryColor,omitempty"`
	LogoBase64     string `json:"logoBase64,omitempty"`
	SupportContact string `json:"supportContact,omitempty"`
	OutputDir      string `json:"outputDir,omitempty"`
	IconPath       string `json:"iconPath,omitempty"`
}

// WhiteLabelOutput is the BuildResult re-spelled in camelCase so the
// frontend doesn't have to rename keys.
type WhiteLabelOutput struct {
	OutputDir     string   `json:"outputDir"`
	BinaryPath    string   `json:"binaryPath"`
	SidecarPath   string   `json:"sidecarPath"`
	BinarySHA256  string   `json:"binarySha256"`
	SidecarSHA256 string   `json:"sidecarSha256"`
	Notes         []string `json:"notes,omitempty"`
}

// BuildWhiteLabelPackage produces a branded EndUser distribution from
// the running Switch binary plus the Reseller's brand inputs.
//
// The built artifacts land in OutputDir (or `<appdata>/lurus-switch/
// whitelabel-builds/<brand-slug>` when empty). On success, the UI gets
// paths + hashes back; the Reseller can ship the directory contents as
// a ZIP or installer payload.
func (a *App) BuildWhiteLabelPackage(in WhiteLabelInputs) (*WhiteLabelOutput, error) {
	base, err := resolveBaseBinaryPath()
	if err != nil {
		return nil, err
	}

	outDir := strings.TrimSpace(in.OutputDir)
	if outDir == "" {
		outDir = defaultBuildOutputDir(in.BrandName)
	}

	key, err := whitelabelHMACKey()
	if err != nil {
		return nil, err
	}

	res, err := whitelabel.Build(whitelabel.BuildOpts{
		Profile: whitelabel.Profile{
			BrandName:      strings.TrimSpace(in.BrandName),
			HubURL:         strings.TrimSpace(in.HubURL),
			TenantSlug:     strings.TrimSpace(in.TenantSlug),
			PrimaryColor:   strings.TrimSpace(in.PrimaryColor),
			LogoBase64:     in.LogoBase64,
			SupportContact: strings.TrimSpace(in.SupportContact),
		},
		HMACKey:        key,
		BaseBinaryPath: base,
		OutputDir:      outDir,
		IconPath:       in.IconPath,
	})
	if err != nil {
		return nil, fmt.Errorf("build white-label package: %w", err)
	}
	return &WhiteLabelOutput{
		OutputDir:     res.OutputDir,
		BinaryPath:    res.BinaryPath,
		SidecarPath:   res.SidecarPath,
		BinarySHA256:  res.SHA256,
		SidecarSHA256: res.SidecarSHA256,
		Notes:         res.Notes,
	}, nil
}

// PreviewWhiteLabelLogo decodes the operator-supplied logo to surface
// useful metadata (size + content-type detection) so the UI can warn
// before running a full Build. Pure validation — no FS writes.
func (a *App) PreviewWhiteLabelLogo(logoBase64 string) (map[string]any, error) {
	if logoBase64 == "" {
		return nil, errors.New("logo is empty")
	}
	raw, err := base64.StdEncoding.DecodeString(logoBase64)
	if err != nil {
		return nil, fmt.Errorf("not valid base64: %w", err)
	}
	mime := "application/octet-stream"
	switch {
	case len(raw) >= 8 && string(raw[:8]) == "\x89PNG\r\n\x1a\n":
		mime = "image/png"
	case len(raw) >= 4 && (string(raw[:4]) == "GIF8"):
		mime = "image/gif"
	case len(raw) >= 3 && raw[0] == 0xFF && raw[1] == 0xD8 && raw[2] == 0xFF:
		mime = "image/jpeg"
	case len(raw) >= 5 && string(raw[:5]) == "<?xml":
		mime = "image/svg+xml"
	}
	return map[string]any{
		"size":      len(raw),
		"mime":      mime,
		"limit":     whitelabel.MaxLogoBytes,
		"oversized": len(raw) > whitelabel.MaxLogoBytes,
	}, nil
}

// resolveBaseBinaryPath finds the source Switch exe to clone. Currently
// uses os.Executable() — the running binary. Future iteration may add
// "download from GitHub releases" once the auto-update channel is wired.
func resolveBaseBinaryPath() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("locate running binary: %w", err)
	}
	return exe, nil
}

// defaultBuildOutputDir produces a per-brand directory under appdata.
// Keeps successive builds tidy and discoverable from the file manager.
func defaultBuildOutputDir(brand string) string {
	base := appDataBaseDir()
	slug := strings.ToLower(strings.TrimSpace(brand))
	if slug == "" {
		slug = "untitled"
	}
	// Reuse the packer's slug rules indirectly via filename safety.
	slug = strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			return r
		}
		return '-'
	}, slug)
	return filepath.Join(base, "whitelabel-builds", slug)
}

// whitelabelHMACKey derives the per-Hub HMAC key. Two scenarios:
//
//   - Reseller mode with AdminToken saved → key = sha256("whitelabel:" + token).
//     Each Reseller's builds are unique-keyed; rotating the token
//     invalidates all prior builds (acceptable: they reissue installers
//     after token rotation anyway).
//   - No saved token (e.g. first launch dev environment) → key derived
//     from the device fingerprint, used only for round-trip testing.
//
// Once the Hub /api/v2/admin/whitelabel/hmac-key endpoint ships, this
// function fetches there instead. No API change for callers.
func whitelabelHMACKey() ([]byte, error) {
	s, err := appconfig.LoadAppSettings()
	if err != nil {
		return nil, fmt.Errorf("load app settings: %w", err)
	}
	tok := strings.TrimSpace(s.Reseller.AdminToken)
	if tok == "" {
		return nil, errors.New("Reseller admin token 未配置：请先完成 Reseller Setup Wizard。")
	}
	sum := sha256.Sum256([]byte("whitelabel:" + tok))
	return sum[:], nil
}

// applyWhiteLabelSidecar runs at app startup. When a signed
// whitelabel.json sits next to the running exe, it locks the app to
// EndUser mode + the embedded Hub URL. Idempotent — already-locked
// installs short-circuit before doing any FS writes.
//
// Failure modes:
//
//   - No sidecar present → silent no-op (this isn't a white-label build).
//   - Sidecar present but verification fails → log + abort startup mode
//     write. EndUser code paths will see no LockedHubURL and surface a
//     clear "白标包损坏" error rather than dialing home.
//
// HMAC key resolution at startup uses the *previously-saved* AdminToken,
// since we have no Hub session yet. Distributors who rotate their token
// must re-pack — that's the documented contract.
func (a *App) applyWhiteLabelSidecar() {
	path := whitelabel.FindSidecarPath()
	if path == "" {
		return
	}
	key, err := whitelabelHMACKey()
	if err != nil {
		// No saved AdminToken yet — first launch on a fresh white-label
		// install can't verify until we know the key. Surface as a
		// startup warning; EndUser activation page will still work via
		// AppSettings.LockedHubURL once it's set elsewhere.
		fmt.Fprintf(os.Stderr, "whitelabel: skipping sidecar (key unavailable): %v\n", err)
		return
	}
	loader := &whitelabel.Loader{HMACKey: key}
	prof, err := loader.Load(path)
	if err != nil {
		if whitelabel.IsNoSidecar(err) {
			return
		}
		fmt.Fprintf(os.Stderr, "whitelabel: sidecar rejected: %v\n", err)
		return
	}
	settings, err := appconfig.LoadAppSettings()
	if err != nil {
		fmt.Fprintf(os.Stderr, "whitelabel: load settings: %v\n", err)
		return
	}
	// Already pinned to this Hub → no-op.
	if settings.LockedHubURL == prof.HubURL && appconfig.AppMode(settings.AppMode) == appconfig.ModeEndUser {
		return
	}
	settings.AppMode = string(appconfig.ModeEndUser)
	settings.LockedHubURL = prof.HubURL
	settings.Reseller.HubURL = prof.HubURL
	settings.Reseller.TenantSlug = prof.TenantSlug
	if err := appconfig.SaveAppSettings(settings); err != nil {
		fmt.Fprintf(os.Stderr, "whitelabel: save settings: %v\n", err)
		return
	}
}
