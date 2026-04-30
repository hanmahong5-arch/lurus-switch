package gateway

import "time"

// Config holds persistent gateway configuration.
type Config struct {
	Port        int             `json:"port"`        // default 19090
	UpstreamURL string          `json:"upstreamUrl"` // Lurus Cloud endpoint, e.g. https://api.lurus.cn
	UserToken   string          `json:"userToken"`   // user's Lurus Cloud bearer token
	AutoStart   bool            `json:"autoStart"`   // start gateway on Switch launch
	Fallbacks   []FallbackEntry `json:"fallbacks"`   // ordered fallback upstreams (tried if primary fails/rate-limits)
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
type UsageFromResponse struct {
	Model        string
	PromptTokens int64
	CompletionTokens int64
	TotalTokens  int64
}

// RequestMeta holds per-request context passed through middleware.
type RequestMeta struct {
	AppID     string
	StartTime time.Time
	Model     string
	ServedBy  string // which upstream actually served this request ("primary" or fallback name)
}
