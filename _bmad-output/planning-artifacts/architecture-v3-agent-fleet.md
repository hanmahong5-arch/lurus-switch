# Architecture Addendum: Agent Fleet Management (v3)

**Date**: 2026-04-10
**Extends**: architecture.md (existing ADRs preserved)
**Aligned**: product-brief v3 (龙虾管理员)

---

## ADR-017: Agent Profile as First-Class Entity

**Context**: Switch v2 manages tools as singletons (one config per tool type). v3 needs multiple independent instances of the same tool, each with its own identity, config, and lifecycle.

**Decision**: Introduce `AgentProfile` as the core domain entity. An agent is a named, configurable, manageable unit that wraps a tool instance.

**Consequences**:
- Tool config moves from global singleton to per-agent variant
- Process monitor tracks agent IDs, not just PIDs
- All metering, logging, budgeting keyed by agent ID
- Existing tool config management preserved for "unmanaged" tools (backwards compatible)

---

## ADR-018: SQLite as Local State Store

**Context**: Current data storage uses individual JSON/TOML files. Agent management requires structured queries (list by status, aggregate by model, filter by project) that file-based storage cannot efficiently support.

**Decision**: Adopt SQLite (WAL mode) as the primary local state store for agent metadata, metering records, audit logs, and project data. File-based storage preserved for tool configs and templates.

**Implementation**:
- Single DB file: `%APPDATA%/lurus-switch/switch.db`
- WAL mode for concurrent read access during gateway operation
- Single-writer goroutine pattern to avoid write conflicts
- Auto-migration on startup (version table tracks schema version)
- `go-sqlite3` with CGO disabled → use `modernc.org/sqlite` (pure Go)

**Consequences**:
- No CGO dependency (critical: `CGO_ENABLED=0` build constraint)
- Structured queries for reporting and analytics
- Backup = copy one file
- 20+ concurrent agents reading is fine; writes serialized

---

## ADR-019: Agent Lifecycle State Machine

**Context**: Agents have complex lifecycle states that need clear transitions.

**Decision**: Define explicit state machine:

```
         ┌──────────────────────────────────────────────┐
         │                                              │
    ┌────▼────┐   start()   ┌─────────┐   crash    ┌───┴────┐
    │ CREATED ├────────────►│ RUNNING ├───────────►│ ERROR  │
    └────┬────┘             └────┬────┘            └───┬────┘
         │                       │                      │
         │                  stop()│                 restart()
         │                       │                      │
         │                  ┌────▼────┐                 │
         │                  │ STOPPED │◄────────────────┘
         │                  └────┬────┘
         │                       │
         │     delete()          │  delete()
         └───────────►──────────►┘
                    [DELETED - removed from DB]
```

**States**:
| State | Meaning | Transitions From | Transitions To |
|-------|---------|------------------|----------------|
| `created` | Profile exists, never started | (initial) | running, deleted |
| `running` | Process alive, healthy | created, stopped, error | stopped, error |
| `stopped` | Gracefully stopped | running, error | running, deleted |
| `error` | Crashed, max restarts exceeded | running | stopped (manual), running (manual restart), deleted |

**Health check loop** (goroutine per running agent):
1. Every 30s: check process alive (signal 0 / WaitForSingleObject)
2. If dead → attempt restart (exponential backoff: 5s, 10s, 30s, 60s, 60s)
3. After 5 failures → transition to `error` state, emit alert

---

## Data Model

### SQLite Schema

