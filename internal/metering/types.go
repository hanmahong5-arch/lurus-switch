package metering

import "time"

// Record captures a single API call through the gateway.
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
}

// DailySummary aggregates usage for one day.
type DailySummary struct {
	Date       string `json:"date"` // YYYY-MM-DD
	TotalCalls int64  `json:"totalCalls"`
	TokensIn   int64  `json:"tokensIn"`
	TokensOut  int64  `json:"tokensOut"`
	CacheHits  int64  `json:"cacheHits"`
}

// AppSummary aggregates usage by app for a time range.
type AppSummary struct {
	AppID      string `json:"appId"`
	TotalCalls int64  `json:"totalCalls"`
	TokensIn   int64  `json:"tokensIn"`
	TokensOut  int64  `json:"tokensOut"`
	CacheHits  int64  `json:"cacheHits"`
}

// ModelSummary aggregates usage by model for a time range.
type ModelSummary struct {
	Model      string `json:"model"`
	TotalCalls int64  `json:"totalCalls"`
	TokensIn   int64  `json:"tokensIn"`
	TokensOut  int64  `json:"tokensOut"`
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
}

// ActivityEntry is a recent API call for the real-time activity feed.
type ActivityEntry struct {
	Timestamp string `json:"timestamp"` // HH:MM
	AppID     string `json:"appId"`
	Model     string `json:"model"`
	Tokens    int64  `json:"tokens"` // total tokens
}
