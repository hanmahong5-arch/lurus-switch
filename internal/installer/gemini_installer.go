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

// GeminiInstaller handles Gemini CLI installation and configuration
type GeminiInstaller struct {
	runtime *BunRuntime
}

// NewGeminiInstaller creates a new GeminiInstaller
func NewGeminiInstaller(rt *BunRuntime) *GeminiInstaller {
	return &GeminiInstaller{runtime: rt}
}

// Detect checks if Gemini CLI is installed and returns its status
func (g *GeminiInstaller) Detect(ctx context.Context) (*ToolStatus, error) {
	status := &ToolStatus{Name: ToolGemini, Installed: false}

	path, err := g.findExecutable()
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

// Install installs Gemini CLI globally via bun
func (g *GeminiInstaller) Install(ctx context.Context) (*InstallResult, error) {
	bunPath, err := g.runtime.EnsureBun(ctx)
	if err != nil {
		return nil, fmt.Errorf("bun required for installation: %w", err)
	}

	installCtx, cancel := context.WithTimeout(ctx, time.Duration(DefaultInstallTimeout)*time.Second)
	defer cancel()

	cmd := exec.CommandContext(installCtx, bunPath, "install", "-g", GeminiNpmPackage)
	hideWindow(cmd)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return &InstallResult{
			Tool:    ToolGemini,
			Success: false,
			Message: fmt.Sprintf("install failed: %s", strings.TrimSpace(string(output))),
		}, nil
	}

	status, _ := g.Detect(ctx)
	if status != nil && status.Installed {
		return &InstallResult{
			Tool:    ToolGemini,
			Success: true,
			Version: status.Version,
			Message: "installed successfully",
		}, nil
	}

	return &InstallResult{
		Tool:    ToolGemini,
		Success: false,
		Message: "install command succeeded but binary not found in PATH",
	}, nil
}

// Update updates Gemini CLI to the latest version
func (g *GeminiInstaller) Update(ctx context.Context) (*InstallResult, error) {
	bunPath, err := g.runtime.EnsureBun(ctx)
	if err != nil {
		return nil, fmt.Errorf("bun required for update: %w", err)
	}

	updateCtx, cancel := context.WithTimeout(ctx, time.Duration(DefaultInstallTimeout)*time.Second)
	defer cancel()

	cmd := exec.CommandContext(updateCtx, bunPath, "install", "-g", GeminiNpmPackage+"@latest")
	hideWindow(cmd)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return &InstallResult{
			Tool:    ToolGemini,
			Success: false,
			Message: fmt.Sprintf("update failed: %s", strings.TrimSpace(string(output))),
		}, nil
	}

	status, _ := g.Detect(ctx)
	version := ""
	if status != nil {
		version = status.Version
	}

	return &InstallResult{
		Tool:    ToolGemini,
		Success: true,
		Version: version,
		Message: "updated successfully",
	}, nil
}

// ConfigureProxy writes NewAPI proxy settings for Gemini CLI
func (g *GeminiInstaller) ConfigureProxy(ctx context.Context, endpoint, apiKey string) error {
	// Gemini CLI uses environment variables for API configuration.
	// We set them in the process environment; the proxy manager persists them.
	if apiKey != "" {
		os.Setenv("GEMINI_API_KEY", apiKey)
	}
	if endpoint != "" {
		os.Setenv("GEMINI_API_ENDPOINT", endpoint)
	}

	// Also write a settings file that the packager can use
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	configDir := filepath.Join(home, ".gemini")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create gemini config directory: %w", err)
	}

	// Write a simple JSON settings file
	settingsContent := fmt.Sprintf(`{
  "apiKey": %q,
  "apiEndpoint": %q
}
`, apiKey, endpoint)

	settingsPath := filepath.Join(configDir, "settings.json")
	if err := os.WriteFile(settingsPath, []byte(settingsContent), 0600); err != nil {
		return fmt.Errorf("failed to write gemini settings: %w", err)
	}

	return nil
}

// findExecutable locates the gemini binary
func (g *GeminiInstaller) findExecutable() (string, error) {
	if path, err := exec.LookPath("gemini"); err == nil {
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
			filepath.Join(home, ".bun", "bin", "gemini.exe"),
			filepath.Join(os.Getenv("APPDATA"), "npm", "gemini.cmd"),
		}
	default:
		candidates = []string{
			filepath.Join(home, ".bun", "bin", "gemini"),
			"/usr/local/bin/gemini",
		}
	}

	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}
	}

	return "", fmt.Errorf("gemini executable not found")
}
