package models

import "time"

// User represents a user in the system
type User struct {
	ID           string    `json:"id" db:"id"`
	NewAPIUserID string    `json:"newapi_user_id,omitempty" db:"newapi_user_id"`
	Username     string    `json:"username" db:"username"`
	Email        string    `json:"email,omitempty" db:"email"`
	AvatarURL    string    `json:"avatar_url,omitempty" db:"avatar_url"`
	Plan         string    `json:"plan" db:"plan"`
	QuotaTotal   float64   `json:"quota_total" db:"quota_total"`
	QuotaUsed    float64   `json:"quota_used" db:"quota_used"`
	IsAdmin      bool      `json:"is_admin" db:"is_admin"`
	IsDisabled   bool      `json:"is_disabled" db:"is_disabled"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time `json:"updated_at" db:"updated_at"`
}

// UserQuota represents user's quota information
type UserQuota struct {
	UserID       string    `json:"user_id"`
	QuotaTotal   float64   `json:"quota_total"`
	QuotaUsed    float64   `json:"quota_used"`
	QuotaRemain  float64   `json:"quota_remain"`
	DailyLimit   float64   `json:"daily_limit,omitempty"`
	DailyUsed    float64   `json:"daily_used,omitempty"`
	ResetAt      time.Time `json:"reset_at,omitempty"`
	LastUpdated  time.Time `json:"last_updated"`
}

// UserPlan constants
const (
	PlanFree       = "free"
	PlanBasic      = "basic"
	PlanPro        = "pro"
	PlanEnterprise = "enterprise"
)

// Device represents a user's device
type Device struct {
	ID            string    `json:"id" db:"id"`
	UserID        string    `json:"user_id" db:"user_id"`
	DeviceID      string    `json:"device_id" db:"device_id"`
	DeviceName    string    `json:"device_name" db:"device_name"`
	DeviceType    string    `json:"device_type" db:"device_type"` // desktop, mobile, cli, web
	ClientVersion string    `json:"client_version" db:"client_version"`
	LastSeenAt    time.Time `json:"last_seen_at" db:"last_seen_at"`
	LastIP        string    `json:"last_ip" db:"last_ip"`
	CreatedAt     time.Time `json:"created_at" db:"created_at"`
}

// DeviceType constants
const (
	DeviceTypeDesktop = "desktop"
	DeviceTypeMobile  = "mobile"
	DeviceTypeCLI     = "cli"
	DeviceTypeWeb     = "web"
)

// PresenceStatus represents user online status
type PresenceStatus string

const (
	PresenceOnline  PresenceStatus = "online"
	PresenceOffline PresenceStatus = "offline"
	PresenceAway    PresenceStatus = "away"
)

// Presence represents user's online presence
type Presence struct {
	UserID        string         `json:"user_id"`
	DeviceID      string         `json:"device_id"`
	DeviceType    string         `json:"device_type"`
	Status        PresenceStatus `json:"status"`
	ClientVersion string         `json:"client_version"`
	LastSeenAt    time.Time      `json:"last_seen_at"`
}
