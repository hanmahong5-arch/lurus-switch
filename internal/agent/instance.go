package agent

import (
	"context"
	"fmt"
	"log"
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

// processLister is the subset of *process.Monitor that SyncStatuses needs.
// Pulled out so tests can inject a fake without spawning real OS processes.
type processLister interface {
	ListCLIProcesses(ctx context.Context) ([]process.ProcessInfo, error)
}

// InstanceManager bridges agent profiles with the process monitor.
// It tracks which agents are currently running and updates their status
// when processes exit.
type InstanceManager struct {
	store      *Store
	configMgr  *ConfigManager
	processMon *process.Monitor

	// liveProcs defaults to processMon. Tests override via setLiveLister.
	// Kept separate so unit tests can avoid shelling out to tasklist/ps.
	liveProcs processLister

	mu        sync.Mutex
	instances map[string]*Instance // agentID -> Instance
}

// NewInstanceManager creates an instance manager.
func NewInstanceManager(store *Store, configMgr *ConfigManager, processMon *process.Monitor) *InstanceManager {
	m := &InstanceManager{
		store:      store,
		configMgr:  configMgr,
		processMon: processMon,
		instances:  make(map[string]*Instance),
	}
	// Only wire the live lister when the monitor itself is non-nil — a
	// typed-nil concrete value stored in an interface field would still
	// satisfy `field != nil`, then panic on call. Keeping liveProcs as
	// the untyped nil lets SyncStatuses skip the OS lookup cleanly.
	if processMon != nil {
		m.liveProcs = processMon
	}
	return m
}

// setLiveLister overrides the source of "OS processes alive right now".
// Test-only seam — kept lower-case so it stays in-package.
func (m *InstanceManager) setLiveLister(p processLister) {
	m.liveProcs = p
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
	args, err := buildLaunchArgs(profile)
	if err != nil {
		return fmt.Errorf("build launch args: %w", err)
	}

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
//
// Two-layer signal:
//  1. In-memory `instances` map: if we have a session for an agent in this
//     process, it's running (we're the source of truth).
//  2. OS-level CLI tool processes (via ListCLIProcesses): if an agent's
//     ToolType has *no* live OS process at all, the agent definitely isn't
//     running. If processes exist but we don't own them, the agent still
//     can't be ours — be conservative and mark stopped.
//
// Called on startup to fix stale "running" statuses left behind by a crash.
// Never panics; individual failures are logged and skipped so one bad
// profile doesn't abort the whole reconciliation.
func (m *InstanceManager) SyncStatuses() error {
	running := StatusRunning
	agents, err := m.store.List(&ListFilter{Status: &running})
	if err != nil {
		return fmt.Errorf("list running agents: %w", err)
	}

	// Snapshot OS-level CLI processes once. Bounded 5s timeout so a hung
	// tasklist/ps can't block startup. ListCLIProcesses failure is non-
	// fatal — we fall back to the in-memory map only.
	listCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	liveByTool := make(map[string]bool)
	if m.liveProcs != nil {
		procs, lerr := m.liveProcs.ListCLIProcesses(listCtx)
		if lerr != nil {
			log.Printf("agent: SyncStatuses ListCLIProcesses: %v", lerr)
		}
		for _, p := range procs {
			liveByTool[p.Tool] = true
		}
	}

	var firstErr error
	for _, a := range agents {
		m.mu.Lock()
		_, tracked := m.instances[a.ID]
		m.mu.Unlock()
		if tracked {
			// We launched this session in this process — leave it alone.
			continue
		}

		// Not tracked. Either we crashed and restarted, or another instance
		// of Switch owned it. Either way, this process can no longer drive
		// it — mark stopped. The OS-level check below adds a confidence
		// note to the audit trail but doesn't change the decision (we
		// can't reattach to a foreign session).
		hasLive := liveByTool[string(a.ToolType)]
		if err := m.store.SetStatus(a.ID, StatusStopped); err != nil {
			log.Printf("agent: SyncStatuses SetStatus(%s) failed: %v", a.ID, err)
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		if hasLive {
			log.Printf("agent: %s (%s) marked stopped despite live %s process — not owned by this Switch instance",
				a.ID, a.Name, a.ToolType)
		} else {
			log.Printf("agent: %s (%s) marked stopped — no live %s process",
				a.ID, a.Name, a.ToolType)
		}
	}
	return firstErr
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

// buildLaunchArgs constructs CLI arguments for launching the tool process
// backing an agent profile. Returns the args only — environment variables
// (ANTHROPIC_API_KEY, OPENAI_API_KEY, etc.) are layered on by the caller
// via os.Environ() before exec.
//
// Conventions:
//   - Empty ModelID → omit the --model flag rather than passing an empty
//     string; CLI tools either reject "" or fall back to a default that
//     contradicts the profile.
//   - claude / codex / gemini support a documented `--model <id>` flag.
//   - The *claw family (picoclaw / nullclaw / openclaw / zeroclaw) reads
//     all configuration from its on-disk JSON/TOML config file written by
//     the corresponding generator; the CLI surface has no published model
//     flag, so we pass no args and rely on the config file. TODO: revisit
//     once each tool's `--help` output is checked end-to-end.
func buildLaunchArgs(p *Profile) ([]string, error) {
	if p == nil {
		return nil, fmt.Errorf("nil profile")
	}

	switch p.ToolType {
	case ToolClaude:
		// Claude Code CLI accepts `--model <id>`. Per-agent config dir is
		// communicated to the binary via CLAUDE_CONFIG_DIR (env, set by
		// caller), not an arg, so the args stay minimal.
		// TODO: verify Claude Code's CLI flag surface against `claude --help`.
		var args []string
		if p.ModelID != "" {
			args = append(args, "--model", p.ModelID)
		}
		return args, nil

	case ToolCodex:
		// Codex CLI accepts `--model <id>`. Auth + provider come from
		// env (OPENAI_API_KEY, OPENAI_BASE_URL) or the generated
		// config.toml — neither belongs in args.
		// TODO: verify Codex CLI's exact flag surface.
		var args []string
		if p.ModelID != "" {
			args = append(args, "--model", p.ModelID)
		}
		return args, nil

	case ToolGemini:
		// Gemini CLI accepts `--model <id>`. Auth via env
		// (GOOGLE_AI_API_KEY) or ADC.
		// TODO: verify Gemini CLI's exact flag surface.
		var args []string
		if p.ModelID != "" {
			args = append(args, "--model", p.ModelID)
		}
		return args, nil

	case ToolPicoClaw, ToolNullClaw, ToolOpenClaw, ToolZeroClaw:
		// The *claw family reads its model list and provider config from a
		// generator-written JSON/TOML file under the agent's config dir.
		// No published `--model` style CLI flag, so we pass nothing and
		// trust the file. The caller is responsible for pointing the
		// binary at the right config (via env or working dir).
		// TODO: verify args for picoclaw / nullclaw / openclaw / zeroclaw
		// once those CLIs ship a `--help`.
		return nil, nil

	default:
		return nil, fmt.Errorf("unknown tool kind: %s", p.ToolType)
	}
}
