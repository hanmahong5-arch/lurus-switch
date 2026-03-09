package mcp

// MCPServer describes a single MCP (Model Context Protocol) server definition
type MCPServer struct {
	Name    string            `json:"name"`
	Command string            `json:"command,omitempty"` // stdio transport: executable
	Args    []string          `json:"args,omitempty"`
	Env     map[string]string `json:"env,omitempty"`
	URL     string            `json:"url,omitempty"`  // SSE/HTTP transport
	Type    string            `json:"type"`            // "stdio" | "sse" | "http"
}

// MCPPreset is a named, shareable MCP server configuration
type MCPPreset struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Server      MCPServer `json:"server"`
	Tags        []string  `json:"tags"`
}

// BuiltinPresets returns the built-in MCP server presets bundled with the app
func BuiltinPresets() []MCPPreset {
	return []MCPPreset{
		{
			ID:          "builtin-filesystem",
			Name:        "Filesystem",
			Description: "Read and write files on the local filesystem",
			Server: MCPServer{
				Name:    "filesystem",
				Type:    "stdio",
				Command: "npx",
				Args:    []string{"-y", "@modelcontextprotocol/server-filesystem", "/tmp"},
			},
			Tags: []string{"files", "builtin"},
		},
		{
			ID:          "builtin-github",
			Name:        "GitHub",
			Description: "Access GitHub repositories, PRs, and issues",
			Server: MCPServer{
				Name:    "github",
				Type:    "stdio",
				Command: "npx",
				Args:    []string{"-y", "@modelcontextprotocol/server-github"},
				Env:     map[string]string{"GITHUB_PERSONAL_ACCESS_TOKEN": ""},
			},
			Tags: []string{"github", "vcs", "builtin"},
		},
		{
			ID:          "builtin-memory",
			Name:        "Memory",
			Description: "Persistent key-value memory store for the AI",
			Server: MCPServer{
				Name:    "memory",
				Type:    "stdio",
				Command: "npx",
				Args:    []string{"-y", "@modelcontextprotocol/server-memory"},
			},
			Tags: []string{"memory", "persistence", "builtin"},
		},
		{
			ID:          "builtin-sequential-thinking",
			Name:        "Sequential Thinking",
			Description: "Structured step-by-step reasoning support",
			Server: MCPServer{
				Name:    "sequential-thinking",
				Type:    "stdio",
				Command: "npx",
				Args:    []string{"-y", "@modelcontextprotocol/server-sequential-thinking"},
			},
			Tags: []string{"reasoning", "builtin"},
		},
		{
			ID:          "builtin-postgres",
			Name:        "PostgreSQL",
			Description: "Read-only access to a PostgreSQL database",
			Server: MCPServer{
				Name:    "postgres",
				Type:    "stdio",
				Command: "npx",
				Args:    []string{"-y", "@modelcontextprotocol/server-postgres", "postgresql://localhost/mydb"},
			},
			Tags: []string{"database", "sql", "builtin"},
		},
	}
}
