package gateway

import "time"

// Config holds persistent gateway configuration.
type Config struct {
	Port        int    `json:"port"`        // default 19090
	UpstreamURL string `json:"upstreamUrl"` // Lurus Cloud endpoint, e.g. https://api.lurus.cn
	UserToken   string `json:"userToken"`   // user's Lurus Cloud bearer token
	AutoStart   bool   `json:"autoStart"`   // start gateway on Switch launch

	// Deprecated: ordered fallback upstreams. Wave 3 (PR-W3.1) replaced
	// this with the relay router's per-rule chain; Wave 4 (PR-W4.2)
	// migrates the field one-shot into the relay store and then never
	// reads it again. The field stays for back-compat decoding of older
	// gateway.json files. `omitempty` lets new saves drop it cleanly.
	Fallbacks []FallbackEntry `json:"fallbacks,omitempty"`
}

// DefaultConfig returns production defaults.
func DefaultConfig() Config {
	return Config{
		Port:       19090,
		UpstreamURL: "",
		UserToken:  "",
		AutoStart:  false,
	}
}

// Status describes the current state of the local gateway.
type Status struct {
	Running       bool   `json:"running"`
	Port          int    `json:"port"`
	URL           string `json:"url"`           // "http://localhost:PORT" or ""
	Uptime        int64  `json:"uptime"`        // seconds since start
	TotalRequests int64  `json:"totalRequests"` // lifetime request count
	ActiveConns   int32  `json:"activeConns"`   // current in-flight requests
}

// UsageFromResponse captures token usage parsed from an upstream API response.
//
// PromptTokens / CompletionTokens are the raw OpenAI-shape counts. CachedTokens
// is the prompt-cache subset already inside PromptTokens; ReasoningTokens is the
// "thinking" subset already inside CompletionTokens. The normalization into
// billable (TokensIn, CacheReadTokens) lives in recordUsage — see the comment
// there — so this struct stays a faithful mirror of the upstream payload.
type UsageFromResponse struct {
	Model            string
	PromptTokens     int64
	CompletionTokens int64
	TotalTokens      int64
	CachedTokens     int64 // prompt_tokens_details.cached_tokens (subset of PromptTokens)
	ReasoningTokens  int64 // completion_tokens_details.reasoning_tokens (subset of CompletionTokens)
}

// RequestMeta holds per-request context passed through middleware.
type RequestMeta struct {
	AppID     string
	StartTime time.Time
	Model     string
	ServedBy  string // which upstream actually served this request ("primary" or fallback name)
	MatchedBy string // relay rule name that selected the primary upstream; empty for cfg / mapping defaults

	// Enterprise dimensions sourced from the per-app registry record.
	// Empty in Personal/Reseller installs; the chargeback report
	// buckets unattributed traffic separately.
	OwnerEmployeeID string
	CostCenter      string
}
