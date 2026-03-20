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

// PicoClawInstaller handles PicoClaw CLI installation via GitHub Releases binary download
type PicoClawInstaller struct {
	overrideURL    string
	expectedSHA256 string
	progressFn     func(int64, int64, int)
}

// SetOverrideURL sets a manifest-provided download URL, bypassing the GitHub API.
func (p *PicoClawInstaller) SetOverrideURL(url string) { p.overrideURL = url }

// SetExpectedSHA256 sets the expected SHA-256 hex digest for integrity verification.
func (p *PicoClawInstaller) SetExpectedSHA256(hash string) { p.expectedSHA256 = hash }

// SetProgressFn attaches a download-progress callback.
func (p *PicoClawInstaller) SetProgressFn(fn func(int64, int64, int)) { p.progressFn = fn }

// NewPicoClawInstaller creates a new PicoClawInstaller
func NewPicoClawInstaller() *PicoClawInstaller {
	return &PicoClawInstaller{}
}

func (p *PicoClawInstaller) binaryConfig() BinaryToolConfig {
	return BinaryToolConfig{
		Name:         ToolPicoClaw,
		GitHubOwner:  PicoClawGitHubOwner,
		GitHubRepo:   PicoClawGitHubRepo,
		BinaryName:   PicoClawBinaryName,
		DefaultModel: DefaultPicoClawModel,
	}
}

// Detect checks if PicoClaw is installed and returns its status
func (p *PicoClawInstaller) Detect(ctx context.Context) (*ToolStatus, error) {
	status := &ToolStatus{Name: ToolPicoClaw, Installed: false}

	path, err := findBinaryExecutable(PicoClawBinaryName)
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

// Install downloads the PicoClaw binary from GitHub Releases
func (p *PicoClawInstaller) Install(ctx context.Context) (*InstallResult, error) {
	installCtx, cancel := context.WithTimeout(ctx, time.Duration(DefaultInstallTimeout)*time.Second)
	defer cancel()

	result, err := downloadAndInstallBinary(installCtx, p.binaryConfig(), p.overrideURL, p.expectedSHA256, p.progressFn)
	if err != nil {
		return result, err
	}

	// If download succeeded, verify with Detect
	if result.Success {
		if status, _ := p.Detect(ctx); status != nil && status.Installed && status.Version != "unknown" {
			result.Version = status.Version
		}
	}
	return result, nil
}

// Update re-downloads the latest PicoClaw binary
func (p *PicoClawInstaller) Update(ctx context.Context) (*InstallResult, error) {
	_ = removeManagedBinary(PicoClawBinaryName)
	return p.Install(ctx)
}

// Uninstall removes the PicoClaw binary and cached download
func (p *PicoClawInstaller) Uninstall(_ context.Context) (*InstallResult, error) {
	if err := removeManagedBinary(PicoClawBinaryName); err != nil {
		return &InstallResult{
			Tool:    ToolPicoClaw,
			Success: false,
			Message: fmt.Sprintf("failed to remove binary: %v", err),
		}, nil
	}
	_ = os.RemoveAll(toolCacheDir(ToolPicoClaw))
	return &InstallResult{Tool: ToolPicoClaw, Success: true, Message: "uninstalled successfully"}, nil
}

// ConfigureModel updates the model_name for the "code-switch" entry in PicoClaw's config
func (p *PicoClawInstaller) ConfigureModel(ctx context.Context, model string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	configPath := filepath.Join(home, ".picoclaw", "config.json")

	cfg := make(map[string]interface{})
	if data, err := os.ReadFile(configPath); err == nil {
		_ = json.Unmarshal(data, &cfg)
	}

	modelList, _ := cfg["model_list"].([]interface{})
	found := false
	for i, m := range modelList {
		if entry, ok := m.(map[string]interface{}); ok {
			if entry["name"] == "code-switch" {
				entry["model_name"] = model
				modelList[i] = entry
				found = true
				break
			}
		}
	}
	if !found {
		modelList = append(modelList, map[string]interface{}{
			"name":       "code-switch",
			"model_name": model,
		})
	}
	cfg["model_list"] = modelList

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal picoclaw config: %w", err)
	}

	return os.WriteFile(configPath, data, 0600)
}

// ConfigureProxy writes NewAPI proxy settings into PicoClaw's config
func (p *PicoClawInstaller) ConfigureProxy(_ context.Context, endpoint, apiKey string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	configDir := filepath.Join(home, ".picoclaw")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create picoclaw config directory: %w", err)
	}

	configPath := filepath.Join(configDir, "config.json")

	// Load existing config or start fresh
	cfg := make(map[string]interface{})
	if data, err := os.ReadFile(configPath); err == nil {
		if err := json.Unmarshal(data, &cfg); err != nil {
			// Existing config is corrupt, start fresh
			cfg = make(map[string]interface{})
		}
	}

	// Upsert "code-switch" entry in model_list
	modelList, _ := cfg["model_list"].([]interface{})
	switchEntry := map[string]interface{}{
		"name":       "code-switch",
		"api_base":   endpoint,
		"api_key":    apiKey,
		"model_name": DefaultPicoClawModel,
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
		return fmt.Errorf("failed to marshal picoclaw config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write picoclaw config: %w", err)
	}

	return nil
}