```sql
-- Schema version tracking
CREATE TABLE schema_version (
  version INTEGER PRIMARY KEY,
  applied_at TEXT NOT NULL DEFAULT (datetime('now'))
);

-- Core agent table
CREATE TABLE agents (
  id          TEXT PRIMARY KEY,          -- UUID v4
  name        TEXT NOT NULL,
  icon        TEXT NOT NULL DEFAULT '🤖', -- emoji or icon identifier
  tags        TEXT NOT NULL DEFAULT '[]', -- JSON array of strings
  tool_type   TEXT NOT NULL,             -- claude|codex|gemini|openclaw|zeroclaw|picoclaw|nullclaw
  model_id    TEXT NOT NULL,             -- e.g. "claude-sonnet-4-6"
  system_prompt TEXT NOT NULL DEFAULT '',
  mcp_servers TEXT NOT NULL DEFAULT '[]', -- JSON array of MCP server configs
  permissions TEXT NOT NULL DEFAULT '{}', -- JSON: {allowShell: bool, allowFiles: bool, ...}
  budget_limit_tokens  INTEGER,          -- NULL = unlimited
  budget_limit_currency REAL,            -- NULL = unlimited
  budget_period TEXT DEFAULT 'monthly',  -- daily|monthly|total
  budget_policy TEXT DEFAULT 'pause',    -- pause|degrade|notify_only
  project_id  TEXT,                      -- FK to projects.id, NULL = unassigned
  status      TEXT NOT NULL DEFAULT 'created', -- created|running|stopped|error
  config_dir  TEXT,                      -- path to per-agent config directory
  created_at  TEXT NOT NULL DEFAULT (datetime('now')),
  updated_at  TEXT NOT NULL DEFAULT (datetime('now')),
  FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE SET NULL
);

CREATE INDEX idx_agents_status ON agents(status);
CREATE INDEX idx_agents_tool_type ON agents(tool_type);
CREATE INDEX idx_agents_project ON agents(project_id);

-- Project workspace
CREATE TABLE projects (
  id          TEXT PRIMARY KEY,
  name        TEXT NOT NULL,
  description TEXT NOT NULL DEFAULT '',
  context_dir TEXT,                      -- path to shared context files
  created_at  TEXT NOT NULL DEFAULT (datetime('now')),
  updated_at  TEXT NOT NULL DEFAULT (datetime('now'))
);

-- Metering records (per-request granularity)
CREATE TABLE metering (
  id          INTEGER PRIMARY KEY AUTOINCREMENT,
  agent_id    TEXT NOT NULL,
  app_id      TEXT,                      -- gateway app token ID
  model       TEXT NOT NULL,
  tokens_in   INTEGER NOT NULL DEFAULT 0,
  tokens_out  INTEGER NOT NULL DEFAULT 0,
  cached_hit  INTEGER NOT NULL DEFAULT 0, -- boolean
  latency_ms  INTEGER NOT NULL DEFAULT 0,
  status_code INTEGER NOT NULL DEFAULT 200,
  error_msg   TEXT,
  timestamp   TEXT NOT NULL DEFAULT (datetime('now')),
  FOREIGN KEY (agent_id) REFERENCES agents(id) ON DELETE CASCADE
);

CREATE INDEX idx_metering_agent ON metering(agent_id, timestamp);
CREATE INDEX idx_metering_timestamp ON metering(timestamp);

-- Budget tracking (daily aggregates)
CREATE TABLE budget_usage (
  agent_id    TEXT NOT NULL,
  date        TEXT NOT NULL,             -- YYYY-MM-DD
  tokens_used INTEGER NOT NULL DEFAULT 0,
  cost_usd    REAL NOT NULL DEFAULT 0,
  PRIMARY KEY (agent_id, date),
  FOREIGN KEY (agent_id) REFERENCES agents(id) ON DELETE CASCADE
);

-- Audit log
CREATE TABLE audit_log (
  id          INTEGER PRIMARY KEY AUTOINCREMENT,
  event_type  TEXT NOT NULL,             -- agent.created|agent.started|agent.stopped|budget.changed|...
  agent_id    TEXT,
  detail      TEXT NOT NULL DEFAULT '{}', -- JSON payload
  timestamp   TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX idx_audit_timestamp ON audit_log(timestamp);
CREATE INDEX idx_audit_agent ON audit_log(agent_id);

-- Agent templates
CREATE TABLE templates (
  id          TEXT PRIMARY KEY,
  name        TEXT NOT NULL,
  description TEXT NOT NULL DEFAULT '',
  icon        TEXT NOT NULL DEFAULT '📋',
  category    TEXT NOT NULL DEFAULT 'custom', -- development|writing|analysis|productivity|automation|custom
  tool_type   TEXT NOT NULL,
  model_id    TEXT NOT NULL,
  system_prompt TEXT NOT NULL DEFAULT '',
  mcp_servers TEXT NOT NULL DEFAULT '[]',
  permissions TEXT NOT NULL DEFAULT '{}',
  budget_suggestion_tokens INTEGER,
  is_builtin  INTEGER NOT NULL DEFAULT 0, -- boolean
  version     INTEGER NOT NULL DEFAULT 1,
  created_at  TEXT NOT NULL DEFAULT (datetime('now')),
  updated_at  TEXT NOT NULL DEFAULT (datetime('now'))
);

-- Alert rules
CREATE TABLE alert_rules (
  id          TEXT PRIMARY KEY,
  name        TEXT NOT NULL,
  condition   TEXT NOT NULL,             -- JSON: {type: "crash_count", threshold: 3, window_minutes: 10}
  action      TEXT NOT NULL DEFAULT 'notify', -- notify|pause_agent|degrade_model
  enabled     INTEGER NOT NULL DEFAULT 1,
  created_at  TEXT NOT NULL DEFAULT (datetime('now'))
);

-- Alert history
CREATE TABLE alert_history (
  id          INTEGER PRIMARY KEY AUTOINCREMENT,
  rule_id     TEXT,
  agent_id    TEXT,
  message     TEXT NOT NULL,
  acknowledged INTEGER NOT NULL DEFAULT 0,
  timestamp   TEXT NOT NULL DEFAULT (datetime('now'))
);
```

