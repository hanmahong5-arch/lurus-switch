package whitelabel

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"image"
	"image/color"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/tc-hib/winres"
)

// TestProfile_SignVerify_RoundTrip locks in the contract that Sign output
// is what Verify expects. Without this, an HMAC implementation drift
// (e.g. accidentally including HMAC field in the canonical bytes) would
// be silently incompatible across the same process — bad for diagnosis.
func TestProfile_SignVerify_RoundTrip(t *testing.T) {
	key := sha256Bytes("rotate-me")
	p := &Profile{
		Version:      SidecarVersion,
		BrandName:    "Acme",
		HubURL:       "https://hub.acme.example",
		PrimaryColor: "#9333ea",
	}
	sig, err := p.Sign(key)
	if err != nil {
		t.Fatalf("Sign: %v", err)
	}
	p.HMAC = sig
	if err := p.Verify(key); err != nil {
		t.Errorf("Verify after Sign: %v", err)
	}

	// A different key must reject.
	if err := p.Verify(sha256Bytes("other")); err == nil {
		t.Error("Verify accepted wrong key")
	}
}

// TestProfile_Verify_DetectsTampering ensures any change to the signed
// fields after signing is caught — that's the entire point of the HMAC.
func TestProfile_Verify_DetectsTampering(t *testing.T) {
	key := sha256Bytes("rotate-me")
	p := &Profile{
		Version:   SidecarVersion,
		BrandName: "Acme",
		HubURL:    "https://hub.acme.example",
	}
	sig, _ := p.Sign(key)
	p.HMAC = sig

	p.HubURL = "https://attacker.example"
	if err := p.Verify(key); err == nil {
		t.Error("Verify accepted tampered HubURL")
	}
}

// TestProfile_Validate_RequiresKeyFields walks the validator's rules so
// the UI can rely on early feedback instead of HTTP errors at deploy time.
func TestProfile_Validate_RequiresKeyFields(t *testing.T) {
	cases := []struct {
		name string
		p    Profile
		ok   bool
	}{
		{"complete", Profile{BrandName: "A", HubURL: "https://h"}, true},
		{"missing brand", Profile{HubURL: "https://h"}, false},
		{"missing hub", Profile{BrandName: "A"}, false},
		{"non-http hub", Profile{BrandName: "A", HubURL: "ftp://h"}, false},
		{"good color", Profile{BrandName: "A", HubURL: "https://h", PrimaryColor: "#abc"}, true},
		{"bad color", Profile{BrandName: "A", HubURL: "https://h", PrimaryColor: "potato"}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.p.Validate()
			if tc.ok && err != nil {
				t.Errorf("expected ok, got %v", err)
			}
			if !tc.ok && err == nil {
				t.Error("expected error, got nil")
			}
		})
	}
}

// TestBuild_RoundTrip exercises the full Build → Loader.Load path.
func TestBuild_RoundTrip(t *testing.T) {
	tmp := t.TempDir()
	base := filepath.Join(tmp, "switch-base.exe")
	if err := os.WriteFile(base, []byte("STUB BINARY CONTENT"), 0o644); err != nil {
		t.Fatalf("seed base: %v", err)
	}

	out := filepath.Join(tmp, "out")
	res, err := Build(BuildOpts{
		Profile: Profile{
			BrandName:    "Acme Corp",
			HubURL:       "https://hub.acme.example",
			TenantSlug:   "acme",
			PrimaryColor: "#9333ea",
		},
		HMACKey:        sha256Bytes("k"),
		BaseBinaryPath: base,
		OutputDir:      out,
	})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if !strings.HasSuffix(res.BinaryPath, ".exe") {
		t.Errorf("binary path %q missing .exe suffix (base had .exe)", res.BinaryPath)
	}
	if !strings.Contains(filepath.Base(res.BinaryPath), "acme-corp") {
		t.Errorf("binary path %q did not include slug", res.BinaryPath)
	}

	loader := &Loader{HMACKey: sha256Bytes("k")}
	loaded, err := loader.Load(res.SidecarPath)
	if err != nil {
		t.Fatalf("Loader.Load: %v", err)
	}
	if loaded.HubURL != "https://hub.acme.example" {
		t.Errorf("HubURL round-trip mismatch: %q", loaded.HubURL)
	}
	if loaded.BrandName != "Acme Corp" {
		t.Errorf("BrandName round-trip mismatch: %q", loaded.BrandName)
	}
}

