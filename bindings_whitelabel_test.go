package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"lurus-switch/internal/appconfig"
	"lurus-switch/internal/whitelabel"
)

// ============================
// Track 2.5 — Switch HMAC key fetched from Hub endpoint (with fallback).
// ============================
//
// Tests cover two surfaces:
//
//  1. fetchHubHMACKey alone — happy path + every error classification.
//  2. whitelabelHMACKey wiring — that fallback to the baked secret kicks
//     in on every failure mode (200-success-false, 404, malformed body,
//     transport error). The integration test routes through the real
//     fetchHubHMACKey using httptest, swapping only hubHMACKeyHTTPClient
//     and the per-test settings via t.Setenv on HOME/APPDATA.

// makeValidHexKey returns a 64-char (32-byte) hex string suitable as a
// canonical Hub HMAC key payload.
func makeValidHexKey() string {
	raw := sha256.Sum256([]byte("track-2.5-test-key"))
	return hex.EncodeToString(raw[:])
}

// hubHMACKeyServer wraps an httptest.Server with helpers for setting
// the next response. Each test gets its own server.
type hubHMACKeyServer struct {
	srv         *httptest.Server
	gotPath     string
	gotQuery    string
	gotAuth     string
	statusCode  int
	body        string
	contentType string
}

func newHubHMACKeyServer(t *testing.T) *hubHMACKeyServer {
	t.Helper()
	h := &hubHMACKeyServer{statusCode: http.StatusOK, contentType: "application/json"}
	h.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h.gotPath = r.URL.Path
		h.gotQuery = r.URL.RawQuery
		h.gotAuth = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", h.contentType)
		w.WriteHeader(h.statusCode)
		_, _ = w.Write([]byte(h.body))
	}))
	t.Cleanup(h.srv.Close)
	return h
}

// withHubHMACKeyClient installs a custom http.Client for the duration
// of a test. The default Switch client carries a 5s timeout; tests use
// a 2s budget so a hang shows up as a fail, not a 2-minute wait.
func withHubHMACKeyClient(t *testing.T, c *http.Client) {
	t.Helper()
	orig := hubHMACKeyHTTPClient
	hubHMACKeyHTTPClient = c
	t.Cleanup(func() { hubHMACKeyHTTPClient = orig })
}

// ── fetchHubHMACKey direct tests ────────────────────────────────────

func TestFetchHubHMACKey_Success(t *testing.T) {
	h := newHubHMACKeyServer(t)
	hex := makeValidHexKey()
	h.body = fmt.Sprintf(`{"success":true,"message":"","data":{"hmac_key":%q}}`, hex)
	withHubHMACKeyClient(t, &http.Client{Timeout: 2 * time.Second})

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	got, err := fetchHubHMACKey(ctx, h.srv.URL, "acme", "admin-tok")
	if err != nil {
		t.Fatalf("fetchHubHMACKey: %v", err)
	}
	if len(got) != sha256.Size {
		t.Errorf("len(got) = %d, want %d", len(got), sha256.Size)
	}
	// Sanity: the bytes round-trip the hex.
	if encoded := encodeHexHelper(got); encoded != hex {
		t.Errorf("decoded bytes round-trip mismatch: got %s want %s", encoded, hex)
	}
	// Wiring sanity: path + query + auth header all reach the Hub.
	if h.gotPath != "/api/v2/admin/whitelabel/hmac-key" {
		t.Errorf("server saw path %q, want /api/v2/admin/whitelabel/hmac-key", h.gotPath)
	}
	if !strings.Contains(h.gotQuery, "tenant_slug=acme") {
		t.Errorf("server saw query %q, want tenant_slug=acme", h.gotQuery)
	}
	if h.gotAuth != "admin-tok" {
		t.Errorf("server saw auth %q, want admin-tok", h.gotAuth)
	}
}

