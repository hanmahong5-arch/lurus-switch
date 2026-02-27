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

// apiResponse is the generic lurus-api V2 response envelope
type apiResponse struct {
	Success bool            `json:"success"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}
