package agent

import (
	"context"
	"fmt"
	"sync"
	"time"

	"lurus-switch/internal/process"
)

// Instance tracks the runtime state of a launched agent.
type Instance struct {
	AgentID   string
	SessionID string // process.Monitor session ID
	ToolType  ToolType
}

// InstanceManager bridges agent profiles with the process monitor.
// It tracks which agents are currently running and updates their status
// when processes exit.
type InstanceManager struct {
	store      *Store
	configMgr  *ConfigManager
	processMon *process.Monitor

	mu        sync.Mutex
	instances map[string]*Instance // agentID -> Instance
}

// NewInstanceManager creates an instance manager.
func NewInstanceManager(store *Store, configMgr *ConfigManager, processMon *process.Monitor) *InstanceManager {
	return &InstanceManager{
		store:      store,
		configMgr:  configMgr,
		processMon: processMon,
		instances:  make(map[string]*Instance),
	}
}

// Launch starts the tool process for an agent.
// It generates a per-agent config directory, launches the tool,
// and tracks the instance.
func (m *InstanceManager) Launch(ctx context.Context, agentID string) error {
	profile, err := m.store.Get(agentID)
	if err != nil {
		return fmt.Errorf("get agent: %w", err)
	}

	if profile.Status == StatusRunning {
		// Check if the process is actually alive.
		m.mu.Lock()
		inst, exists := m.instances[agentID]
		m.mu.Unlock()
		if exists && inst.SessionID != "" {
			return fmt.Errorf("agent %q is already running", profile.Name)
		}
		// Process gone but status stale — allow restart.
	}

	// Ensure agent has a config directory.
	dir, err := m.configMgr.AgentDir(agentID)
	if err != nil {
		return fmt.Errorf("prepare config dir: %w", err)
	}

	// Record config dir in profile.
	m.store.SetConfigDir(agentID, dir)

	// Build tool-specific launch arguments.
	args := buildLaunchArgs(profile)

	// Launch the tool process.
	sessionID, err := m.processMon.LaunchTool(ctx, string(profile.ToolType), args)
	if err != nil {
		return fmt.Errorf("launch tool: %w", err)
	}

	// Track the instance.
	m.mu.Lock()
	m.instances[agentID] = &Instance{
		AgentID:   agentID,
		SessionID: sessionID,
		ToolType:  profile.ToolType,
	}
	m.mu.Unlock()

	// Update status to running.
	m.store.SetStatus(agentID, StatusRunning)

	// Monitor for exit in background.
	go m.watchExit(agentID, sessionID)

	return nil
}

// Stop gracefully stops an agent's tool process.
func (m *InstanceManager) Stop(agentID string) error {
	m.mu.Lock()
	inst, exists := m.instances[agentID]
	m.mu.Unlock()

	if !exists {
		// Not tracked — just update status.
		return m.store.SetStatus(agentID, StatusStopped)
	}

	// Stop the process.
	if err := m.processMon.StopSession(inst.SessionID); err != nil {
		return fmt.Errorf("stop session: %w", err)
	}

	// Cleanup is handled by watchExit goroutine.
	return nil
}

// GetInstance returns the runtime instance for an agent, or nil if not running.
func (m *InstanceManager) GetInstance(agentID string) *Instance {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.instances[agentID]
}

// GetOutput returns recent output lines from an agent's process.
func (m *InstanceManager) GetOutput(agentID string, maxLines int) ([]string, error) {
	m.mu.Lock()
	inst, exists := m.instances[agentID]
	m.mu.Unlock()

	if !exists {
		return nil, fmt.Errorf("agent %q is not running", agentID)
	}

	return m.processMon.GetOutput(inst.SessionID, maxLines)
}

// RunningAgentIDs returns the IDs of all currently running agents.
func (m *InstanceManager) RunningAgentIDs() []string {
	m.mu.Lock()
	defer m.mu.Unlock()

	ids := make([]string, 0, len(m.instances))
	for id := range m.instances {
		ids = append(ids, id)
	}
	return ids
}

// SyncStatuses reconciles agent statuses with actual process state.
// Call this on startup to fix stale "running" statuses from a previous crash.
func (m *InstanceManager) SyncStatuses() error {
	running := StatusRunning
	agents, err := m.store.List(&ListFilter{Status: &running})
	if err != nil {
		return err
	}

	for _, a := range agents {
		m.mu.Lock()
		_, tracked := m.instances[a.ID]
		m.mu.Unlock()

		if !tracked {
			// Agent claims to be running but we have no process — it crashed.
			m.store.SetStatus(a.ID, StatusStopped)
		}
	}
	return nil
}

// watchExit waits for a process session to end and updates agent status.
// Polls the process monitor every 2s; exits when the session is gone.
// A channel-based signal would be preferable but process.Monitor's API
// does not currently expose one.
func (m *InstanceManager) watchExit(agentID, sessionID string) {
	const pollInterval = 2 * time.Second
	for {
		if _, err := m.processMon.GetOutput(sessionID, 1); err != nil {
			break
		}
		time.Sleep(pollInterval)
	}

	m.mu.Lock()
	delete(m.instances, agentID)
	m.mu.Unlock()

	// Determine exit status — for now, transition to stopped.
	// Health check (Phase 1) will distinguish crash vs graceful exit.
	current, err := m.store.Get(agentID)
	if err != nil {
		return
	}
	if current.Status == StatusRunning {
		m.store.SetStatus(agentID, StatusStopped)
	}
}

// buildLaunchArgs constructs CLI arguments for launching a tool.
// In the future this will include config dir, model overrides, etc.
func buildLaunchArgs(p *Profile) []string {
	// For now, return empty args. Tools will be launched with their default config.
	// Phase 1 will add: --config-dir, --model, etc. per tool type.
	return nil
}
