package installer

const (
	// npm package names for CLI tools
	ClaudeNpmPackage = "@anthropic-ai/claude-code"
	CodexNpmPackage  = "@openai/codex"
	GeminiNpmPackage = "@google/gemini-cli"

	// npm registry base URL for version checking
	NpmRegistryURL = "https://registry.npmjs.org"

	// Bun installation commands per platform
	BunInstallWindows = "irm bun.sh/install.ps1 | iex"
	BunInstallUnix    = "curl -fsSL https://bun.sh/install | bash"

	// Default timeout for install operations (seconds)
	DefaultInstallTimeout = 300

	// Version extraction regex pattern
	VersionPattern = `(\d+\.\d+\.\d+)`

	// Tool names as constants
	ToolClaude   = "claude"
	ToolCodex    = "codex"
	ToolGemini   = "gemini"
	ToolPicoClaw = "picoclaw"
	ToolNullClaw = "nullclaw"
	ToolZeroClaw = "zeroclaw"
	ToolOpenClaw = "openclaw"

	// PicoClaw GitHub release source
	PicoClawGitHubOwner = "picoclaw-labs"
	PicoClawGitHubRepo  = "picoclaw"
	PicoClawBinaryName  = "pclaw"

	// NullClaw GitHub release source
	NullClawGitHubOwner = "nullclaw-labs"
	NullClawGitHubRepo  = "nullclaw"
	NullClawBinaryName  = "nclaw"

	// ZeroClaw GitHub release source
	ZeroClawGitHubOwner = "zeroclaw-labs"
	ZeroClawGitHubRepo  = "zeroclaw"
	ZeroClawBinaryName  = "zeroclaw"

	// OpenClaw npm package
	OpenClawNpmPackage = "openclaw"
	OpenClawBinaryName = "openclaw"

	// Default models for proxy configuration
	DefaultPicoClawModel = "claude-sonnet-4-20250514"
	DefaultNullClawModel = "claude-sonnet-4-20250514"
	DefaultZeroClawModel = "claude-sonnet-4-20250514"
	DefaultOpenClawModel = "claude-sonnet-4-20250514"

	// Node.js minimum required major version
	NodeMinVersion = 22
)
