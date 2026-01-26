package config

// ClaudeConfig represents Claude Code CLI configuration (settings.json)
// Reference: https://github.com/anthropics/claude-code
type ClaudeConfig struct {
	// Core settings
	Model              string   `json:"model,omitempty"`               // Default model (e.g., "claude-sonnet-4-20250514")
	CustomInstructions string   `json:"customInstructions,omitempty"`  // Custom instructions for all conversations
	APIKey             string   `json:"apiKey,omitempty"`              // Anthropic API key
	MaxTokens          int      `json:"maxTokens,omitempty"`           // Max tokens per response

	// Permissions
	Permissions ClaudePermissions `json:"permissions,omitempty"`

	// MCP (Model Context Protocol) servers
	MCPServers map[string]MCPServer `json:"mcpServers,omitempty"`

	// Sandbox settings
	Sandbox ClaudeSandbox `json:"sandbox,omitempty"`

	// Advanced settings
	Advanced ClaudeAdvanced `json:"advanced,omitempty"`
}

// ClaudePermissions defines what the CLI is allowed to do
type ClaudePermissions struct {
	// Allow execution of bash commands
	AllowBash bool `json:"allowBash,omitempty"`
	// Allow file read operations
	AllowRead bool `json:"allowRead,omitempty"`
	// Allow file write operations
	AllowWrite bool `json:"allowWrite,omitempty"`
	// Allow web fetch operations
	AllowWebFetch bool `json:"allowWebFetch,omitempty"`
	// Trusted directories for file operations
	TrustedDirectories []string `json:"trustedDirectories,omitempty"`
	// Allowed bash commands (glob patterns)
	AllowedBashCommands []string `json:"allowedBashCommands,omitempty"`
	// Denied bash commands (glob patterns)
	DeniedBashCommands []string `json:"deniedBashCommands,omitempty"`
}

// MCPServer represents a Model Context Protocol server configuration
type MCPServer struct {
	Command string            `json:"command"`
	Args    []string          `json:"args,omitempty"`
	Env     map[string]string `json:"env,omitempty"`
}

// ClaudeSandbox defines sandbox settings for command execution
type ClaudeSandbox struct {
	// Enable sandbox mode
	Enabled bool `json:"enabled,omitempty"`
	// Sandbox type: "docker", "wsl", "none"
	Type string `json:"type,omitempty"`
	// Docker image to use for sandbox
	DockerImage string `json:"dockerImage,omitempty"`
	// Mount points for sandbox
	Mounts []SandboxMount `json:"mounts,omitempty"`
}

// SandboxMount defines a mount point in sandbox mode
type SandboxMount struct {
	Source      string `json:"source"`
	Destination string `json:"destination"`
	ReadOnly    bool   `json:"readOnly,omitempty"`
}

// ClaudeAdvanced defines advanced settings
type ClaudeAdvanced struct {
	// Enable verbose logging
	Verbose bool `json:"verbose,omitempty"`
	// Disable telemetry
	DisableTelemetry bool `json:"disableTelemetry,omitempty"`
	// Custom API endpoint
	APIEndpoint string `json:"apiEndpoint,omitempty"`
	// Timeout for operations (seconds)
	Timeout int `json:"timeout,omitempty"`
	// Enable experimental features
	ExperimentalFeatures bool `json:"experimentalFeatures,omitempty"`
}

// NewClaudeConfig creates a new Claude configuration with sensible defaults
func NewClaudeConfig() *ClaudeConfig {
	return &ClaudeConfig{
		Model:     "claude-sonnet-4-20250514",
		MaxTokens: 8192,
		Permissions: ClaudePermissions{
			AllowBash:     true,
			AllowRead:     true,
			AllowWrite:    true,
			AllowWebFetch: false,
		},
		Sandbox: ClaudeSandbox{
			Enabled: false,
			Type:    "none",
		},
		Advanced: ClaudeAdvanced{
			Verbose:          false,
			DisableTelemetry: false,
			Timeout:          300,
		},
	}
}
