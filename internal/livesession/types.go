// Package livesession turns the JSONL transcripts that Claude / Codex /
// Gemini write to disk into a real-time "what is the AI doing right now"
// view for the Switch GUI.
//
// The package is deliberately read-only — it never writes to a transcript.
// Tail-style polling (every 2s) keeps the implementation portable across
// Windows (where fsnotify on user-home directories is unreliable) and
// non-disruptive (no inotify watchers leaking into the session-file owner).
package livesession

import "time"

// LiveSession is the public snapshot of one in-flight or recently-active
// CLI session. The Switch UI renders a card per LiveSession.
type LiveSession struct {
	SessionID      string    `json:"sessionId"`
	Tool           string    `json:"tool"` // "claude" | "codex" | "gemini"
	Cwd            string    `json:"cwd"`
	ProjectName    string    `json:"projectName"` // last segment of cwd, for card title
	StartedAt      time.Time `json:"startedAt"`
	LastActivity   time.Time `json:"lastActivity"`
	Model          string    `json:"model,omitempty"`
	TranscriptPath string    `json:"transcriptPath"`

	// Live activity ------------------------------------------------------
	// Status is a coarse state derived from event timestamps:
	//   "running"     — last event < 10s old, no pending tool_use
	//   "tool_call"   — assistant emitted a tool_use that has no matching
	//                   tool_result yet (Claude is waiting for Bash/etc.)
	//   "awaiting_user" — last event was a tool_result and no follow-up
	//                     assistant message has arrived within 30s
	//   "idle"        — last event > 5min old
	Status      string        `json:"status"`
	PendingTool *PendingTool  `json:"pendingTool,omitempty"`
	Recent      []EventSummary `json:"recent"` // tail of last ~8 events, oldest first

	// Aggregates ---------------------------------------------------------
	MessageCount        int     `json:"messageCount"`
	ToolCallCount       int     `json:"toolCallCount"`
	InputTokens         int64   `json:"inputTokens"`
	OutputTokens        int64   `json:"outputTokens"`
	CacheCreateTokens   int64   `json:"cacheCreateTokens"`
	CacheReadTokens     int64   `json:"cacheReadTokens"`
	EstimatedUSD        float64 `json:"estimatedUsd"`
	// Models seen this session in order of first appearance. When >1 the
	// UI surfaces a "混合模型" qualifier so the user doesn't read the cost
	// as if it's all priced at one rate.
	ModelsSeen []string `json:"modelsSeen,omitempty"`

	// Last 20 bash commands so the UI can flag long-running shell calls
	// without having to re-walk the transcript.
	BashCommands []string `json:"bashCommands,omitempty"`

	// Files this session has read/edited, ranked by edit count. Cap at 16
	// entries to keep the payload tight.
	FilesTouched []FileTouch `json:"filesTouched,omitempty"`
}

// PendingTool tracks a tool_use that hasn't seen its tool_result yet.
// A non-nil PendingTool is the strongest signal that "Claude is busy".
type PendingTool struct {
	Name      string    `json:"name"`
	Preview   string    `json:"preview"` // truncated args (bash command / file path / etc.)
	StartedAt time.Time `json:"startedAt"`
}

// EventSummary is a UI-ready one-liner for the recent-activity list.
// Keep it small — the watcher emits this on every poll tick.
type EventSummary struct {
	Time    time.Time `json:"time"`
	Kind    string    `json:"kind"`    // "user" | "assistant" | "tool" | "result" | "system"
	Label   string    `json:"label"`   // one-line summary
	Details string    `json:"details,omitempty"` // optional second line (e.g. tool args preview)
}

// FileTouch is one file the session has read or written, with how many
// times. Counts let the UI prioritise hot files at the top.
type FileTouch struct {
	Path  string `json:"path"`
	Count int    `json:"count"`
	Kind  string `json:"kind"` // "read" | "edit" | "write"
}

// IsActive reports whether a session counts as "live" for UI purposes —
// any activity within activeWindow. Older sessions are kept in memory but
// filtered out of the default view.
func (s *LiveSession) IsActive(now time.Time, activeWindow time.Duration) bool {
	return now.Sub(s.LastActivity) <= activeWindow
}
