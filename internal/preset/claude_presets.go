package preset

import "lurus-switch/internal/config"

// ClaudePresets returns the list of available Claude preset descriptors.
func ClaudePresets() []Preset {
	return []Preset{
		{
			ID:          "quick-start",
			Name:        "Quick Start",
			Description: "Out-of-the-box defaults — all permissions on, no sandbox",
		},
		{
			ID:          "security",
			Name:        "Security",
			Description: "Locked-down mode — bash and web fetch disabled, sandbox on",
		},
		{
			ID:          "performance",
			Name:        "Performance",
			Description: "Speed-optimized — Haiku model, capped tokens, telemetry off",
		},
		{
			ID:          "budget",
			Name:        "Budget",
			Description: "Cost-efficient — Haiku model, minimal tokens, web fetch off",
		},
	}
}

// ApplyClaudePreset fills and returns a ClaudeConfig for the requested preset ID.
// Returns (nil, error) for unknown IDs.
func ApplyClaudePreset(id string) (*config.ClaudeConfig, error) {
	switch id {
	case "quick-start":
		return claudeQuickStart(), nil
	case "security":
		return claudeSecurity(), nil
	case "performance":
		return claudePerformance(), nil
	case "budget":
		return claudeBudget(), nil
	default:
		return nil, unknownPreset("claude", id)
	}
}

func claudeQuickStart() *config.ClaudeConfig {
	c := config.NewClaudeConfig()
	c.Permissions.AllowBash = true
	c.Permissions.AllowRead = true
	c.Permissions.AllowWrite = true
	c.Permissions.AllowWebFetch = true
	c.Sandbox.Enabled = false
	c.Sandbox.Type = "none"
	return c
}

func claudeSecurity() *config.ClaudeConfig {
	c := config.NewClaudeConfig()
	c.Permissions.AllowBash = false
	c.Permissions.AllowRead = true
	c.Permissions.AllowWrite = false
	c.Permissions.AllowWebFetch = false
	c.Sandbox.Enabled = true
	c.Sandbox.Type = "docker"
	c.Advanced.DisableTelemetry = true
	return c
}

func claudePerformance() *config.ClaudeConfig {
	c := config.NewClaudeConfig()
	c.Model = "claude-haiku-4-20250514"
	c.MaxTokens = 4096
	c.Advanced.DisableTelemetry = true
	c.Permissions.AllowWebFetch = false
	return c
}

func claudeBudget() *config.ClaudeConfig {
	c := config.NewClaudeConfig()
	c.Model = "claude-haiku-4-20250514"
	c.MaxTokens = 2048
	c.Permissions.AllowWebFetch = false
	c.Advanced.DisableTelemetry = true
	return c
}
