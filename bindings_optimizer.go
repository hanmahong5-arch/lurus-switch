package main

import (
	"fmt"
	"strings"

	"lurus-switch/internal/installer"
	"lurus-switch/internal/optimizer"
	"lurus-switch/internal/toolhealth"
)

// AnalyzeOptimizations gathers environment data and returns actionable optimization items.
func (a *App) AnalyzeOptimizations() *optimizer.AnalysisResult {
	statuses, _ := a.instMgr.DetectAll(a.ctx)
	healthResults := toolhealth.CheckAll()

	var depCheck *installer.DepCheckResult
	if dc, err := a.instMgr.CheckDependencies(a.ctx); err == nil {
		depCheck = dc
	}

	gwRunning := false
	gwURL := a.gatewayBaseURL()
	installedCount := 0
	boundCount := 0

	if a.gatewaySrv != nil {
		st := a.gatewaySrv.Status()
		gwRunning = st.Running
	}

	for _, tool := range managedTools {
		if ts, ok := statuses[tool]; ok && ts.Installed {
			installedCount++
			if isToolBoundToGateway(tool, gwURL) {
				boundCount++
			}
		}
	}

	deps := &optimizer.Deps{
		ToolStatuses:   statuses,
		HealthResults:  healthResults,
		DepCheck:       depCheck,
		GatewayRunning: gwRunning,
		GatewayURL:     gwURL,
		InstalledCount: installedCount,
		BoundCount:     boundCount,
	}

	return optimizer.Analyze(deps)
}

// ApplyOptimization dispatches a single optimization fix by its ID.
func (a *App) ApplyOptimization(id string) *optimizer.FixResult {
	switch {
	case strings.HasPrefix(id, "install-runtime-"):
		target := strings.TrimPrefix(id, "install-runtime-")
		result, err := a.instMgr.InstallDependency(a.ctx, target)
		if err != nil {
			return &optimizer.FixResult{ID: id, Status: optimizer.FixFailed, Error: err.Error()}
		}
		if result != nil && result.Success {
			return &optimizer.FixResult{ID: id, Status: optimizer.FixSuccess, Message: result.Message}
		}
		msg := "install failed"
		if result != nil {
			msg = result.Message
		}
		return &optimizer.FixResult{ID: id, Status: optimizer.FixFailed, Error: msg}

	case strings.HasPrefix(id, "install-tool-"):
		target := strings.TrimPrefix(id, "install-tool-")
		result, err := a.instMgr.InstallTool(a.ctx, target)
		if err != nil {
			return &optimizer.FixResult{ID: id, Status: optimizer.FixFailed, Error: err.Error()}
		}
		if result != nil && result.Success {
			return &optimizer.FixResult{ID: id, Status: optimizer.FixSuccess, Message: result.Message}
		}
		msg := "install failed"
		if result != nil {
			msg = result.Message
		}
		return &optimizer.FixResult{ID: id, Status: optimizer.FixFailed, Error: msg}

	case strings.HasPrefix(id, "update-tool-"):
		target := strings.TrimPrefix(id, "update-tool-")
		result, err := a.instMgr.UpdateTool(a.ctx, target)
		if err != nil {
			return &optimizer.FixResult{ID: id, Status: optimizer.FixFailed, Error: err.Error()}
		}
		if result != nil && result.Success {
			return &optimizer.FixResult{ID: id, Status: optimizer.FixSuccess, Message: result.Message}
		}
		msg := "update failed"
		if result != nil {
			msg = result.Message
		}
		return &optimizer.FixResult{ID: id, Status: optimizer.FixFailed, Error: msg}

	case id == "start-gateway":
		if err := a.StartGateway(); err != nil {
			return &optimizer.FixResult{ID: id, Status: optimizer.FixFailed, Error: err.Error()}
		}
		return &optimizer.FixResult{ID: id, Status: optimizer.FixSuccess, Message: "gateway started"}

	case id == "connect-all-tools":
		results := a.AutoConfigureToolsForGateway()
		allOK := true
		var msgs []string
		for _, r := range results {
			if !r.Success {
				allOK = false
				msgs = append(msgs, fmt.Sprintf("%s: %s", r.Tool, r.Message))
			}
		}
		if allOK {
			return &optimizer.FixResult{ID: id, Status: optimizer.FixSuccess, Message: "all tools connected"}
		}
		return &optimizer.FixResult{ID: id, Status: optimizer.FixFailed, Error: strings.Join(msgs, "; ")}

	case strings.HasPrefix(id, "fix-config-"):
		target := strings.TrimPrefix(id, "fix-config-")
		result, err := a.AutoFixToolConfig(target)
		if err != nil {
			return &optimizer.FixResult{ID: id, Status: optimizer.FixFailed, Error: err.Error()}
		}
		if result != nil && result.Success {
			return &optimizer.FixResult{ID: id, Status: optimizer.FixSuccess, Message: result.Message}
		}
		msg := "fix failed"
		if result != nil {
			msg = result.Message
		}
		return &optimizer.FixResult{ID: id, Status: optimizer.FixFailed, Error: msg}

	case id == "install-git":
		return &optimizer.FixResult{
			ID:     id,
			Status: optimizer.FixSkipped,
			Error:  "git must be installed manually; download from https://git-scm.com",
		}

	default:
		return &optimizer.FixResult{
			ID:     id,
			Status: optimizer.FixFailed,
			Error:  fmt.Sprintf("unknown optimization ID: %s", id),
		}
	}
}

// ApplyAllOptimizations iterates all auto-fixable optimizations and applies each.
func (a *App) ApplyAllOptimizations() []optimizer.FixResult {
	analysis := a.AnalyzeOptimizations()
	var results []optimizer.FixResult

	for _, opt := range analysis.Optimizations {
		if !opt.AutoFixable {
			results = append(results, optimizer.FixResult{
				ID:     opt.ID,
				Status: optimizer.FixSkipped,
				Error:  "not auto-fixable",
			})
			continue
		}
		result := a.ApplyOptimization(opt.ID)
		results = append(results, *result)
	}

	return results
}