// TestBuild_RejectsOversizeLogo ensures the operator-facing error is
// helpful rather than producing a 50MB sidecar that fails at the deploy
// boundary.
func TestBuild_RejectsOversizeLogo(t *testing.T) {
	tmp := t.TempDir()
	base := filepath.Join(tmp, "switch.exe")
	_ = os.WriteFile(base, []byte("X"), 0o644)

	huge := strings.Repeat("A", MaxLogoBytes+1)
	encoded := base64.StdEncoding.EncodeToString([]byte(huge))
	_, err := Build(BuildOpts{
		Profile:        Profile{BrandName: "A", HubURL: "https://h", LogoBase64: encoded},
		HMACKey:        sha256Bytes("k"),
		BaseBinaryPath: base,
		OutputDir:      filepath.Join(tmp, "out"),
	})
	if err == nil || !strings.Contains(err.Error(), "logo exceeds") {
		t.Errorf("expected logo size error, got %v", err)
	}
}

// TestBuild_Idempotent verifies that running Build twice with the same
// inputs (CreatedAt pinned) produces byte-identical sidecars. Important
// because Resellers re-pack after each Switch upgrade — drifting hashes
// would break their own distribution-manifest tooling.
func TestBuild_Idempotent(t *testing.T) {
	tmp := t.TempDir()
	base := filepath.Join(tmp, "switch.exe")
	_ = os.WriteFile(base, []byte("STUB"), 0o644)

	pinned, _ := time.Parse(time.RFC3339, "2026-01-01T00:00:00Z")
	prof := Profile{
		BrandName: "Acme",
		HubURL:    "https://h",
		CreatedAt: pinned,
	}
	build := func() string {
		out := t.TempDir()
		res, err := Build(BuildOpts{
			Profile:        prof,
			HMACKey:        sha256Bytes("k"),
			BaseBinaryPath: base,
			OutputDir:      out,
		})
		if err != nil {
			t.Fatalf("Build: %v", err)
		}
		return res.SidecarSHA256
	}
	a, b := build(), build()
	if a != b {
		t.Errorf("idempotency broken: %s vs %s", a, b)
	}
}

// TestLoader_NoSidecar_DistinctError exposes the contract the bindings
// layer relies on to tell "not a white-label build" from "sidecar
// rejected" — so the latter can hard-fail while the former silently proceeds.
func TestLoader_NoSidecar_DistinctError(t *testing.T) {
	loader := &Loader{HMACKey: sha256Bytes("k")}
	_, err := loader.Load(filepath.Join(t.TempDir(), "nope.json"))
	if !IsNoSidecar(err) {
		t.Errorf("expected ErrNoSidecar, got %v", err)
	}
}

// TestLoader_RejectsUnsupportedVersion catches a future schema-bump from
// being silently accepted by an old EndUser binary.
func TestLoader_RejectsUnsupportedVersion(t *testing.T) {
	tmp := t.TempDir()
	p := Profile{
		Version:   SidecarVersion + 5,
		BrandName: "A", HubURL: "https://h",
	}
	sig, _ := p.Sign(sha256Bytes("k"))
	p.HMAC = sig
	raw, _ := json.Marshal(&p)
	path := filepath.Join(tmp, SidecarFilename)
	_ = os.WriteFile(path, raw, 0o644)

	loader := &Loader{HMACKey: sha256Bytes("k")}
	_, err := loader.Load(path)
	if err == nil || !strings.Contains(err.Error(), "version") {
		t.Errorf("expected version error, got %v", err)
	}
}

// TestLoader_RejectsTamperedSidecar — full integration view: write a
// good sidecar, mutate its HubURL on disk, expect verification to fail.
func TestLoader_RejectsTamperedSidecar(t *testing.T) {
	tmp := t.TempDir()
	p := Profile{
		Version:   SidecarVersion,
		BrandName: "Acme", HubURL: "https://hub.acme.example",
	}
	sig, _ := p.Sign(sha256Bytes("k"))
	p.HMAC = sig
	raw, _ := json.Marshal(&p)
	path := filepath.Join(tmp, SidecarFilename)
	_ = os.WriteFile(path, raw, 0o644)

	tampered := strings.Replace(string(raw), "hub.acme.example", "attacker.example", 1)
	_ = os.WriteFile(path, []byte(tampered), 0o644)

	loader := &Loader{HMACKey: sha256Bytes("k")}
	_, err := loader.Load(path)
	if err == nil {
		t.Error("expected verification failure on tampered sidecar")
	}
	if errors.Is(err, ErrNoSidecar) {
		t.Error("tampered should not be classified as missing")
	}
}

