package installer

import (
	"context"
	"runtime"
	"testing"
)

func TestZeroClawInstaller_Detect_NotInstalled(t *testing.T) {
	inst := NewZeroClawInstaller()
	status, err := inst.Detect(context.Background())
	if err != nil {
		t.Fatalf("Detect returned unexpected error: %v", err)
	}
	if status == nil {
		t.Fatal("Detect returned nil status")
	}
	if status.Name != ToolZeroClaw {
		t.Errorf("expected Name=%q, got %q", ToolZeroClaw, status.Name)
	}
}

func TestZeroClawInstaller_BinaryFilename(t *testing.T) {
	name := binaryFilename(ZeroClawBinaryName)
	if name == "" {
		t.Error("binaryFilename returned empty string")
	}
	expected := ZeroClawBinaryName
	if runtime.GOOS == "windows" {
		expected += ".exe"
	}
	if name != expected {
		t.Errorf("binaryFilename = %q, want %q", name, expected)
	}
}

func TestZeroClawInstaller_CacheDir(t *testing.T) {
	dir := toolCacheDir(ToolZeroClaw)
	if dir == "" {
		t.Error("toolCacheDir returned empty string")
	}
}

func TestZeroClawInstaller_BinaryConfig(t *testing.T) {
	inst := NewZeroClawInstaller()
	cfg := inst.binaryConfig()
	if cfg.Name != ToolZeroClaw {
		t.Errorf("Name = %q, want %q", cfg.Name, ToolZeroClaw)
	}
	if cfg.GitHubOwner != ZeroClawGitHubOwner {
		t.Errorf("GitHubOwner = %q, want %q", cfg.GitHubOwner, ZeroClawGitHubOwner)
	}
}

func TestFindPlatformAsset_ZeroClaw(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows-only asset test")
	}

	assets := []GitHubAsset{
		{Name: "zeroclaw-x86_64-pc-windows-msvc.zip", BrowserDownloadURL: "https://example.com/win.zip"},
		{Name: "zeroclaw-x86_64-unknown-linux-gnu.tar.gz", BrowserDownloadURL: "https://example.com/linux.tar.gz"},
		{Name: "zeroclaw-aarch64-apple-darwin.tar.gz", BrowserDownloadURL: "https://example.com/mac.tar.gz"},
	}

	url, name := findPlatformAsset(assets)
	if url == "" {
		t.Error("findPlatformAsset failed to find Windows asset")
	}
	if name == "" {
		t.Error("findPlatformAsset returned empty asset name")
	}
}
