package healthscore

// CategoryScore holds the score for a single health dimension.
type CategoryScore struct {
	Category string   `json:"category"` // e.g. "runtimes", "tools", "config", "gateway", "system"
	Score    int      `json:"score"`    // 0-max for this category
	Max      int      `json:"max"`      // max possible score
	Label    string   `json:"label"`    // human-readable label (i18n key)
	Issues   []string `json:"issues"`   // actionable issue descriptions
}

// ScoreReport is the complete health assessment.
type ScoreReport struct {
	TotalScore  int             `json:"totalScore"`  // 0-100
	MaxScore    int             `json:"maxScore"`     // always 100
	Categories  []CategoryScore `json:"categories"`
	Suggestions []Suggestion    `json:"suggestions"` // ordered by priority
}

// Suggestion is an actionable optimization recommendation.
type Suggestion struct {
	ID       string `json:"id"`       // unique key, e.g. "install-bun"
	Priority int    `json:"priority"` // 1=critical, 2=warning, 3=info
	Title    string `json:"title"`    // one-line description
	Action   string `json:"action"`   // action type: "install-tool", "connect-gateway", "configure-key", "update-tool", "install-runtime"
	Target   string `json:"target"`   // tool name or runtime ID
}
