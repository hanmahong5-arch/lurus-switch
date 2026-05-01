package whitelabel

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// SidecarFilename is the standard name the EndUser binary looks for next
// to itself. Hardcoded — there's no Reseller knob for this.
const SidecarFilename = "whitelabel.json"

// BuildResult is what the packager returns to the UI for display +
// distribution.
type BuildResult struct {
	// OutputDir is where the binary + sidecar were written.
	OutputDir string `json:"output_dir"`
	// BinaryPath is the absolute path of the (renamed) Switch exe.
	BinaryPath string `json:"binary_path"`
	// SidecarPath is the absolute path of whitelabel.json.
	SidecarPath string `json:"sidecar_path"`
	// SHA256 is the hex digest of the binary, for distribution integrity.
	SHA256 string `json:"sha256"`
	// SidecarSHA256 is the hex digest of the sidecar.
	SidecarSHA256 string `json:"sidecar_sha256"`
	// Notes carries non-fatal messages — e.g. "icon replacement skipped:
	// rcedit not on PATH". The UI surfaces these as warnings.
	Notes []string `json:"notes,omitempty"`
}

// BuildOpts is the call-site configuration for Build().
type BuildOpts struct {
	// Profile holds branding inputs. Mutated by Build (timestamps + HMAC),
	// caller should pass a fresh struct each call.
	Profile Profile

	// HMACKey is used to sign + verify the sidecar. In production this
	// comes from the Hub admin endpoint; for testing or air-gapped builds
	// the Reseller can paste a key directly.
	HMACKey []byte

	// BaseBinaryPath is the source Switch exe. Required.
	BaseBinaryPath string

	// OutputDir is where the result goes. Created if missing.
	OutputDir string

	// IconPath is an optional .ico file to embed (Windows only). When
	// empty or rcedit isn't available, the original icon is preserved
	// and a note is added to BuildResult.Notes.
	IconPath string
}

// brandSlugPattern matches the safe character set for filenames derived
// from BrandName. Anything else is replaced with `-`.
var brandSlugPattern = regexp.MustCompile(`[^a-zA-Z0-9._-]+`)

// Build produces a branded Switch distribution: copies the base binary,
// optionally swaps its icon, and writes the signed sidecar. Idempotent
// — running twice with the same inputs produces byte-identical outputs.
func Build(opts BuildOpts) (*BuildResult, error) {
	if err := opts.Profile.Validate(); err != nil {
		return nil, fmt.Errorf("profile invalid: %w", err)
	}
	if len(opts.HMACKey) == 0 {
		return nil, errors.New("hmac key is required")
	}
	if opts.BaseBinaryPath == "" {
		return nil, errors.New("base binary path is required")
	}
	if _, err := os.Stat(opts.BaseBinaryPath); err != nil {
		return nil, fmt.Errorf("base binary not accessible: %w", err)
	}

	// Validate logo size early — cheap check that prevents a 50MB sidecar.
	if opts.Profile.LogoBase64 != "" {
		decoded, err := base64.StdEncoding.DecodeString(opts.Profile.LogoBase64)
		if err != nil {
			return nil, fmt.Errorf("logo_base64 is not valid base64: %w", err)
		}
		if len(decoded) > MaxLogoBytes {
			return nil, fmt.Errorf("logo exceeds %d bytes after decoding (got %d)", MaxLogoBytes, len(decoded))
		}
	}

	// Stamp version + creation time so the same Profile struct can be
	// passed in multiple times, with timestamp the only diff.
	opts.Profile.Version = SidecarVersion
	if opts.Profile.CreatedAt.IsZero() {
		// Truncate to seconds so a build run that takes 200ms doesn't
		// produce a different sidecar than one that takes 800ms.
		opts.Profile.CreatedAt = time.Now().UTC().Truncate(time.Second)
	}

	if err := os.MkdirAll(opts.OutputDir, 0o755); err != nil {
		return nil, fmt.Errorf("create output dir: %w", err)
	}

	binaryName := slugify(opts.Profile.BrandName) + "-Switch"
	if strings.HasSuffix(strings.ToLower(opts.BaseBinaryPath), ".exe") {
		binaryName += ".exe"
	}
	binaryPath := filepath.Join(opts.OutputDir, binaryName)

	notes := []string{}

	if err := copyFile(opts.BaseBinaryPath, binaryPath); err != nil {
		return nil, fmt.Errorf("copy base binary: %w", err)
	}

	// Icon replacement — best effort. rcedit is the Windows-side tool
	// the build pipeline expects. When it isn't present (non-Windows
	// host, dev machine without it installed), we skip and document.
	if opts.IconPath != "" {
		note, err := tryReplaceIcon(binaryPath, opts.IconPath)
		if err != nil {
			notes = append(notes, "icon replacement failed: "+err.Error())
		} else if note != "" {
			notes = append(notes, note)
		}
	}

	// Sign + write the sidecar.
	hmacSig, err := opts.Profile.Sign(opts.HMACKey)
	if err != nil {
		return nil, fmt.Errorf("sign sidecar: %w", err)
	}
	opts.Profile.HMAC = hmacSig
	sidecarBytes, err := json.MarshalIndent(&opts.Profile, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal sidecar: %w", err)
	}
	sidecarPath := filepath.Join(opts.OutputDir, SidecarFilename)
	if err := os.WriteFile(sidecarPath, sidecarBytes, 0o644); err != nil {
		return nil, fmt.Errorf("write sidecar: %w", err)
	}

	binSum, err := fileSHA256(binaryPath)
	if err != nil {
		return nil, fmt.Errorf("hash binary: %w", err)
	}
	sideSum, err := fileSHA256(sidecarPath)
	if err != nil {
		return nil, fmt.Errorf("hash sidecar: %w", err)
	}

	return &BuildResult{
		OutputDir:     opts.OutputDir,
		BinaryPath:    binaryPath,
		SidecarPath:   sidecarPath,
		SHA256:        binSum,
		SidecarSHA256: sideSum,
		Notes:         notes,
	}, nil
}

// slugify reduces an arbitrary brand name to a filename-safe slug. Falls
// back to "switch" when the input contains zero usable characters.
func slugify(s string) string {
	out := brandSlugPattern.ReplaceAllString(strings.TrimSpace(s), "-")
	out = strings.Trim(out, "-")
	if out == "" {
		return "switch"
	}
	return strings.ToLower(out)
}

// copyFile is a buffered, atomic file copy. Atomic via tmp+rename so a
// crash mid-build doesn't leave a half-written exe in OutputDir.
func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	tmp := dst + ".tmp"
	out, err := os.Create(tmp)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		_ = os.Remove(tmp)
		return err
	}
	if err := out.Close(); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return os.Rename(tmp, dst)
}

// fileSHA256 streams the file through a SHA-256 hasher. Used for both
// the integrity-check display and any future "verify-on-launch" logic.
func fileSHA256(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// tryReplaceIcon is the rcedit shim. Returns ("", nil) on success, a
// note string on intentional skip, or an error on failure.
//
// Currently a stub: rcedit integration depends on shipping rcedit-x64.exe
// alongside Switch and is deferred to a follow-up sprint where we can
// test on a real Windows host with the tool installed. The stub returns
// a note so the BuildResult surfaces the deferral to the operator.
func tryReplaceIcon(_, iconPath string) (string, error) {
	if _, err := os.Stat(iconPath); err != nil {
		return "", fmt.Errorf("icon %q not found: %w", iconPath, err)
	}
	return "icon replacement deferred: rcedit integration not yet wired (tracked in S-Xc.1 follow-up)", nil
}
