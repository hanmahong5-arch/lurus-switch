package models

import "time"

// LLMRequestEvent represents an LLM request event published to NATS
type LLMRequestEvent struct {
	TraceID   string    `json:"trace_id"`
	RequestID string    `json:"request_id"`
	UserID    string    `json:"user_id"`
	Platform  string    `json:"platform"` // claude, codex, gemini
	Model     string    `json:"model"`
	Provider  string    `json:"provider"`
	IsStream  bool      `json:"is_stream"`
	Timestamp time.Time `json:"timestamp"`
}

// LLMResponseEvent represents an LLM response event published to NATS
type LLMResponseEvent struct {
	TraceID       string    `json:"trace_id"`
	RequestID     string    `json:"request_id"`
	UserID        string    `json:"user_id"`
	Platform      string    `json:"platform"`
	Model         string    `json:"model"`
	Provider      string    `json:"provider"`
	HTTPCode      int       `json:"http_code"`
	InputTokens   int       `json:"input_tokens"`
	OutputTokens  int       `json:"output_tokens"`
	TotalCost     float64   `json:"total_cost"`
	DurationMs    int64     `json:"duration_ms"`
	FinishReason  string    `json:"finish_reason,omitempty"`
	ErrorType     string    `json:"error_type,omitempty"`
	ErrorMessage  string    `json:"error_message,omitempty"`
	Timestamp     time.Time `json:"timestamp"`
}

// LogWriteEvent represents a log write event for async persistence
type LogWriteEvent struct {
	Log       *RequestLog `json:"log"`
	Timestamp time.Time   `json:"timestamp"`
}

// BillingUsageEvent represents a usage event for billing
type BillingUsageEvent struct {
	UserID       string    `json:"user_id"`
	TraceID      string    `json:"trace_id"`
	Platform     string    `json:"platform"`
	Model        string    `json:"model"`
	InputTokens  int       `json:"input_tokens"`
	OutputTokens int       `json:"output_tokens"`
	TotalCost    float64   `json:"total_cost"`
	Timestamp    time.Time `json:"timestamp"`
}

// QuotaChangeEvent represents a quota change notification
type QuotaChangeEvent struct {
	UserID      string    `json:"user_id"`
	QuotaTotal  float64   `json:"quota_total"`
	QuotaUsed   float64   `json:"quota_used"`
	QuotaRemain float64   `json:"quota_remain"`
	Reason      string    `json:"reason"` // usage, recharge, admin_adjust
	Timestamp   time.Time `json:"timestamp"`
}

// SyncMessageEvent represents a message sync event
type SyncMessageEvent struct {
	SessionID string    `json:"session_id"`
	UserID    string    `json:"user_id"`
	Message   *Message  `json:"message"`
	Action    string    `json:"action"` // create, update, delete
	Timestamp time.Time `json:"timestamp"`
}

// SyncSessionEvent represents a session sync event
type SyncSessionEvent struct {
	UserID    string    `json:"user_id"`
	Session   *Session  `json:"session"`
	Action    string    `json:"action"` // create, update, delete, archive
	Timestamp time.Time `json:"timestamp"`
}

// AlertEvent represents an alert triggered by monitoring
type AlertEvent struct {
	ID          string                 `json:"id"`
	RuleID      string                 `json:"rule_id"`
	RuleName    string                 `json:"rule_name"`
	Severity    string                 `json:"severity"` // info, warning, critical
	Metric      string                 `json:"metric"`
	Value       float64                `json:"value"`
	Threshold   float64                `json:"threshold"`
	Message     string                 `json:"message"`
	Labels      map[string]string      `json:"labels,omitempty"`
	Annotations map[string]string      `json:"annotations,omitempty"`
	Timestamp   time.Time              `json:"timestamp"`
}

// AlertSeverity constants
const (
	AlertSeverityInfo     = "info"
	AlertSeverityWarning  = "warning"
	AlertSeverityCritical = "critical"
)

// EventAction constants
const (
	ActionCreate  = "create"
	ActionUpdate  = "update"
	ActionDelete  = "delete"
	ActionArchive = "archive"
)

// QuotaChangeReason constants
const (
	QuotaReasonUsage       = "usage"
	QuotaReasonRecharge    = "recharge"
	QuotaReasonAdminAdjust = "admin_adjust"
	QuotaReasonRefund      = "refund"
)
