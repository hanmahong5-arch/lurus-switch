package installer

import (
	"context"
	"fmt"
)

// ToolStatus represents the current status of a CLI tool
type ToolStatus struct {
	Name            string `json:"name"`
	Installed       bool   `json:"installed"`
	Version         string `json:"version"`
	LatestVersion   string `json:"latestVersion"`
	UpdateAvailable bool   `json:"updateAvailable"`
	Path            string `json:"path"`
}

// InstallResult represents the outcome of an install/update/uninstall operation
type InstallResult struct {
	Tool    string `json:"tool"`
	Success bool   `json:"success"`
	Version string `json:"version"`
	Message string `json:"message"`
}

// ToolInstaller defines the interface for installing and managing a CLI tool
type ToolInstaller interface {
	// Detect checks if the tool is installed and returns its status
	Detect(ctx context.Context) (*ToolStatus, error)
	// Install installs the tool via bun or GitHub binary download
	Install(ctx context.Context) (*InstallResult, error)
	// Update updates the tool to the latest version
	Update(ctx context.Context) (*InstallResult, error)
	// Uninstall removes the tool
	Uninstall(ctx context.Context) (*InstallResult, error)
	// ConfigureProxy writes proxy/API settings into the tool's config
	ConfigureProxy(ctx context.Context, endpoint, apiKey string) error
}

// Manager holds all tool installers and provides aggregate operations
type Manager struct {
	installers  map[string]ToolInstaller
	runtime     *BunRuntime
	nodeRuntime *NodeRuntime
}

// NewManager creates a new installer manager with all tool installers
func NewManager() *Manager {
	rt := NewBunRuntime()
	nrt := NewNodeRuntime()
	return &Manager{
		installers: map[string]ToolInstaller{
			ToolClaude:   NewClaudeInstaller(rt),
			ToolCodex:    NewCodexInstaller(rt),
			ToolGemini:   NewGeminiInstaller(rt),
			ToolPicoClaw: NewPicoClawInstaller(),
			ToolNullClaw: NewNullClawInstaller(),
			ToolZeroClaw: NewZeroClawInstaller(),
			ToolOpenClaw: NewOpenClawInstaller(rt),
		},
		runtime:     rt,
		nodeRuntime: nrt,
	}
}

// DetectAll checks the installation status of all tools
func (m *Manager) DetectAll(ctx context.Context) (map[string]*ToolStatus, error) {
	results := make(map[string]*ToolStatus)
	for name, inst := range m.installers {
		status, err := inst.Detect(ctx)
		if err != nil {
			// Return a not-installed status on detection error rather than failing entirely
			results[name] = &ToolStatus{
				Name:      name,
				Installed: false,
				Version:   "",
			}
			continue
		}
		results[name] = status
	}
	return results, nil
}

// InstallTool installs a specific tool by name, resolving runtime dependencies first
func (m *Manager) InstallTool(ctx context.Context, name string) (*InstallResult, error) {
	inst, ok := m.installers[name]
	if !ok {
		return nil, fmt.Errorf("unknown tool: %s, expected one of: claude, codex, gemini, picoclaw, nullclaw, zeroclaw, openclaw", name)
	}

	// Resolve runtime dependencies before installing
	depResults, err := m.EnsureToolDependencies(ctx, name)
	if err != nil {
		return &InstallResult{
			Tool:    name,
			Success: false,
			Message: fmt.Sprintf("dependency resolution failed: %v", err),
		}, nil
	}
	for _, dr := range depResults {
		if !dr.Success {
			return &InstallResult{
				Tool:    name,
				Success: false,
				Message: fmt.Sprintf("required dependency %s not available: %s", dr.RuntimeID, dr.Message),
			}, nil
		}
	}

	return inst.Install(ctx)
}

// InstallAll installs all tools sequentially to avoid bun global install conflicts
func (m *Manager) InstallAll(ctx context.Context) []InstallResult {
	order := []string{ToolClaude, ToolCodex, ToolGemini, ToolPicoClaw, ToolNullClaw, ToolZeroClaw, ToolOpenClaw}
	var results []InstallResult
	for _, name := range order {
		result, err := m.installers[name].Install(ctx)
		if err != nil {
			results = append(results, InstallResult{
				Tool:    name,
				Success: false,
				Message: fmt.Sprintf("install failed: %v", err),
			})
			continue
		}
		results = append(results, *result)
	}
	return results
}

// UpdateTool updates a specific tool by name
func (m *Manager) UpdateTool(ctx context.Context, name string) (*InstallResult, error) {
	inst, ok := m.installers[name]
	if !ok {
		return nil, fmt.Errorf("unknown tool: %s, expected one of: claude, codex, gemini, picoclaw, nullclaw, zeroclaw, openclaw", name)
	}
	return inst.Update(ctx)
}

// UpdateAll updates all tools sequentially
func (m *Manager) UpdateAll(ctx context.Context) []InstallResult {
	order := []string{ToolClaude, ToolCodex, ToolGemini, ToolPicoClaw, ToolNullClaw, ToolZeroClaw, ToolOpenClaw}
	var results []InstallResult
	for _, name := range order {
		result, err := m.installers[name].Update(ctx)
		if err != nil {
			results = append(results, InstallResult{
				Tool:    name,
				Success: false,
				Message: fmt.Sprintf("update failed: %v", err),
			})
			continue
		}
		results = append(results, *result)
	}
	return results
}

// UninstallTool uninstalls a specific tool by name
func (m *Manager) UninstallTool(ctx context.Context, name string) (*InstallResult, error) {
	inst, ok := m.installers[name]
	if !ok {
		return nil, fmt.Errorf("unknown tool: %s, expected one of: claude, codex, gemini, picoclaw, nullclaw, zeroclaw, openclaw", name)
	}
	return inst.Uninstall(ctx)
}

// UninstallAll uninstalls all tools sequentially
func (m *Manager) UninstallAll(ctx context.Context) []InstallResult {
	order := []string{ToolClaude, ToolCodex, ToolGemini, ToolPicoClaw, ToolNullClaw, ToolZeroClaw, ToolOpenClaw}
	var results []InstallResult
	for _, name := range order {
		result, err := m.installers[name].Uninstall(ctx)
		if err != nil {
			results = append(results, InstallResult{
				Tool:    name,
				Success: false,
				Message: fmt.Sprintf("uninstall failed: %v", err),
			})
			continue
		}
		results = append(results, *result)
	}
	return results
}

// ConfigureAllProxy applies proxy settings to all installed tools, skipping uninstalled ones
func (m *Manager) ConfigureAllProxy(ctx context.Context, endpoint, apiKey string) map[string]error {
	errs := make(map[string]error)
	for name, inst := range m.installers {
		status, err := inst.Detect(ctx)
		if err != nil || !status.Installed {
			continue
		}
		if err := inst.ConfigureProxy(ctx, endpoint, apiKey); err != nil {
			errs[name] = err
		}
	}
	return errs
}

// GetRuntime returns the bun runtime manager
func (m *Manager) GetRuntime() *BunRuntime {
	return m.runtime
}