---

## Go Package Structure (New)

```
internal/
├── agent/                    # Agent lifecycle management
│   ├── profile.go           # AgentProfile struct + CRUD
│   ├── instance.go          # Process lifecycle (start/stop/restart)
│   ├── health.go            # Health check loop + auto-restart
│   ├── budget.go            # Per-agent budget enforcement
│   ├── store.go             # SQLite persistence (agents table)
│   └── agent_test.go
│
├── project/                  # Project workspace
│   ├── workspace.go         # Project CRUD + agent binding
│   ├── context.go           # Shared context injection into agent prompts
│   ├── store.go             # SQLite persistence (projects table)
│   └── project_test.go
│
├── template/                 # Agent templates
│   ├── builtin.go           # 15+ built-in templates (embedded)
│   ├── custom.go            # User template CRUD
│   ├── store.go             # SQLite persistence (templates table)
│   ├── export.go            # JSON import/export
│   └── template_test.go
│
├── monitor/                  # Monitoring & alerting
│   ├── collector.go         # Metrics aggregation from metering data
│   ├── alerter.go           # Alert rule engine + evaluation loop
│   ├── report.go            # Cost report generation + CSV export
│   ├── store.go             # SQLite persistence (alert_rules, alert_history)
│   └── monitor_test.go
│
├── logstream/                # Agent log aggregation
│   ├── stream.go            # Per-agent ring buffer + Wails event streaming
│   ├── aggregator.go        # Multi-agent combined view
│   └── logstream_test.go
│
├── db/                       # SQLite database management
│   ├── db.go                # Open, migrate, close
│   ├── migrations.go        # Schema migration definitions (embedded SQL)
│   └── db_test.go
│
├── ... (existing packages unchanged)
```

### Wails Bindings (New)

```
bindings_agent.go            # CreateAgent, ListAgents, GetAgent, UpdateAgent, DeleteAgent
                             # LaunchAgent, StopAgent, RestartAgent, CloneAgent
                             # GetAgentLogs, BatchStartAgents, BatchStopAgents

bindings_project.go          # CreateProject, ListProjects, GetProject, UpdateProject, DeleteProject
                             # BindAgentToProject, GetProjectAgents, GetProjectContext

bindings_template.go         # ListTemplates, GetTemplate, CreateFromTemplate
                             # SaveAsTemplate, ImportTemplate, ExportTemplate

bindings_monitor.go          # GetBurnRate, GetCostReport, ExportCostCSV
                             # ListAlertRules, CreateAlertRule, GetAlertHistory
                             # GetAuditLog

bindings_budget.go           # SetAgentBudget, GetBudgetStatus, SetGlobalBudget
                             # GetGlobalBudgetStatus
```

