package livesession

import (
	"encoding/json"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"lurus-switch/internal/conversation"
)

// maxRecentEvents caps the per-session "recent activity" list. 10 is
// enough to convey what the assistant just did without overwhelming the
// card and keeps the JSON payload to the frontend small.
const maxRecentEvents = 10

// maxBashHistory caps the bash-command tail. Same trade-off — long enough
// to spot a hung command, short enough to scroll.
const maxBashHistory = 20

// maxFilesTouched caps the FilesTouched list. After 32 unique files the
// list stops being useful as a glance — UI surfaces "+ N more" instead.
const maxFilesTouched = 32

// sessionState is the watcher-internal accumulator. It maps 1-1 to a
// transcript file. snapshot() projects it to a LiveSession suitable for
// the frontend.
type sessionState struct {
	id             string
	tool           string
	cwd            string
	transcriptPath string

	startedAt    time.Time
	lastActivity time.Time
	model        string

	// modelsSeen preserves the order each distinct model was first seen,
	// so the UI can flag "混合模型" sessions where total $ would be
	// misleading if read as one-rate.
	modelsSeen []string

	messages   int
	toolCalls  int
	inputTok   int64
	outputTok  int64
	cacheCreateTok int64
	cacheReadTok   int64
	// cost is accumulated PER MESSAGE using that message's own model. We
	// can't just multiply session totals by lookupPrice(s.model) at the
	// end — a session that toggled opus→sonnet would otherwise see all
	// historical tokens priced at the latest model.
	cost float64

	recent  []EventSummary // ring buffer (size <= maxRecentEvents)
	bashes  []string       // ring buffer (size <= maxBashHistory)
	files   map[string]*FileTouch
	pending *PendingTool // nil when no tool call awaits a result

	// idsAwaitingResult tracks tool_use IDs whose tool_result hasn't been
	// observed yet. Claude may emit multiple tool_use blocks in one
	// assistant turn; we treat any unresolved one as "pending".
	idsAwaitingResult map[string]string // tool_use id → tool name
}

func newState(id, tool, cwd, transcriptPath string) *sessionState {
	return &sessionState{
		id:                id,
		tool:              tool,
		cwd:               cwd,
		transcriptPath:    transcriptPath,
		files:             map[string]*FileTouch{},
		idsAwaitingResult: map[string]string{},
	}
}

// applyEvents folds a chronological slice of parsed events into the state.
// All transcript schemas yield conversation.Events with a Type and a
// Timestamp; everything below works off that uniform view.
func (s *sessionState) applyEvents(evs []conversation.Event) {
	for _, ev := range evs {
		s.applyEvent(ev)
	}
}

func (s *sessionState) applyEvent(ev conversation.Event) {
	if s.startedAt.IsZero() && !ev.Timestamp.IsZero() {
		s.startedAt = ev.Timestamp
	}
	if !ev.Timestamp.IsZero() && ev.Timestamp.After(s.lastActivity) {
		s.lastActivity = ev.Timestamp
	}
	if ev.Model != "" {
		s.model = ev.Model
		if !contains(s.modelsSeen, ev.Model) {
			s.modelsSeen = append(s.modelsSeen, ev.Model)
		}
	}
	// Accumulate raw token counts for the UI display, and accumulate
	// $-cost per-event using the event's OWN model so a session that
	// toggled models is priced correctly piecewise.
	s.inputTok += ev.InputTokens
	s.outputTok += ev.OutputTokens
	s.cacheCreateTok += ev.CacheCreationTokens
	s.cacheReadTok += ev.CacheReadTokens
	if ev.InputTokens > 0 || ev.OutputTokens > 0 || ev.CacheCreationTokens > 0 || ev.CacheReadTokens > 0 {
		model := ev.Model
		if model == "" {
			// Some lines (system / tool_result) don't carry a model; fall
			// back to the last assistant's model rather than the cheapest
			// fallback price.
			model = s.model
		}
		s.cost += eventCost(model, ev.InputTokens, ev.OutputTokens, ev.CacheCreationTokens, ev.CacheReadTokens)
	}

	switch ev.Type {
	case conversation.EventUser:
		s.messages++
		s.pushRecent(EventSummary{
			Time:  ev.Timestamp,
			Kind:  "user",
			Label: previewText("用户消息", ev.Content, 80),
		})
	case conversation.EventAssistant:
		s.messages++
		s.pushRecent(EventSummary{
			Time:  ev.Timestamp,
			Kind:  "assistant",
			Label: previewText("助手回复", ev.Content, 80),
		})
	case conversation.EventToolUse:
		s.toolCalls++
		preview := summariseToolArgs(ev.ToolName, ev.ToolArgs)
		s.pushRecent(EventSummary{
			Time:    ev.Timestamp,
			Kind:    "tool",
			Label:   "调用工具 " + ev.ToolName,
			Details: preview,
		})
		s.idsAwaitingResult[toolUseID(ev)] = ev.ToolName
		s.pending = &PendingTool{
			Name:      ev.ToolName,
			Preview:   preview,
			StartedAt: ev.Timestamp,
		}
		// Side-effect bookkeeping: bash history, file touches.
		s.recordToolSideEffect(ev)
	case conversation.EventToolResult:
		// Mark the matching tool_use as resolved. If we don't have an ID
		// match we still clear `pending` once any result lands — Claude's
		// tool_result lines don't always echo the parent UUID predictably.
		if id := toolUseID(ev); id != "" {
			delete(s.idsAwaitingResult, id)
		}
		if len(s.idsAwaitingResult) == 0 {
			s.pending = nil
		}
		s.pushRecent(EventSummary{
			Time:  ev.Timestamp,
			Kind:  "result",
			Label: "工具返回",
			Details: previewText("", ev.Content, 120),
		})
	case conversation.EventSystem:
		// System messages are usually compaction / model-switch markers.
		// Surface them dimly so the user knows where context got cut.
		s.pushRecent(EventSummary{
			Time:  ev.Timestamp,
			Kind:  "system",
			Label: previewText("系统", ev.Content, 80),
		})
	}
}

