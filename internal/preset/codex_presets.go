package preset

import "lurus-switch/internal/config"

// CodexPresets returns the list of available Codex preset descriptors.
func CodexPresets() []Preset {
	return []Preset{
		{
			ID:          "quick-start",
			Name:        "Quick Start",
			Description: "Suggest mode, local network, command execution on",
		},
		{
			ID:          "security",
			Name:        "Security",
			Description: "Full-auto off, network disabled, sandbox on",
		},
		{
			ID:          "performance",
			Name:        "Performance",
			Description: "Full-auto mode, full network, MCP on",
		},
		{
			ID:          "budget",
			Name:        "Budget",
			Description: "Suggest mode, network off, minimal history",
		},
	}
}

// ApplyCodexPreset fills and returns a CodexConfig for the requested preset ID.
func ApplyCodexPreset(id string) (*config.CodexConfig, error) {
	switch id {
	case "quick-start":
		return codexQuickStart(), nil
	case "security":
		return codexSecurity(), nil
	case "performance":
		return codexPerformance(), nil
	case "budget":
		return codexBudget(), nil
	default:
		return nil, unknownPreset("codex", id)
	}
}

func codexQuickStart() *config.CodexConfig {
	c := config.NewCodexConfig()
	c.ApprovalMode = "suggest"
	c.Security.NetworkAccess = "local"
	c.Security.CommandExecution.Enabled = true
	return c
}

func codexSecurity() *config.CodexConfig {
	c := config.NewCodexConfig()
	c.ApprovalMode = "suggest"
	c.Security.NetworkAccess = "off"
	c.Security.CommandExecution.Enabled = false
	c.Sandbox.Enabled = true
	c.Sandbox.Type = "seatbelt"
	return c
}

func codexPerformance() *config.CodexConfig {
	c := config.NewCodexConfig()
	c.ApprovalMode = "full-auto"
	c.Security.NetworkAccess = "full"
	c.MCP.Enabled = true
	c.History.Enabled = true
	c.History.MaxEntries = 5000
	return c
}

func codexBudget() *config.CodexConfig {
	c := config.NewCodexConfig()
	c.Model = "o4-mini"
	c.ApprovalMode = "suggest"
	c.Security.NetworkAccess = "off"
	c.History.Enabled = true
	c.History.MaxEntries = 100
	return c
}
