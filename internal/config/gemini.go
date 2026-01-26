package config

// GeminiConfig represents Gemini CLI configuration (GEMINI.md)
// Reference: https://github.com/google-gemini/gemini-cli
type GeminiConfig struct {
	// Core settings
	Model       string `json:"model"`              // Default model (e.g., "gemini-2.0-flash")
	APIKey      string `json:"apiKey"`             // Google AI API key
	ProjectID   string `json:"projectId,omitempty"` // Google Cloud project ID (optional)

	// Authentication
	Auth GeminiAuth `json:"auth"`

	// Behavior settings
	Behavior GeminiBehavior `json:"behavior"`

	// Custom instructions (written to GEMINI.md)
	Instructions GeminiInstructions `json:"instructions"`

	// Theme and display
	Display GeminiDisplay `json:"display"`
}

// GeminiAuth defines authentication settings
type GeminiAuth struct {
	// Auth type: "api_key", "oauth", "adc" (Application Default Credentials)
	Type string `json:"type"`
	// OAuth client ID (if using OAuth)
	OAuthClientID string `json:"oauthClientId,omitempty"`
	// Path to service account key (if using ADC)
	ServiceAccountPath string `json:"serviceAccountPath,omitempty"`
}

// GeminiBehavior defines CLI behavior settings
type GeminiBehavior struct {
	// Sandbox mode
	Sandbox bool `json:"sandbox"`
	// Auto-approve certain operations
	AutoApprove []string `json:"autoApprove,omitempty"`
	// Enable yolo mode (no confirmations)
	YoloMode bool `json:"yoloMode"`
	// Maximum file size to read (bytes)
	MaxFileSize int64 `json:"maxFileSize,omitempty"`
	// Allowed file extensions
	AllowedExtensions []string `json:"allowedExtensions,omitempty"`
}

// GeminiInstructions defines custom instructions for GEMINI.md
type GeminiInstructions struct {
	// Project description
	ProjectDescription string `json:"projectDescription,omitempty"`
	// Tech stack information
	TechStack string `json:"techStack,omitempty"`
	// Code style guidelines
	CodeStyle string `json:"codeStyle,omitempty"`
	// Custom rules and constraints
	CustomRules []string `json:"customRules,omitempty"`
	// File structure notes
	FileStructure string `json:"fileStructure,omitempty"`
	// Testing guidelines
	TestingGuidelines string `json:"testingGuidelines,omitempty"`
}

// GeminiDisplay defines display and theme settings
type GeminiDisplay struct {
	// Theme: "dark", "light", "auto"
	Theme string `json:"theme"`
	// Enable syntax highlighting
	SyntaxHighlight bool `json:"syntaxHighlight"`
	// Enable markdown rendering
	MarkdownRender bool `json:"markdownRender"`
}

// NewGeminiConfig creates a new Gemini configuration with sensible defaults
func NewGeminiConfig() *GeminiConfig {
	return &GeminiConfig{
		Model: "gemini-2.0-flash",
		Auth: GeminiAuth{
			Type: "api_key",
		},
		Behavior: GeminiBehavior{
			Sandbox:     false,
			YoloMode:    false,
			MaxFileSize: 10 * 1024 * 1024, // 10MB
		},
		Instructions: GeminiInstructions{
			CustomRules: []string{},
		},
		Display: GeminiDisplay{
			Theme:           "auto",
			SyntaxHighlight: true,
			MarkdownRender:  true,
		},
	}
}

// GenerateMarkdown generates the GEMINI.md content from the configuration
func (c *GeminiConfig) GenerateMarkdown() string {
	md := "# GEMINI.md\n\n"
	md += "This file provides guidance to Gemini CLI when working with this repository.\n\n"

	if c.Instructions.ProjectDescription != "" {
		md += "## Project Description\n\n"
		md += c.Instructions.ProjectDescription + "\n\n"
	}

	if c.Instructions.TechStack != "" {
		md += "## Tech Stack\n\n"
		md += c.Instructions.TechStack + "\n\n"
	}

	if c.Instructions.CodeStyle != "" {
		md += "## Code Style\n\n"
		md += c.Instructions.CodeStyle + "\n\n"
	}

	if len(c.Instructions.CustomRules) > 0 {
		md += "## Rules\n\n"
		for _, rule := range c.Instructions.CustomRules {
			md += "- " + rule + "\n"
		}
		md += "\n"
	}

	if c.Instructions.FileStructure != "" {
		md += "## File Structure\n\n"
		md += c.Instructions.FileStructure + "\n\n"
	}

	if c.Instructions.TestingGuidelines != "" {
		md += "## Testing\n\n"
		md += c.Instructions.TestingGuidelines + "\n\n"
	}

	return md
}
