package conversation

import (
	"bufio"
	"encoding/json"
	"io"
	"os"
	"strings"
	"time"
)

// EventType is a coarse classification of an entry in the JSONL stream.
// CLI vendors disagree on every detail (key names, role taxonomy, tool
// envelope shape) so we collapse the noise to a stable enum the frontend
// can render against.
type EventType string

const (
	EventUser       EventType = "user"
	EventAssistant  EventType = "assistant"
	EventSystem     EventType = "system"
	EventToolUse    EventType = "tool_use"
	EventToolResult EventType = "tool_result"
	EventMeta       EventType = "meta" // session-meta, summary, anything else
)

// Event is one parsed JSONL line. Raw preserves the original record so
// the UI / export path can reach for fields we don't yet model.
type Event struct {
	Type        EventType       `json:"type"`
	MessageUUID string          `json:"messageUUID,omitempty"`
	ParentUUID  string          `json:"parentUUID,omitempty"`
	Timestamp   time.Time       `json:"timestamp"`
	Content     string          `json:"content,omitempty"`
	ToolName    string          `json:"toolName,omitempty"`
	ToolArgs    json.RawMessage `json:"toolArgs,omitempty"`
	Model       string          `json:"model,omitempty"`
	// InputTokens / OutputTokens are the *fresh* (uncached) input and the
	// output respectively, as Anthropic's API returns them. Don't fold
	// these together with the cache fields — they are billed at different
	// rates (see internal/livesession/pricing.go).
	InputTokens         int64           `json:"inputTokens,omitempty"`
	OutputTokens        int64           `json:"outputTokens,omitempty"`
	CacheCreationTokens int64           `json:"cacheCreationTokens,omitempty"`
	CacheReadTokens     int64           `json:"cacheReadTokens,omitempty"`
	Raw                 json.RawMessage `json:"raw,omitempty"`
}

// rawClaudeLine is the union of fields we've observed in Claude Code
// JSONLs. Anything unknown lives in Raw, which Parse always populates.
type rawClaudeLine struct {
	Type      string          `json:"type"`
	UUID      string          `json:"uuid"`
	ParentUUID string         `json:"parentUuid"`
	Timestamp string          `json:"timestamp"`
	Cwd       string          `json:"cwd"`
	SessionID string          `json:"sessionId"`
	Model     string          `json:"model"`
	Message   *struct {
		ID      string          `json:"id"`
		Role    string          `json:"role"`
		Model   string          `json:"model"`
		Content json.RawMessage `json:"content"`
		Usage   *struct {
			InputTokens              int64 `json:"input_tokens"`
			OutputTokens             int64 `json:"output_tokens"`
			CacheCreationInputTokens int64 `json:"cache_creation_input_tokens"`
			CacheReadInputTokens     int64 `json:"cache_read_input_tokens"`
		} `json:"usage"`
	} `json:"message,omitempty"`
	ToolUseID string          `json:"toolUseID,omitempty"`
	Content   json.RawMessage `json:"content,omitempty"`
}

// Parse streams a JSONL file and returns the parsed Event list. The
// reader is closed by the caller. Lines that fail to decode are skipped
// rather than aborting — the UI surfaces what could be parsed.
func Parse(r io.Reader) ([]Event, error) {
	scanner := bufio.NewScanner(r)
	// Some Claude sessions emit very long lines (tool args >1 MiB). Bump
	// the buffer well past the default 64 KiB so we don't lose them.
	buf := make([]byte, 0, 1024*1024)
	scanner.Buffer(buf, 16*1024*1024)

	var events []Event
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		ev, ok := parseLine(line)
		if !ok {
			continue
		}
		events = append(events, ev)
	}
	if err := scanner.Err(); err != nil {
		return events, err
	}
	return events, nil
}

// ParseFile opens `path` and streams it through Parse.
func ParseFile(path string) ([]Event, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return Parse(f)
}

