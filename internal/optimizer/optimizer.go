package optimizer

import (
	"os/exec"
	"sort"

	"lurus-switch/internal/installer"
	"lurus-switch/internal/toolhealth"
)

// Deps holds all data needed to produce optimization suggestions.
// The caller gathers this data and passes it in — the optimizer is stateless.
type Deps struct {
	ToolStatuses   map[string]*installer.ToolStatus
	HealthResults  map[string]*toolhealth.HealthResult
	DepCheck       *installer.DepCheckResult
	GatewayRunning bool
	GatewayURL     string
	InstalledCount int
	BoundCount     int
}

// autoFixableRuntimes lists runtimes that can be installed automatically.
var autoFixableRuntimes = map[string]bool{
	"bun":    true,
	"nodejs": true,
}

// Analyze examines the current environment and returns actionable optimizations.
func Analyze(d *Deps) *AnalysisResult {
	if d == nil {
		return &AnalysisResult{Optimizations: []Optimization{}}
	}

	var opts []Optimization

	opts = append(opts, analyzeRuntimes(d)...)
	opts = append(opts, analyzeTools(d)...)
	opts = append(opts, analyzeGateway(d)...)
	opts = append(opts, analyzeConfig(d)...)
	opts = append(opts, analyzeSystem()...)

	// Sort by priority ascending (critical first), then by ID for stable order.
	sort.Slice(opts, func(i, j int) bool {
		if opts[i].Priority != opts[j].Priority {
			return opts[i].Priority < opts[j].Priority
		}
		return opts[i].ID < opts[j].ID
	})

	fixable := 0
	for i := range opts {
		if opts[i].AutoFixable {
			fixable++
		}
	}

	return &AnalysisResult{
		Optimizations: opts,
		FixableCount:  fixable,
		TotalCount:    len(opts),
	}
}

// analyzeRuntimes checks for missing required runtimes.
func analyzeRuntimes(d *Deps) []Optimization {
	if d.DepCheck == nil {
		return nil
	}
	var opts []Optimization
	for _, rt := range d.DepCheck.Runtimes {
		if rt.Required && !rt.Installed {
			opts = append(opts, Optimization{
				ID:          "install-runtime-" + rt.ID,
				Category:    "runtime",
				Priority:    1,
				Title:       "optimizer.installRuntime.title",
				Description: "optimizer.installRuntime.desc",
				Action:      "install-runtime",
				Target:      rt.ID,
				AutoFixable: autoFixableRuntimes[rt.ID],
				Status:      FixPending,
			})
		}
	}
	return opts
}

// analyzeTools checks for tools that are not installed or have updates available.
func analyzeTools(d *Deps) []Optimization {
	var opts []Optimization
	for name, ts := range d.ToolStatuses {
		if !ts.Installed {
			opts = append(opts, Optimization{
				ID:          "install-tool-" + name,
				Category:    "tool",
				Priority:    3,
				Title:       "optimizer.installTool.title",
				Description: "optimizer.installTool.desc",
				Action:      "install-tool",
				Target:      name,
				AutoFixable: true,
				Status:      FixPending,
			})
		} else if ts.UpdateAvailable {
			opts = append(opts, Optimization{
				ID:          "update-tool-" + name,
				Category:    "tool",
				Priority:    2,
				Title:       "optimizer.updateTool.title",
				Description: "optimizer.updateTool.desc",
				Action:      "update-tool",
				Target:      name,
				AutoFixable: true,
				Status:      FixPending,
			})
		}
	}
	return opts
}

// analyzeGateway checks whether the gateway is running and tools are connected.
func analyzeGateway(d *Deps) []Optimization {
	var opts []Optimization

	if !d.GatewayRunning {
		opts = append(opts, Optimization{
			ID:          "start-gateway",
			Category:    "gateway",
			Priority:    1,
			Title:       "optimizer.startGateway.title",
			Description: "optimizer.startGateway.desc",
			Action:      "start-gateway",
			Target:      "",
			AutoFixable: true,
			Status:      FixPending,
		})
	}

	// Only suggest connecting tools when the gateway is running and some tools are not bound.
	if d.GatewayRunning && d.InstalledCount > 0 && d.BoundCount < d.InstalledCount {
		opts = append(opts, Optimization{
			ID:          "connect-all-tools",
			Category:    "gateway",
			Priority:    1,
			Title:       "optimizer.connectTools.title",
			Description: "optimizer.connectTools.desc",
			Action:      "connect-gateway",
			Target:      "",
			AutoFixable: true,
			Status:      FixPending,
		})
	}

	return opts
}

// analyzeConfig checks health results for configuration issues.
func analyzeConfig(d *Deps) []Optimization {
	var opts []Optimization
	for tool, hr := range d.HealthResults {
		ts, ok := d.ToolStatuses[tool]
		if !ok || !ts.Installed {
			continue
		}
		switch hr.Status {
		case toolhealth.StatusRed:
			opts = append(opts, Optimization{
				ID:          "fix-config-" + tool,
				Category:    "config",
				Priority:    1,
				Title:       "optimizer.fixConfig.title",
				Description: "optimizer.fixConfig.desc",
				Action:      "fix-config",
				Target:      tool,
				AutoFixable: true,
				Status:      FixPending,
			})
		case toolhealth.StatusYellow:
			opts = append(opts, Optimization{
				ID:          "fix-config-" + tool,
				Category:    "config",
				Priority:    2,
				Title:       "optimizer.fixConfig.title",
				Description: "optimizer.fixConfig.desc",
				Action:      "fix-config",
				Target:      tool,
				AutoFixable: true,
				Status:      FixPending,
			})
		}
	}
	return opts
}

// analyzeSystem checks for system-level prerequisites like git.
func analyzeSystem() []Optimization {
	var opts []Optimization
	if _, err := exec.LookPath("git"); err != nil {
		opts = append(opts, Optimization{
			ID:          "install-git",
			Category:    "system",
			Priority:    2,
			Title:       "optimizer.installGit.title",
			Description: "optimizer.installGit.desc",
			Action:      "install-git",
			Target:      "git",
			AutoFixable: false,
			Status:      FixPending,
		})
	}
	return opts
}