func sha256Bytes(s string) []byte {
	h := sha256.Sum256([]byte(s))
	return h[:]
}

// TestTryReplaceIcon_MissingIcon hard-fails fast when the operator points
// at a non-existent icon — without this guard the real failure surfaces
// later from inside winres with a confusing message.
func TestTryReplaceIcon_MissingIcon(t *testing.T) {
	tmp := t.TempDir()
	exe := filepath.Join(tmp, "fake.exe")
	if err := os.WriteFile(exe, []byte("STUB"), 0o644); err != nil {
		t.Fatalf("seed exe: %v", err)
	}
	_, err := tryReplaceIcon(exe, filepath.Join(tmp, "nope.ico"))
	if err == nil {
		t.Fatal("expected error for missing icon, got nil")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected 'not found' in error, got %q", err)
	}
}

// TestTryReplaceIcon_NonPEBase verifies the "skip with note" path the
// caller relies on: when the base binary isn't a PE (dev machine running
// a non-Windows base, or a stub used in another test), the build still
// produces a working sidecar and the operator gets a clear note.
func TestTryReplaceIcon_NonPEBase(t *testing.T) {
	tmp := t.TempDir()
	exe := filepath.Join(tmp, "not-a-pe.exe")
	if err := os.WriteFile(exe, []byte("definitely not a PE"), 0o644); err != nil {
		t.Fatalf("seed exe: %v", err)
	}
	icoPath := writeTestICO(t, filepath.Join(tmp, "brand.ico"))

	note, err := tryReplaceIcon(exe, icoPath)
	if err != nil {
		t.Fatalf("expected nil error on non-PE skip, got %v", err)
	}
	if note == "" || !strings.Contains(note, "skipped") {
		t.Errorf("expected skip note, got %q", note)
	}
}

// TestBuild_IconReplacementSurfacedAsNote checks the wiring from BuildOpts
// through Build() to BuildResult.Notes — operators rely on Notes to learn
// when icon replacement was a no-op so they can act on it.
func TestBuild_IconReplacementSurfacedAsNote(t *testing.T) {
	tmp := t.TempDir()
	base := filepath.Join(tmp, "switch.exe")
	if err := os.WriteFile(base, []byte("STUB"), 0o644); err != nil {
		t.Fatalf("seed base: %v", err)
	}
	icoPath := writeTestICO(t, filepath.Join(tmp, "brand.ico"))

	res, err := Build(BuildOpts{
		Profile:        Profile{BrandName: "Acme", HubURL: "https://h"},
		HMACKey:        sha256Bytes("k"),
		BaseBinaryPath: base,
		OutputDir:      filepath.Join(tmp, "out"),
		IconPath:       icoPath,
	})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if len(res.Notes) == 0 {
		t.Fatal("expected a note about icon replacement skip on stub binary, got none")
	}
	found := false
	for _, n := range res.Notes {
		if strings.Contains(n, "skipped") || strings.Contains(n, "replacement") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("notes did not mention icon outcome: %v", res.Notes)
	}
}

