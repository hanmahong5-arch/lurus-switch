package installer

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
)

// ZeroClawInstaller handles ZeroClaw CLI installation via GitHub Releases
type ZeroClawInstaller struct{}

// NewZeroClawInstaller creates a new ZeroClawInstaller
func NewZeroClawInstaller() *ZeroClawInstaller {
	return &ZeroClawInstaller{}
}

func (z *ZeroClawInstaller) binaryConfig() BinaryToolConfig {
	return BinaryToolConfig{
		Name:         ToolZeroClaw,
		GitHubOwner:  ZeroClawGitHubOwner,
		GitHubRepo:   ZeroClawGitHubRepo,
		BinaryName:   ZeroClawBinaryName,
		DefaultModel: DefaultZeroClawModel,
	}
}

// Detect checks if ZeroClaw is installed and returns its status
func (z *ZeroClawInstaller) Detect(ctx context.Context) (*ToolStatus, error) {
	status := &ToolStatus{Name: ToolZeroClaw, Installed: false}

	path, err := findBinaryExecutable(ZeroClawBinaryName)
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

// Install downloads the ZeroClaw binary from GitHub Releases and places it in PATH
func (z *ZeroClawInstaller) Install(ctx context.Context) (*InstallResult, error) {
	installCtx, cancel := context.WithTimeout(ctx, time.Duration(DefaultInstallTimeout)*time.Second)
	defer cancel()

	result, err := downloadAndInstallBinary(installCtx, z.binaryConfig())
	if err != nil {
		return result, err
	}

	// If download succeeded, verify with Detect
	if result.Success {
		if status, _ := z.Detect(ctx); status != nil && status.Installed && status.Version != "unknown" {
			result.Version = status.Version
		}
	}
	return result, nil
}

// Update re-downloads the latest ZeroClaw binary
func (z *ZeroClawInstaller) Update(ctx context.Context) (*InstallResult, error) {
	_ = removeManagedBinary(ZeroClawBinaryName)
	return z.Install(ctx)
}

// Uninstall removes the ZeroClaw binary and cached download
func (z *ZeroClawInstaller) Uninstall(_ context.Context) (*InstallResult, error) {
	if err := removeManagedBinary(ZeroClawBinaryName); err != nil {
		return &InstallResult{
			Tool:    ToolZeroClaw,
			Success: false,
			Message: fmt.Sprintf("failed to remove binary: %v", err),
		}, nil
	}
	_ = os.RemoveAll(toolCacheDir(ToolZeroClaw))
	return &InstallResult{Tool: ToolZeroClaw, Success: true, Message: "uninstalled successfully"}, nil
}

// ConfigureProxy writes proxy/API settings into ZeroClaw's config.toml
func (z *ZeroClawInstaller) ConfigureProxy(_ context.Context, endpoint, apiKey string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	configDir := filepath.Join(home, ".zeroclaw")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create zeroclaw config directory: %w", err)
	}

	configPath := filepath.Join(configDir, "config.toml")

	// Load existing config as a generic map to preserve unknown fields
	cfg := make(map[string]interface{})
	if data, err := os.ReadFile(configPath); err == nil {
		_ = toml.Unmarshal(data, &cfg)
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

	var buf strings.Builder
	enc := toml.NewEncoder(&buf)
	if err := enc.Encode(cfg); err != nil {
		return fmt.Errorf("failed to encode zeroclaw config: %w", err)
	}

	if err := os.WriteFile(configPath, []byte(buf.String()), 0600); err != nil {
		return fmt.Errorf("failed to write zeroclaw config: %w", err)
	}

	return nil
}
