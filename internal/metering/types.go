package metering

import "time"

// Record captures a single API call through the gateway.
//
// Enterprise-mode dimensions (CostCenter / EmployeeID / ProjectTag) are
// optional and only populated when the gateway request meta carries
// them. Personal / Reseller deployments leave them empty.
type Record struct {
	ID           string    `json:"id"`
	AppID        string    `json:"appId"`
	Model        string    `json:"model"`
	TokensIn     int64     `json:"tokensIn"`
	TokensOut    int64     `json:"tokensOut"`
	LatencyMs    int64     `json:"latencyMs"`
	CachedHit    bool      `json:"cachedHit"`
	StatusCode   int       `json:"statusCode"`
	ErrorMessage string    `json:"errorMessage,omitempty"`
	Timestamp    time.Time `json:"timestamp"`

	// Cache-stream token counts, billed at the cache rates by pricing.Cost.
	// Anthropic reports cache-create / cache-read independently of
	// input_tokens (so they ADD to TokensIn). The OpenAI-compat path reports
	// a cached_tokens subset that is already inside prompt_tokens — the
	// gateway subtracts it out of TokensIn and lands it here so the cache-read
	// stream bills at the discounted rate instead of full input price. Both
	// default 0 for tools / sessions that don't cache.
	CacheCreateTokens int64 `json:"cacheCreateTokens"`
	CacheReadTokens   int64 `json:"cacheReadTokens"`

	// ReasoningTokens is a display-only breakdown of the output stream. It is
	// ALREADY counted in TokensOut (OpenAI completion_tokens) and is never
	// billed a second time — kept so dashboards can show "thinking" volume.
	ReasoningTokens int64 `json:"reasoningTokens,omitempty"`

	// Enterprise dimensions (optional).
	CostCenter string `json:"costCenter,omitempty"` // e.g. "ENG-PLATFORM-001"
	EmployeeID string `json:"employeeId,omitempty"` // SSO sub claim
	ProjectTag string `json:"projectTag,omitempty"` // free-form, set by user/agent

	// Routing dimensions populated by the gateway when the relay router
	// is active. ServedBy carries the endpoint display name (e.g.
	// "DeepSeek backup"), MatchedBy the rule name that selected the
	// primary. Both empty when the cfg-driven path served the request.
	ServedBy  string `json:"servedBy,omitempty"`
	MatchedBy string `json:"matchedBy,omitempty"`
}

// DailySummary aggregates usage for one day. CostUSD is computed at
// query time from the internal/pricing table — not persisted, so
// price-table updates apply retroactively when historical data is
// recomputed.
type DailySummary struct {
	Date       string  `json:"date"` // YYYY-MM-DD
	TotalCalls int64   `json:"totalCalls"`
	TokensIn   int64   `json:"tokensIn"`
	TokensOut  int64   `json:"tokensOut"`
	CacheHits  int64   `json:"cacheHits"`
	CostUSD    float64 `json:"costUSD"`
}

// AppSummary aggregates usage by app for a time range.
type AppSummary struct {
	AppID      string  `json:"appId"`
	TotalCalls int64   `json:"totalCalls"`
	TokensIn   int64   `json:"tokensIn"`
	TokensOut  int64   `json:"tokensOut"`
	CacheHits  int64   `json:"cacheHits"`
	CostUSD    float64 `json:"costUSD"`
}

// CostCenterSummary aggregates usage by cost-center for chargeback
// reporting. Only meaningful in Enterprise mode.
type CostCenterSummary struct {
	CostCenter string `json:"costCenter"`
	TotalCalls int64  `json:"totalCalls"`
	TokensIn   int64  `json:"tokensIn"`
	TokensOut  int64  `json:"tokensOut"`
	UniqueEmps int    `json:"uniqueEmployees"` // distinct employee IDs in the bucket
}

// EmployeeSummary aggregates per-employee usage for the second view
// of the chargeback dashboard. The CostCenter field is included so
// the UI can color-band employees by department without a second
// lookup.
type EmployeeSummary struct {
	EmployeeID string `json:"employeeId"`
	CostCenter string `json:"costCenter"`
	TotalCalls int64  `json:"totalCalls"`
	TokensIn   int64  `json:"tokensIn"`
	TokensOut  int64  `json:"tokensOut"`
}

// ModelSummary aggregates usage by model for a time range.
type ModelSummary struct {
	Model      string  `json:"model"`
	TotalCalls int64   `json:"totalCalls"`
	TokensIn   int64   `json:"tokensIn"`
	TokensOut  int64   `json:"tokensOut"`
	CostUSD    float64 `json:"costUSD"`
}

// InsightsRaw holds raw aggregated data for cost/rate-limit/latency insights.
type InsightsRaw struct {
	TotalCalls      int64            `json:"totalCalls"`
	TotalTokensIn   int64            `json:"totalTokensIn"`
	TotalTokensOut  int64            `json:"totalTokensOut"`
	CacheHits       int64            `json:"cacheHits"`
	RateLimitEvents int64            `json:"rateLimitEvents"` // HTTP 429 count
	ErrorEvents     int64            `json:"errorEvents"`     // HTTP 5xx count
	TotalLatencyMs  int64            `json:"-"`               // internal sum for avg calc
	AvgLatencyMs    int64            `json:"avgLatencyMs"`
	ModelTokensIn   map[string]int64 `json:"modelTokensIn"`
	ModelTokensOut  map[string]int64 `json:"modelTokensOut"`
	TotalCostUSD    float64          `json:"totalCostUSD"`
}

// ActivityEntry is a recent API call for the real-time activity feed.
type ActivityEntry struct {
	Timestamp string `json:"timestamp"` // HH:MM
	AppID     string `json:"appId"`
	Model     string `json:"model"`
	Tokens    int64  `json:"tokens"` // total tokens
}
