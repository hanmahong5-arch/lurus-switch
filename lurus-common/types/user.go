// Package types provides common data types shared across Lurus services.
package types

import "time"

// UserQuotaStatus represents the current quota status for a user.
// This is the canonical type used across all Lurus services.
type UserQuotaStatus struct {
	UserID          int    `json:"user_id"`
	Username        string `json:"username,omitempty"`
	Email           string `json:"email,omitempty"`
	
	// Total quota (from subscription)
	Quota     int64 `json:"quota"`
	UsedQuota int64 `json:"used_quota"`
	
	// Current effective group
	Group string `json:"group"`
	
	// Daily quota management (managed by new-api)
	DailyQuota     int64 `json:"daily_quota"`
	DailyUsed      int64 `json:"daily_used"`
	DailyRemaining int64 `json:"daily_remaining"`
	LastDailyReset int64 `json:"last_daily_reset"` // Unix timestamp
	NeedsReset     bool  `json:"needs_reset"`
	
	// Group management
	BaseGroup       string `json:"base_group"`
	FallbackGroup   string `json:"fallback_group"`
	IsUsingFallback bool   `json:"is_using_fallback"`
	
	// Status
	Status int `json:"status"` // 1=enabled, 2=disabled
}

// SubscriptionConfig represents the configuration to sync from subscription-service to new-api.
// This is the unified API for subscription state management.
type SubscriptionConfig struct {
	DailyQuota    int64  `json:"daily_quota"`
	BaseGroup     string `json:"base_group"`
	FallbackGroup string `json:"fallback_group"`
	Quota         int64  `json:"quota,omitempty"` // Optional: update total quota
}

// DailyQuotaInfo represents the daily quota information for a user.
type DailyQuotaInfo struct {
	UserID          int    `json:"user_id"`
	DailyQuota      int64  `json:"daily_quota"`
	DailyUsed       int64  `json:"daily_used"`
	DailyRemaining  int64  `json:"daily_remaining"`
	LastDailyReset  int64  `json:"last_daily_reset"`
	NeedsReset      bool   `json:"needs_reset"`
	CurrentGroup    string `json:"current_group"`
	BaseGroup       string `json:"base_group"`
	FallbackGroup   string `json:"fallback_group"`
	IsUsingFallback bool   `json:"is_using_fallback"`
}

// User represents a user from new-api.
type User struct {
	ID        int    `json:"id"`
	Username  string `json:"username"`
	Email     string `json:"email"`
	Status    int    `json:"status"`
	Role      int    `json:"role"`
	Group     string `json:"group"`
	
	// Quota
	Quota     int64 `json:"quota"`
	UsedQuota int64 `json:"used_quota"`
	
	// Daily quota (new fields)
	DailyQuota     int64 `json:"daily_quota"`
	DailyUsed      int64 `json:"daily_used"`
	LastDailyReset int64 `json:"last_daily_reset"`
	BaseGroup      string `json:"base_group"`
	FallbackGroup  string `json:"fallback_group"`
	
	// Timestamps
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Token represents an API token from new-api.
type Token struct {
	ID          int    `json:"id"`
	Key         string `json:"key"`
	Name        string `json:"name"`
	UserID      int    `json:"user_id"`
	RemainQuota int64  `json:"remain_quota"`
	UsedQuota   int64  `json:"used_quota"`
	ExpiredTime int64  `json:"expired_time"`
	Status      int    `json:"status"`
}