func TestFetchHubHMACKey_EndpointAbsent_404(t *testing.T) {
	h := newHubHMACKeyServer(t)
	h.statusCode = http.StatusNotFound
	h.body = `{"success":false,"message":"not found"}`
	withHubHMACKeyClient(t, &http.Client{Timeout: 2 * time.Second})

	_, err := fetchHubHMACKey(context.Background(), h.srv.URL, "acme", "tok")
	if !errors.Is(err, errHubHMACKeyEndpointAbsent) {
		t.Fatalf("expected errHubHMACKeyEndpointAbsent, got %v", err)
	}
}

func TestFetchHubHMACKey_EndpointAbsent_405(t *testing.T) {
	h := newHubHMACKeyServer(t)
	h.statusCode = http.StatusMethodNotAllowed
	h.body = `{"success":false,"message":"method not allowed"}`
	withHubHMACKeyClient(t, &http.Client{Timeout: 2 * time.Second})

	_, err := fetchHubHMACKey(context.Background(), h.srv.URL, "acme", "tok")
	if !errors.Is(err, errHubHMACKeyEndpointAbsent) {
		t.Fatalf("expected errHubHMACKeyEndpointAbsent on 405, got %v", err)
	}
}

func TestFetchHubHMACKey_HubError_SuccessFalse(t *testing.T) {
	h := newHubHMACKeyServer(t)
	h.body = `{"success":false,"message":"tenant not provisioned"}`
	withHubHMACKeyClient(t, &http.Client{Timeout: 2 * time.Second})

	_, err := fetchHubHMACKey(context.Background(), h.srv.URL, "acme", "tok")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if errors.Is(err, errHubHMACKeyEndpointAbsent) {
		t.Fatalf("hub-error should NOT match endpoint-absent: %v", err)
	}
	if !strings.Contains(err.Error(), "tenant not provisioned") {
		t.Errorf("error should carry hub message, got %v", err)
	}
}

func TestFetchHubHMACKey_MalformedResponse_InvalidHex(t *testing.T) {
	h := newHubHMACKeyServer(t)
	// 64 chars but the last one is non-hex 'z'.
	bogus := strings.Repeat("ab", 31) + "az"
	h.body = fmt.Sprintf(`{"success":true,"data":{"hmac_key":%q}}`, bogus)
	withHubHMACKeyClient(t, &http.Client{Timeout: 2 * time.Second})

	_, err := fetchHubHMACKey(context.Background(), h.srv.URL, "acme", "tok")
	if err == nil {
		t.Fatal("expected malformed-response error, got nil")
	}
	if !strings.Contains(err.Error(), "invalid hex") {
		t.Errorf("expected 'invalid hex' in error, got %v", err)
	}
}

func TestFetchHubHMACKey_MalformedResponse_WrongLength(t *testing.T) {
	h := newHubHMACKeyServer(t)
	// Only 32 hex chars instead of 64.
	short := strings.Repeat("ab", 16)
	h.body = fmt.Sprintf(`{"success":true,"data":{"hmac_key":%q}}`, short)
	withHubHMACKeyClient(t, &http.Client{Timeout: 2 * time.Second})

	_, err := fetchHubHMACKey(context.Background(), h.srv.URL, "acme", "tok")
	if err == nil {
		t.Fatal("expected wrong-length error, got nil")
	}
	if !strings.Contains(err.Error(), "expected") {
		t.Errorf("expected message mentioning expected length, got %v", err)
	}
}

func TestFetchHubHMACKey_MalformedResponse_NotJSON(t *testing.T) {
	h := newHubHMACKeyServer(t)
	h.contentType = "text/html"
	h.body = "<html>oops</html>"
	withHubHMACKeyClient(t, &http.Client{Timeout: 2 * time.Second})

	_, err := fetchHubHMACKey(context.Background(), h.srv.URL, "acme", "tok")
	if err == nil {
		t.Fatal("expected JSON-decode error, got nil")
	}
	if !strings.Contains(err.Error(), "malformed") {
		t.Errorf("expected 'malformed' in error, got %v", err)
	}
}

