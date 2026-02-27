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

// PicoClawInstaller handles PicoClaw CLI installation and configuration via pip
type PicoClawInstaller struct{}

// NewPicoClawInstaller creates a new PicoClawInstaller
func NewPicoClawInstaller() *PicoClawInstaller {
	return &PicoClawInstaller{}
}

// Detect checks if PicoClaw is installed and returns its status
func (p *PicoClawInstaller) Detect(ctx context.Context) (*ToolStatus, error) {
	status := &ToolStatus{Name: ToolPicoClaw, Installed: false}

	path, err := p.findExecutable()
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

// Install installs PicoClaw globally via pip
func (p *PicoClawInstaller) Install(ctx context.Context) (*InstallResult, error) {
	pythonPath, err := p.findPython()
	if err != nil {
		return nil, fmt.Errorf("python required for installation: %w", err)
	}

	installCtx, cancel := context.WithTimeout(ctx, time.Duration(DefaultInstallTimeout)*time.Second)
	defer cancel()

	cmd := exec.CommandContext(installCtx, pythonPath, "-m", "pip", "install", PicoClawPipPackage)
	hideWindow(cmd)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return &InstallResult{
			Tool:    ToolPicoClaw,
			Success: false,
			Message: fmt.Sprintf("install failed: %s", strings.TrimSpace(string(output))),
		}, nil
	}

	// Verify installation
	status, _ := p.Detect(ctx)
	if status != nil && status.Installed {
		return &InstallResult{
			Tool:    ToolPicoClaw,
			Success: true,
			Version: status.Version,
			Message: "installed successfully",
		}, nil
	}

	return &InstallResult{
		Tool:    ToolPicoClaw,
		Success: false,
		Message: "install command succeeded but binary not found in PATH",
	}, nil
}

// Update updates PicoClaw to the latest version
func (p *PicoClawInstaller) Update(ctx context.Context) (*InstallResult, error) {
	pythonPath, err := p.findPython()
	if err != nil {
		return nil, fmt.Errorf("python required for update: %w", err)
	}

	updateCtx, cancel := context.WithTimeout(ctx, time.Duration(DefaultInstallTimeout)*time.Second)
	defer cancel()

	cmd := exec.CommandContext(updateCtx, pythonPath, "-m", "pip", "install", "--upgrade", PicoClawPipPackage)
	hideWindow(cmd)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return &InstallResult{
			Tool:    ToolPicoClaw,
			Success: false,
			Message: fmt.Sprintf("update failed: %s", strings.TrimSpace(string(output))),
		}, nil
	}

	status, _ := p.Detect(ctx)
	version := ""
	if status != nil {
		version = status.Version
	}

	return &InstallResult{
		Tool:    ToolPicoClaw,
		Success: true,
		Version: version,
		Message: "updated successfully",
	}, nil
}

// ConfigureProxy writes NewAPI proxy settings into PicoClaw's config
func (p *PicoClawInstaller) ConfigureProxy(ctx context.Context, endpoint, apiKey string) error {
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

// findExecutable locates the pclaw binary
func (p *PicoClawInstaller) findExecutable() (string, error) {
	if path, err := exec.LookPath("pclaw"); err == nil {
		return path, nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	var candidates []string
	switch runtime.GOOS {
	case "windows":
		// Check Python Scripts directories
		localAppData := os.Getenv("LOCALAPPDATA")
		appData := os.Getenv("APPDATA")
		candidates = append(candidates, p.globPythonScripts(localAppData, "pclaw.exe")...)
		candidates = append(candidates, p.globPythonScripts(appData, "pclaw.exe")...)
		candidates = append(candidates,
			filepath.Join(home, ".local", "bin", "pclaw.exe"),
		)
	default:
		candidates = []string{
			filepath.Join(home, ".local", "bin", "pclaw"),
			"/usr/local/bin/pclaw",
		}
	}

	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}
	}

	return "", fmt.Errorf("pclaw executable not found")
}

// findPython locates the python interpreter
func (p *PicoClawInstaller) findPython() (string, error) {
	// Try python3 first, then python
	for _, name := range []string{"python3", "python"} {
		if path, err := exec.LookPath(name); err == nil {
			return path, nil
		}
	}
	return "", fmt.Errorf("python not found in PATH")
}

// globPythonScripts searches for the executable in Python Scripts directories
func (p *PicoClawInstaller) globPythonScripts(baseDir, exeName string) []string {
	if baseDir == "" {
		return nil
	}

	var results []string
	pattern := filepath.Join(baseDir, "Programs", "Python", "*", "Scripts", exeName)
	if matches, err := filepath.Glob(pattern); err == nil {
		results = append(results, matches...)
	}

	pattern = filepath.Join(baseDir, "Python", "*", "Scripts", exeName)
	if matches, err := filepath.Glob(pattern); err == nil {
		results = append(results, matches...)
	}

	return results
}
