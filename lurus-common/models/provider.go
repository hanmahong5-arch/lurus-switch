package models

import "time"

// Provider represents an LLM provider configuration
type Provider struct {
	ID        int64     `json:"id" db:"id"`
	Name      string    `json:"name" db:"name"`
	APIURL    string    `json:"api_url" db:"api_url"`
	APIKey    string    `json:"api_key" db:"api_key"`
	Site      string    `json:"site,omitempty" db:"site"`
	Icon      string    `json:"icon,omitempty" db:"icon"`
	Tint      string    `json:"tint,omitempty" db:"tint"`
	Accent    string    `json:"accent,omitempty" db:"accent"`
	Enabled   bool      `json:"enabled" db:"enabled"`
	Platform  string    `json:"platform" db:"platform"` // claude, codex, gemini
	Level     int       `json:"level,omitempty" db:"level"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`

	// Model whitelist - provider natively supported models
	// Use map for O(1) lookup
	SupportedModels map[string]bool `json:"supported_models,omitempty" db:"-"`

	// Model mapping - external model name -> provider internal model name
	// Supports exact match and wildcard (e.g. "claude-*" -> "anthropic/claude-*")
	ModelMapping map[string]string `json:"model_mapping,omitempty" db:"-"`
}

// ProviderHealth represents provider health check status
type ProviderHealth struct {
	ProviderID   int64     `json:"provider_id"`
	ProviderName string    `json:"provider_name"`
	IsHealthy    bool      `json:"is_healthy"`
	Latency      int64     `json:"latency_ms"`
	LastCheck    time.Time `json:"last_check"`
	ErrorMessage string    `json:"error_message,omitempty"`
}

// ProviderStats represents provider usage statistics
type ProviderStats struct {
	ProviderID    int64   `json:"provider_id"`
	ProviderName  string  `json:"provider_name"`
	TotalRequests int64   `json:"total_requests"`
	SuccessRate   float64 `json:"success_rate"`
	AvgLatency    float64 `json:"avg_latency_ms"`
	TotalCost     float64 `json:"total_cost"`
	TotalTokens   int64   `json:"total_tokens"`
}

// Platform constants for LLM platforms
const (
	PlatformClaude = "claude"
	PlatformCodex  = "codex"
	PlatformGemini = "gemini"
)