func TestFetchHubHMACKey_TransportError_IsEndpointAbsent(t *testing.T) {
	// Point at a closed port — connection refused is treated as
	// endpoint-absent so the caller falls back silently.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	closedURL := srv.URL
	srv.Close() // immediate close → subsequent requests fail to connect

	withHubHMACKeyClient(t, &http.Client{Timeout: 500 * time.Millisecond})

	_, err := fetchHubHMACKey(context.Background(), closedURL, "acme", "tok")
	if !errors.Is(err, errHubHMACKeyEndpointAbsent) {
		t.Fatalf("expected errHubHMACKeyEndpointAbsent on connection refused, got %v", err)
	}
}

func TestFetchHubHMACKey_RejectsEmptyArgs(t *testing.T) {
	cases := []struct {
		name string
		url  string
		slug string
		tok  string
	}{
		{"empty url", "", "acme", "tok"},
		{"empty slug", "https://hub.example", "", "tok"},
		{"empty token", "https://hub.example", "acme", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := fetchHubHMACKey(context.Background(), tc.url, tc.slug, tc.tok)
			if !errors.Is(err, errHubHMACKeyEndpointAbsent) {
				t.Fatalf("expected endpoint-absent, got %v", err)
			}
		})
	}
}

// ── whitelabelHMACKey integration tests ───────────────────────────

// withIsolatedAppData redirects every home-dir env var to a TempDir so
// appconfig.LoadAppSettings reads/writes a per-test file. On Windows
// the resolver consults USERPROFILE (via os.UserHomeDir) plus APPDATA;
// on Unix it uses HOME. We set them all so a single helper covers
// every platform the CI matrix can land on.
func withIsolatedAppData(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("APPDATA", dir)
	t.Setenv("HOME", dir)
	t.Setenv("USERPROFILE", dir)
	t.Setenv("XDG_CONFIG_HOME", dir)
	return dir
}

// writeResellerSettings persists a Reseller-mode app-settings.json via
// the production SaveAppSettings. This exercises the real serialization
// path and stays resilient to schema additions.
func writeResellerSettings(t *testing.T, hubURL, slug, token string) {
	t.Helper()
	s := appconfig.DefaultAppSettings()
	s.AppMode = "reseller"
	s.Reseller = appconfig.ResellerConfig{
		HubURL:     hubURL,
		AdminToken: token,
		TenantSlug: slug,
	}
	if err := appconfig.SaveAppSettings(s); err != nil {
		t.Fatalf("SaveAppSettings: %v", err)
	}
}

// TestWhitelabelHMACKey_BakedSecretEvenWhenHubReachable documents the
// intentional post-fix behavior: whitelabelHMACKey always returns the
// baked secret, even when Hub credentials are present and the Hub
// endpoint would return a valid remote key. This guarantees signing on
// a Reseller machine and verification on an EndUser machine always use
// the same key (the EndUser machine has no Hub credentials).
func TestWhitelabelHMACKey_BakedSecretEvenWhenHubReachable(t *testing.T) {
	_ = withIsolatedAppData(t)

	hexKey := makeValidHexKey()
	h := newHubHMACKeyServer(t)
	h.body = fmt.Sprintf(`{"success":true,"data":{"hmac_key":%q}}`, hexKey)
	withHubHMACKeyClient(t, &http.Client{Timeout: 2 * time.Second})

	writeResellerSettings(t, h.srv.URL, "acme", "admin-tok")

	got, err := whitelabelHMACKey()
	if err != nil {
		t.Fatalf("whitelabelHMACKey: %v", err)
	}
	expected := sha256.Sum256([]byte(whitelabelBuildSecret))
	if !bytesEqual(got, expected[:]) {
		t.Errorf("expected baked secret, got remote key or different bytes")
	}
}

