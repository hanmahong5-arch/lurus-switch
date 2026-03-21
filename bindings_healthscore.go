package main

import (
	"lurus-switch/internal/healthscore"
	"lurus-switch/internal/installer"
	"lurus-switch/internal/toolhealth"
)

// ComputeHealthScore gathers environment data and returns a full health score report.
func (a *App) ComputeHealthScore() *healthscore.ScoreReport {
	// Gather all dependencies for the score computation.
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

	// Check env keys: scan all managed tools for configured API keys.
	envKeys := make(map[string]bool)
	if a.envMgr != nil {
		keys, err := a.envMgr.ListAllKeys(managedTools)
		if err == nil {
			for _, entry := range keys {
				if entry.MaskedValue != "" {
					envKeys[entry.Tool] = true
				}
			}
		}
	}

	deps := &healthscore.Deps{
		ToolStatuses:   statuses,
		HealthResults:  healthResults,
		DepCheck:       depCheck,
		GatewayRunning: gwRunning,
		GatewayURL:     gwURL,
		AllToolsBound:  installedCount > 0 && boundCount == installedCount,
		InstalledCount: installedCount,
		BoundCount:     boundCount,
		EnvKeys:        envKeys,
	}

	return healthscore.Compute(deps)
}
