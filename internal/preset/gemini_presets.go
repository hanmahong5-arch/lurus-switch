package preset

import "lurus-switch/internal/config"

// GeminiPresets returns the list of available Gemini preset descriptors.
func GeminiPresets() []Preset {
	return []Preset{
		{
			ID:          "quick-start",
			Name:        "Quick Start",
			Description: "Flash model, API key auth, markdown and syntax highlight on",
		},
		{
			ID:          "security",
			Name:        "Security",
			Description: "Sandbox on, yolo off, no auto-approve actions",
		},
		{
			ID:          "performance",
			Name:        "Performance",
			Description: "Flash model, yolo mode, large file support",
		},
		{
			ID:          "budget",
			Name:        "Budget",
			Description: "Flash model, 1 MB file cap, minimal extensions",
		},
	}
}

// ApplyGeminiPreset fills and returns a GeminiConfig for the requested preset ID.
func ApplyGeminiPreset(id string) (*config.GeminiConfig, error) {
	switch id {
	case "quick-start":
		return geminiQuickStart(), nil
	case "security":
		return geminiSecurity(), nil
	case "performance":
		return geminiPerformance(), nil
	case "budget":
		return geminiBudget(), nil
	default:
		return nil, unknownPreset("gemini", id)
	}
}

func geminiQuickStart() *config.GeminiConfig {
	c := config.NewGeminiConfig()
	c.Auth.Type = "api_key"
	c.Behavior.Sandbox = false
	c.Behavior.YoloMode = false
	c.Display.SyntaxHighlight = true
	c.Display.MarkdownRender = true
	return c
}

func geminiSecurity() *config.GeminiConfig {
	c := config.NewGeminiConfig()
	c.Behavior.Sandbox = true
	c.Behavior.YoloMode = false
	c.Behavior.AutoApprove = []string{}
	c.Behavior.AllowedExtensions = []string{".go", ".ts", ".tsx", ".js", ".py", ".md"}
	return c
}

func geminiPerformance() *config.GeminiConfig {
	c := config.NewGeminiConfig()
	c.Model = "gemini-2.0-flash"
	c.Behavior.YoloMode = true
	c.Behavior.MaxFileSize = 50 * 1024 * 1024 // 50 MB
	return c
}

func geminiBudget() *config.GeminiConfig {
	c := config.NewGeminiConfig()
	c.Model = "gemini-2.0-flash"
	c.Behavior.MaxFileSize = 1 * 1024 * 1024 // 1 MB
	c.Behavior.AllowedExtensions = []string{".go", ".ts", ".js", ".py"}
	return c
}
