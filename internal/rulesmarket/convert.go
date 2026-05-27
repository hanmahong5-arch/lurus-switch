package rulesmarket

import (
	"fmt"
	"strings"
)

// Convert transforms rule content from one format to another.
// The windsurf format is recognised as an input but any output to windsurf
// is treated the same as cursorrules (plain markdown, no frontmatter).
//
// All three supported formats store rules as Markdown.  The differences are:
//   - agents_md  — opens with a `# Project Rules` H1 heading (AAIF convention)
//   - claude_md  — opens with a `# CLAUDE.md` H1 heading
//   - cursorrules — plain markdown, no mandatory heading; a leading H1 is allowed
//
// Convert rewrites only the leading H1 (if present) and the first section
// separator.  Body content is preserved verbatim.
func Convert(content string, from, to Format) (string, error) {
	if from == to {
		return content, nil
	}
	if content == "" {
		return "", fmt.Errorf("rulesmarket.Convert: content must not be empty")
	}

	// Normalise line endings
	body := strings.ReplaceAll(content, "\r\n", "\n")

	// Strip a known heading so we can re-add the target one
	body = stripLeadingHeading(body, from)

	switch to {
	case FormatAgentsMD:
		return "# Project Rules\n\n" + strings.TrimLeft(body, "\n"), nil
	case FormatClaudeMD:
		return "# CLAUDE.md\n\n" + strings.TrimLeft(body, "\n"), nil
	case FormatCursorRules, FormatWindsurf:
		// Plain markdown — preserve whatever heading the body still carries
		return strings.TrimLeft(body, "\n"), nil
	default:
		return "", fmt.Errorf("rulesmarket.Convert: unknown target format %q", to)
	}
}

// stripLeadingHeading removes the canonical H1 for a given format, leaving the
// rest of the document intact.
func stripLeadingHeading(body string, from Format) string {
	var prefix string
	switch from {
	case FormatAgentsMD:
		prefix = "# Project Rules"
	case FormatClaudeMD:
		prefix = "# CLAUDE.md"
	default:
		// cursorrules / windsurf have no mandatory heading
		return body
	}

	if strings.HasPrefix(body, prefix) {
		body = strings.TrimPrefix(body, prefix)
		// Eat the newline(s) immediately after the heading
		body = strings.TrimLeft(body, "\n")
	}
	return body
}

// TargetFileName returns the conventional file name for a given format.
func TargetFileName(format Format) string {
	switch format {
	case FormatAgentsMD:
		return "AGENTS.md"
	case FormatClaudeMD:
		return "CLAUDE.md"
	case FormatCursorRules:
		return ".cursorrules"
	case FormatWindsurf:
		return ".windsurfrules"
	default:
		return ".cursorrules"
	}
}
