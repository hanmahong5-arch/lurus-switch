package installer

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// OpenClawInstaller handles OpenClaw CLI installation via npm/bun
type OpenClawInstaller struct {
	runtime *BunRuntime
}

// NewOpenClawInstaller creates a new OpenClawInstaller
func NewOpenClawInstaller(rt *BunRuntime) *OpenClawInstaller {
	return &OpenClawInstaller{runtime: rt}
}

// Detect checks if OpenClaw is installed and returns its status
func (o *OpenClawInstaller) Detect(ctx context.Context) (*ToolStatus, error) {
	status := &ToolStatus{Name: ToolOpenClaw, Installed: false}

	path, err := o.findExecutable()
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

// Install installs OpenClaw globally via bun/npm
func (o *OpenClawInstaller) Install(ctx context.Context) (*InstallResult, error) {
	bunPath, err := o.runtime.EnsureBun(ctx)
	if err != nil {
		return nil, fmt.Errorf("bun required for OpenClaw installation: %w", err)
	}

	installCtx, cancel := context.WithTimeout(ctx, time.Duration(DefaultInstallTimeout)*time.Second)
	defer cancel()

	cmd := exec.CommandContext(installCtx, bunPath, "install", "-g", OpenClawNpmPackage+"@latest")
	hideWindow(cmd)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return &InstallResult{
			Tool:    ToolOpenClaw,
			Success: false,
			Message: fmt.Sprintf("install failed: %s", strings.TrimSpace(string(output))),
		}, nil
	}

	// Verify installation
	status, _ := o.Detect(ctx)
	if status != nil && status.Installed {
		msg := "installed successfully"
		if runtime.GOOS == "windows" {
			msg += ". Note: some OpenClaw features work best with WSL2 on Windows"
		}
		return &InstallResult{
			Tool:    ToolOpenClaw,
			Success: true,
			Version: status.Version,
			Message: msg,
		}, nil
	}

	return &InstallResult{
		Tool:    ToolOpenClaw,
		Success: false,
		Message: "install command succeeded but binary not found in PATH",
	}, nil
}

// Update updates OpenClaw to the latest version
func (o *OpenClawInstaller) Update(ctx context.Context) (*InstallResult, error) {
	bunPath, err := o.runtime.EnsureBun(ctx)
	if err != nil {
		return nil, fmt.Errorf("bun required for OpenClaw update: %w", err)
	}

	updateCtx, cancel := context.WithTimeout(ctx, time.Duration(DefaultInstallTimeout)*time.Second)
	defer cancel()

	cmd := exec.CommandContext(updateCtx, bunPath, "install", "-g", OpenClawNpmPackage+"@latest")
	hideWindow(cmd)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return &InstallResult{
			Tool:    ToolOpenClaw,
			Success: false,
			Message: fmt.Sprintf("update failed: %s", strings.TrimSpace(string(output))),
		}, nil
	}

	status, _ := o.Detect(ctx)
	version := ""
	if status != nil {
		version = status.Version
	}

	return &InstallResult{
		Tool:    ToolOpenClaw,
		Success: true,
		Version: version,
		Message: "updated successfully",
	}, nil
}

// Uninstall removes OpenClaw via bun/npm
func (o *OpenClawInstaller) Uninstall(ctx context.Context) (*InstallResult, error) {
	bunPath, err := o.runtime.EnsureBun(ctx)
	if err != nil {
		return nil, fmt.Errorf("bun required for OpenClaw uninstall: %w", err)
	}

	uninstallCtx, cancel := context.WithTimeout(ctx, time.Duration(DefaultInstallTimeout)*time.Second)
	defer cancel()

	cmd := exec.CommandContext(uninstallCtx, bunPath, "uninstall", "-g", OpenClawNpmPackage)
	hideWindow(cmd)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return &InstallResult{
			Tool:    ToolOpenClaw,
			Success: false,
			Message: fmt.Sprintf("uninstall failed: %s", strings.TrimSpace(string(output))),
		}, nil
	}

	return &InstallResult{Tool: ToolOpenClaw, Success: true, Message: "uninstalled successfully"}, nil
}

// ConfigureModel writes the model ID into OpenClaw's openclaw.json provider section
func (o *OpenClawInstaller) ConfigureModel(ctx context.Context, model string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	configPath := filepath.Join(home, ".openclaw", "openclaw.json")

	cfg := make(map[string]interface{})
	if data, err := os.ReadFile(configPath); err == nil {
		_ = json.Unmarshal(data, &cfg)
	}

	provider, _ := cfg["provider"].(map[string]interface{})
	if provider == nil {
		provider = make(map[string]interface{})
	}
	provider["model"] = model
	cfg["provider"] = provider

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal openclaw config: %w", err)
	}

	return os.WriteFile(configPath, data, 0600)
}

// ConfigureProxy writes proxy/API settings into OpenClaw's openclaw.json
func (o *OpenClawInstaller) ConfigureProxy(_ context.Context, endpoint, apiKey string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	configDir := filepath.Join(home, ".openclaw")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create openclaw config directory: %w", err)
	}

	configPath := filepath.Join(configDir, "openclaw.json")

	// Load existing config as generic map to preserve unknown fields
	cfg := make(map[string]interface{})
	if data, err := os.ReadFile(configPath); err == nil {
		if err := json.Unmarshal(data, &cfg); err != nil {
			cfg = make(map[string]interface{})
		}
	}

	provider, _ := cfg["provider"].(map[string]interface{})
	if provider == nil {
		provider = make(map[string]interface{})
	}
	provider["type"] = "anthropic"
	if apiKey != "" {
		provider["api_key"] = apiKey
	}
	if endpoint != "" {
		provider["base_url"] = endpoint
	}
	cfg["provider"] = provider

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal openclaw config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write openclaw config: %w", err)
	}

	return nil
}

// findExecutable locates the openclaw binary
func (o *OpenClawInstaller) findExecutable() (string, error) {
	// Try the primary binary name
	if path, err := exec.LookPath("openclaw"); err == nil {
		return path, nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	var candidates []string
	switch runtime.GOOS {
	case "windows":
		// Bun global bin and npm global
		candidates = []string{
			filepath.Join(home, ".bun", "bin", "openclaw.exe"),
			filepath.Join(os.Getenv("APPDATA"), "npm", "openclaw.cmd"),
			filepath.Join(os.Getenv("APPDATA"), "npm", "openclaw.exe"),
		}
	default:
		candidates = []string{
			filepath.Join(home, ".bun", "bin", "openclaw"),
			"/usr/local/bin/openclaw",
		}
	}

	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}
	}

	return "", fmt.Errorf("openclaw executable not found")
}