// TestTryReplaceIcon_PatchesPEInPlace is the positive integration test:
// synthesize a real PE with a known starter icon, run tryReplaceIcon, and
// verify the result still parses as a PE and the GROUP_ICON entry now has
// the dimensions of the replacement icon.
//
// We use a real PE fixture from the test binary itself — Go test binaries
// on Windows are valid PEs, so we can copy the test exe, seed it with a
// starter icon via winres, then exercise the replace path.
func TestTryReplaceIcon_PatchesPEInPlace(t *testing.T) {
	self, err := os.Executable()
	if err != nil {
		t.Skipf("os.Executable unavailable: %v", err)
	}
	if !looksLikePE(t, self) {
		t.Skip("test binary is not a PE on this host")
	}

	tmp := t.TempDir()
	target := filepath.Join(tmp, "fixture.exe")
	if err := copyFile(self, target); err != nil {
		t.Fatalf("copy test binary: %v", err)
	}

	// Seed: add an initial 16x16 icon to the copied PE so tryReplaceIcon
	// has something to replace. If the copied binary already has a .rsrc
	// section we extend it; otherwise we get ErrNoResources and skip
	// (Go test binaries on Windows commonly have no resources, in which
	// case the positive path can't be exercised here — that gap is fine
	// because the negative tests above already cover the wiring).
	if err := seedIconResource(target, 16); err != nil {
		if errors.Is(err, winres.ErrNoResources) {
			t.Skip("test PE has no .rsrc section; positive patch path is exercised by real wails-built binaries in production")
		}
		t.Fatalf("seed icon: %v", err)
	}

	// Now build a 32x32 replacement icon and patch.
	icoPath := writeTestICOAtSize(t, filepath.Join(tmp, "brand32.ico"), 32)
	note, err := tryReplaceIcon(target, icoPath)
	if err != nil {
		t.Fatalf("tryReplaceIcon: %v", err)
	}
	if note != "" {
		t.Fatalf("expected clean replacement, got note %q", note)
	}

	// Verify the patched file is still a parseable PE with a GROUP_ICON
	// of size 32 (proves the replacement actually happened).
	patched, err := os.Open(target)
	if err != nil {
		t.Fatalf("reopen patched: %v", err)
	}
	defer patched.Close()
	rs, err := winres.LoadFromEXE(patched)
	if err != nil {
		t.Fatalf("LoadFromEXE on patched: %v", err)
	}
	var sawIcon bool
	rs.WalkType(winres.RT_GROUP_ICON, func(resID winres.Identifier, _ uint16, data []byte) bool {
		// First 6 bytes are the GROUP_ICONDIR header; entries follow at
		// offset 6 with Width/Height as the first two bytes.
		if len(data) < 8 {
			return true
		}
		// Width is the byte at offset 6.
		w := data[6]
		// 0 in PE icon dirs means 256 — for 32 we expect a literal 32.
		if w == 32 {
			sawIcon = true
		}
		return true
	})
	if !sawIcon {
		t.Errorf("expected RT_GROUP_ICON with width 32 in patched PE, none found")
	}
}

// writeTestICO emits a minimal valid .ico with one 32x32 image and
// returns the path written. Used by tests that need a real .ico file
// without committing a binary fixture.
func writeTestICO(t *testing.T, path string) string {
	t.Helper()
	return writeTestICOAtSize(t, path, 32)
}

func writeTestICOAtSize(t *testing.T, path string, size int) string {
	t.Helper()
	img := image.NewNRGBA(image.Rect(0, 0, size, size))
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			img.Set(x, y, color.NRGBA{R: 0x9c, G: 0x33, B: 0xea, A: 0xff})
		}
	}
	icon, err := winres.NewIconFromImages([]image.Image{img})
	if err != nil {
		t.Fatalf("build test icon: %v", err)
	}
	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("create test icon file: %v", err)
	}
	defer f.Close()
	if err := icon.SaveICO(f); err != nil {
		t.Fatalf("write test icon: %v", err)
	}
	return path
}

// looksLikePE returns true when path is a parseable Windows PE. Used to
// gate the positive-path test on hosts where the test binary happens to
// be a different format.
func looksLikePE(t *testing.T, path string) bool {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()
	_, err = winres.IsSignedEXE(f)
	return err == nil
}

// seedIconResource adds a starter icon to an existing PE by reading its
// resource set, adding a fresh RT_GROUP_ICON, and writing the result back
// over the file. Returns ErrNoResources if the PE has no .rsrc section
// to extend, since adding a brand new section is not supported by the
// underlying library.
func seedIconResource(target string, size int) error {
	src, err := os.Open(target)
	if err != nil {
		return err
	}
	rs, err := winres.LoadFromEXE(src)
	src.Close()
	if err != nil {
		return err
	}

	img := image.NewNRGBA(image.Rect(0, 0, size, size))
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			img.Set(x, y, color.NRGBA{R: 0x10, G: 0x10, B: 0x10, A: 0xff})
		}
	}
	icon, err := winres.NewIconFromImages([]image.Image{img})
	if err != nil {
		return err
	}
	if err := rs.SetIcon(winres.RT_ICON, icon); err != nil {
		return err
	}

	src, err = os.Open(target)
	if err != nil {
		return err
	}
	defer src.Close()
	var buf bytes.Buffer
	if err := rs.WriteToEXE(&buf, src); err != nil {
		return err
	}
	return os.WriteFile(target, buf.Bytes(), 0o644)
}
