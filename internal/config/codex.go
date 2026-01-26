package config

// CodexConfig represents OpenAI Codex CLI configuration (config.toml)
// Reference: https://github.com/openai/codex
type CodexConfig struct {
	// Core settings
	Model         string `toml:"model" json:"model"`                    // Default model (e.g., "o4-mini")
	APIKey        string `toml:"api_key" json:"apiKey"`                 // OpenAI API key
	ApprovalMode  string `toml:"approval_mode" json:"approvalMode"`     // "suggest", "auto-edit", "full-auto"

	// Provider settings
	Provider CodexProvider `toml:"provider" json:"provider"`

	// Security settings
	Security CodexSecurity `toml:"security" json:"security"`

	// MCP settings
	MCP CodexMCP `toml:"mcp" json:"mcp"`

	// Sandbox settings
	Sandbox CodexSandbox `toml:"sandbox" json:"sandbox"`

	// History settings
	History CodexHistory `toml:"history" json:"history"`
}

// CodexProvider defines API provider settings
type CodexProvider struct {
	// Provider type: "openai", "azure", "openrouter", etc.
	Type string `toml:"type" json:"type"`
	// Base URL for API
	BaseURL string `toml:"base_url" json:"baseUrl,omitempty"`
	// Azure deployment name
	AzureDeployment string `toml:"azure_deployment" json:"azureDeployment,omitempty"`
	// Azure API version
	AzureAPIVersion string `toml:"azure_api_version" json:"azureApiVersion,omitempty"`
}

// CodexSecurity defines security and permission settings
type CodexSecurity struct {
	// Network access: "off", "local", "full"
	NetworkAccess string `toml:"network_access" json:"networkAccess"`
	// File access scope
	FileAccess CodexFileAccess `toml:"file_access" json:"fileAccess"`
	// Command execution settings
	CommandExecution CodexCommandExecution `toml:"command_execution" json:"commandExecution"`
}

// CodexFileAccess defines file access permissions
type CodexFileAccess struct {
	// Allowed directories
	AllowedDirs []string `toml:"allowed_dirs" json:"allowedDirs,omitempty"`
	// Denied patterns
	DeniedPatterns []string `toml:"denied_patterns" json:"deniedPatterns,omitempty"`
	// Read-only directories
	ReadOnlyDirs []string `toml:"read_only_dirs" json:"readOnlyDirs,omitempty"`
}

// CodexCommandExecution defines command execution permissions
type CodexCommandExecution struct {
	// Enable command execution
	Enabled bool `toml:"enabled" json:"enabled"`
	// Allowed commands (patterns)
	AllowedCommands []string `toml:"allowed_commands" json:"allowedCommands,omitempty"`
	// Denied commands (patterns)
	DeniedCommands []string `toml:"denied_commands" json:"deniedCommands,omitempty"`
}

// CodexMCP defines Model Context Protocol settings
type CodexMCP struct {
	// Enable MCP
	Enabled bool `toml:"enabled" json:"enabled"`
	// MCP servers
	Servers []CodexMCPServer `toml:"servers" json:"servers,omitempty"`
}

// CodexMCPServer defines a single MCP server
type CodexMCPServer struct {
	Name    string            `toml:"name" json:"name"`
	Command string            `toml:"command" json:"command"`
	Args    []string          `toml:"args" json:"args,omitempty"`
	Env     map[string]string `toml:"env" json:"env,omitempty"`
}

// CodexSandbox defines sandbox settings
type CodexSandbox struct {
	// Enable sandbox mode
	Enabled bool `toml:"enabled" json:"enabled"`
	// Sandbox type: "seatbelt" (macOS), "landlock" (Linux), "none"
	Type string `toml:"type" json:"type"`
}

// CodexHistory defines history and logging settings
type CodexHistory struct {
	// Enable history
	Enabled bool `toml:"enabled" json:"enabled"`
	// History file path
	FilePath string `toml:"file_path" json:"filePath,omitempty"`
	// Max history entries
	MaxEntries int `toml:"max_entries" json:"maxEntries,omitempty"`
}

// NewCodexConfig creates a new Codex configuration with sensible defaults
func NewCodexConfig() *CodexConfig {
	return &CodexConfig{
		Model:        "o4-mini",
		ApprovalMode: "suggest",
		Provider: CodexProvider{
			Type: "openai",
		},
		Security: CodexSecurity{
			NetworkAccess: "local",
			FileAccess: CodexFileAccess{
				AllowedDirs:    []string{"."},
				DeniedPatterns: []string{"**/.env", "**/*.key", "**/secrets/**"},
			},
			CommandExecution: CodexCommandExecution{
				Enabled: true,
			},
		},
		MCP: CodexMCP{
			Enabled: false,
		},
		Sandbox: CodexSandbox{
			Enabled: true,
			Type:    "none",
		},
		History: CodexHistory{
			Enabled:    true,
			MaxEntries: 1000,
		},
	}
}
