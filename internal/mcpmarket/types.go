package mcpmarket

import "time"

// TargetTool identifies an AI coding tool whose MCP config can be written.
type TargetTool string

const (
	ToolClaudeCode TargetTool = "claude_code"
	ToolCursor     TargetTool = "cursor"
	ToolGemini     TargetTool = "gemini"
	ToolAntigravity TargetTool = "antigravity"
)

// AllTargetTools is the canonical ordered list of supported installation targets.
var AllTargetTools = []TargetTool{ToolClaudeCode, ToolCursor, ToolGemini, ToolAntigravity}

// MarketServer describes a single MCP server from the registry or builtin list.
type MarketServer struct {
	// ID is a stable opaque identifier. For registry entries this is the UUID
	// returned by the API; for builtins it is a stable slug like "github".
	ID string `json:"id"`

	// QualifiedName is the registry slug (e.g. "@modelcontextprotocol/server-github").
	// Empty for builtin-only entries.
	QualifiedName string `json:"qualifiedName,omitempty"`

	// Name is the human-readable display name.
	Name string `json:"name"`

	// Description is a short plaintext description.
	Description string `json:"description"`

	// Category is an informal grouping tag (e.g. "database", "files", "web").
	Category string `json:"category"`

	// Author identifies the publisher; may be empty for builtin entries.
	Author string `json:"author,omitempty"`

	// InstallCommand is the Smithery CLI install hint, e.g.
	// "npx -y @smithery/cli@latest install @modelcontextprotocol/server-github".
	// Populated by the client for registry entries; may be empty for builtins.
	InstallCommand string `json:"installCommand,omitempty"`

	// ConfigSchema is a JSON Schema object describing user-configurable parameters
	// (e.g. API keys, connection strings).  Empty when no configuration is required.
	ConfigSchema map[string]any `json:"configSchema,omitempty"`

	// Capabilities is a free-form list of capability keywords (e.g. "tools", "resources").
	Capabilities []string `json:"capabilities,omitempty"`

	// Stars approximates usage popularity.  Maps to useCount from the registry API.
	Stars int `json:"stars"`

	// Verified indicates the registry has vetted this server.
	Verified bool `json:"verified"`

	// Homepage is the canonical URL for documentation / source.
	Homepage string `json:"homepage,omitempty"`

	// Builtin is true when this entry comes from the embedded manifest rather
	// than the live registry.
	Builtin bool `json:"builtin"`

	// FetchedAt records when this entry was pulled from the registry.
	FetchedAt time.Time `json:"fetchedAt,omitempty"`
}

// ToolInstallStatus records the outcome of installing an MCP server into one tool.
type ToolInstallStatus struct {
	// Tool is the target tool identifier.
	Tool TargetTool `json:"tool"`
	// OK is true when the write succeeded.
	OK bool `json:"ok"`
	// Path is the config file that was written.
	Path string `json:"path,omitempty"`
	// Error contains the error message when OK is false.
	Error string `json:"error,omitempty"`
}

// InstallReport is the aggregate result of InstallToTools.
type InstallReport struct {
	// Statuses contains one entry per requested target tool, in request order.
	Statuses []ToolInstallStatus `json:"statuses"`
}

// registryListResponse mirrors the top-level shape returned by
// GET /servers on the public registry API.
type registryListResponse struct {
	Servers    []registryServer    `json:"servers"`
	Pagination registryPagination  `json:"pagination"`
}

type registryPagination struct {
	CurrentPage int `json:"currentPage"`
	PageSize    int `json:"pageSize"`
	TotalPages  int `json:"totalPages"`
	TotalCount  int `json:"totalCount"`
}

// registryServer mirrors one element of the servers array from the registry API.
// Only the fields used by Switch are present; unknown fields are silently dropped.
type registryServer struct {
	ID            string    `json:"id"`
	QualifiedName string    `json:"qualifiedName"`
	DisplayName   string    `json:"displayName"`
	Description   string    `json:"description"`
	IconURL       string    `json:"iconUrl,omitempty"`
	Verified      bool      `json:"verified"`
	UseCount      int       `json:"useCount"`
	Remote        bool      `json:"remote"`
	IsDeployed    bool      `json:"isDeployed"`
	Homepage      string    `json:"homepage,omitempty"`
	BySmithery    bool      `json:"bySmithery"`
	Owner         string    `json:"owner,omitempty"`
	Namespace     string    `json:"namespace,omitempty"`
	CreatedAt     time.Time `json:"createdAt"`
}