func TestWhitelabelHMACKey_FallsBackOn404(t *testing.T) {
	_ = withIsolatedAppData(t)

	h := newHubHMACKeyServer(t)
	h.statusCode = http.StatusNotFound
	h.body = `{"success":false,"message":"not found"}`
	withHubHMACKeyClient(t, &http.Client{Timeout: 2 * time.Second})

	writeResellerSettings(t, h.srv.URL, "acme", "admin-tok")

	got, err := whitelabelHMACKey()
	if err != nil {
		t.Fatalf("whitelabelHMACKey: %v", err)
	}
	expected := sha256.Sum256([]byte(whitelabelBuildSecret))
	if !bytesEqual(got, expected[:]) {
		t.Errorf("expected fallback baked-secret, got different bytes")
	}
}

func TestWhitelabelHMACKey_FallsBackOnConnectionRefused(t *testing.T) {
	_ = withIsolatedAppData(t)

	// Closed server URL → connection refused.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	closedURL := srv.URL
	srv.Close()

	withHubHMACKeyClient(t, &http.Client{Timeout: 500 * time.Millisecond})
	writeResellerSettings(t, closedURL, "acme", "admin-tok")

	got, err := whitelabelHMACKey()
	if err != nil {
		t.Fatalf("whitelabelHMACKey: %v", err)
	}
	expected := sha256.Sum256([]byte(whitelabelBuildSecret))
	if !bytesEqual(got, expected[:]) {
		t.Errorf("expected fallback baked-secret on transport error")
	}
}

func TestWhitelabelHMACKey_FallsBackWhenNoResellerConfig(t *testing.T) {
	_ = withIsolatedAppData(t)
	// No settings file → LoadAppSettings returns defaults with empty
	// Reseller block. Must fall straight back to the baked secret
	// without ever attempting a network call.
	withHubHMACKeyClient(t, &http.Client{Timeout: 2 * time.Second})

	got, err := whitelabelHMACKey()
	if err != nil {
		t.Fatalf("whitelabelHMACKey: %v", err)
	}
	expected := sha256.Sum256([]byte(whitelabelBuildSecret))
	if !bytesEqual(got, expected[:]) {
		t.Errorf("expected fallback baked-secret when no Reseller config")
	}
}

func TestWhitelabelHMACKey_FallsBackOnMalformedResponse(t *testing.T) {
	_ = withIsolatedAppData(t)

	h := newHubHMACKeyServer(t)
	h.body = `not even json`
	withHubHMACKeyClient(t, &http.Client{Timeout: 2 * time.Second})
	writeResellerSettings(t, h.srv.URL, "acme", "admin-tok")

	got, err := whitelabelHMACKey()
	if err != nil {
		t.Fatalf("whitelabelHMACKey: %v", err)
	}
	expected := sha256.Sum256([]byte(whitelabelBuildSecret))
	if !bytesEqual(got, expected[:]) {
		t.Errorf("expected fallback baked-secret on malformed response")
	}
}

// ── build-sign → EndUser-verify round-trip ─────────────────────────

