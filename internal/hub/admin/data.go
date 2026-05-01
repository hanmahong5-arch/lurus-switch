package admin

import (
	"context"
	"net/http"
	"net/url"
)

// DashboardSummary is the V1 `/api/data/` response — it's a flat object
// derived from newapi/newhub's `controller.DashboardData`. Field types
// match the upstream JSON tags so `json.Unmarshal` populates everything
// in one shot.
type DashboardSummary struct {
	UserCount    int   `json:"user_count"`
	ChannelCount int   `json:"channel_count"`
	TokenCount   int   `json:"token_count"`
	TodayRequest int64 `json:"today_request"`
	TodayQuota   int64 `json:"today_quota"`
	TodayTokens  int64 `json:"today_tokens"`
}

// GetDashboardSummary fetches the lightweight dashboard counters.
func (c *Client) GetDashboardSummary(ctx context.Context) (*DashboardSummary, error) {
	var out DashboardSummary
	if err := c.do(ctx, http.MethodGet, "/api/data/", nil, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// QuotaDate is one row of the 14-day usage table. ModelUsage is opaque
// (model name → request count) — the dashboard chart just renders the
// `quota` field for now, so nested decoding stays generous.
type QuotaDate struct {
	Date         string         `json:"date"`
	Quota        int64          `json:"quota"`
	RequestCount int            `json:"request_count"`
	TokenCount   int64          `json:"token_count"`
	ModelUsage   map[string]int `json:"model_usage"`
}

// GetQuotaDates returns the per-day usage rollup for [startDate, endDate]
// inclusive. Dates are formatted "YYYY-MM-DD" — Hub does the bucketing.
func (c *Client) GetQuotaDates(ctx context.Context, startDate, endDate string) ([]QuotaDate, error) {
	q := url.Values{}
	q.Set("start_date", startDate)
	q.Set("end_date", endDate)
	var out []QuotaDate
	if err := c.do(ctx, http.MethodGet, "/api/data/quota_dates", q, nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// PerformanceStats wraps Hub's runtime metrics — useful for ops panels but
// reflects only the Hub process, not aggregate fleet metrics. Reseller
// dashboards may surface this read-only.
type PerformanceStats struct {
	Goroutines     int     `json:"goroutines"`
	MemoryAlloc    int64   `json:"memory_alloc"`
	Uptime         int64   `json:"uptime"`
	RequestsTotal  int64   `json:"requests_total"`
	RequestsPerSec float64 `json:"requests_per_sec"`
}

// GetPerformanceStats returns the Hub process runtime stats.
func (c *Client) GetPerformanceStats(ctx context.Context) (*PerformanceStats, error) {
	var out PerformanceStats
	if err := c.do(ctx, http.MethodGet, "/api/performance/stats", nil, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