// toolUseID pulls the tool_use ID out of an event when present. The
// parser stores it under different keys depending on direction; we check
// both so a missing ID doesn't break pending-state tracking.
func toolUseID(ev conversation.Event) string {
	// Defensive: peek the raw line for either of the IDs Claude/Codex use.
	var probe struct {
		ToolUseID string `json:"tool_use_id"`
		ID        string `json:"id"`
	}
	if len(ev.Raw) > 0 {
		_ = json.Unmarshal(ev.Raw, &probe)
		if probe.ToolUseID != "" {
			return probe.ToolUseID
		}
		if probe.ID != "" {
			return probe.ID
		}
	}
	return ev.MessageUUID
}

// recordToolSideEffect updates the FilesTouched map and BashCommands tail
// when a tool_use targets a file/shell. Unknown tool names are no-ops.
func (s *sessionState) recordToolSideEffect(ev conversation.Event) {
	if len(ev.ToolArgs) == 0 {
		return
	}
	var args map[string]json.RawMessage
	if err := json.Unmarshal(ev.ToolArgs, &args); err != nil {
		return
	}
	switch ev.ToolName {
	case "Bash":
		var cmd string
		if raw, ok := args["command"]; ok {
			_ = json.Unmarshal(raw, &cmd)
		}
		if cmd != "" {
			s.bashes = append(s.bashes, cmd)
			if len(s.bashes) > maxBashHistory {
				s.bashes = s.bashes[len(s.bashes)-maxBashHistory:]
			}
		}
	case "Read":
		s.bumpFile(args, "file_path", "read")
	case "Edit", "NotebookEdit":
		s.bumpFile(args, "file_path", "edit")
	case "Write":
		s.bumpFile(args, "file_path", "write")
	}
}

func (s *sessionState) bumpFile(args map[string]json.RawMessage, key, kind string) {
	raw, ok := args[key]
	if !ok {
		return
	}
	var p string
	if err := json.Unmarshal(raw, &p); err != nil || p == "" {
		return
	}
	if existing, ok := s.files[p]; ok {
		existing.Count++
		// Upgrade kind: read→edit→write. Once a file is written, count it as written.
		if kindRank(kind) > kindRank(existing.Kind) {
			existing.Kind = kind
		}
		return
	}
	if len(s.files) >= maxFilesTouched*2 {
		// Prevent map blowup on a runaway session — drop the lowest-count
		// entries periodically. The UI only renders maxFilesTouched anyway.
		s.pruneFiles()
	}
	s.files[p] = &FileTouch{Path: p, Count: 1, Kind: kind}
}

func kindRank(k string) int {
	switch k {
	case "write":
		return 3
	case "edit":
		return 2
	case "read":
		return 1
	}
	return 0
}

func (s *sessionState) pruneFiles() {
	type kv struct {
		path  string
		count int
	}
	all := make([]kv, 0, len(s.files))
	for p, f := range s.files {
		all = append(all, kv{p, f.Count})
	}
	sort.Slice(all, func(i, j int) bool { return all[i].count > all[j].count })
	keep := all
	if len(keep) > maxFilesTouched {
		keep = keep[:maxFilesTouched]
	}
	pruned := make(map[string]*FileTouch, len(keep))
	for _, kv := range keep {
		pruned[kv.path] = s.files[kv.path]
	}
	s.files = pruned
}

func (s *sessionState) pushRecent(ev EventSummary) {
	s.recent = append(s.recent, ev)
	if len(s.recent) > maxRecentEvents {
		s.recent = s.recent[len(s.recent)-maxRecentEvents:]
	}
}

