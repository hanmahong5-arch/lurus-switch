package toolmanifest

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

func TestCurrentPlatform(t *testing.T) {
	got := CurrentPlatform()
	if got == "" {
		t.Fatal("CurrentPlatform returned empty string")
	}
	// Must contain exactly one "/"
	if len(got) < 3 || got[0] == '/' || got[len(got)-1] == '/' {
		t.Fatalf("unexpected format: %q", got)
	}
	// On Windows ARM64, expect "windows/amd64" (mapped)
	if runtime.GOOS == "windows" && runtime.GOARCH == "arm64" {
		if got != "windows/amd64" {
			t.Fatalf("expected windows/amd64 for Windows ARM64, got %q", got)
		}
	}
}

func TestIsSupportedPlatform(t *testing.T) {
	ok, reason := IsSupportedPlatform()
	// On 64-bit systems this should always return true
	if runtime.GOARCH != "386" {
		if !ok {
			t.Fatalf("expected supported, got unsupported with reason: %s", reason)
		}
		if reason != "" {
			t.Fatalf("expected empty reason, got %q", reason)
		}
	}
}

func TestBuiltin(t *testing.T) {
	mf := Builtin()
	if mf == nil {
		t.Fatal("Builtin returned nil")
	}
	if len(mf.Tools) == 0 {
		t.Fatal("Builtin manifest has no tools")
	}
	if mf.GeneratedAt == "" {
		t.Fatal("Builtin manifest has empty generated_at")
	}
	// Verify key tools exist
	for _, name := range []string{"claude", "picoclaw", "nullclaw", "zeroclaw"} {
		if _, ok := mf.Tools[name]; !ok {
			t.Errorf("missing expected tool %q in builtin manifest", name)
		}
	}
}

func TestManifest_GetPlatformURL(t *testing.T) {
	mf := &Manifest{
		Tools: map[string]ToolEntry{
			"test-tool": {
				Type: "binary",
				Platforms: map[string]PlatformAsset{
					"windows/amd64": {URL: "https://example.com/win.zip", SHA256: "abc123"},
					"linux/amd64":   {URL: "https://example.com/linux.tar.gz"},
				},
			},
		},
	}

	if url := mf.GetPlatformURL("test-tool", "windows/amd64"); url != "https://example.com/win.zip" {
		t.Errorf("unexpected URL: %q", url)
	}
	if url := mf.GetPlatformURL("test-tool", "darwin/arm64"); url != "" {
		t.Errorf("expected empty for missing platform, got %q", url)
	}
	if url := mf.GetPlatformURL("nonexistent", "windows/amd64"); url != "" {
		t.Errorf("expected empty for missing tool, got %q", url)
	}

	// nil manifest
	var nilMf *Manifest
	if url := nilMf.GetPlatformURL("test-tool", "windows/amd64"); url != "" {
		t.Errorf("expected empty for nil manifest, got %q", url)
	}
}

func TestManifest_GetPlatformAsset(t *testing.T) {
	mf := &Manifest{
		Tools: map[string]ToolEntry{
			"test-tool": {
				Type: "binary",
				Platforms: map[string]PlatformAsset{
					"windows/amd64": {URL: "https://example.com/win.zip", SHA256: "abc123"},
					"linux/amd64":   {URL: "https://example.com/linux.tar.gz"},
				},
			},
		},
	}

	// Returns full asset with SHA256
	asset := mf.GetPlatformAsset("test-tool", "windows/amd64")
	if asset == nil {
		t.Fatal("expected non-nil asset")
	}
	if asset.URL != "https://example.com/win.zip" {
		t.Errorf("unexpected URL: %q", asset.URL)
	}
	if asset.SHA256 != "abc123" {
		t.Errorf("unexpected SHA256: %q", asset.SHA256)
	}

	// Asset without SHA256
	asset = mf.GetPlatformAsset("test-tool", "linux/amd64")
	if asset == nil {
		t.Fatal("expected non-nil asset for linux")
	}
	if asset.SHA256 != "" {
		t.Errorf("expected empty SHA256, got %q", asset.SHA256)
	}

	// Missing platform
	if a := mf.GetPlatformAsset("test-tool", "darwin/arm64"); a != nil {
		t.Errorf("expected nil for missing platform, got %+v", a)
	}

	// Missing tool
	if a := mf.GetPlatformAsset("nonexistent", "windows/amd64"); a != nil {
		t.Errorf("expected nil for missing tool, got %+v", a)
	}

	// nil manifest
	var nilMf *Manifest
	if a := nilMf.GetPlatformAsset("test-tool", "windows/amd64"); a != nil {
		t.Errorf("expected nil for nil manifest, got %+v", a)
	}
}

