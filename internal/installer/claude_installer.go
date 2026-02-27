package installer

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"
)

// ClaudeInstaller handles Claude Code CLI installation and configuration
type ClaudeInstaller struct {
	runtime *BunRuntime
}

// NewClaudeInstaller creates a new ClaudeInstaller
func NewClaudeInstaller(rt *BunRuntime) *ClaudeInstaller {
	return &ClaudeInstaller{runtime: rt}
}

// Detect checks if Claude Code is installed and returns its status
func (c *ClaudeInstaller) Detect(ctx context.Context) (*ToolStatus, error) {
	status := &ToolStatus{Name: ToolClaude, Installed: false}

	path, err := c.findExecutable()
	if err != nil {
		return status, nil
	}
	status.Path = path

	// Get version
	verCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	verCmd := exec.CommandContext(verCtx, path, "--version")
	hideWindow(verCmd)
	out, err := verCmd.CombinedOutput()
	if err != nil {
		// Binary found but version check failed — still mark as installed
		status.Installed = true
		status.Version = "unknown"
		return status, nil
	}

	version := extractVersion(string(out))
	status.Installed = true
	status.Version = version
	return status, nil
}

// Install installs Claude Code globally via bun
func (c *ClaudeInstaller) Install(ctx context.Context) (*InstallResult, error) {
	bunPath, err := c.runtime.EnsureBun(ctx)
	if err != nil {
		return nil, fmt.Errorf("bun required for installation: %w", err)
	}

	installCtx, cancel := context.WithTimeout(ctx, time.Duration(DefaultInstallTimeout)*time.Second)
	defer cancel()

	cmd := exec.CommandContext(installCtx, bunPath, "install", "-g", ClaudeNpmPackage)
	hideWindow(cmd)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return &InstallResult{
			Tool:    ToolClaude,
			Success: false,
			Message: fmt.Sprintf("install failed: %s", strings.TrimSpace(string(output))),
		}, nil
	}

	// Verify installation
	status, _ := c.Detect(ctx)
	if status != nil && status.Installed {
		return &InstallResult{
			Tool:    ToolClaude,
			Success: true,
			Version: status.Version,
			Message: "installed successfully",
		}, nil
	}

	return &InstallResult{
		Tool:    ToolClaude,
		Success: false,
		Message: "install command succeeded but binary not found in PATH",
	}, nil
}

// Update updates Claude Code to the latest version
func (c *ClaudeInstaller) Update(ctx context.Context) (*InstallResult, error) {
	bunPath, err := c.runtime.EnsureBun(ctx)
	if err != nil {
		return nil, fmt.Errorf("bun required for update: %w", err)
	}

	updateCtx, cancel := context.WithTimeout(ctx, time.Duration(DefaultInstallTimeout)*time.Second)
	defer cancel()

	cmd := exec.CommandContext(updateCtx, bunPath, "install", "-g", ClaudeNpmPackage+"@latest")
	hideWindow(cmd)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return &InstallResult{
			Tool:    ToolClaude,
			Success: false,
			Message: fmt.Sprintf("update failed: %s", strings.TrimSpace(string(output))),
		}, nil
	}

	status, _ := c.Detect(ctx)
	version := ""
	if status != nil {
		version = status.Version
	}

	return &InstallResult{
		Tool:    ToolClaude,
		Success: true,
		Version: version,
		Message: "updated successfully",
	}, nil
}

// ConfigureProxy writes NewAPI proxy settings into Claude's config
func (c *ClaudeInstaller) ConfigureProxy(ctx context.Context, endpoint, apiKey string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	configDir := filepath.Join(home, ".claude")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create claude config directory: %w", err)
	}

	settingsPath := filepath.Join(configDir, "settings.json")

	// Load existing settings or start fresh
	settings := make(map[string]interface{})
	if data, err := os.ReadFile(settingsPath); err == nil {
		if err := json.Unmarshal(data, &settings); err != nil {
			settings = make(map[string]interface{})
		}
	}

	// Set API endpoint and key via env section
	env, ok := settings["env"].(map[string]interface{})
	if !ok {
		env = make(map[string]interface{})
	}
	if apiKey != "" {
		env["ANTHROPIC_API_KEY"] = apiKey
	}
	if endpoint != "" {
		env["ANTHROPIC_BASE_URL"] = endpoint
	}
	settings["env"] = env

	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal claude settings: %w", err)
	}

	if err := os.WriteFile(settingsPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write claude settings: %w", err)
	}

	return nil
}

// findExecutable locates the claude binary
func (c *ClaudeInstaller) findExecutable() (string, error) {
	if path, err := exec.LookPath("claude"); err == nil {
		return path, nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	var candidates []string
	switch runtime.GOOS {
	case "windows":
		candidates = []string{
			filepath.Join(home, ".bun", "bin", "claude.exe"),
			filepath.Join(os.Getenv("APPDATA"), "npm", "claude.cmd"),
		}
	default:
		candidates = []string{
			filepath.Join(home, ".bun", "bin", "claude"),
			"/usr/local/bin/claude",
		}
	}

	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}
	}

	return "", fmt.Errorf("claude executable not found")
}

// extractVersion extracts a semver version string from command output
func extractVersion(output string) string {
	re := regexp.MustCompile(VersionPattern)
	match := re.FindString(output)
	if match == "" {
		return "unknown"
	}
	return match
}
