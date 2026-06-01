package updater

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"testing"
)

// newSelfUpdaterWithServer creates a SelfUpdater whose internal GitHub checker
// is redirected to a test server (same package access to private fields).
func newSelfUpdaterWithServer(srv *httptest.Server, version string) *SelfUpdater {
	checker := &GitHubChecker{
		owner: selfUpdateOwner,
		repo:  selfUpdateRepo,
		client: &http.Client{
			Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
				u, _ := url.Parse(srv.URL + r.URL.RequestURI())
				r2 := r.Clone(r.Context())
				r2.URL = u
				r2.Host = srv.Listener.Addr().String()
				return srv.Client().Transport.RoundTrip(r2)
			}),
		},
	}
	return &SelfUpdater{checker: checker, currentVersion: version}
}

func TestSelfUpdater_CheckUpdate_UpdateAvailable(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rel := githubRelease{
			TagName: "v99.0.0",
			HTMLURL: "https://github.com/lurus-dev/lurus-switch",
		}
		json.NewEncoder(w).Encode(rel)
	}))
	defer srv.Close()

	s := newSelfUpdaterWithServer(srv, "1.0.0")
	info, err := s.CheckUpdate()
	if err != nil {
		t.Fatalf("CheckUpdate error: %v", err)
	}
	if !info.UpdateAvailable {
		t.Error("UpdateAvailable should be true")
	}
	if info.LatestVersion != "99.0.0" {
		t.Errorf("LatestVersion = %q, want 99.0.0", info.LatestVersion)
	}
}

func TestSelfUpdater_CheckUpdate_NoUpdate(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rel := githubRelease{TagName: "v1.0.0", HTMLURL: "https://github.com/lurus-dev/lurus-switch"}
		json.NewEncoder(w).Encode(rel)
	}))
	defer srv.Close()

	s := newSelfUpdaterWithServer(srv, "1.0.0")
	info, err := s.CheckUpdate()
	if err != nil {
		t.Fatalf("CheckUpdate error: %v", err)
	}
	if info.UpdateAvailable {
		t.Error("UpdateAvailable should be false when at same version")
	}
}

func TestSelfUpdater_CheckUpdate_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	s := newSelfUpdaterWithServer(srv, "1.0.0")
	_, err := s.CheckUpdate()
	if err == nil {
		t.Error("expected error for HTTP 500")
	}
}

// TestDownloadFile tests the downloadFile helper with a mock HTTP server
func TestDownloadFile_Success(t *testing.T) {
	content := []byte("fake binary content")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write(content)
	}))
	defer srv.Close()

	tmp := t.TempDir()
	dest := filepath.Join(tmp, "downloaded.bin")

	if err := downloadFile(srv.URL+"/file", dest); err != nil {
		t.Fatalf("downloadFile error: %v", err)
	}

	got, err := os.ReadFile(dest)
	if err != nil {
		t.Fatalf("ReadFile error: %v", err)
	}
	if string(got) != string(content) {
		t.Errorf("downloaded content = %q, want %q", got, content)
	}
}

func TestDownloadFile_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	tmp := t.TempDir()
	dest := filepath.Join(tmp, "file.bin")

	if err := downloadFile(srv.URL+"/file", dest); err == nil {
		t.Error("expected error for HTTP 404")
	}
}

func TestDownloadFile_BadURL(t *testing.T) {
	tmp := t.TempDir()
	dest := filepath.Join(tmp, "file.bin")

	// Non-existent server
	err := downloadFile("http://127.0.0.1:1/nonexistent", dest)
	if err == nil {
		t.Error("expected error for unreachable server")
	}
}

func TestDownloadFile_InvalidDestDir(t *testing.T) {
	content := []byte("data")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(content)
	}))
	defer srv.Close()

	// Destination in a non-existent directory
	err := downloadFile(srv.URL+"/file", "/nonexistent/deep/path/file.bin")
	if err == nil {
		t.Error("expected error for non-existent destination directory")
	}
}

// TestDownloadFile_SizeCap verifies that downloadFile stops reading after maxDownloadBytes.
// We temporarily lower the cap to 10 bytes to keep the test fast.
func TestDownloadFile_SizeCap(t *testing.T) {
	const capBytes = 10
	orig := maxDownloadBytes
	maxDownloadBytes = capBytes
	defer func() { maxDownloadBytes = orig }()

	// Server sends 100 bytes.
	payload := bytes.Repeat([]byte("x"), 100)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(payload)
	}))
	defer srv.Close()

	tmp := t.TempDir()
	dest := filepath.Join(tmp, "capped.bin")

	if err := downloadFile(srv.URL+"/big", dest); err != nil {
		t.Fatalf("downloadFile error: %v", err)
	}

	got, err := os.ReadFile(dest)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if int64(len(got)) > capBytes {
		t.Errorf("downloaded %d bytes, cap was %d — LimitReader not enforced", len(got), capBytes)
	}
}
