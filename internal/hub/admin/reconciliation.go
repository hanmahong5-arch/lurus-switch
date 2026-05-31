package admin

import (
	"context"
	"net/http"
)

// Hub-side usage reconciliation (Wave 1 W1.2).
//
// FetchUsage hits the public Switch reconciliation endpoint, which authenticates
// with a raw user_token (Token.Key) — the SAME token the Switch gateway routes
// upstream through — and returns that token's owning user's consume-log totals
// for a window. The caller compares this against Switch's local metering to
// surface aggregate drift. Read-only on the Hub side; no admin role required.

// switchReconciliationPath is the public POST endpoint (inline token auth).
const switchReconciliationPath = "/api/v2/switch/reconciliation"

// hubQuotaPerUnit is newapi's quota↔USD scale (500000 quota = $1). Used to
// turn the Hub's quota totals into the USD figure the reconcile report compares.
const hubQuotaPerUnit = 500000.0

// reconcileRequest is the POST body: an inclusive Unix-second window.
type reconcileRequest struct {
	StartTime int64 `json:"start_time"`
	EndTime   int64 `json:"end_time"`
}

// SwitchUsageModel is the per-model breakdown row.
type SwitchUsageModel struct {
	ModelName        string `json:"model_name"`
	Quota            int64  `json:"quota"`
	PromptTokens     int64  `json:"prompt_tokens"`
	CompletionTokens int64  `json:"completion_tokens"`
	RequestCount     int64  `json:"request_count"`
}

// SwitchUsageAgg is the Hub-side aggregate for the requested window.
type SwitchUsageAgg struct {
	TotalQuota            int64              `json:"total_quota"`
	TotalPromptTokens     int64              `json:"total_prompt_tokens"`
	TotalCompletionTokens int64              `json:"total_completion_tokens"`
	RequestCount          int64              `json:"request_count"`
	Models                []SwitchUsageModel `json:"models"`
}

// CostUSD converts the Hub's quota total into USD using newapi's standard
// quota-per-unit scale.
func (a *SwitchUsageAgg) CostUSD() float64 {
	if a == nil {
		return 0
	}
	return float64(a.TotalQuota) / hubQuotaPerUnit
}

// FetchUsage queries the Hub's consume-log totals for [startUnix, endUnix]
// (inclusive, Unix seconds) for the user_token this client is configured with.
func (c *Client) FetchUsage(ctx context.Context, startUnix, endUnix int64) (*SwitchUsageAgg, error) {
	body := reconcileRequest{StartTime: startUnix, EndTime: endUnix}
	var out SwitchUsageAgg
	if err := c.do(ctx, http.MethodPost, switchReconciliationPath, nil, body, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
