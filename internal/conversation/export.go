package conversation

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// ExportFormat is the user-selectable export shape. Markdown is intended
// for human consumption (paste into a report / share with a teammate);
// JSON is the round-trippable form for tooling.
type ExportFormat string

const (
	ExportMarkdown ExportFormat = "markdown"
	ExportJSON     ExportFormat = "json"
)

// ExportOptions captures the toggles available to the caller.
type ExportOptions struct {
	Format            ExportFormat
	OutputDir         string // directory to write into; required
	RedactToolResults bool   // strip tool_result bodies (default true)
}

// Export renders a session to the chosen format and writes it under
// OutputDir. Returns the absolute path of the file produced.
func Export(meta ConversationMeta, events []Event, opts ExportOptions) (string, error) {
	if opts.OutputDir == "" {
		return "", fmt.Errorf("export: OutputDir is required")
	}
	if err := os.MkdirAll(opts.OutputDir, 0o755); err != nil {
		return "", fmt.Errorf("export: mkdir output dir: %w", err)
	}
	stamp := time.Now().Format("20060102-150405")
	stem := fmt.Sprintf("%s-%s-%s", meta.Tool, sanitize(meta.SessionID), stamp)

	switch opts.Format {
	case ExportMarkdown, "":
		path := filepath.Join(opts.OutputDir, stem+".md")
		body := renderMarkdown(meta, events, opts.RedactToolResults)
		if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
			return "", err
		}
		return path, nil
	case ExportJSON:
		path := filepath.Join(opts.OutputDir, stem+".json")
		payload := map[string]any{
			"meta":   meta,
			"events": maybeRedact(events, opts.RedactToolResults),
		}
		data, err := json.MarshalIndent(payload, "", "  ")
		if err != nil {
			return "", err
		}
		if err := os.WriteFile(path, data, 0o600); err != nil {
			return "", err
		}
		return path, nil
	default:
		return "", fmt.Errorf("export: unknown format %q", opts.Format)
	}
}

func renderMarkdown(meta ConversationMeta, events []Event, redact bool) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "# %s session %s\n\n", meta.Tool, meta.SessionID)
	if meta.Cwd != "" {
		fmt.Fprintf(&sb, "- Project: `%s`\n", meta.Cwd)
	}
	if meta.Model != "" {
		fmt.Fprintf(&sb, "- Model: `%s`\n", meta.Model)
	}
	if !meta.StartedAt.IsZero() {
		fmt.Fprintf(&sb, "- Started: %s\n", meta.StartedAt.Format(time.RFC3339))
	}
	if !meta.EndedAt.IsZero() {
		fmt.Fprintf(&sb, "- Ended:   %s\n", meta.EndedAt.Format(time.RFC3339))
	}
	fmt.Fprintf(&sb, "- Messages: %d  ·  Tokens: %d\n\n", meta.MessageCount, meta.TotalTokens)

	for _, e := range events {
		switch e.Type {
		case EventUser:
			fmt.Fprintf(&sb, "## 👤 User · %s\n\n%s\n\n", fmtTime(e.Timestamp), e.Content)
		case EventAssistant:
			fmt.Fprintf(&sb, "## 🤖 Assistant · %s\n\n%s\n\n", fmtTime(e.Timestamp), e.Content)
		case EventToolUse:
			fmt.Fprintf(&sb, "### 🛠 tool_use → `%s` · %s\n\n", e.ToolName, fmtTime(e.Timestamp))
			if len(e.ToolArgs) > 0 {
				sb.WriteString("```json\n")
				sb.Write(e.ToolArgs)
				sb.WriteString("\n```\n\n")
			}
		case EventToolResult:
			fmt.Fprintf(&sb, "### 📤 tool_result · %s\n\n", fmtTime(e.Timestamp))
			if redact {
				sb.WriteString("_(content redacted)_\n\n")
			} else {
				fmt.Fprintf(&sb, "%s\n\n", e.Content)
			}
		case EventSystem:
			fmt.Fprintf(&sb, "> _system · %s_\n>\n> %s\n\n", fmtTime(e.Timestamp), e.Content)
		}
	}
	return sb.String()
}

func maybeRedact(events []Event, redact bool) []Event {
	if !redact {
		return events
	}
	out := make([]Event, len(events))
	copy(out, events)
	for i := range out {
		if out[i].Type == EventToolResult {
			out[i].Content = "(redacted)"
			out[i].Raw = nil
		}
	}
	return out
}

func fmtTime(t time.Time) string {
	if t.IsZero() {
		return "—"
	}
	return t.Format("2006-01-02 15:04:05")
}

// sanitize replaces filesystem-hostile characters in a session ID so
// the export filename is safe on Windows. We don't try to be clever —
// any non-alphanumeric/hyphen char becomes underscore.
func sanitize(s string) string {
	out := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z', c >= '0' && c <= '9', c == '-':
			out[i] = c
		default:
			out[i] = '_'
		}
	}
	return string(out)
}
