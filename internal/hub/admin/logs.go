package admin

import (
	"context"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

// LogQuery scopes a log search. Zero-valued fields are omitted from the
// request — Hub's defaults apply (typically last 30 days, all types).
type LogQuery struct {
	Page         int
	PageSize     int
	Username     string    // filter by user (admin-only effective)
	TokenName    string    // filter by token name
	ModelName    string    // filter by model
	Type         int       // 1 user, 2 admin, 3 system, 4 consume, 5 manage, 6 error
	StartAt      time.Time // inclusive lower bound (Unix seconds)
	EndAt        time.Time // exclusive upper bound
	ChannelID    int       // filter by channel
	Group        string    // filter by user group
	IP           string    // filter by client IP
	OnlyMine     bool      // when true, hits /api/log/self/ instead of /api/log/
}

// ListLogs queries Hub's log table. When q.OnlyMine is true, it hits the
// self endpoint (works with any UserAuth role); otherwise the admin
// endpoint requires admin role.
func (c *Client) ListLogs(ctx context.Context, q LogQuery) (*LogPage, error) {
	v := url.Values{}
	if q.Page > 0 {
		v.Set("p", strconv.Itoa(q.Page))
	}
	if q.PageSize > 0 {
		v.Set("page_size", strconv.Itoa(q.PageSize))
	}
	if q.Username != "" {
		v.Set("username", q.Username)
	}
	if q.TokenName != "" {
		v.Set("token_name", q.TokenName)
	}
	if q.ModelName != "" {
		v.Set("model_name", q.ModelName)
	}
	if q.Type != 0 {
		v.Set("type", strconv.Itoa(q.Type))
	}
	if !q.StartAt.IsZero() {
		v.Set("start_timestamp", strconv.FormatInt(q.StartAt.Unix(), 10))
	}
	if !q.EndAt.IsZero() {
		v.Set("end_timestamp", strconv.FormatInt(q.EndAt.Unix(), 10))
	}
	if q.ChannelID > 0 {
		v.Set("channel", strconv.Itoa(q.ChannelID))
	}
	if q.Group != "" {
		v.Set("group", q.Group)
	}
	if q.IP != "" {
		v.Set("ip", q.IP)
	}

	path := "/api/log/"
	if q.OnlyMine {
		path = "/api/log/self/"
	}
	var out LogPage
	if err := c.do(ctx, http.MethodGet, path, v, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// LogStat is the daily breakdown returned by /api/log/self/stat.
type LogStat struct {
	Day   string  `json:"day"`
	Quota int64   `json:"quota"`
	Count int64   `json:"count"`
	RPM   float64 `json:"rpm"`
}

// GetSelfLogStats fetches the per-day usage stats for the authenticated
// user (used by the Reseller dashboard "我的近 30 天" panel).
func (c *Client) GetSelfLogStats(ctx context.Context) ([]LogStat, error) {
	var out []LogStat
	if err := c.do(ctx, http.MethodGet, "/api/log/self/stat", nil, nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}
