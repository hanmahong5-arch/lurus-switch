package models

import "time"

// Session represents a chat session
type Session struct {
	ID            string    `json:"id" db:"id"`
	UserID        string    `json:"user_id" db:"user_id"`
	Title         string    `json:"title" db:"title"`
	Summary       string    `json:"summary,omitempty" db:"summary"`
	Model         string    `json:"model,omitempty" db:"model"`
	Provider      string    `json:"provider,omitempty" db:"provider"`
	MessageCount  int       `json:"message_count" db:"message_count"`
	TokenCount    int       `json:"token_count" db:"token_count"`
	Cost          float64   `json:"cost" db:"cost"`
	IsPinned      bool      `json:"is_pinned" db:"is_pinned"`
	IsArchived    bool      `json:"is_archived" db:"is_archived"`
	LastMessageAt time.Time `json:"last_message_at,omitempty" db:"last_message_at"`
	CreatedAt     time.Time `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time `json:"updated_at" db:"updated_at"`
}

// Message represents a chat message
type Message struct {
	ID              string                 `json:"id" db:"id"`
	SessionID       string                 `json:"session_id" db:"session_id"`
	UserID          string                 `json:"user_id" db:"user_id"`
	Role            string                 `json:"role" db:"role"` // user, assistant, system
	Content         string                 `json:"content" db:"content"`
	ContentType     string                 `json:"content_type" db:"content_type"` // text, markdown, code
	Model           string                 `json:"model,omitempty" db:"model"`
	Provider        string                 `json:"provider,omitempty" db:"provider"`
	TokensInput     int                    `json:"tokens_input,omitempty" db:"tokens_input"`
	TokensOutput    int                    `json:"tokens_output,omitempty" db:"tokens_output"`
	TokensReasoning int                    `json:"tokens_reasoning,omitempty" db:"tokens_reasoning"`
	Cost            float64                `json:"cost,omitempty" db:"cost"`
	DurationMs      int                    `json:"duration_ms,omitempty" db:"duration_ms"`
	FinishReason    string                 `json:"finish_reason,omitempty" db:"finish_reason"`
	Metadata        map[string]interface{} `json:"metadata,omitempty" db:"-"`
	CreatedAt       time.Time              `json:"created_at" db:"created_at"`
}

// MessageRole constants
const (
	RoleUser      = "user"
	RoleAssistant = "assistant"
	RoleSystem    = "system"
)

// ContentType constants
const (
	ContentTypeText     = "text"
	ContentTypeMarkdown = "markdown"
	ContentTypeCode     = "code"
)

// SessionStatusType represents session processing status
type SessionStatusType string

const (
	SessionIdle      SessionStatusType = "idle"
	SessionThinking  SessionStatusType = "thinking"
	SessionStreaming SessionStatusType = "streaming"
	SessionError     SessionStatusType = "error"
	SessionCompleted SessionStatusType = "completed"
)

// SessionStatusEvent represents a session status change event
type SessionStatusEvent struct {
	SessionID string            `json:"session_id"`
	UserID    string            `json:"user_id"`
	Status    SessionStatusType `json:"status"`
	Model     string            `json:"model,omitempty"`
	Progress  int               `json:"progress,omitempty"` // 0-100
	Timestamp time.Time         `json:"timestamp"`
}

// TypingEvent represents a typing indicator event
type TypingEvent struct {
	SessionID string    `json:"session_id"`
	UserID    string    `json:"user_id"`
	DeviceID  string    `json:"device_id"`
	IsTyping  bool      `json:"is_typing"`
	Timestamp time.Time `json:"timestamp"`
}
