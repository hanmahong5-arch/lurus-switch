package admin

import "encoding/json"

// ListOpts carries the pagination + filter parameters accepted by Hub's list
// endpoints. Extra is a passthrough for resource-specific filters (e.g.
// `status`, `tag_mode`, `id_sort`).
type ListOpts struct {
	Page     int
	PageSize int
	Keyword  string
	Extra    map[string]string
}

// Page is a generic-shaped wrapper around Hub's PageInfo response.
// Wails generates separate types for the concrete instantiations below so the
// frontend gets named TypeScript types — not raw maps.
type ChannelPage struct {
	Items    []Channel `json:"items"`
	Page     int       `json:"page"`
	PageSize int       `json:"page_size"`
	Total    int       `json:"total"`
}

type TokenPage struct {
	Items    []Token `json:"items"`
	Page     int     `json:"page"`
	PageSize int     `json:"page_size"`
	Total    int     `json:"total"`
}

type RedemptionPage struct {
	Items    []Redemption `json:"items"`
	Page     int          `json:"page"`
	PageSize int          `json:"page_size"`
	Total    int          `json:"total"`
}

type LogPage struct {
	Items    []LogEntry `json:"items"`
	Page     int        `json:"page"`
	PageSize int        `json:"page_size"`
	Total    int        `json:"total"`
}

type TenantList struct {
	Items []Tenant `json:"items"`
	Total int      `json:"total"`
}

// Channel is the subset of Hub's Channel entity Switch surfaces in
// GatewayChannelPage. Fields not modeled here are passed through verbatim
// via Raw — frontend may render them, but Switch logic does not branch on
// them. See 2b-svc-newhub/internal/domain/entity/channel.go for the full
// schema.
type Channel struct {
	ID           int             `json:"id"`
	Name         string          `json:"name"`
	Type         int             `json:"type"`     // upstream provider code (constant.ChannelType*)
	Status       int             `json:"status"`   // 1 enabled, 0+ disabled
	Group        string          `json:"group"`
	Models       string          `json:"models"`   // comma-separated model IDs
	Weight       *uint           `json:"weight"`
	Priority     *int64          `json:"priority"`
	BaseURL      *string         `json:"base_url"`
	Balance      float64         `json:"balance"`
	UsedQuota    int64           `json:"used_quota"`
	Tag          *string         `json:"tag"`
	Remark       *string         `json:"remark"`
	CreatedTime  int64           `json:"created_time"`
	TestTime     int64           `json:"test_time"`
	ResponseTime int             `json:"response_time"`
	Raw          json.RawMessage `json:"-"` // populated by ListChannels for unmodeled fields when callers ask
}

// Token mirrors the V1 /api/token entity Switch displays in GatewayTokenPage.
type Token struct {
	ID             int    `json:"id"`
	UserID         int    `json:"user_id"`
	Key            string `json:"key"`
	Name           string `json:"name"`
	Status         int    `json:"status"`
	UsedQuota      int64  `json:"used_quota"`
	RemainQuota    int64  `json:"remain_quota"`
	UnlimitedQuota bool   `json:"unlimited_quota"`
	ExpiredTime    int64  `json:"expired_time"`
	CreatedTime    int64  `json:"created_time"`
	AccessedTime   int64  `json:"accessed_time"`
	ModelLimits    string `json:"model_limits"`
	Group          string `json:"group"`
}

// Redemption is a single activation code. CreateRedemptionBatch issues N at
// a time; UI lists them paginated.
type Redemption struct {
	ID          int    `json:"id"`
	UserID      int    `json:"user_id"`
	Key         string `json:"key"`
	Name        string `json:"name"`
	Status      int    `json:"status"` // 1 unused, 2 used, 3 disabled
	Quota       int64  `json:"quota"`
	UsedID      int    `json:"used_id"`
	UsedTime    int64  `json:"used_time"`
	ExpiredTime int64  `json:"expired_time"`
	CreatedTime int64  `json:"created_time"`
}

// LogEntry covers the fields rendered in GatewayLogPage. Hub's actual log
// table has many more (model fingerprints, cache hits, etc.) — those flow
// via wails JSON serialization unchanged when present.
type LogEntry struct {
	ID                int    `json:"id"`
	UserID            int    `json:"user_id"`
	Username          string `json:"username"`
	CreatedAt         int64  `json:"created_at"`
	Type              int    `json:"type"` // 1 user, 2 admin, 3 system, 4 consume, 5 manage, 6 error
	Content           string `json:"content"`
	ModelName         string `json:"model_name"`
	TokenName         string `json:"token_name"`
	Quota             int64  `json:"quota"`
	PromptTokens      int    `json:"prompt_tokens"`
	CompletionTokens  int    `json:"completion_tokens"`
	UseTime           int    `json:"use_time"` // request latency in seconds
	IsStream          bool   `json:"is_stream"`
	ChannelID         int    `json:"channel"`
	IP                string `json:"ip"`
	Group             string `json:"group"`
}

// Tenant is a V2 multi-tenant entry created by platform admins (Switch
// Reseller-mode pulls a list to populate its tenant slug picker).
type Tenant struct {
	ID        string `json:"id"`
	Slug      string `json:"slug"`
	Name      string `json:"name"`
	Status    string `json:"status"`
	CreatedAt int64  `json:"created_at"`
}

// CreateChannelInput is the minimal payload for AddChannel. Hub accepts a
// large channel struct; callers populate only the fields they care about
// and Hub fills in defaults. Use map for forward compatibility.
type CreateChannelInput map[string]any

// CreateTokenInput is the AddToken payload.
type CreateTokenInput map[string]any

// CreateRedemptionInput is the AddRedemption payload. Hub generates Count
// codes when Count > 1.
type CreateRedemptionInput struct {
	Name        string `json:"name"`
	Quota       int64  `json:"quota"`
	Count       int    `json:"count"`
	ExpiredTime int64  `json:"expired_time,omitempty"`
}