func TestManifest_GetLatestVersion(t *testing.T) {
	mf := &Manifest{
		Tools: map[string]ToolEntry{
			"claude": {LatestVersion: "1.0.26"},
		},
	}
	if v := mf.GetLatestVersion("claude"); v != "1.0.26" {
		t.Errorf("expected 1.0.26, got %q", v)
	}
	if v := mf.GetLatestVersion("nonexistent"); v != "" {
		t.Errorf("expected empty, got %q", v)
	}
}

func TestFetch_ReturnsManifest(t *testing.T) {
	// Set up a test HTTP server
	testManifest := Manifest{
		GeneratedAt: "2026-03-17T00:00:00Z",
		Tools: map[string]ToolEntry{
			"test": {Type: "npm", LatestVersion: "1.0.0"},
		},
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != manifestEndpoint {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(testManifest)
	}))
	defer srv.Close()

	cacheDir := t.TempDir()
	ctx := context.Background()

	mf, err := Fetch(ctx, srv.URL, cacheDir)
	if err != nil {
		t.Fatalf("Fetch failed: %v", err)
	}
	if mf.GeneratedAt != "2026-03-17T00:00:00Z" {
		t.Errorf("unexpected GeneratedAt: %q", mf.GeneratedAt)
	}
	if _, ok := mf.Tools["test"]; !ok {
		t.Error("expected 'test' tool in fetched manifest")
	}

	// Verify cache was written
	cachePath := filepath.Join(cacheDir, cacheFilename)
	if _, err := os.Stat(cachePath); err != nil {
		t.Errorf("cache file not created: %v", err)
	}
}

func TestFetch_UsesFreshCache(t *testing.T) {
	cacheDir := t.TempDir()
	cachePath := filepath.Join(cacheDir, cacheFilename)

	// Write a fresh cache entry
	entry := cacheEntry{
		FetchedAt: time.Now().UTC(),
		Manifest: Manifest{
			GeneratedAt: "cached-version",
			Tools:       map[string]ToolEntry{"cached": {Type: "npm"}},
		},
	}
	data, _ := json.Marshal(entry)
	os.WriteFile(cachePath, data, 0600)

	ctx := context.Background()
	// Pass an invalid API base — should not matter because cache is fresh
	mf, err := Fetch(ctx, "http://invalid-should-not-reach.test", cacheDir)
	if err != nil {
		t.Fatalf("Fetch failed: %v", err)
	}
	if mf.GeneratedAt != "cached-version" {
		t.Errorf("expected cached manifest, got %q", mf.GeneratedAt)
	}
}

func TestFetch_FallsBackOnError(t *testing.T) {
	// No cache, unreachable server → should fall back to Builtin
	cacheDir := t.TempDir()
	ctx := context.Background()

	mf, err := Fetch(ctx, "http://127.0.0.1:1", cacheDir) // port 1 should be unreachable
	if err != nil {
		t.Fatalf("Fetch should not error (falls back to builtin): %v", err)
	}
	// Should get the builtin manifest
	builtin := Builtin()
	if len(mf.Tools) != len(builtin.Tools) {
		t.Errorf("expected builtin tools count %d, got %d", len(builtin.Tools), len(mf.Tools))
	}
}

func TestFetch_UsesStaleCache(t *testing.T) {
	cacheDir := t.TempDir()
	cachePath := filepath.Join(cacheDir, cacheFilename)

	// Write a stale cache entry (past TTL)
	entry := cacheEntry{
		FetchedAt: time.Now().Add(-24 * time.Hour), // stale
		Manifest: Manifest{
			GeneratedAt: "stale-version",
			Tools:       map[string]ToolEntry{"stale": {Type: "npm"}},
		},
	}
	data, _ := json.Marshal(entry)
	os.WriteFile(cachePath, data, 0600)

	ctx := context.Background()
	// Unreachable server + stale cache → should use stale cache
	mf, err := Fetch(ctx, "http://127.0.0.1:1", cacheDir)
	if err != nil {
		t.Fatalf("Fetch failed: %v", err)
	}
	if mf.GeneratedAt != "stale-version" {
		t.Errorf("expected stale cache, got %q", mf.GeneratedAt)
	}
}
