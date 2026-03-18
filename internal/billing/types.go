package billing

import "encoding/json"

// UserInfo represents user account information from lurus-api V2
type UserInfo struct {
	Quota          int64             `json:"quota"`
	UsedQuota      int64             `json:"used_quota"`
	RemainingQuota int64             `json:"remaining_quota"`
	DailyQuota     int64             `json:"daily_quota"`
	DailyUsed      int64             `json:"daily_used"`
	Group          string            `json:"group"`
	Username       string            `json:"username"`
	DisplayName    string            `json:"display_name"`
	AffCode        string            `json:"aff_code"`
	Subscription   *SubscriptionInfo `json:"subscription,omitempty"`
}

// SubscriptionInfo represents a user's current subscription
type SubscriptionInfo struct {
	ID         int    `json:"id"`
	PlanCode   string `json:"plan_code"`
	PlanName   string `json:"plan_name"`
	Status     string `json:"status"`
	ExpiresAt  string `json:"expires_at"`
	AutoRenew  bool   `json:"auto_renew"`
	DailyQuota int64  `json:"daily_quota"`
	TotalQuota int64  `json:"total_quota"`
}

// SubscriptionPlan represents an available subscription plan
type SubscriptionPlan struct {
	Code       string   `json:"code"`
	Name       string   `json:"name"`
	Currency   string   `json:"currency"`
	Duration   string   `json:"duration"`
	Price      float64  `json:"price"`
	DailyQuota int64    `json:"daily_quota"`
	TotalQuota int64    `json:"total_quota"`
	Features   []string `json:"features"`
}

// TopUpInfo contains available top-up methods and options
type TopUpInfo struct {
	PayMethods    []map[string]string `json:"pay_methods"`
	AmountOptions []int               `json:"amount_options"`
	MinTopup      int                 `json:"min_topup"`
	Discount      float64             `json:"discount"`
}

// PaymentResult represents the result of a payment request
type PaymentResult struct {
	TradeNo    string `json:"trade_no"`
	PaymentURL string `json:"payment_url"`
	Message    string `json:"message"`
}

// QuotaSummary provides a quick overview of user quota for dashboard display
type QuotaSummary struct {
	Quota          int64  `json:"quota"`
	UsedQuota      int64  `json:"used_quota"`
	RemainingQuota int64  `json:"remaining_quota"`
	DailyQuota     int64  `json:"daily_quota"`
	DailyUsed      int64  `json:"daily_used"`
	Username       string `json:"username"`
}

// IdentityOverview is the aggregated account overview from lurus-identity,
// proxied via lurus-api GET /api/v2/user/identity-overview.
type IdentityOverview struct {
	Account struct {
		ID          int64  `json:"id"`
		LurusID     string `json:"lurus_id"`
		DisplayName string `json:"display_name"`
		AvatarURL   string `json:"avatar_url"`
	} `json:"account"`
	VIP struct {
		Level          int16  `json:"level"`
		LevelName      string `json:"level_name"`
		LevelEN        string `json:"level_en"`
		Points         int64  `json:"points"`
		LevelExpiresAt string `json:"level_expires_at,omitempty"`
	} `json:"vip"`
	Wallet struct {
		Balance float64 `json:"balance"`
		Frozen  float64 `json:"frozen"`
	} `json:"wallet"`
	Subscription *struct {
		ProductID string `json:"product_id"`
		PlanCode  string `json:"plan_code"`
		Status    string `json:"status"`
		ExpiresAt string `json:"expires_at,omitempty"`
		AutoRenew bool   `json:"auto_renew"`
	} `json:"subscription"`
	TopupURL string `json:"topup_url"`
}

// AffiliateStats holds referral/affiliate statistics from newapi.
type AffiliateStats struct {
	TotalReferrals int     `json:"total_referrals"`
	TotalEarned    float64 `json:"total_earned"`
	PendingEarned  float64 `json:"pending_earned"`
}

// apiResponse is the generic lurus-api V2 response envelope
type apiResponse struct {
	Success bool            `json:"success"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}
