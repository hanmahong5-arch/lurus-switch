package installer

import (
	"context"
	"runtime"
	"testing"
)

func TestNullClawInstaller_BinaryConfig(t *testing.T) {
	inst := NewNullClawInstaller()
	cfg := inst.binaryConfig()

	if cfg.Name != ToolNullClaw {
		t.Errorf("Name = %q, want %q", cfg.Name, ToolNullClaw)
	}
	if cfg.GitHubOwner != NullClawGitHubOwner {
		t.Errorf("GitHubOwner = %q, want %q", cfg.GitHubOwner, NullClawGitHubOwner)
	}
	if cfg.GitHubRepo != NullClawGitHubRepo {
		t.Errorf("GitHubRepo = %q, want %q", cfg.GitHubRepo, NullClawGitHubRepo)
	}
	if cfg.BinaryName != NullClawBinaryName {
		t.Errorf("BinaryName = %q, want %q", cfg.BinaryName, NullClawBinaryName)
	}
}

func TestNullClawInstaller_BinaryFilename(t *testing.T) {
	expected := NullClawBinaryName
	if runtime.GOOS == "windows" {
		expected += ".exe"
	}

	result := binaryFilename(NullClawBinaryName)
	if result != expected {
		t.Errorf("binaryFilename(%q) = %q, want %q", NullClawBinaryName, result, expected)
	}
}

func TestNullClawInstaller_CacheDir(t *testing.T) {
	dir := toolCacheDir(ToolNullClaw)
	if dir == "" {
		t.Error("toolCacheDir should return non-empty path")
	}
}

func TestNullClawInstaller_Detect_ReturnsStatus(t *testing.T) {
	inst := NewNullClawInstaller()
	ctx := context.Background()

	status, err := inst.Detect(ctx)
	if err != nil {
		t.Fatalf("Detect should not error: %v", err)
	}
	if status == nil {
		t.Fatal("Detect should return non-nil status")
	}
	if status.Name != ToolNullClaw {
		t.Errorf("expected name %q, got %q", ToolNullClaw, status.Name)
	}
}
