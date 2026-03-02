package installer

import "context"

// RuntimeID identifies a runtime dependency
type RuntimeID string

const (
	RuntimeNodeJS RuntimeID = "nodejs"
	RuntimeBun    RuntimeID = "bun"
	RuntimeNone   RuntimeID = "none"
)

// depGraph maps each tool to its required runtimes (in dependency order)
var depGraph = map[string][]RuntimeID{
	ToolClaude:   {RuntimeNodeJS, RuntimeBun},
	ToolCodex:    {RuntimeNodeJS, RuntimeBun},
	ToolGemini:   {RuntimeNodeJS, RuntimeBun},
	ToolOpenClaw: {RuntimeNodeJS, RuntimeBun},
	ToolPicoClaw: {RuntimeNone},
	ToolNullClaw: {RuntimeNone},
	ToolZeroClaw: {RuntimeNone},
}

// RuntimeStatus describes the current state of a single runtime dependency
type RuntimeStatus struct {
	ID        string   `json:"id"`
	Name      string   `json:"name"`
	Installed bool     `json:"installed"`
	Version   string   `json:"version"`
	Path      string   `json:"path"`
	Required  bool     `json:"required"`
	Tools     []string `json:"tools"`
}

// DepCheckResult is the complete dependency tree status returned to the frontend
type DepCheckResult struct {
	Runtimes []RuntimeStatus `json:"runtimes"`
	AllMet   bool            `json:"allMet"`
}

// DepInstallResult describes the result of installing a single runtime
type DepInstallResult struct {
	RuntimeID string `json:"runtimeId"`
	Success   bool   `json:"success"`
	Version   string `json:"version"`
	Message   string `json:"message"`
}

// CheckDependencies returns the full dependency tree status
func (m *Manager) CheckDependencies(ctx context.Context) (*DepCheckResult, error) {
	// Collect which tools need each runtime
	runtimeTools := make(map[RuntimeID][]string)
	for tool, deps := range depGraph {
		for _, rid := range deps {
			if rid != RuntimeNone {
				runtimeTools[rid] = append(runtimeTools[rid], tool)
			}
		}
	}

	// Deduplicate tool lists
	for rid, tools := range runtimeTools {
		runtimeTools[rid] = dedup(tools)
	}

	var runtimes []RuntimeStatus
	allMet := true

	// Check Node.js
	if tools, ok := runtimeTools[RuntimeNodeJS]; ok {
		nodeStatus := RuntimeStatus{
			ID:       string(RuntimeNodeJS),
			Name:     "Node.js",
			Required: true,
			Tools:    tools,
		}
		if m.nodeRuntime != nil {
			if path, err := m.nodeRuntime.FindNode(); err == nil {
				nodeStatus.Installed = true
				nodeStatus.Path = path
				if ver, err := m.nodeRuntime.GetVersion(ctx); err == nil {
					nodeStatus.Version = ver
				}
			}
		}
		if !nodeStatus.Installed {
			allMet = false
		}
		runtimes = append(runtimes, nodeStatus)
	}

	// Check Bun
	if tools, ok := runtimeTools[RuntimeBun]; ok {
		bunStatus := RuntimeStatus{
			ID:       string(RuntimeBun),
			Name:     "Bun",
			Required: true,
			Tools:    tools,
		}
		if m.runtime != nil {
			if path, err := m.runtime.FindBun(); err == nil {
				bunStatus.Installed = true
				bunStatus.Path = path
			}
		}
		if !bunStatus.Installed {
			allMet = false
		}
		runtimes = append(runtimes, bunStatus)
	}

	// Add standalone tools info
	var standaloneTools []string
	for tool, deps := range depGraph {
		if len(deps) == 1 && deps[0] == RuntimeNone {
			standaloneTools = append(standaloneTools, tool)
		}
	}
	if len(standaloneTools) > 0 {
		runtimes = append(runtimes, RuntimeStatus{
			ID:        string(RuntimeNone),
			Name:      "Standalone",
			Installed: true,
			Required:  false,
			Tools:     dedup(standaloneTools),
		})
	}

	return &DepCheckResult{Runtimes: runtimes, AllMet: allMet}, nil
}

// EnsureToolDependencies resolves all runtime dependencies for a given tool
func (m *Manager) EnsureToolDependencies(ctx context.Context, tool string) ([]DepInstallResult, error) {
	deps, ok := depGraph[tool]
	if !ok {
		return nil, nil
	}

	var results []DepInstallResult
	for _, rid := range deps {
		result, err := m.InstallDependency(ctx, string(rid))
		if err != nil {
			results = append(results, DepInstallResult{
				RuntimeID: string(rid),
				Success:   false,
				Message:   err.Error(),
			})
			return results, err
		}
		if result != nil {
			results = append(results, *result)
			if !result.Success {
				return results, nil
			}
		}
	}
	return results, nil
}

// InstallDependency installs a single runtime by ID
func (m *Manager) InstallDependency(ctx context.Context, runtimeID string) (*DepInstallResult, error) {
	rid := RuntimeID(runtimeID)

	switch rid {
	case RuntimeNodeJS:
		if m.nodeRuntime != nil && m.nodeRuntime.IsInstalled() {
			ver, _ := m.nodeRuntime.GetVersion(ctx)
			return &DepInstallResult{RuntimeID: runtimeID, Success: true, Version: ver, Message: "already installed"}, nil
		}
		if m.nodeRuntime == nil {
			return &DepInstallResult{RuntimeID: runtimeID, Success: false, Message: "node runtime not initialized"}, nil
		}
		path, err := m.nodeRuntime.InstallNode(ctx)
		if err != nil {
			return &DepInstallResult{RuntimeID: runtimeID, Success: false, Message: err.Error()}, nil
		}
		ver, _ := m.nodeRuntime.GetVersion(ctx)
		return &DepInstallResult{RuntimeID: runtimeID, Success: true, Version: ver, Message: "installed at " + path}, nil

	case RuntimeBun:
		if m.runtime != nil && m.runtime.IsInstalled() {
			return &DepInstallResult{RuntimeID: runtimeID, Success: true, Message: "already installed"}, nil
		}
		if m.runtime == nil {
			return &DepInstallResult{RuntimeID: runtimeID, Success: false, Message: "bun runtime not initialized"}, nil
		}
		path, err := m.runtime.InstallBun(ctx)
		if err != nil {
			return &DepInstallResult{RuntimeID: runtimeID, Success: false, Message: err.Error()}, nil
		}
		return &DepInstallResult{RuntimeID: runtimeID, Success: true, Message: "installed at " + path}, nil

	case RuntimeNone:
		return &DepInstallResult{RuntimeID: runtimeID, Success: true, Message: "no runtime needed"}, nil

	default:
		return &DepInstallResult{RuntimeID: runtimeID, Success: false, Message: "unknown runtime: " + runtimeID}, nil
	}
}

// GetNodeRuntime returns the node runtime manager
func (m *Manager) GetNodeRuntime() *NodeRuntime {
	return m.nodeRuntime
}

// GetToolDependencies returns the runtime dependencies for a given tool name
func GetToolDependencies(tool string) []RuntimeID {
	deps, ok := depGraph[tool]
	if !ok {
		return nil
	}
	return deps
}

// dedup removes duplicate strings from a slice
func dedup(items []string) []string {
	seen := make(map[string]bool, len(items))
	var result []string
	for _, item := range items {
		if !seen[item] {
			seen[item] = true
			result = append(result, item)
		}
	}
	return result
}
