package updater

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func init() {
	// Speed up tests: no real sleep between retry attempts.
	checksumRetryDelay = 0
}

// TestVerifyFileChecksum_NetworkError tests that a network/transport error is NOT silently
// swallowed — the function must return a non-nil error.
func TestVerifyFileChecksum_NetworkError(t *testing.T) {
	// Use a URL that will immediately fail (no server listening).
	// localPath is never reached (fails at GET), so any value is fine.
	tmp := t.TempDir()
	client := &http.Client{}
	err := VerifyFileChecksum(client, "http://127.0.0.1:1/fake.bin", filepath.Join(tmp, "unused"))
	if err == nil {
		t.Fatal("expected error on network/transport failure, got nil")
	}
}

// TestVerifyFileChecksum_404GraceAllows tests that HTTP 404 still returns nil (rollout grace).
// The function appends ".sha256" to the downloadURL, so we pass a URL without the suffix.
func TestVerifyFileChecksum_404GraceAllows(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	// downloadURL = srv.URL+"/bin" → function will fetch srv.URL+"/bin.sha256" → 404 grace.
	err := VerifyFileChecksum(srv.Client(), srv.URL+"/bin", "/dev/null")
	if err != nil {
		t.Fatalf("expected nil error for 404 grace, got: %v", err)
	}
}

// TestVerifyFileChecksum_Mismatch tests that a checksum mismatch returns an error and
// removes the local file.
func TestVerifyFileChecksum_Mismatch(t *testing.T) {
	const badChecksum = "aabbccdd" + "aabbccdd" + "aabbccdd" + "aabbccdd" +
		"aabbccdd" + "aabbccdd" + "aabbccdd" + "aabbccdd" // 64 hex chars

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(badChecksum + "  fake.bin\n"))
	}))
	defer srv.Close()

	tmp := t.TempDir()
	localFile := filepath.Join(tmp, "fake.bin")
	if err := os.WriteFile(localFile, []byte("content"), 0644); err != nil {
		t.Fatal(err)
	}

	err := VerifyFileChecksum(srv.Client(), srv.URL+"/fake.bin", localFile)
	if err == nil {
		t.Fatal("expected mismatch error, got nil")
	}

	// File should be removed on mismatch
	if _, statErr := os.Stat(localFile); !os.IsNotExist(statErr) {
		t.Error("expected local file to be removed after mismatch")
	}
}

// TestVerifyFileChecksum_Match tests the happy path: correct checksum passes.
func TestVerifyFileChecksum_Match(t *testing.T) {
	// sha256("hello") = 2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824
	const helloSHA = "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(helloSHA + "  hello.bin\n"))
	}))
	defer srv.Close()

	tmp := t.TempDir()
	localFile := filepath.Join(tmp, "hello.bin")
	if err := os.WriteFile(localFile, []byte("hello"), 0644); err != nil {
		t.Fatal(err)
	}

	if err := VerifyFileChecksum(srv.Client(), srv.URL+"/hello.bin", localFile); err != nil {
		t.Fatalf("expected nil for matching checksum, got: %v", err)
	}
}
