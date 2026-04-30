package db

import (
	"database/sql"
	"fmt"
)

// migrations is an ordered list of SQL migration statements.
// Each entry is applied exactly once, tracked by its index (1-based version).
// Append-only: never edit or remove existing entries.
var migrations = []string{
	// v1: core agent management tables
	`CREATE TABLE IF NOT EXISTS agents (
		id              TEXT PRIMARY KEY,
		name            TEXT NOT NULL,
		icon            TEXT NOT NULL DEFAULT '🤖',
		tags            TEXT NOT NULL DEFAULT '[]',
		tool_type       TEXT NOT NULL,
		model_id        TEXT NOT NULL,
		system_prompt   TEXT NOT NULL DEFAULT '',
		mcp_servers     TEXT NOT NULL DEFAULT '[]',
		permissions     TEXT NOT NULL DEFAULT '{}',
		budget_limit_tokens   INTEGER,
		budget_limit_currency REAL,
		budget_period   TEXT DEFAULT 'monthly',
		budget_policy   TEXT DEFAULT 'pause',
		project_id      TEXT,
		status          TEXT NOT NULL DEFAULT 'created',
		config_dir      TEXT,
		created_at      TEXT NOT NULL DEFAULT (datetime('now')),
		updated_at      TEXT NOT NULL DEFAULT (datetime('now'))
	);
	CREATE INDEX IF NOT EXISTS idx_agents_status ON agents(status);
	CREATE INDEX IF NOT EXISTS idx_agents_tool ON agents(tool_type);
	CREATE INDEX IF NOT EXISTS idx_agents_project ON agents(project_id);`,

	// v2: projects table
	`CREATE TABLE IF NOT EXISTS projects (
		id          TEXT PRIMARY KEY,
		name        TEXT NOT NULL,
		description TEXT NOT NULL DEFAULT '',
		context_dir TEXT,
		created_at  TEXT NOT NULL DEFAULT (datetime('now')),
		updated_at  TEXT NOT NULL DEFAULT (datetime('now'))
	);`,

	// v3: budget usage tracking (daily aggregates)
	`CREATE TABLE IF NOT EXISTS budget_usage (
		agent_id    TEXT NOT NULL,
		date        TEXT NOT NULL,
		tokens_used INTEGER NOT NULL DEFAULT 0,
		cost_usd    REAL NOT NULL DEFAULT 0,
		PRIMARY KEY (agent_id, date)
	);`,

	// v4: audit log
	`CREATE TABLE IF NOT EXISTS audit_log (
		id          INTEGER PRIMARY KEY AUTOINCREMENT,
		event_type  TEXT NOT NULL,
		agent_id    TEXT,
		detail      TEXT NOT NULL DEFAULT '{}',
		timestamp   TEXT NOT NULL DEFAULT (datetime('now'))
	);
	CREATE INDEX IF NOT EXISTS idx_audit_ts ON audit_log(timestamp);
	CREATE INDEX IF NOT EXISTS idx_audit_agent ON audit_log(agent_id);`,
}

// migrate applies all pending migrations.
func (d *DB) migrate() error {
	// Ensure the version tracking table exists.
	if _, err := d.conn.Exec(`CREATE TABLE IF NOT EXISTS schema_version (
		version    INTEGER PRIMARY KEY,
		applied_at TEXT NOT NULL DEFAULT (datetime('now'))
	)`); err != nil {
		return fmt.Errorf("create schema_version table: %w", err)
	}

	// Determine current version.
	var current int
	row := d.conn.QueryRow("SELECT COALESCE(MAX(version), 0) FROM schema_version")
	if err := row.Scan(&current); err != nil {
		return fmt.Errorf("read schema version: %w", err)
	}

	// Apply pending migrations.
	for i := current; i < len(migrations); i++ {
		version := i + 1
		if err := d.applyMigration(version, migrations[i]); err != nil {
			return fmt.Errorf("migration v%d: %w", version, err)
		}
	}

	return nil
}

func (d *DB) applyMigration(version int, ddl string) error {
	tx, err := d.conn.Begin()
	if err != nil {
		return err
	}

	if _, err := tx.Exec(ddl); err != nil {
		tx.Rollback()
		return fmt.Errorf("execute DDL: %w", err)
	}

	if _, err := tx.Exec("INSERT INTO schema_version (version) VALUES (?)", version); err != nil {
		tx.Rollback()
		return fmt.Errorf("record version: %w", err)
	}

	return tx.Commit()
}

// SchemaVersion returns the current schema version.
func (d *DB) SchemaVersion() (int, error) {
	var v int
	err := d.conn.QueryRow("SELECT COALESCE(MAX(version), 0) FROM schema_version").Scan(&v)
	return v, err
}

// ExecWrite is a convenience wrapper for single-statement writes.
func (d *DB) ExecWrite(query string, args ...any) (sql.Result, error) {
	var result sql.Result
	err := d.WriteTx(func(tx *sql.Tx) error {
		var e error
		result, e = tx.Exec(query, args...)
		return e
	})
	return result, err
}
