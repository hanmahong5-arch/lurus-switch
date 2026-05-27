package rulesmarket

import "time"

// Format identifies which AI coding tool a rule file targets.
// The values are intentionally lowercase to match common file-name conventions.
type Format string

const (
	FormatAgentsMD    Format = "agents_md"    // AGENTS.md
	FormatClaudeMD    Format = "claude_md"    // CLAUDE.md
	FormatCursorRules Format = "cursorrules"  // .cursorrules
	FormatWindsurf    Format = "windsurf_rules" // .windsurfrules (recognized but not converted)
)

// RuleTemplate is a single entry from the market manifest or the builtin list.
type RuleTemplate struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Category    string    `json:"category"`   // e.g. "framework" | "language" | "custom"
	Framework   string    `json:"framework"`  // e.g. "Next.js" | "Go" | "Rust"
	Description string    `json:"description"`
	Format      Format    `json:"format"`     // original/canonical format of the Content
	SourceURL   string    `json:"source_url"` // empty for builtin templates
	Content     string    `json:"content"`
	ETag        string    `json:"etag,omitempty"`
	FetchedAt   time.Time `json:"fetched_at,omitempty"`
}

// WriteResult describes the outcome of a WriteRuleToProject call.
type WriteResult struct {
	// Path is the absolute path that was written.
	Path string `json:"path"`
	// Appended is true when the content was appended to an existing file.
	Appended bool `json:"appended"`
	// Skipped is true when the file already existed and the caller requested no overwrite.
	Skipped bool `json:"skipped"`
}
