package installer

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// BunRuntime manages Bun installation and detection
type BunRuntime struct {
	bunPath string
}

// NewBunRuntime creates a new BunRuntime (does not require Bun to be present yet)
func NewBunRuntime() *BunRuntime {
	return &BunRuntime{}
}

// FindBun locates the Bun executable, checking PATH then known install locations
func (r *BunRuntime) FindBun() (string, error) {
	if r.bunPath != "" {
		return r.bunPath, nil
	}

	// Check PATH first
	if path, err := exec.LookPath("bun"); err == nil {
		r.bunPath = path
		return path, nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}

	var candidates []string
	switch runtime.GOOS {
	case "windows":
		candidates = []string{
			filepath.Join(home, ".bun", "bin", "bun.exe"),
			filepath.Join(os.Getenv("LOCALAPPDATA"), "bun", "bin", "bun.exe"),
		}
	default:
		candidates = []string{
			filepath.Join(home, ".bun", "bin", "bun"),
			"/usr/local/bin/bun",
			"/usr/bin/bun",
		}
	}

	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			r.bunPath = candidate
			return candidate, nil
		}
	}

	return "", fmt.Errorf("bun not found: install Bun first (https://bun.sh)")
}

// InstallBun installs Bun using the official installer script
func (r *BunRuntime) InstallBun(ctx context.Context) (string, error) {
	timeout := time.Duration(DefaultInstallTimeout) * time.Second
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.CommandContext(ctx, "powershell", "-NoProfile", "-Command", BunInstallWindows)
	default:
		cmd = exec.CommandContext(ctx, "bash", "-c", BunInstallUnix)
	}
	hideWindow(cmd)

	output, err := cmd.CombinedOutput()
	if err != nil {
		outputStr := strings.TrimSpace(string(output))
		if runtime.GOOS == "windows" && strings.Contains(outputStr, "ExecutionPolicy") {
			return "", fmt.Errorf("PowerShell execution policy blocked Bun install: run 'Set-ExecutionPolicy RemoteSigned -Scope CurrentUser' first, then retry")
		}
		return "", fmt.Errorf("bun install failed: %w, output: %s", err, outputStr)
	}

	// After install, locate the bun binary at the known path instead of relying on PATH
	r.bunPath = ""
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory after bun install: %w", err)
	}

	var expectedPath string
	switch runtime.GOOS {
	case "windows":
		expectedPath = filepath.Join(home, ".bun", "bin", "bun.exe")
	default:
		expectedPath = filepath.Join(home, ".bun", "bin", "bun")
	}

	if _, err := os.Stat(expectedPath); err == nil {
		r.bunPath = expectedPath
		return expectedPath, nil
	}

	// Fallback to FindBun which checks all candidates
	return r.FindBun()
}

// EnsureBun checks if Bun is available; if not, installs it and returns the path
func (r *BunRuntime) EnsureBun(ctx context.Context) (string, error) {
	if path, err := r.FindBun(); err == nil {
		return path, nil
	}
	return r.InstallBun(ctx)
}

// GetPath returns the cached bun path (empty if not yet located)
func (r *BunRuntime) GetPath() string {
	return r.bunPath
}

// IsInstalled returns true if Bun can be found on the system
func (r *BunRuntime) IsInstalled() bool {
	_, err := r.FindBun()
	return err == nil
}
