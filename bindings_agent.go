package main

import (
	"fmt"
	"time"

	"lurus-switch/internal/agent"
	"lurus-switch/internal/analytics"
)

// --- Agent CRUD ---

// CreateAgent creates a new agent profile from the given parameters.
func (a *App) CreateAgent(params agent.CreateParams) (*agent.Profile, error) {
	if a.agentStore == nil {
		return nil, fmt.Errorf("agent store not initialized")
	}
	p, err := a.agentStore.Create(params)
	if err != nil {
		return nil, err
	}
	// Track analytics event.
	a.trackAgentEvent(string(params.ToolType), "create")
	return p, nil
}

// ListAgents returns all agents, optionally filtered.
func (a *App) ListAgents(filter *agent.ListFilter) ([]*agent.Profile, error) {
	if a.agentStore == nil {
		return nil, fmt.Errorf("agent store not initialized")
	}
	return a.agentStore.List(filter)
}

// GetAgent returns a single agent by ID.
func (a *App) GetAgent(id string) (*agent.Profile, error) {
	if a.agentStore == nil {
		return nil, fmt.Errorf("agent store not initialized")
	}
	return a.agentStore.Get(id)
}

// UpdateAgent updates an agent's fields.
func (a *App) UpdateAgent(id string, params agent.UpdateParams) (*agent.Profile, error) {
	if a.agentStore == nil {
		return nil, fmt.Errorf("agent store not initialized")
	}
	return a.agentStore.Update(id, params)
}

// DeleteAgent removes an agent. Stops it first if running.
func (a *App) DeleteAgent(id string) error {
	if a.agentStore == nil {
		return fmt.Errorf("agent store not initialized")
	}

	// Stop if running.
	if a.agentInstMgr != nil {
		if inst := a.agentInstMgr.GetInstance(id); inst != nil {
			a.agentInstMgr.Stop(id)
		}
	}

	// Clean up config directory.
	if a.agentConfigMgr != nil {
		a.agentConfigMgr.Remove(id)
	}

	return a.agentStore.Delete(id)
}

// --- Agent Lifecycle ---

// LaunchAgent starts an agent's tool process.
func (a *App) LaunchAgent(id string) error {
	if a.agentInstMgr == nil {
		return fmt.Errorf("agent instance manager not initialized")
	}
	err := a.agentInstMgr.Launch(a.ctx, id)
	if err != nil {
		return err
	}
	a.trackAgentEvent("", "launch")
	return nil
}

// StopAgent gracefully stops an agent's tool process.
func (a *App) StopAgent(id string) error {
	if a.agentInstMgr == nil {
		return fmt.Errorf("agent instance manager not initialized")
	}
	return a.agentInstMgr.Stop(id)
}

// GetAgentOutput returns recent output lines from an agent's process.
func (a *App) GetAgentOutput(id string, maxLines int) ([]string, error) {
	if a.agentInstMgr == nil {
		return nil, fmt.Errorf("agent instance manager not initialized")
	}
	return a.agentInstMgr.GetOutput(id, maxLines)
}

// --- Agent Stats ---

// AgentStats contains summary counts for the dashboard.
type AgentStats struct {
	Total   int `json:"total"`
	Running int `json:"running"`
	Stopped int `json:"stopped"`
	Error   int `json:"error"`
	Created int `json:"created"`
}

// GetAgentStats returns agent count by status.
func (a *App) GetAgentStats() (*AgentStats, error) {
	if a.agentStore == nil {
		return &AgentStats{}, nil
	}
	counts, err := a.agentStore.CountByStatus()
	if err != nil {
		return nil, err
	}
	total, _ := a.agentStore.Count()
	return &AgentStats{
		Total:   total,
		Running: counts[agent.StatusRunning],
		Stopped: counts[agent.StatusStopped],
		Error:   counts[agent.StatusError],
		Created: counts[agent.StatusCreated],
	}, nil
}

// --- Agent Clone ---

// CloneAgent creates a copy of an agent with a new name.
func (a *App) CloneAgent(sourceID, newName string) (*agent.Profile, error) {
	if a.agentStore == nil {
		return nil, fmt.Errorf("agent store not initialized")
	}

	src, err := a.agentStore.Get(sourceID)
	if err != nil {
		return nil, fmt.Errorf("source agent: %w", err)
	}

	return a.agentStore.Create(agent.CreateParams{
		Name:                newName,
		Icon:                src.Icon,
		Tags:                src.Tags,
		ToolType:            src.ToolType,
		ModelID:             src.ModelID,
		SystemPrompt:        src.SystemPrompt,
		MCPServers:          src.MCPServers,
		Permissions:         src.Permissions,
		ProjectID:           src.ProjectID,
		BudgetLimitTokens:   src.BudgetLimitTokens,
		BudgetLimitCurrency: src.BudgetLimitCurrency,
		BudgetPeriod:        src.BudgetPeriod,
		BudgetPolicy:        src.BudgetPolicy,
	})
}

// trackAgentEvent records an analytics event for agent operations.
func (a *App) trackAgentEvent(tool, action string) {
	if a.tracker == nil {
		return
	}
	go func() {
		a.tracker.Record(analytics.Event{
			Timestamp: time.Now().Format(time.RFC3339),
			Tool:      tool,
			Action:    action,
			Success:   true,
		})
	}()
}