func parseLine(line []byte) (Event, bool) {
	var raw rawClaudeLine
	if err := json.Unmarshal(line, &raw); err != nil {
		return Event{}, false
	}
	ev := Event{
		MessageUUID: raw.UUID,
		ParentUUID:  raw.ParentUUID,
		Raw:         append(json.RawMessage(nil), line...),
		Model:       raw.Model,
	}
	if t, err := time.Parse(time.RFC3339Nano, raw.Timestamp); err == nil {
		ev.Timestamp = t
	}
	// Type discovery. Claude uses "type":"user"/"assistant"/"summary",
	// Codex uses richer payloads under "type":"message"/"tool". When
	// nothing matches we tag it as Meta so the UI can still display it.
	switch strings.ToLower(raw.Type) {
	case "user":
		ev.Type = EventUser
		ev.Content = extractTextContent(raw)
	case "assistant":
		ev.Type = EventAssistant
		ev.Content = extractTextContent(raw)
		if raw.Message != nil {
			if raw.Message.Model != "" {
				ev.Model = raw.Message.Model
			}
			if raw.Message.Usage != nil {
				ev.InputTokens = raw.Message.Usage.InputTokens
				ev.OutputTokens = raw.Message.Usage.OutputTokens
				ev.CacheCreationTokens = raw.Message.Usage.CacheCreationInputTokens
				ev.CacheReadTokens = raw.Message.Usage.CacheReadInputTokens
			}
		}
		if name, args, ok := extractToolUse(raw); ok {
			ev.ToolName = name
			ev.ToolArgs = args
			ev.Type = EventToolUse
		}
	case "tool_use":
		ev.Type = EventToolUse
		if name, args, ok := extractToolUse(raw); ok {
			ev.ToolName = name
			ev.ToolArgs = args
		}
	case "tool_result", "tool-result":
		ev.Type = EventToolResult
		ev.Content = extractTextContent(raw)
	case "system":
		ev.Type = EventSystem
		ev.Content = extractTextContent(raw)
	default:
		ev.Type = EventMeta
	}
	return ev, true
}

// extractTextContent flattens whatever shape the JSONL line uses for
// "the message text" into a single string. Claude wraps content in an
// array of typed blocks; older Codex sessions just inline a string.
func extractTextContent(raw rawClaudeLine) string {
	if raw.Message != nil && len(raw.Message.Content) > 0 {
		return flattenContent(raw.Message.Content)
	}
	if len(raw.Content) > 0 {
		return flattenContent(raw.Content)
	}
	return ""
}

func flattenContent(b json.RawMessage) string {
	// String form: "content":"hello"
	var s string
	if json.Unmarshal(b, &s) == nil {
		return s
	}
	// Array of blocks form: [{"type":"text","text":"hi"},...]
	var blocks []struct {
		Type string          `json:"type"`
		Text string          `json:"text"`
		Name string          `json:"name"`
		Input json.RawMessage `json:"input"`
		Content json.RawMessage `json:"content"`
	}
	if json.Unmarshal(b, &blocks) == nil {
		var parts []string
		for _, blk := range blocks {
			if blk.Text != "" {
				parts = append(parts, blk.Text)
				continue
			}
			if len(blk.Content) > 0 {
				parts = append(parts, flattenContent(blk.Content))
			}
		}
		return strings.Join(parts, "\n")
	}
	// Unknown shape — return the raw JSON so it's at least visible.
	return string(b)
}

// extractToolUse pulls the first tool_use block out of an assistant
// message, if any. Returns (name, raw-args, true) on success.
func extractToolUse(raw rawClaudeLine) (string, json.RawMessage, bool) {
	if raw.Message == nil || len(raw.Message.Content) == 0 {
		return "", nil, false
	}
	var blocks []struct {
		Type  string          `json:"type"`
		Name  string          `json:"name"`
		Input json.RawMessage `json:"input"`
	}
	if json.Unmarshal(raw.Message.Content, &blocks) != nil {
		return "", nil, false
	}
	for _, blk := range blocks {
		if blk.Type == "tool_use" && blk.Name != "" {
			return blk.Name, blk.Input, true
		}
	}
	return "", nil, false
}
