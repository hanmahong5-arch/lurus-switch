package agent

import (
	"encoding/json"
	"time"
)

// ToolType represents the type of AI CLI tool an agent wraps.
type ToolType string

const (
	ToolClaude   ToolType = "claude"
	ToolCodex    ToolType = "codex"
	ToolGemini   ToolType = "gemini"
	ToolOpenClaw ToolType = "openclaw"
	ToolZeroClaw ToolType = "zeroclaw"
	ToolPicoClaw ToolType = "picoclaw"
	ToolNullClaw ToolType = "nullclaw"
)

// ValidToolTypes lists all recognized tool types.
var ValidToolTypes = []ToolType{
	ToolClaude, ToolCodex, ToolGemini,
	ToolOpenClaw, ToolZeroClaw, ToolPicoClaw, ToolNullClaw,
}

// IsValid returns true if the tool type is recognized.
func (t ToolType) IsValid() bool {
	for _, v := range ValidToolTypes {
		if t == v {
			return true
		}
	}
	return false
}

// Status represents the lifecycle state of an agent.
type Status string

const (
	StatusCreated Status = "created"
	StatusRunning Status = "running"
	StatusStopped Status = "stopped"
	StatusError   Status = "error"
)

// BudgetPeriod defines the time window for budget enforcement.
type BudgetPeriod string

const (
	BudgetDaily   BudgetPeriod = "daily"
	BudgetMonthly BudgetPeriod = "monthly"
	BudgetTotal   BudgetPeriod = "total"
)

// BudgetPolicy defines what happens when an agent exceeds its budget.
type BudgetPolicy string

const (
	PolicyPause      BudgetPolicy = "pause"
	PolicyDegrade    BudgetPolicy = "degrade"
	PolicyNotifyOnly BudgetPolicy = "notify_only"
)

// Permissions defines what an agent is allowed to do.
type Permissions struct {
	AllowShell   bool `json:"allowShell"`
	AllowFiles   bool `json:"allowFiles"`
	AllowNetwork bool `json:"allowNetwork"`
}

// Profile is the core domain entity representing a managed AI assistant instance.
type Profile struct {
	ID           string       `json:"id"`
	Name         string       `json:"name"`
	Icon         string       `json:"icon"`
	Tags         []string     `json:"tags"`
	ToolType     ToolType     `json:"toolType"`
	ModelID      string       `json:"modelId"`
	SystemPrompt string       `json:"systemPrompt"`
	MCPServers   []string     `json:"mcpServers"`
	Permissions  Permissions  `json:"permissions"`
	ProjectID    string       `json:"projectId,omitempty"`
	Status       Status       `json:"status"`
	ConfigDir    string       `json:"configDir,omitempty"`
	CreatedAt    time.Time    `json:"createdAt"`
	UpdatedAt    time.Time    `json:"updatedAt"`

	// Budget fields (zero value = unlimited)
	BudgetLimitTokens   *int64       `json:"budgetLimitTokens,omitempty"`
	BudgetLimitCurrency *float64     `json:"budgetLimitCurrency,omitempty"`
	BudgetPeriod        BudgetPeriod `json:"budgetPeriod"`
	BudgetPolicy        BudgetPolicy `json:"budgetPolicy"`
}

// CreateParams holds the parameters for creating a new agent.
type CreateParams struct {
	Name         string       `json:"name"`
	Icon         string       `json:"icon"`
	Tags         []string     `json:"tags"`
	ToolType     ToolType     `json:"toolType"`
	ModelID      string       `json:"modelId"`
	SystemPrompt string       `json:"systemPrompt"`
	MCPServers   []string     `json:"mcpServers"`
	Permissions  Permissions  `json:"permissions"`
	ProjectID    string       `json:"projectId"`

	BudgetLimitTokens   *int64       `json:"budgetLimitTokens"`
	BudgetLimitCurrency *float64     `json:"budgetLimitCurrency"`
	BudgetPeriod        BudgetPeriod `json:"budgetPeriod"`
	BudgetPolicy        BudgetPolicy `json:"budgetPolicy"`
}

// UpdateParams holds the fields that can be updated on an existing agent.
// Only non-nil/non-zero fields are applied.
type UpdateParams struct {
	Name         *string      `json:"name,omitempty"`
	Icon         *string      `json:"icon,omitempty"`
	Tags         []string     `json:"tags,omitempty"`
	ModelID      *string      `json:"modelId,omitempty"`
	SystemPrompt *string      `json:"systemPrompt,omitempty"`
	MCPServers   []string     `json:"mcpServers,omitempty"`
	Permissions  *Permissions `json:"permissions,omitempty"`
	ProjectID    *string      `json:"projectId,omitempty"`
	Status       *Status      `json:"status,omitempty"`

	BudgetLimitTokens   *int64       `json:"budgetLimitTokens,omitempty"`
	BudgetLimitCurrency *float64     `json:"budgetLimitCurrency,omitempty"`
	BudgetPeriod        *BudgetPeriod `json:"budgetPeriod,omitempty"`
	BudgetPolicy        *BudgetPolicy `json:"budgetPolicy,omitempty"`
}

// ListFilter defines optional filters for listing agents.
type ListFilter struct {
	Status    *Status   `json:"status,omitempty"`
	ToolType  *ToolType `json:"toolType,omitempty"`
	ProjectID *string   `json:"projectId,omitempty"`
	Tag       *string   `json:"tag,omitempty"`
}

// marshalJSON marshals a value to JSON string (for SQLite TEXT columns).
func marshalJSON(v any) string {
	b, _ := json.Marshal(v)
	return string(b)
}

// unmarshalJSON unmarshals a JSON string from a SQLite TEXT column.
func unmarshalJSON(s string, v any) {
	if s != "" {
		json.Unmarshal([]byte(s), v)
	}
}