### Frontend Structure (New)

```
frontend/src/
├── pages/
│   ├── AgentsPage.tsx           # Agent list/grid view + creation wizard
│   ├── AgentDetailPage.tsx      # Single agent: config, logs, metrics, snapshots
│   ├── ProjectsPage.tsx         # Project list + bound agents + context editor
│   ├── TemplatesPage.tsx        # Template browser + import/export
│   ├── AnalyticsPage.tsx        # Cost reports + performance comparison + audit
│   └── ... (existing pages preserved)
│
├── stores/
│   ├── agentStore.ts            # Agent CRUD + lifecycle actions
│   ├── projectStore.ts          # Project CRUD + context management
│   ├── templateStore.ts         # Template listing + deploy
│   ├── monitorStore.ts          # Burn rate + alerts + audit
│   └── ... (existing stores preserved)
│
├── components/
│   ├── agents/
│   │   ├── AgentCard.tsx        # Agent card (status LED, name, metrics, actions)
│   │   ├── AgentGrid.tsx        # Responsive card grid with filters/sort
│   │   ├── CreationWizard.tsx   # 7-step agent creation flow
│   │   ├── BatchActionBar.tsx   # Multi-select action toolbar
│   │   ├── LogViewer.tsx        # Real-time log stream viewer
│   │   └── BudgetBar.tsx        # Budget progress indicator
│   │
│   ├── projects/
│   │   ├── ProjectCard.tsx      # Project with bound agent count
│   │   ├── ContextEditor.tsx    # Markdown editor for shared context
│   │   └── KnowledgeBase.tsx    # Shared documents manager
│   │
│   ├── templates/
│   │   ├── TemplateCard.tsx     # Template preview card
│   │   ├── TemplateBrowser.tsx  # Category filters + search
│   │   └── DeployDialog.tsx     # One-click deploy confirmation
│   │
│   ├── analytics/
│   │   ├── BurnRateGauge.tsx    # Real-time consumption meter
│   │   ├── CostChart.tsx        # Time series cost visualization
│   │   ├── ComparisonTable.tsx  # Agent efficiency comparison
│   │   └── AuditTimeline.tsx    # Audit event timeline
│   │
│   └── ... (existing components preserved)
```

---

## Gateway Integration

Agent management integrates with the existing gateway via per-agent app tokens:

```
Agent Profile
  │
  ├── tool_type: "claude"
  ├── model_id: "claude-sonnet-4-6"
  ├── budget_limit: 100000 tokens/month
  │
  └──→ On LaunchAgent():
       1. Generate temp config file (claude settings.json) with Switch gateway as endpoint
       2. Register agent as "app" in appreg → get per-agent app token
       3. Launch tool process with generated config
       4. Gateway middleware checks agent budget on each request:
          - Extract app token → resolve to agent ID
          - Query budget_usage for current period
          - If exceeded → enforce policy (429 / degrade model / notify)
       5. After each request → record metering row → update budget_usage aggregate
```

---

## Migration Path

### Phase 0 (Sprint 4): Foundation

1. Add `modernc.org/sqlite` to go.mod
2. Create `internal/db/` with migration support
3. Create `internal/agent/` with Profile + Store
4. Extend `internal/process/monitor.go` with agent binding
5. Add `bindings_agent.go` + `AgentsPage.tsx`
6. **No breaking changes** — existing tools still work as before, agents are additive

### Phase 1-2 (Sprint 5-7): Lifecycle + Budget

1. Creation wizard replaces raw tool config flow for new agents
2. Dashboard redesign (existing ToolCards preserved in "Tools" tab)
3. Budget enforcement added to gateway middleware
4. Metering data migrates from daily JSON files to SQLite

### Phase 3-5 (Sprint 8-10): Context + Monitoring + Templates

1. Projects page adds context management
2. Monitor/alert system runs as background goroutines
3. Templates pre-loaded on first startup

**Backwards compatibility guarantee**: Users who don't use agent management can continue using Switch as before (tool config + gateway). Agent features are opt-in.
