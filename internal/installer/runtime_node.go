package installer

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// NodeRuntime manages Node.js installation and detection
type NodeRuntime struct {
	nodePath string
}

// NewNodeRuntime creates a new NodeRuntime (does not require Node.js to be present yet)
func NewNodeRuntime() *NodeRuntime {
	return &NodeRuntime{}
}

// FindNode locates the Node.js executable, checking PATH then known install locations
func (r *NodeRuntime) FindNode() (string, error) {
	if r.nodePath != "" {
		return r.nodePath, nil
	}

	// Check PATH first
	if path, err := exec.LookPath("node"); err == nil {
		r.nodePath = path
		return path, nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}

	var candidates []string
	switch runtime.GOOS {
	case "windows":
		programFiles := os.Getenv("ProgramFiles")
		if programFiles == "" {
			programFiles = `C:\Program Files`
		}
		candidates = []string{
			filepath.Join(programFiles, "nodejs", "node.exe"),
			filepath.Join(home, "AppData", "Roaming", "fnm", "node-versions", "*", "installation", "node.exe"),
			filepath.Join(home, ".nvm", "versions", "node", "*", "bin", "node.exe"),
		}
		// Expand glob patterns for version managers
		var expanded []string
		for _, c := range candidates {
			if strings.Contains(c, "*") {
				if matches, err := filepath.Glob(c); err == nil {
					expanded = append(expanded, matches...)
				}
			} else {
				expanded = append(expanded, c)
			}
		}
		candidates = expanded
	case "darwin":
		candidates = []string{
			"/usr/local/bin/node",
			"/opt/homebrew/bin/node",
			filepath.Join(home, ".nvm", "versions", "node"),
			filepath.Join(home, ".fnm", "node-versions"),
		}
		// Expand nvm/fnm version directories
		nvmGlob := filepath.Join(home, ".nvm", "versions", "node", "*", "bin", "node")
		if matches, err := filepath.Glob(nvmGlob); err == nil && len(matches) > 0 {
			candidates = append(candidates, matches[len(matches)-1])
		}
		fnmGlob := filepath.Join(home, ".fnm", "node-versions", "*", "installation", "bin", "node")
		if matches, err := filepath.Glob(fnmGlob); err == nil && len(matches) > 0 {
			candidates = append(candidates, matches[len(matches)-1])
		}
	default: // linux
		candidates = []string{
			"/usr/local/bin/node",
			"/usr/bin/node",
		}
		nvmGlob := filepath.Join(home, ".nvm", "versions", "node", "*", "bin", "node")
		if matches, err := filepath.Glob(nvmGlob); err == nil && len(matches) > 0 {
			candidates = append(candidates, matches[len(matches)-1])
		}
	}

	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			r.nodePath = candidate
			return candidate, nil
		}
	}

	return "", fmt.Errorf("node not found: install Node.js 22+ first (https://nodejs.org)")
}

// GetVersion returns the installed Node.js version string (e.g. "22.14.0")
func (r *NodeRuntime) GetVersion(ctx context.Context) (string, error) {
	nodePath, err := r.FindNode()
	if err != nil {
		return "", err
	}

	verCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	cmd := exec.CommandContext(verCtx, nodePath, "--version")
	hideWindow(cmd)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to get node version: %w", err)
	}

	version := extractVersion(strings.TrimSpace(string(out)))
	return version, nil
}

// InstallNode installs Node.js using the platform-appropriate package manager
func (r *NodeRuntime) InstallNode(ctx context.Context) (string, error) {
	timeout := time.Duration(DefaultInstallTimeout) * time.Second
	installCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.CommandContext(installCtx, "winget", "install", "--id", "OpenJS.NodeJS.LTS", "--accept-package-agreements", "--accept-source-agreements")
	case "darwin":
		cmd = exec.CommandContext(installCtx, "brew", "install", "node@22")
	default: // linux
		// Use NodeSource setup script for apt-based systems
		cmd = exec.CommandContext(installCtx, "bash", "-c",
			"curl -fsSL https://deb.nodesource.com/setup_22.x | sudo -E bash - && sudo apt-get install -y nodejs")
	}
	hideWindow(cmd)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("node.js install failed: %w, output: %s", err, strings.TrimSpace(string(output)))
	}

	// Clear cached path and re-discover
	r.nodePath = ""
	return r.FindNode()
}

// EnsureNode checks if Node.js is available; if not, installs it and returns the path
func (r *NodeRuntime) EnsureNode(ctx context.Context) (string, error) {
	if path, err := r.FindNode(); err == nil {
		return path, nil
	}
	return r.InstallNode(ctx)
}

// GetPath returns the cached node path (empty if not yet located)
func (r *NodeRuntime) GetPath() string {
	return r.nodePath
}

// IsInstalled returns true if Node.js can be found on the system
func (r *NodeRuntime) IsInstalled() bool {
	_, err := r.FindNode()
	return err == nil
}

// MeetsMinVersion returns true if the installed Node.js version is >= minMajor
func (r *NodeRuntime) MeetsMinVersion(ctx context.Context, minMajor int) bool {
	version, err := r.GetVersion(ctx)
	if err != nil {
		return false
	}
	parts := strings.SplitN(version, ".", 3)
	if len(parts) < 1 {
		return false
	}
	major, err := strconv.Atoi(parts[0])
	if err != nil {
		return false
	}
	return major >= minMajor
}
