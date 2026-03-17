package installer

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

// NullClawInstaller handles NullClaw CLI installation via GitHub Releases binary download
type NullClawInstaller struct {
	overrideURL string
	progressFn  func(int64, int64, int)
}

// SetOverrideURL sets a manifest-provided download URL, bypassing the GitHub API.
func (n *NullClawInstaller) SetOverrideURL(url string) { n.overrideURL = url }

// SetProgressFn attaches a download-progress callback.
func (n *NullClawInstaller) SetProgressFn(fn func(int64, int64, int)) { n.progressFn = fn }

// NewNullClawInstaller creates a new NullClawInstaller
func NewNullClawInstaller() *NullClawInstaller {
	return &NullClawInstaller{}
}

func (n *NullClawInstaller) binaryConfig() BinaryToolConfig {
	return BinaryToolConfig{
		Name:         ToolNullClaw,
		GitHubOwner:  NullClawGitHubOwner,
		GitHubRepo:   NullClawGitHubRepo,
		BinaryName:   NullClawBinaryName,
		DefaultModel: DefaultNullClawModel,
	}
}

// Detect checks if NullClaw is installed and returns its status
func (n *NullClawInstaller) Detect(ctx context.Context) (*ToolStatus, error) {
	status := &ToolStatus{Name: ToolNullClaw, Installed: false}

	path, err := findBinaryExecutable(NullClawBinaryName)
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

// Install downloads the NullClaw binary from GitHub Releases
func (n *NullClawInstaller) Install(ctx context.Context) (*InstallResult, error) {
	installCtx, cancel := context.WithTimeout(ctx, time.Duration(DefaultInstallTimeout)*time.Second)
	defer cancel()

	result, err := downloadAndInstallBinary(installCtx, n.binaryConfig(), n.overrideURL, n.progressFn)
	if err != nil {
		return result, err
	}

	if result.Success {
		if status, _ := n.Detect(ctx); status != nil && status.Installed && status.Version != "unknown" {
			result.Version = status.Version
		}
	}
	return result, nil
}

// Update re-downloads the latest NullClaw binary
func (n *NullClawInstaller) Update(ctx context.Context) (*InstallResult, error) {
	_ = removeManagedBinary(NullClawBinaryName)
	return n.Install(ctx)
}

// Uninstall removes the NullClaw binary and cached download
func (n *NullClawInstaller) Uninstall(_ context.Context) (*InstallResult, error) {
	if err := removeManagedBinary(NullClawBinaryName); err != nil {
		return &InstallResult{
			Tool:    ToolNullClaw,
			Success: false,
			Message: fmt.Sprintf("failed to remove binary: %v", err),
		}, nil
	}
	_ = os.RemoveAll(toolCacheDir(ToolNullClaw))
	return &InstallResult{Tool: ToolNullClaw, Success: true, Message: "uninstalled successfully"}, nil
}

// ConfigureProxy writes NewAPI proxy settings into NullClaw's config
func (n *NullClawInstaller) ConfigureProxy(_ context.Context, endpoint, apiKey string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	configDir := filepath.Join(home, ".nullclaw")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create nullclaw config directory: %w", err)
	}

	configPath := filepath.Join(configDir, "config.json")

	cfg := make(map[string]interface{})
	if data, err := os.ReadFile(configPath); err == nil {
		if err := json.Unmarshal(data, &cfg); err != nil {
			cfg = make(map[string]interface{})
		}
	}

	modelList, _ := cfg["model_list"].([]interface{})
	switchEntry := map[string]interface{}{
		"name":       "code-switch",
		"api_base":   endpoint,
		"api_key":    apiKey,
		"model_name": DefaultNullClawModel,
	}

	found := false
	for i, m := range modelList {
		if entry, ok := m.(map[string]interface{}); ok {
			if entry["name"] == "code-switch" {
				modelList[i] = switchEntry
				found = true
				break
			}
		}
	}
	if !found {
		modelList = append(modelList, switchEntry)
	}
	cfg["model_list"] = modelList

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal nullclaw config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write nullclaw config: %w", err)
	}

	return nil
}