// snapshot projects the internal state into the public LiveSession form,
// computing derived fields (status, cost, sorted file list).
func (s *sessionState) snapshot(now time.Time) LiveSession {
	files := make([]FileTouch, 0, len(s.files))
	for _, f := range s.files {
		files = append(files, *f)
	}
	sort.Slice(files, func(i, j int) bool {
		if files[i].Count != files[j].Count {
			return files[i].Count > files[j].Count
		}
		return files[i].Path < files[j].Path
	})
	if len(files) > maxFilesTouched {
		files = files[:maxFilesTouched]
	}

	return LiveSession{
		SessionID:         s.id,
		Tool:              s.tool,
		Cwd:               s.cwd,
		ProjectName:       projectNameFromCwd(s.cwd),
		StartedAt:         s.startedAt,
		LastActivity:      s.lastActivity,
		Model:             s.model,
		TranscriptPath:    s.transcriptPath,
		Status:            s.statusAt(now),
		PendingTool:       s.pending,
		Recent:            append([]EventSummary(nil), s.recent...),
		MessageCount:      s.messages,
		ToolCallCount:     s.toolCalls,
		InputTokens:       s.inputTok,
		OutputTokens:      s.outputTok,
		CacheCreateTokens: s.cacheCreateTok,
		CacheReadTokens:   s.cacheReadTok,
		EstimatedUSD:      s.cost,
		ModelsSeen:        append([]string(nil), s.modelsSeen...),
		BashCommands:      append([]string(nil), s.bashes...),
		FilesTouched:      files,
	}
}

// contains is a small helper for tracking model order without pulling in
// a set type. Sessions rarely see more than 2-3 distinct models.
func contains(xs []string, s string) bool {
	for _, x := range xs {
		if x == s {
			return true
		}
	}
	return false
}

// statusAt classifies the session for the UI based on event recency +
// outstanding tool calls. Thresholds are deliberately generous — a 10s
// gap during a long bash command shouldn't flip the badge.
func (s *sessionState) statusAt(now time.Time) string {
	if s.pending != nil {
		return "tool_call"
	}
	age := now.Sub(s.lastActivity)
	switch {
	case age <= 10*time.Second:
		return "running"
	case age <= 30*time.Second:
		return "awaiting_user"
	case age <= 5*time.Minute:
		return "awaiting_user"
	default:
		return "idle"
	}
}

// projectNameFromCwd returns the last path segment of cwd for use as the
// card title. Falls back to the cwd itself when it's not a path.
func projectNameFromCwd(cwd string) string {
	if cwd == "" {
		return "(unknown)"
	}
	cleaned := filepath.ToSlash(cwd)
	cleaned = strings.TrimRight(cleaned, "/")
	if idx := strings.LastIndex(cleaned, "/"); idx >= 0 {
		return cleaned[idx+1:]
	}
	return cleaned
}

// summariseToolArgs builds a short one-line preview of the tool arguments
// for display. Different tools have very different argument shapes, so we
// pick the highest-signal field per tool.
func summariseToolArgs(tool string, args json.RawMessage) string {
	if len(args) == 0 {
		return ""
	}
	var m map[string]json.RawMessage
	if err := json.Unmarshal(args, &m); err != nil {
		return ""
	}
	get := func(key string) string {
		raw, ok := m[key]
		if !ok {
			return ""
		}
		var s string
		_ = json.Unmarshal(raw, &s)
		return s
	}
	switch tool {
	case "Bash":
		return truncate(get("command"), 120)
	case "Read":
		return truncate(get("file_path"), 120)
	case "Edit", "Write", "NotebookEdit":
		return truncate(get("file_path"), 120)
	case "Glob":
		return "pattern: " + truncate(get("pattern"), 100)
	case "Grep":
		p := get("pattern")
		path := get("path")
		s := "/" + p + "/"
		if path != "" {
			s += " in " + path
		}
		return truncate(s, 120)
	case "WebFetch", "WebSearch":
		if u := get("url"); u != "" {
			return truncate(u, 120)
		}
		return truncate(get("query"), 120)
	case "Task":
		return truncate(get("description"), 120)
	}
	// Fallback: take the first short string-valued field we find.
	for _, raw := range m {
		var s string
		if json.Unmarshal(raw, &s) == nil && s != "" {
			return truncate(s, 120)
		}
	}
	return ""
}

func previewText(prefix, content string, max int) string {
	c := strings.TrimSpace(content)
	c = strings.ReplaceAll(c, "\n", " ")
	if max > 0 && len(c) > max {
		c = c[:max] + "…"
	}
	if prefix == "" {
		return c
	}
	if c == "" {
		return prefix
	}
	return prefix + ": " + c
}

func truncate(s string, max int) string {
	s = strings.TrimSpace(strings.ReplaceAll(s, "\n", " "))
	if max > 0 && len(s) > max {
		s = s[:max] + "…"
	}
	return s
}
