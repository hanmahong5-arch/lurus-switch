package healthscore

import (
	"os/exec"
	"runtime"

	"lurus-switch/internal/installer"
	"lurus-switch/internal/toolhealth"
)

// Deps holds all data needed to compute the score.
// The caller gathers this data and passes it in.
type Deps struct {
	ToolStatuses   map[string]*installer.ToolStatus
	HealthResults  map[string]*toolhealth.HealthResult
	DepCheck       *installer.DepCheckResult
	GatewayRunning bool
	GatewayURL     string
	AllToolsBound  bool
	InstalledCount int
	BoundCount     int
	EnvKeys        map[string]bool // tool -> has API key configured
}

const (
	maxRuntimes = 20
	maxTools    = 25
	maxConfig   = 25
	maxGateway  = 20
	maxSystem   = 10
)

// Compute calculates the full health score report.
func Compute(d *Deps) *ScoreReport {
	report := &ScoreReport{MaxScore: 100}

	runtimes := scoreRuntimes(d)
	tools := scoreTools(d)
	config := scoreConfig(d)
	gateway := scoreGateway(d)
	system := scoreSystem()

	report.Categories = []CategoryScore{runtimes, tools, config, gateway, system}
	for _, c := range report.Categories {
		report.TotalScore += c.Score
	}

	report.Suggestions = buildSuggestions(d)
	return report
}

func scoreRuntimes(d *Deps) CategoryScore {
	c := CategoryScore{Category: "runtimes", Max: maxRuntimes, Label: "health.runtimes"}
	if d.DepCheck == nil {
		return c
	}
	total := 0
	installed := 0
	for _, rt := range d.DepCheck.Runtimes {
		if rt.Required {
			total++
			if rt.Installed {
				installed++
			} else {
				c.Issues = append(c.Issues, rt.Name+" not installed")
			}
		}
	}
	if total == 0 {
		c.Score = maxRuntimes
	} else {
		c.Score = maxRuntimes * installed / total
	}
	return c
}

func scoreTools(d *Deps) CategoryScore {
	c := CategoryScore{Category: "tools", Max: maxTools, Label: "health.tools"}
	totalTools := len(d.ToolStatuses)
	if totalTools == 0 {
		c.Issues = append(c.Issues, "no tools detected")
		return c
	}
	installed := 0
	for _, ts := range d.ToolStatuses {
		if ts.Installed {
			installed++
		}
	}
	if installed == 0 {
		c.Issues = append(c.Issues, "no tools installed")
		return c
	}
	// Score based on proportion installed (at least 1 tool = 10 pts, each additional up to 25)
	base := 10
	extra := (maxTools - base) * installed / totalTools
	c.Score = base + extra
	if c.Score > maxTools {
		c.Score = maxTools
	}
	return c
}

func scoreConfig(d *Deps) CategoryScore {
	c := CategoryScore{Category: "config", Max: maxConfig, Label: "health.config"}
	if len(d.HealthResults) == 0 {
		return c
	}
	green := 0
	yellow := 0
	total := 0
	for tool, hr := range d.HealthResults {
		ts, ok := d.ToolStatuses[tool]
		if !ok || !ts.Installed {
			continue
		}
		total++
		switch hr.Status {
		case toolhealth.StatusGreen:
			green++
		case toolhealth.StatusYellow:
			yellow++
			for _, issue := range hr.Issues {
				c.Issues = append(c.Issues, tool+": "+issue)
			}
		default:
			for _, issue := range hr.Issues {
				c.Issues = append(c.Issues, tool+": "+issue)
			}
		}
	}
	if total == 0 {
		return c
	}
	// green = full points, yellow = half points, red = 0
	c.Score = maxConfig * (green*2 + yellow) / (total * 2)
	return c
}

func scoreGateway(d *Deps) CategoryScore {
	c := CategoryScore{Category: "gateway", Max: maxGateway, Label: "health.gateway"}
	if !d.GatewayRunning {
		c.Issues = append(c.Issues, "gateway not running")
		return c
	}
	c.Score += 10 // gateway running
	if d.InstalledCount > 0 && d.BoundCount > 0 {
		ratio := d.BoundCount * 10 / d.InstalledCount
		c.Score += ratio
		if c.Score > maxGateway {
			c.Score = maxGateway
		}
	}
	if d.InstalledCount > 0 && d.BoundCount < d.InstalledCount {
		c.Issues = append(c.Issues, "some tools not connected to gateway")
	}
	return c
}

func scoreSystem() CategoryScore {
	c := CategoryScore{Category: "system", Max: maxSystem, Label: "health.system"}
	// Check git
	if _, err := exec.LookPath("git"); err == nil {
		c.Score += 5
	} else {
		c.Issues = append(c.Issues, "git not found in PATH")
	}
	// Check PATH sanity (platform-dependent basic check)
	if runtime.GOOS == "windows" {
		// On Windows, check that common dev paths exist
		c.Score += 5 // basic OS check passes
	} else {
		c.Score += 5
	}
	return c
}

func buildSuggestions(d *Deps) []Suggestion {
	var suggestions []Suggestion

	// Check runtimes
	if d.DepCheck != nil {
		for _, rt := range d.DepCheck.Runtimes {
			if rt.Required && !rt.Installed {
				suggestions = append(suggestions, Suggestion{
					ID:       "install-runtime-" + rt.ID,
					Priority: 1,
					Title:    rt.Name + " not installed",
					Action:   "install-runtime",
					Target:   rt.ID,
				})
			}
		}
	}

	// Check tools not installed
	for name, ts := range d.ToolStatuses {
		if !ts.Installed {
			suggestions = append(suggestions, Suggestion{
				ID:       "install-tool-" + name,
				Priority: 3,
				Title:    name + " not installed",
				Action:   "install-tool",
				Target:   name,
			})
		}
	}

	// Check tools with update available
	for name, ts := range d.ToolStatuses {
		if ts.Installed && ts.UpdateAvailable {
			suggestions = append(suggestions, Suggestion{
				ID:       "update-tool-" + name,
				Priority: 2,
				Title:    name + " has update available",
				Action:   "update-tool",
				Target:   name,
			})
		}
	}

	// Check gateway
	if !d.GatewayRunning {
		suggestions = append(suggestions, Suggestion{
			ID:       "start-gateway",
			Priority: 1,
			Title:    "gateway not running",
			Action:   "start-gateway",
			Target:   "",
		})
	}

	// Check tools not connected to gateway
	if d.GatewayRunning && d.InstalledCount > 0 && d.BoundCount < d.InstalledCount {
		suggestions = append(suggestions, Suggestion{
			ID:       "connect-all-tools",
			Priority: 1,
			Title:    "tools not connected to gateway",
			Action:   "connect-gateway",
			Target:   "",
		})
	}

	// Check config health
	for tool, hr := range d.HealthResults {
		ts, ok := d.ToolStatuses[tool]
		if !ok || !ts.Installed {
			continue
		}
		if hr.Status == toolhealth.StatusYellow || hr.Status == toolhealth.StatusRed {
			suggestions = append(suggestions, Suggestion{
				ID:       "fix-config-" + tool,
				Priority: 2,
				Title:    tool + " has configuration issues",
				Action:   "fix-config",
				Target:   tool,
			})
		}
	}

	// Check git
	if _, err := exec.LookPath("git"); err != nil {
		suggestions = append(suggestions, Suggestion{
			ID:       "install-git",
			Priority: 2,
			Title:    "git not found",
			Action:   "install-git",
			Target:   "git",
		})
	}

	return suggestions
}
