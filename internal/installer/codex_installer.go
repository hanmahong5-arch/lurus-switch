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

	"github.com/BurntSushi/toml"
)

// CodexInstaller handles Codex CLI installation and configuration
type CodexInstaller struct {
	runtime *BunRuntime
}

// NewCodexInstaller creates a new CodexInstaller
func NewCodexInstaller(rt *BunRuntime) *CodexInstaller {
	return &CodexInstaller{runtime: rt}
}

// Detect checks if Codex is installed and returns its status
func (c *CodexInstaller) Detect(ctx context.Context) (*ToolStatus, error) {
	status := &ToolStatus{Name: ToolCodex, Installed: false}

	path, err := c.findExecutable()
	if err != nil {
		return status, nil
	}
	status.Path = path

	verCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	verCmd := exec.CommandContext(verCtx, path, "--version")
	hideWindow(verCmd)
	out, err := verCmd.CombinedOutput()
	if err != nil {
		status.Installed = true
		status.Version = "unknown"
		return status, nil
	}

	version := extractVersion(string(out))
	status.Installed = true
	status.Version = version
	return status, nil
}

// Install installs Codex globally via bun
func (c *CodexInstaller) Install(ctx context.Context) (*InstallResult, error) {
	bunPath, err := c.runtime.EnsureBun(ctx)
	if err != nil {
		return nil, fmt.Errorf("bun required for installation: %w", err)
	}

	installCtx, cancel := context.WithTimeout(ctx, time.Duration(DefaultInstallTimeout)*time.Second)
	defer cancel()

	cmd := exec.CommandContext(installCtx, bunPath, "install", "-g", CodexNpmPackage)
	hideWindow(cmd)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return &InstallResult{
			Tool:    ToolCodex,
			Success: false,
			Message: fmt.Sprintf("install failed: %s", strings.TrimSpace(string(output))),
		}, nil
	}

	status, _ := c.Detect(ctx)
	if status != nil && status.Installed {
		return &InstallResult{
			Tool:    ToolCodex,
			Success: true,
			Version: status.Version,
			Message: "installed successfully",
		}, nil
	}

	return &InstallResult{
		Tool:    ToolCodex,
		Success: false,
		Message: "install command succeeded but binary not found in PATH",
	}, nil
}

// Update updates Codex to the latest version
func (c *CodexInstaller) Update(ctx context.Context) (*InstallResult, error) {
	bunPath, err := c.runtime.EnsureBun(ctx)
	if err != nil {
		return nil, fmt.Errorf("bun required for update: %w", err)
	}

	updateCtx, cancel := context.WithTimeout(ctx, time.Duration(DefaultInstallTimeout)*time.Second)
	defer cancel()

	cmd := exec.CommandContext(updateCtx, bunPath, "install", "-g", CodexNpmPackage+"@latest")
	hideWindow(cmd)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return &InstallResult{
			Tool:    ToolCodex,
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
		Tool:    ToolCodex,
		Success: true,
		Version: version,
		Message: "updated successfully",
	}, nil
}

// ConfigureProxy writes NewAPI proxy settings into Codex's config
func (c *CodexInstaller) ConfigureProxy(ctx context.Context, endpoint, apiKey string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	configDir := filepath.Join(home, ".codex")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create codex config directory: %w", err)
	}

	configPath := filepath.Join(configDir, "config.toml")

	// Build TOML config structure
	type providerConfig struct {
		Type    string `toml:"type"`
		BaseURL string `toml:"base_url,omitempty"`
	}

	type codexConfig struct {
		Provider providerConfig `toml:"provider"`
	}

	// Load existing config or start fresh
	cfg := codexConfig{}
	if data, err := os.ReadFile(configPath); err == nil {
		toml.Unmarshal(data, &cfg)
	}

	cfg.Provider.Type = "custom"
	if endpoint != "" {
		cfg.Provider.BaseURL = endpoint
	}

	var buf strings.Builder
	enc := toml.NewEncoder(&buf)
	if err := enc.Encode(cfg); err != nil {
		return fmt.Errorf("failed to encode codex config: %w", err)
	}

	if err := os.WriteFile(configPath, []byte(buf.String()), 0600); err != nil {
		return fmt.Errorf("failed to write codex config: %w", err)
	}

	// Also set environment variables for the current session (persisted via proxy settings)
	if apiKey != "" {
		os.Setenv("OPENAI_API_KEY", apiKey)
	}
	if endpoint != "" {
		os.Setenv("OPENAI_BASE_URL", endpoint)
	}

	return nil
}

// findExecutable locates the codex binary
func (c *CodexInstaller) findExecutable() (string, error) {
	if path, err := exec.LookPath("codex"); err == nil {
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
			filepath.Join(home, ".bun", "bin", "codex.exe"),
			filepath.Join(os.Getenv("APPDATA"), "npm", "codex.cmd"),
		}
	default:
		candidates = []string{
			filepath.Join(home, ".bun", "bin", "codex"),
			"/usr/local/bin/codex",
		}
	}

	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}
	}

	return "", fmt.Errorf("codex executable not found")
}