// TestWhitelabelHMACKey_ResellerSignEndUserVerify_RoundTrip is the
// critical E2E contract test: a sidecar signed with whitelabelHMACKey()
// on a Reseller machine (full config present, Hub available) must be
// verifiable by whitelabelHMACKey() on a fresh EndUser machine (no
// Reseller config). Before the fix this failed because the two calls
// returned different keys (Hub remote key vs baked secret).
func TestWhitelabelHMACKey_ResellerSignEndUserVerify_RoundTrip(t *testing.T) {
	tmp := t.TempDir()
	base := filepath.Join(tmp, "switch-base.exe")
	if err := os.WriteFile(base, []byte("STUB BINARY CONTENT"), 0o644); err != nil {
		t.Fatalf("seed base: %v", err)
	}

	// — Reseller side: Hub is reachable and returns a remote key.
	remoteHexKey := makeValidHexKey()
	h := newHubHMACKeyServer(t)
	h.body = fmt.Sprintf(`{"success":true,"data":{"hmac_key":%q}}`, remoteHexKey)
	withHubHMACKeyClient(t, &http.Client{Timeout: 2 * time.Second})

	resellerDir := t.TempDir()
	t.Setenv("APPDATA", resellerDir)
	t.Setenv("HOME", resellerDir)
	t.Setenv("USERPROFILE", resellerDir)
	t.Setenv("XDG_CONFIG_HOME", resellerDir)
	writeResellerSettings(t, h.srv.URL, "acme", "admin-tok")

	signingKey, err := whitelabelHMACKey()
	if err != nil {
		t.Fatalf("Reseller: whitelabelHMACKey: %v", err)
	}

	out := filepath.Join(tmp, "out")
	res, err := whitelabel.Build(whitelabel.BuildOpts{
		Profile: whitelabel.Profile{
			BrandName:  "Acme Corp",
			HubURL:     "https://hub.acme.example",
			TenantSlug: "acme",
		},
		HMACKey:        signingKey,
		BaseBinaryPath: base,
		OutputDir:      out,
	})
	if err != nil {
		t.Fatalf("whitelabel.Build: %v", err)
	}

	// — EndUser side: fresh machine, no Reseller config.
	endUserDir := t.TempDir()
	t.Setenv("APPDATA", endUserDir)
	t.Setenv("HOME", endUserDir)
	t.Setenv("USERPROFILE", endUserDir)
	t.Setenv("XDG_CONFIG_HOME", endUserDir)

	verifyKey, err := whitelabelHMACKey()
	if err != nil {
		t.Fatalf("EndUser: whitelabelHMACKey: %v", err)
	}

	loader := &whitelabel.Loader{HMACKey: verifyKey}
	prof, err := loader.Load(res.SidecarPath)
	if err != nil {
		t.Fatalf("EndUser: Loader.Load: %v (signing key == remote: %v)",
			err, encodeHexHelper(signingKey) == remoteHexKey)
	}
	if prof.HubURL != "https://hub.acme.example" {
		t.Errorf("HubURL round-trip mismatch: %q", prof.HubURL)
	}
}

// TestWhitelabelHMACKey_AlwaysBakedSecret ensures whitelabelHMACKey
// returns the sha256(whitelabelBuildSecret) key in all cases — both
// when Reseller config is absent AND when it is present (even if Hub
// is reachable). This guarantees signing and verification always use
// the same deterministic key regardless of machine state.
func TestWhitelabelHMACKey_AlwaysBakedSecret(t *testing.T) {
	bakedKey := sha256.Sum256([]byte(whitelabelBuildSecret))
	expected := bakedKey[:]

	t.Run("no_reseller_config", func(t *testing.T) {
		_ = withIsolatedAppData(t)
		got, err := whitelabelHMACKey()
		if err != nil {
			t.Fatalf("whitelabelHMACKey: %v", err)
		}
		if !bytesEqual(got, expected) {
			t.Errorf("expected baked secret, got different bytes")
		}
	})

	t.Run("reseller_config_present_hub_reachable", func(t *testing.T) {
		_ = withIsolatedAppData(t)
		// Hub is reachable and returns a valid remote key — but we still
		// expect the baked secret after the fix.
		hexKey := makeValidHexKey()
		h := newHubHMACKeyServer(t)
		h.body = fmt.Sprintf(`{"success":true,"data":{"hmac_key":%q}}`, hexKey)
		withHubHMACKeyClient(t, &http.Client{Timeout: 2 * time.Second})
		writeResellerSettings(t, h.srv.URL, "acme", "admin-tok")

		got, err := whitelabelHMACKey()
		if err != nil {
			t.Fatalf("whitelabelHMACKey: %v", err)
		}
		if !bytesEqual(got, expected) {
			t.Errorf("expected baked secret even when Hub reachable, got remote key")
		}
	})
}

// ── helpers ────────────────────────────────────────────────────────

func encodeHexHelper(b []byte) string { return hex.EncodeToString(b) }

func bytesEqual(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
