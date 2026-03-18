package installer

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestPicoClawInstaller_BinaryConfig(t *testing.T) {
	inst := NewPicoClawInstaller()
	cfg := inst.binaryConfig()

	if cfg.Name != ToolPicoClaw {
		t.Errorf("Name = %q, want %q", cfg.Name, ToolPicoClaw)
	}
	if cfg.GitHubOwner != PicoClawGitHubOwner {
		t.Errorf("GitHubOwner = %q, want %q", cfg.GitHubOwner, PicoClawGitHubOwner)
	}
	if cfg.GitHubRepo != PicoClawGitHubRepo {
		t.Errorf("GitHubRepo = %q, want %q", cfg.GitHubRepo, PicoClawGitHubRepo)
	}
	if cfg.BinaryName != PicoClawBinaryName {
		t.Errorf("BinaryName = %q, want %q", cfg.BinaryName, PicoClawBinaryName)
	}
}

func TestPicoClawInstaller_BinaryFilename(t *testing.T) {
	expected := PicoClawBinaryName
	if runtime.GOOS == "windows" {
		expected += ".exe"
	}

	result := binaryFilename(PicoClawBinaryName)
	if result != expected {
		t.Errorf("binaryFilename(%q) = %q, want %q", PicoClawBinaryName, result, expected)
	}
}

func TestPicoClawInstaller_CacheDir(t *testing.T) {
	dir := toolCacheDir(ToolPicoClaw)
	if dir == "" {
		t.Error("toolCacheDir should return non-empty path")
	}
}

func TestFindPlatformAsset_Windows(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows-only test")
	}

	assets := []GitHubAsset{
		{Name: "pclaw-x86_64-pc-windows-msvc.zip", BrowserDownloadURL: "https://example.com/win.zip"},
		{Name: "pclaw-x86_64-unknown-linux-gnu.tar.gz", BrowserDownloadURL: "https://example.com/linux.tar.gz"},
		{Name: "pclaw-aarch64-apple-darwin.tar.gz", BrowserDownloadURL: "https://example.com/mac.tar.gz"},
	}

	url, name := findPlatformAsset(assets)
	if url == "" {
		t.Fatal("findPlatformAsset should find Windows asset")
	}
	if name != "pclaw-x86_64-pc-windows-msvc.zip" {
		t.Errorf("found asset = %q, want Windows asset", name)
	}
}

func TestFindPlatformAsset_Empty(t *testing.T) {
	url, name := findPlatformAsset(nil)
	if url != "" || name != "" {
		t.Errorf("findPlatformAsset(nil) = (%q, %q), want empty", url, name)
	}
}

func TestFindPlatformAsset_NoMatch(t *testing.T) {
	assets := []GitHubAsset{
		{Name: "checksums.txt", BrowserDownloadURL: "https://example.com/checksums.txt"},
	}
	url, name := findPlatformAsset(assets)
	if url != "" || name != "" {
		t.Errorf("findPlatformAsset with no matching assets = (%q, %q), want empty", url, name)
	}
}

func TestVerifySHA256_EmptyHash(t *testing.T) {
	// Empty expected hash should always pass (no verification)
	if err := verifySHA256("nonexistent-file", ""); err != nil {
		t.Errorf("empty hash should skip verification, got: %v", err)
	}
}

func TestVerifySHA256_Correct(t *testing.T) {
	content := []byte("hello world")
	h := sha256.Sum256(content)
	expectedHex := hex.EncodeToString(h[:])

	tmpFile := filepath.Join(t.TempDir(), "test.bin")
	if err := os.WriteFile(tmpFile, content, 0644); err != nil {
		t.Fatal(err)
	}

	if err := verifySHA256(tmpFile, expectedHex); err != nil {
		t.Errorf("correct hash should pass, got: %v", err)
	}
}

func TestVerifySHA256_Mismatch(t *testing.T) {
	content := []byte("hello world")
	tmpFile := filepath.Join(t.TempDir(), "test.bin")
	if err := os.WriteFile(tmpFile, content, 0644); err != nil {
		t.Fatal(err)
	}

	err := verifySHA256(tmpFile, "0000000000000000000000000000000000000000000000000000000000000000")
	if err == nil {
		t.Fatal("wrong hash should fail verification")
	}
	if !testing.Verbose() {
		return
	}
	t.Logf("expected error: %v", err)
}

func TestVerifySHA256_CaseInsensitive(t *testing.T) {
	content := []byte("test data")
	h := sha256.Sum256(content)
	upperHex := hex.EncodeToString(h[:])
	// Convert to uppercase to test case-insensitive comparison
	upperHash := ""
	for _, c := range upperHex {
		if c >= 'a' && c <= 'f' {
			upperHash += string(c - 32)
		} else {
			upperHash += string(c)
		}
	}

	tmpFile := filepath.Join(t.TempDir(), "test.bin")
	if err := os.WriteFile(tmpFile, content, 0644); err != nil {
		t.Fatal(err)
	}

	if err := verifySHA256(tmpFile, upperHash); err != nil {
		t.Errorf("uppercase hash should pass (case-insensitive), got: %v", err)
	}
}

func TestVerifySHA256_FileNotFound(t *testing.T) {
	err := verifySHA256(filepath.Join(t.TempDir(), "missing.bin"), "abc123")
	if err == nil {
		t.Fatal("missing file should fail")
	}
}
