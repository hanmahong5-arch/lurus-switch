package whitelabel

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
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
