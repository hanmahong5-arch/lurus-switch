package appreg

import "time"

// Tier classifies how deeply Switch integrates with a tool.
type Tier int

const (
	// TierAuto: Switch auto-detects and injects config (Claude Code, Codex, etc.)
	TierAuto Tier = 1
	// TierGuided: user pastes URL/key following Switch instructions (Cursor, Windsurf, etc.)
	TierGuided Tier = 2
	// TierEnvVar: Switch sets system environment variables (Python openai SDK, LangChain, etc.)
	TierEnvVar Tier = 3
	// TierManual: user manually inputs localhost:PORT + token (any OpenAI-compatible app)
	TierManual Tier = 4
)

// AppKind distinguishes how the app was registered.
type AppKind string

const (
	KindBuiltin AppKind = "builtin" // pre-defined tool (Claude, Codex, etc.)
	KindUser    AppKind = "user"    // manually registered by user
)

// App represents a registered application that uses the Switch gateway.
type App struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Kind        AppKind   `json:"kind"`
	Tier        Tier      `json:"tier"`
	Token       string    `json:"token"`       // sk-switch-{random}, used for gateway auth
	Icon        string    `json:"icon"`        // optional icon identifier or emoji
	Description string    `json:"description"` // optional description
	CreatedAt   time.Time `json:"createdAt"`
	LastSeenAt  time.Time `json:"lastSeenAt,omitempty"` // last API call timestamp
	Connected   bool      `json:"connected"`            // whether currently configured to use Switch
}

// BuiltinTool defines a pre-known tool that Switch can auto-detect and configure.
type BuiltinTool struct {
	ID          string
	Name        string
	Tier        Tier
	Icon        string
	Description string
	// DetectFunc returns true if this tool is installed on the system.
	DetectFunc func() bool
	// ConfigPath returns the tool's config file path (empty if not file-based).
	ConfigPath func() string
}
