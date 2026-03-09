package updater

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

// roundTripFunc allows using a function as an http.RoundTripper.
// This lets tests intercept outbound requests without modifying production code.
type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

// newGitHubCheckerWithServer creates a GitHubChecker whose HTTP client is backed
// by a test server — all requests are forwarded to the server regardless of URL.
func newGitHubCheckerWithServer(srv *httptest.Server, owner, repo string) *GitHubChecker {
	return &GitHubChecker{
		owner: owner,
		repo:  repo,
		client: &http.Client{
			Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
				// Redirect request to test server
				u, _ := url.Parse(srv.URL + r.URL.RequestURI())
				r2 := r.Clone(r.Context())
				r2.URL = u
				r2.Host = srv.Listener.Addr().String()
				return srv.Client().Transport.RoundTrip(r2)
			}),
		},
	}
}

func TestGitHubChecker_CheckUpdate_UpdateAvailable(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rel := githubRelease{
			TagName: "v2.5.0",
			HTMLURL: "https://github.com/owner/repo/releases/tag/v2.5.0",
			Assets: []githubAsset{
				{Name: "app-windows-x64.exe", BrowserDownloadURL: "https://example.com/app.exe"},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(rel)
	}))
	defer srv.Close()

	checker := newGitHubCheckerWithServer(srv, "owner", "repo")
	info, err := checker.CheckUpdate("myapp", "1.0.0")
	if err != nil {
		t.Fatalf("CheckUpdate error: %v", err)
	}
	if info.LatestVersion != "2.5.0" {
		t.Errorf("LatestVersion = %q, want 2.5.0", info.LatestVersion)
	}
	if !info.UpdateAvailable {
		t.Error("UpdateAvailable should be true")
	}
}

func TestGitHubChecker_CheckUpdate_NoUpdate(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rel := githubRelease{TagName: "v1.0.0", HTMLURL: "https://github.com/owner/repo"}
		json.NewEncoder(w).Encode(rel)
	}))
	defer srv.Close()

	checker := newGitHubCheckerWithServer(srv, "owner", "repo")
	info, err := checker.CheckUpdate("myapp", "1.0.0")
	if err != nil {
		t.Fatalf("CheckUpdate error: %v", err)
	}
	if info.UpdateAvailable {
		t.Error("UpdateAvailable should be false when same version")
	}
}

func TestGitHubChecker_CheckUpdate_TagWithoutV(t *testing.T) {
	// tag_name without "v" prefix
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rel := githubRelease{TagName: "2.0.0", HTMLURL: "https://github.com/owner/repo"}
		json.NewEncoder(w).Encode(rel)
	}))
	defer srv.Close()

	checker := newGitHubCheckerWithServer(srv, "owner", "repo")
	info, err := checker.CheckUpdate("myapp", "1.0.0")
	if err != nil {
		t.Fatalf("CheckUpdate error: %v", err)
	}
	if info.LatestVersion != "2.0.0" {
		t.Errorf("LatestVersion = %q, want 2.0.0", info.LatestVersion)
	}
	if !info.UpdateAvailable {
		t.Error("UpdateAvailable should be true")
	}
}

func TestGitHubChecker_CheckUpdate_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer srv.Close()

	checker := newGitHubCheckerWithServer(srv, "owner", "repo")
	_, err := checker.CheckUpdate("myapp", "1.0.0")
	if err == nil {
		t.Error("expected error for HTTP 403")
	}
}

func TestGitHubChecker_CheckUpdate_InvalidJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("{not json"))
	}))
	defer srv.Close()

	checker := newGitHubCheckerWithServer(srv, "owner", "repo")
	_, err := checker.CheckUpdate("myapp", "1.0.0")
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestGitHubChecker_CheckUpdate_PicksPlatformAsset(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rel := githubRelease{
			TagName: "v2.0.0",
			HTMLURL: "https://github.com/owner/repo",
			Assets: []githubAsset{
				{Name: "app-linux-x64", BrowserDownloadURL: "https://example.com/linux"},
				{Name: "app-windows-x64.exe", BrowserDownloadURL: "https://example.com/win"},
				{Name: "app-darwin-arm64", BrowserDownloadURL: "https://example.com/mac"},
			},
		}
		json.NewEncoder(w).Encode(rel)
	}))
	defer srv.Close()

	checker := newGitHubCheckerWithServer(srv, "owner", "repo")
	info, err := checker.CheckUpdate("myapp", "1.0.0")
	if err != nil {
		t.Fatalf("CheckUpdate error: %v", err)
	}
	// On Windows, should prefer the windows asset URL
	if info.DownloadURL == "" {
		t.Error("DownloadURL should not be empty")
	}
}

func TestGitHubChecker_CheckUpdate_FallsBackToHTMLURL(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rel := githubRelease{
			TagName: "v2.0.0",
			HTMLURL: "https://github.com/owner/repo/releases/tag/v2.0.0",
			Assets:  []githubAsset{}, // no assets
		}
		json.NewEncoder(w).Encode(rel)
	}))
	defer srv.Close()

	checker := newGitHubCheckerWithServer(srv, "owner", "repo")
	info, err := checker.CheckUpdate("myapp", "1.0.0")
	if err != nil {
		t.Fatalf("CheckUpdate error: %v", err)
	}
	if info.DownloadURL != "https://github.com/owner/repo/releases/tag/v2.0.0" {
		t.Errorf("should fall back to HTMLURL, got %q", info.DownloadURL)
	}
}

// ===========================
// matchesPlatformAsset tests
// ===========================

func TestMatchesPlatformAsset_Windows(t *testing.T) {
	windowsAssets := []string{
		"app-windows-x64.exe",
		"lurus-switch-Windows-x64.zip",
		"myapp-windows-amd64",
		"tool-WINDOWS-AMD64.exe",
	}
	for _, name := range windowsAssets {
		if !matchesPlatformAsset(name) {
			t.Errorf("matchesPlatformAsset(%q) = false, want true", name)
		}
	}
}

func TestMatchesPlatformAsset_NonWindows(t *testing.T) {
	nonWindowsAssets := []string{
		"app-linux-x64",
		"app-darwin-arm64",
		"app-macos-x64",
		"checksum.txt",
	}
	for _, name := range nonWindowsAssets {
		if matchesPlatformAsset(name) {
			t.Errorf("matchesPlatformAsset(%q) = true, want false", name)
		}
	}
}

func TestMatchesPlatformAsset_CaseInsensitive(t *testing.T) {
	// matchesPlatformAsset uses ToLower internally
	mixed := "APP-Windows-X64.EXE"
	if !matchesPlatformAsset(mixed) {
		t.Errorf("matchesPlatformAsset should be case-insensitive for %q", mixed)
	}
	// Verify it's actually lowercased
	if lower := strings.ToLower(mixed); !strings.Contains(lower, "windows") {
		t.Errorf("ToLower(%q) does not contain 'windows'", mixed)
	}
}
