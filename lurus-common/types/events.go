package types

import "time"

// EventType defines the type of event.
type EventType string

const (
	// User events
	EventUserCreated          EventType = "user.created"
	EventUserUpdated          EventType = "user.updated"
	EventUserDeleted          EventType = "user.deleted"
	EventUserQuotaChanged     EventType = "user.quota.changed"
	EventUserGroupChanged     EventType = "user.group.changed"
	EventUserDailyQuotaReset  EventType = "user.daily_quota.reset"
	EventUserDailyQuotaExhausted EventType = "user.daily_quota.exhausted"
	
	// Subscription events
	EventSubscriptionCreated  EventType = "subscription.created"
	EventSubscriptionRenewed  EventType = "subscription.renewed"
	EventSubscriptionCancelled EventType = "subscription.cancelled"
	EventSubscriptionExpired  EventType = "subscription.expired"
	
	// Billing events
	EventPaymentSucceeded     EventType = "payment.succeeded"
	EventPaymentFailed        EventType = "payment.failed"
	EventUsageRecorded        EventType = "usage.recorded"
	
	// LLM request events
	EventLLMRequestStarted    EventType = "llm.request.started"
	EventLLMRequestCompleted  EventType = "llm.request.completed"
	EventLLMRequestFailed     EventType = "llm.request.failed"
)

// BaseEvent contains common fields for all events.
type BaseEvent struct {
	ID        string    `json:"id"`         // Unique event ID
	Type      EventType `json:"type"`       // Event type
	Source    string    `json:"source"`     // Service that generated the event
	Timestamp time.Time `json:"timestamp"`  // When the event occurred
	TraceID   string    `json:"trace_id,omitempty"` // For distributed tracing
}

// UserEvent represents a user-related event.
type UserEvent struct {
	BaseEvent
	UserID    int             `json:"user_id"`
	Data      *UserEventData  `json:"data,omitempty"`
}

// UserEventData contains user event payload.
type UserEventData struct {
	OldQuota      int64  `json:"old_quota,omitempty"`
	NewQuota      int64  `json:"new_quota,omitempty"`
	OldGroup      string `json:"old_group,omitempty"`
	NewGroup      string `json:"new_group,omitempty"`
	DailyQuota    int64  `json:"daily_quota,omitempty"`
	DailyUsed     int64  `json:"daily_used,omitempty"`
	BaseGroup     string `json:"base_group,omitempty"`
	FallbackGroup string `json:"fallback_group,omitempty"`
}

// SubscriptionEvent represents a subscription-related event.
type SubscriptionEvent struct {
	BaseEvent
	UserID         int                `json:"user_id"`
	SubscriptionID int64              `json:"subscription_id"`
	PlanCode       string             `json:"plan_code"`
	Status         SubscriptionStatus `json:"status"`
	ExpiresAt      time.Time          `json:"expires_at,omitempty"`
}

// UsageEvent represents a usage recording event.
type UsageEvent struct {
	BaseEvent
	UserID       int     `json:"user_id"`
	TokenID      int     `json:"token_id,omitempty"`
	Platform     string  `json:"platform"`     // claude, codex, gemini
	Provider     string  `json:"provider"`     // Provider name
	Model        string  `json:"model"`        // Model name
	InputTokens  int     `json:"input_tokens"`
	OutputTokens int     `json:"output_tokens"`
	TotalCost    float64 `json:"total_cost"`   // Cost in USD
	DurationMs   int64   `json:"duration_ms"`  // Request duration
}

// LLMRequestEvent represents an LLM request event.
type LLMRequestEvent struct {
	BaseEvent
	UserID       int             `json:"user_id"`
	TokenID      int             `json:"token_id,omitempty"`
	Platform     string          `json:"platform"`
	Provider     string          `json:"provider"`
	Model        string          `json:"model"`
	IsStream     bool            `json:"is_stream"`
	Request      *LLMRequestData `json:"request,omitempty"`
	Response     *LLMResponseData `json:"response,omitempty"`
	Error        string          `json:"error,omitempty"`
}

// LLMRequestData contains LLM request metadata.
type LLMRequestData struct {
	InputTokens  int    `json:"input_tokens"`
	MaxTokens    int    `json:"max_tokens,omitempty"`
	Temperature  float32 `json:"temperature,omitempty"`
}

// LLMResponseData contains LLM response metadata.
type LLMResponseData struct {
	OutputTokens    int     `json:"output_tokens"`
	TotalTokens     int     `json:"total_tokens"`
	Cost            float64 `json:"cost"`
	DurationMs      int64   `json:"duration_ms"`
	FirstTokenMs    int64   `json:"first_token_ms,omitempty"`
	StopReason      string  `json:"stop_reason,omitempty"`
}
