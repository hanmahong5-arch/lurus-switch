package optimizer

// FixStatus represents the current state of an optimization fix.
type FixStatus string

const (
	FixPending FixStatus = "pending"
	FixRunning FixStatus = "running"
	FixSuccess FixStatus = "success"
	FixFailed  FixStatus = "failed"
	FixSkipped FixStatus = "skipped"
)

// Optimization describes a single actionable improvement suggestion.
type Optimization struct {
	ID          string    `json:"id"`          // e.g. "install-bun", "connect-claude-gateway"
	Category    string    `json:"category"`    // "runtime", "tool", "config", "gateway", "system"
	Priority    int       `json:"priority"`    // 1=critical, 2=warning, 3=info
	Title       string    `json:"title"`       // i18n key for display
	Description string    `json:"description"` // i18n key for detailed explanation
	Action      string    `json:"action"`      // machine action type
	Target      string    `json:"target"`      // tool name or runtime ID
	AutoFixable bool      `json:"autoFixable"` // can be fixed automatically
	Status      FixStatus `json:"status"`
	Error       string    `json:"error,omitempty"`
}

// AnalysisResult holds all discovered optimizations from a single analysis pass.
type AnalysisResult struct {
	Optimizations []Optimization `json:"optimizations"`
	FixableCount  int            `json:"fixableCount"`
	TotalCount    int            `json:"totalCount"`
}

// FixResult holds the outcome of applying a single optimization fix.
type FixResult struct {
	ID      string    `json:"id"`
	Status  FixStatus `json:"status"`
	Message string    `json:"message,omitempty"`
	Error   string    `json:"error,omitempty"`
}
