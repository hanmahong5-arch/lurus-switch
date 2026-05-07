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

	"github.com/tc-hib/winres"
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

	// IconPath is an optional .ico file to embed. When empty, the base
	// binary's original icon is preserved. When set, every RT_GROUP_ICON
	// resource in the PE is replaced with the new icon (pure-Go patch via
	// github.com/tc-hib/winres — works on any Windows-targeted PE, not
	// just Wails builds).
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

	// Icon replacement — best effort. The new icon is patched into the PE
	// .rsrc section in pure Go. Failures are surfaced to the operator as
	// notes rather than aborting the whole build, because a missing icon
	// is cosmetic; the binary still runs and the sidecar still validates.
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

// tryReplaceIcon patches the .rsrc section of targetExe so every existing
// RT_GROUP_ICON entry points to the images from iconPath. Pure Go — no
// rcedit / ResourceHacker / cgo dependency.
//
// Returns ("", nil) on a successful patch, a non-empty note for an
// intentional skip (e.g. the base PE has no icon resources to replace),
// or an error when the patch was attempted but failed.
//
// The base binary is preserved on failure: the rewrite is staged in a
// temp file and only renamed over targetExe once the new PE is fully
// written. A pre-flight signature check on targetExe means signed bases
// surface a clear error rather than producing an exe with an invalid
// signature trailer (Authenticode signing of white-label builds is a
// post-build step the Reseller workflow handles separately).
func tryReplaceIcon(targetExe, iconPath string) (string, error) {
	if _, err := os.Stat(iconPath); err != nil {
		return "", fmt.Errorf("icon %q not found: %w", iconPath, err)
	}

	// Refuse to touch a signed binary. Patching strips/invalidates the
	// Authenticode trailer; better to fail loudly than ship a broken sig.
	exeFile, err := os.Open(targetExe)
	if err != nil {
		return "", fmt.Errorf("open target exe: %w", err)
	}
	signed, sigErr := winres.IsSignedEXE(exeFile)
	if sigErr != nil {
		exeFile.Close()
		// Likely "not a PE" — caller probably handed us a non-Windows
		// base on a non-Windows host. Skip with a note rather than fail
		// the whole build; the binary still runs, just with the original
		// icon.
		return "base binary is not a Windows PE — icon replacement skipped", nil
	}
	if signed {
		exeFile.Close()
		return "", errors.New("base binary is Authenticode-signed; sign the white-label exe AFTER building, not before")
	}

	// Load the existing resource set so we can see which RT_GROUP_ICON
	// resIDs the base actually uses (Wails uses ID(3); other toolchains
	// may differ). Replacing in place — same resIDs, same languages —
	// keeps any code that resolves icons by ID still working.
	if _, err := exeFile.Seek(0, io.SeekStart); err != nil {
		exeFile.Close()
		return "", fmt.Errorf("rewind exe: %w", err)
	}
	rs, err := winres.LoadFromEXE(exeFile)
	exeFile.Close()
	if err != nil {
		if errors.Is(err, winres.ErrNoResources) {
			// No .rsrc section at all — there's nothing to replace and
			// adding a fresh resource section here would be a much
			// bigger surgery. Skip with a note.
			return "base binary has no resource section — icon replacement skipped", nil
		}
		return "", fmt.Errorf("read base resources: %w", err)
	}

	// Load the new icon images.
	icoFile, err := os.Open(iconPath)
	if err != nil {
		return "", fmt.Errorf("open icon: %w", err)
	}
	newIcon, err := winres.LoadICO(icoFile)
	icoFile.Close()
	if err != nil {
		return "", fmt.Errorf("parse %q as ICO: %w", iconPath, err)
	}

	// Collect every RT_GROUP_ICON identifier in the base. Replacing them
	// in a separate pass after collection avoids mutating the map while
	// Walk is iterating.
	var groupIDs []winres.Identifier
	rs.WalkType(winres.RT_GROUP_ICON, func(resID winres.Identifier, _ uint16, _ []byte) bool {
		groupIDs = append(groupIDs, resID)
		return true
	})
	if len(groupIDs) == 0 {
		return "base binary has no RT_GROUP_ICON entries — icon replacement skipped", nil
	}
	for _, id := range groupIDs {
		if err := rs.SetIcon(id, newIcon); err != nil {
			return "", fmt.Errorf("set icon for resID %v: %w", id, err)
		}
	}

	// Stream the patched PE through a temp file. Atomic rename means a
	// crash mid-write doesn't corrupt the binary the build already
	// produced.
	src, err := os.Open(targetExe)
	if err != nil {
		return "", fmt.Errorf("reopen target exe: %w", err)
	}
	tmp := targetExe + ".rsrc.tmp"
	dst, err := os.Create(tmp)
	if err != nil {
		_ = src.Close()
		return "", fmt.Errorf("create temp exe: %w", err)
	}
	if err := rs.WriteToEXE(dst, src); err != nil {
		_ = dst.Close()
		_ = src.Close()
		_ = os.Remove(tmp)
		return "", fmt.Errorf("patch exe: %w", err)
	}
	if err := dst.Close(); err != nil {
		_ = src.Close()
		_ = os.Remove(tmp)
		return "", fmt.Errorf("close temp exe: %w", err)
	}
	// Close the source before rename — Windows won't let us replace a
	// file with an open read handle.
	if err := src.Close(); err != nil {
		_ = os.Remove(tmp)
		return "", fmt.Errorf("close source exe: %w", err)
	}
	if err := os.Rename(tmp, targetExe); err != nil {
		_ = os.Remove(tmp)
		return "", fmt.Errorf("swap patched exe: %w", err)
	}

	return "", nil
}
