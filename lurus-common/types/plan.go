package types

import "time"

// PlanType defines the subscription plan type.
type PlanType string

const (
	PlanTypeFree    PlanType = "free"
	PlanTypeMonthly PlanType = "monthly"
	PlanTypeYearly  PlanType = "yearly"
)

// Plan represents a subscription plan.
type Plan struct {
	ID            int64    `json:"id"`
	Code          string   `json:"code"`           // Unique plan code (e.g., "pro_monthly")
	Name          string   `json:"name"`           // Display name
	Description   string   `json:"description"`
	Type          PlanType `json:"type"`           // monthly, yearly, free
	Price         int64    `json:"price"`          // Price in cents
	Currency      string   `json:"currency"`       // USD, EUR, etc.
	
	// Quota settings
	Quota         int64    `json:"quota"`          // Total quota per billing period
	DailyQuota    int64    `json:"daily_quota"`    // Daily quota limit (0 = unlimited)
	
	// Group settings
	GroupName     string   `json:"group_name"`     // new-api group to assign
	FallbackGroup string   `json:"fallback_group"` // Group to use when daily quota exhausted
	
	// Metadata
	Features      []string `json:"features,omitempty"`
	SortOrder     int      `json:"sort_order"`
	Status        int      `json:"status"`         // 1=active, 0=inactive
	
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// SubscriptionStatus defines the subscription status.
type SubscriptionStatus string

const (
	SubscriptionStatusActive    SubscriptionStatus = "active"
	SubscriptionStatusExpired   SubscriptionStatus = "expired"
	SubscriptionStatusCancelled SubscriptionStatus = "cancelled"
	SubscriptionStatusPending   SubscriptionStatus = "pending"
)

// Subscription represents a user's subscription.
type Subscription struct {
	ID           int64              `json:"id"`
	UserID       int                `json:"user_id"`
	PlanID       int64              `json:"plan_id"`
	PlanCode     string             `json:"plan_code,omitempty"`
	PlanName     string             `json:"plan_name,omitempty"`
	Status       SubscriptionStatus `json:"status"`
	
	// Billing period
	StartedAt    time.Time          `json:"started_at"`
	ExpiresAt    time.Time          `json:"expires_at"`
	AutoRenew    bool               `json:"auto_renew"`
	
	// Total quota (subscription-level tracking)
	CurrentQuota int64              `json:"current_quota"`
	UsedQuota    int64              `json:"used_quota"`
	LastResetAt  time.Time          `json:"last_reset_at"`
	
	// Daily quota config (state is in new-api)
	DailyQuota   int64              `json:"daily_quota"`
	
	// Cancellation
	CancelledAt  *time.Time         `json:"cancelled_at,omitempty"`
	CancelReason string             `json:"cancel_reason,omitempty"`
	
	// External reference
	ExternalID   string             `json:"external_id,omitempty"`
	
	CreatedAt    time.Time          `json:"created_at"`
	UpdatedAt    time.Time          `json:"updated_at"`
}

// QuotaStatus represents the current quota status for a user (combined view).
type QuotaStatus struct {
	UserID           int       `json:"user_id"`
	PlanCode         string    `json:"plan_code"`
	PlanName         string    `json:"plan_name"`
	HasQuota         bool      `json:"has_quota"`
	CurrentGroup     string    `json:"current_group"`
	FallbackGroup    string    `json:"fallback_group,omitempty"`
	DailyQuota       int64     `json:"daily_quota"`
	DailyUsed        int64     `json:"daily_used"`
	DailyRemaining   int64     `json:"daily_remaining"`
	TotalQuota       int64     `json:"total_quota"`
	TotalUsed        int64     `json:"total_used"`
	ExpiresAt        time.Time `json:"expires_at"`
	IsFallback       bool      `json:"is_fallback"`
	LastDailyResetAt time.Time `json:"last_daily_reset_at"`
}
