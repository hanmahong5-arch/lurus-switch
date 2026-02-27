package installer

const (
	// npm package names for CLI tools
	ClaudeNpmPackage = "@anthropic-ai/claude-code"
	CodexNpmPackage  = "@openai/codex"
	GeminiNpmPackage = "@google/gemini-cli"

	// pip package name for PicoClaw
	PicoClawPipPackage = "picoclaw"

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

	// DefaultPicoClawModel is the default model used for PicoClaw proxy configuration
	DefaultPicoClawModel = "claude-sonnet-4-20250514"
)
