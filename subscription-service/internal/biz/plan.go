package biz

import (
	"time"
)

// PlanType defines subscription plan types
type PlanType string

const (
	PlanTypeMonthly PlanType = "monthly"
	PlanTypeYearly  PlanType = "yearly"
)

// Plan represents a subscription plan definition
type Plan struct {
	ID          int64     `json:"id" gorm:"primaryKey"`
	Name        string    `json:"name" gorm:"size:50;not null"`                 // Display name: "Basic Monthly"
	Code        string    `json:"code" gorm:"size:20;uniqueIndex;not null"`     // Internal code: "basic_monthly"
	Type        PlanType  `json:"type" gorm:"size:20;not null"`                 // monthly / yearly
	Quota       int64     `json:"quota" gorm:"not null"`                        // Total quota per period (for reference)
	DailyQuota  int64     `json:"daily_quota" gorm:"not null"`                  // Daily quota limit (tokens or cents)
	PriceCents  int       `json:"price_cents" gorm:"not null"`                  // Price in cents
	Currency    string    `json:"currency" gorm:"size:3;default:'CNY'"`         // CNY / USD
	GroupName   string    `json:"group_name" gorm:"size:50"`                    // new-api user group (premium models)
	FallbackGroup string  `json:"fallback_group" gorm:"size:50;default:'free'"` // Fallback group when quota exhausted
	Features    string    `json:"features" gorm:"type:text"`                    // JSON features list
	Description string    `json:"description" gorm:"type:text"`                 // Plan description
	SortOrder   int       `json:"sort_order" gorm:"default:0"`                  // Display order
	Status      int       `json:"status" gorm:"default:1"`                      // 1=active, 0=disabled
	CreatedAt   time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt   time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}

func (Plan) TableName() string {
	return "plans"
}

// PlanFeatures represents plan feature flags
type PlanFeatures struct {
	MaxConcurrentRequests int      `json:"max_concurrent_requests"`
	AllowedModels         []string `json:"allowed_models"`
	PrioritySupport       bool     `json:"priority_support"`
	APIRateLimit          int      `json:"api_rate_limit"` // requests per minute
	CustomPrompts         bool     `json:"custom_prompts"`
}

// DefaultPlans returns predefined subscription plans
// DailyQuota is in cents (1 USD = 100 cents), representing daily spending limit
func DefaultPlans() []Plan {
	return []Plan{
		{
			Name:          "Free",
			Code:          "free",
			Type:          PlanTypeMonthly,
			Quota:         0,             // No paid quota
			DailyQuota:    0,             // No daily limit (only free models)
			PriceCents:    0,             // Free
			Currency:      "CNY",
			GroupName:     "free",        // Free model pool only
			FallbackGroup: "free",        // Already on free
			Description:   "Free tier with access to free models only",
			SortOrder:     0,
			Status:        1,
		},
		{
			Name:          "Basic Monthly",
			Code:          "basic_monthly",
			Type:          PlanTypeMonthly,
			Quota:         2900,          // ¥29 total per month
			DailyQuota:    100,           // ¥1/day (~$0.14)
			PriceCents:    2900,          // ¥29/month
			Currency:      "CNY",
			GroupName:     "basic",       // Access to basic paid models
			FallbackGroup: "free",        // Fallback to free models
			Description:   "¥1/day quota, fallback to free models when exhausted",
			SortOrder:     1,
			Status:        1,
		},
		{
			Name:          "Pro Monthly",
			Code:          "pro_monthly",
			Type:          PlanTypeMonthly,
			Quota:         9900,          // ¥99 total per month
			DailyQuota:    330,           // ¥3.3/day
			PriceCents:    9900,          // ¥99/month
			Currency:      "CNY",
			GroupName:     "pro",         // Access to pro models (GPT-4, Claude)
			FallbackGroup: "basic",       // Fallback to basic models
			Description:   "¥3.3/day quota, fallback to basic models when exhausted",
			SortOrder:     2,
			Status:        1,
		},
		{
			Name:          "Team Monthly",
			Code:          "team_monthly",
			Type:          PlanTypeMonthly,
			Quota:         29900,         // ¥299 total per month
			DailyQuota:    1000,          // ¥10/day
			PriceCents:    29900,         // ¥299/month
			Currency:      "CNY",
			GroupName:     "team",        // Access to all models
			FallbackGroup: "pro",         // Fallback to pro models
			Description:   "¥10/day quota, fallback to pro models when exhausted",
			SortOrder:     3,
			Status:        1,
		},
		{
			Name:          "Enterprise Monthly",
			Code:          "enterprise_monthly",
			Type:          PlanTypeMonthly,
			Quota:         99900,         // ¥999 total per month
			DailyQuota:    5000,          // ¥50/day
			PriceCents:    99900,         // ¥999/month
			Currency:      "CNY",
			GroupName:     "enterprise",  // Priority access to all models
			FallbackGroup: "team",        // Fallback to team models
			Description:   "¥50/day quota, priority support, fallback to team models",
			SortOrder:     4,
			Status:        1,
		},
		{
			Name:          "Basic Yearly",
			Code:          "basic_yearly",
			Type:          PlanTypeYearly,
			Quota:         29000,         // ¥290/year (2 months free)
			DailyQuota:    100,           // ¥1/day
			PriceCents:    29000,         // ¥290/year
			Currency:      "CNY",
			GroupName:     "basic",
			FallbackGroup: "free",
			Description:   "Basic yearly plan, 2 months free",
			SortOrder:     10,
			Status:        1,
		},
		{
			Name:          "Pro Yearly",
			Code:          "pro_yearly",
			Type:          PlanTypeYearly,
			Quota:         99000,         // ¥990/year (2 months free)
			DailyQuota:    330,           // ¥3.3/day
			PriceCents:    99000,         // ¥990/year
			Currency:      "CNY",
			GroupName:     "pro",
			FallbackGroup: "basic",
			Description:   "Pro yearly plan, 2 months free",
			SortOrder:     11,
			Status:        1,
		},
	}
}
