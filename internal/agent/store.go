package agent

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"

	"lurus-switch/internal/db"
)

// Store provides CRUD operations for agent profiles backed by SQLite.
type Store struct {
	db *db.DB
}

// NewStore creates a new agent store.
func NewStore(database *db.DB) *Store {
	return &Store{db: database}
}

// Create persists a new agent profile and returns the created profile.
func (s *Store) Create(p CreateParams) (*Profile, error) {
	if p.Name == "" {
		return nil, fmt.Errorf("agent name is required")
	}
	if !p.ToolType.IsValid() {
		return nil, fmt.Errorf("invalid tool type: %q", p.ToolType)
	}
	if p.ModelID == "" {
		return nil, fmt.Errorf("model ID is required")
	}

	id := uuid.New().String()
	now := time.Now().UTC()
	icon := p.Icon
	if icon == "" {
		icon = "🤖"
	}
	tags := p.Tags
	if tags == nil {
		tags = []string{}
	}
	mcpServers := p.MCPServers
	if mcpServers == nil {
		mcpServers = []string{}
	}
	budgetPeriod := p.BudgetPeriod
	if budgetPeriod == "" {
		budgetPeriod = BudgetMonthly
	}
	budgetPolicy := p.BudgetPolicy
	if budgetPolicy == "" {
		budgetPolicy = PolicyPause
	}

	err := s.db.WriteTx(func(tx *sql.Tx) error {
		_, err := tx.Exec(`INSERT INTO agents
			(id, name, icon, tags, tool_type, model_id, system_prompt, mcp_servers,
			 permissions, budget_limit_tokens, budget_limit_currency, budget_period,
			 budget_policy, project_id, status, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			id, p.Name, icon, marshalJSON(tags), string(p.ToolType), p.ModelID,
			p.SystemPrompt, marshalJSON(mcpServers), marshalJSON(p.Permissions),
			p.BudgetLimitTokens, p.BudgetLimitCurrency, string(budgetPeriod),
			string(budgetPolicy), nullStr(p.ProjectID),
			string(StatusCreated), now.Format(time.RFC3339), now.Format(time.RFC3339),
		)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("insert agent: %w", err)
	}

	return s.Get(id)
}

// Get retrieves a single agent by ID.
func (s *Store) Get(id string) (*Profile, error) {
	row := s.db.Conn().QueryRow(`SELECT
		id, name, icon, tags, tool_type, model_id, system_prompt, mcp_servers,
		permissions, budget_limit_tokens, budget_limit_currency, budget_period,
		budget_policy, project_id, status, config_dir, created_at, updated_at
		FROM agents WHERE id = ?`, id)

	return scanProfile(row)
}

// List returns agents matching the optional filter criteria.
func (s *Store) List(filter *ListFilter) ([]*Profile, error) {
	query := `SELECT
		id, name, icon, tags, tool_type, model_id, system_prompt, mcp_servers,
		permissions, budget_limit_tokens, budget_limit_currency, budget_period,
		budget_policy, project_id, status, config_dir, created_at, updated_at
		FROM agents WHERE 1=1`
	var args []any

	if filter != nil {
		if filter.Status != nil {
			query += " AND status = ?"
			args = append(args, string(*filter.Status))
		}
		if filter.ToolType != nil {
			query += " AND tool_type = ?"
			args = append(args, string(*filter.ToolType))
		}
		if filter.ProjectID != nil {
			query += " AND project_id = ?"
			args = append(args, *filter.ProjectID)
		}
		if filter.Tag != nil {
			// JSON array contains check via LIKE (SQLite has no native JSON array contains).
			query += ` AND tags LIKE ?`
			args = append(args, fmt.Sprintf(`%%"%s"%%`, *filter.Tag))
		}
	}

	query += " ORDER BY created_at DESC"

	rows, err := s.db.Conn().Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("query agents: %w", err)
	}
	defer rows.Close()

	var profiles []*Profile
	for rows.Next() {
		p, err := scanProfileRows(rows)
		if err != nil {
			return nil, err
		}
		profiles = append(profiles, p)
	}
	return profiles, rows.Err()
}

// Update modifies an existing agent. Only non-nil fields in params are applied.
func (s *Store) Update(id string, p UpdateParams) (*Profile, error) {
	return s.updateFields(id, p)
}

// SetStatus updates only the status field of an agent.
func (s *Store) SetStatus(id string, status Status) error {
	_, err := s.db.ExecWrite(
		"UPDATE agents SET status = ?, updated_at = ? WHERE id = ?",
		string(status), time.Now().UTC().Format(time.RFC3339), id,
	)
	return err
}

// SetConfigDir sets the agent's configuration directory.
func (s *Store) SetConfigDir(id, dir string) error {
	_, err := s.db.ExecWrite(
		"UPDATE agents SET config_dir = ?, updated_at = ? WHERE id = ?",
		dir, time.Now().UTC().Format(time.RFC3339), id,
	)
	return err
}

// Delete removes an agent by ID.
func (s *Store) Delete(id string) error {
	result, err := s.db.ExecWrite("DELETE FROM agents WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("delete agent: %w", err)
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return fmt.Errorf("agent not found: %s", id)
	}
	return nil
}

// Count returns the total number of agents.
func (s *Store) Count() (int, error) {
	var n int
	err := s.db.Conn().QueryRow("SELECT COUNT(*) FROM agents").Scan(&n)
	return n, err
}

// CountByStatus returns agent counts grouped by status.
func (s *Store) CountByStatus() (map[Status]int, error) {
	rows, err := s.db.Conn().Query("SELECT status, COUNT(*) FROM agents GROUP BY status")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	counts := make(map[Status]int)
	for rows.Next() {
		var status string
		var count int
		if err := rows.Scan(&status, &count); err != nil {
			return nil, err
		}
		counts[Status(status)] = count
	}
	return counts, rows.Err()
}

// updateFields builds a dynamic UPDATE query from non-nil params.
func (s *Store) updateFields(id string, p UpdateParams) (*Profile, error) {
	sets := []string{}
	args := []any{}

	if p.Name != nil {
		sets = append(sets, "name = ?")
		args = append(args, *p.Name)
	}
	if p.Icon != nil {
		sets = append(sets, "icon = ?")
		args = append(args, *p.Icon)
	}
	if p.Tags != nil {
		sets = append(sets, "tags = ?")
		args = append(args, marshalJSON(p.Tags))
	}
	if p.ModelID != nil {
		sets = append(sets, "model_id = ?")
		args = append(args, *p.ModelID)
	}
	if p.SystemPrompt != nil {
		sets = append(sets, "system_prompt = ?")
		args = append(args, *p.SystemPrompt)
	}
	if p.MCPServers != nil {
		sets = append(sets, "mcp_servers = ?")
		args = append(args, marshalJSON(p.MCPServers))
	}
	if p.Permissions != nil {
		sets = append(sets, "permissions = ?")
		args = append(args, marshalJSON(*p.Permissions))
	}
	if p.ProjectID != nil {
		sets = append(sets, "project_id = ?")
		args = append(args, nullStr(*p.ProjectID))
	}
	if p.Status != nil {
		sets = append(sets, "status = ?")
		args = append(args, string(*p.Status))
	}
	if p.BudgetLimitTokens != nil {
		sets = append(sets, "budget_limit_tokens = ?")
		args = append(args, *p.BudgetLimitTokens)
	}
	if p.BudgetLimitCurrency != nil {
		sets = append(sets, "budget_limit_currency = ?")
		args = append(args, *p.BudgetLimitCurrency)
	}
	if p.BudgetPeriod != nil {
		sets = append(sets, "budget_period = ?")
		args = append(args, string(*p.BudgetPeriod))
	}
	if p.BudgetPolicy != nil {
		sets = append(sets, "budget_policy = ?")
		args = append(args, string(*p.BudgetPolicy))
	}

	if len(sets) == 0 {
		return s.Get(id)
	}

	sets = append(sets, "updated_at = ?")
	args = append(args, time.Now().UTC().Format(time.RFC3339))
	args = append(args, id)

	query := "UPDATE agents SET "
	for i, set := range sets {
		if i > 0 {
			query += ", "
		}
		query += set
	}
	query += " WHERE id = ?"

	result, err := s.db.ExecWrite(query, args...)
	if err != nil {
		return nil, fmt.Errorf("update agent: %w", err)
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return nil, fmt.Errorf("agent not found: %s", id)
	}

	return s.Get(id)
}

// scanner is an interface satisfied by both *sql.Row and *sql.Rows.
type scanner interface {
	Scan(dest ...any) error
}

func scanProfile(row *sql.Row) (*Profile, error) {
	p := &Profile{}
	var tagsJSON, mcpJSON, permsJSON string
	var projectID, configDir sql.NullString
	var budgetTokens sql.NullInt64
	var budgetCurrency sql.NullFloat64
	var createdAt, updatedAt string

	err := row.Scan(
		&p.ID, &p.Name, &p.Icon, &tagsJSON, (*string)(&p.ToolType), &p.ModelID,
		&p.SystemPrompt, &mcpJSON, &permsJSON,
		&budgetTokens, &budgetCurrency, (*string)(&p.BudgetPeriod),
		(*string)(&p.BudgetPolicy), &projectID, (*string)(&p.Status),
		&configDir, &createdAt, &updatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("agent not found")
		}
		return nil, fmt.Errorf("scan agent: %w", err)
	}

	unmarshalJSON(tagsJSON, &p.Tags)
	unmarshalJSON(mcpJSON, &p.MCPServers)
	unmarshalJSON(permsJSON, &p.Permissions)
	if p.Tags == nil {
		p.Tags = []string{}
	}
	if p.MCPServers == nil {
		p.MCPServers = []string{}
	}
	if projectID.Valid {
		p.ProjectID = projectID.String
	}
	if configDir.Valid {
		p.ConfigDir = configDir.String
	}
	if budgetTokens.Valid {
		p.BudgetLimitTokens = &budgetTokens.Int64
	}
	if budgetCurrency.Valid {
		p.BudgetLimitCurrency = &budgetCurrency.Float64
	}
	p.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	p.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)

	return p, nil
}

func scanProfileRows(rows *sql.Rows) (*Profile, error) {
	p := &Profile{}
	var tagsJSON, mcpJSON, permsJSON string
	var projectID, configDir sql.NullString
	var budgetTokens sql.NullInt64
	var budgetCurrency sql.NullFloat64
	var createdAt, updatedAt string

	err := rows.Scan(
		&p.ID, &p.Name, &p.Icon, &tagsJSON, (*string)(&p.ToolType), &p.ModelID,
		&p.SystemPrompt, &mcpJSON, &permsJSON,
		&budgetTokens, &budgetCurrency, (*string)(&p.BudgetPeriod),
		(*string)(&p.BudgetPolicy), &projectID, (*string)(&p.Status),
		&configDir, &createdAt, &updatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("scan agent row: %w", err)
	}

	unmarshalJSON(tagsJSON, &p.Tags)
	unmarshalJSON(mcpJSON, &p.MCPServers)
	unmarshalJSON(permsJSON, &p.Permissions)
	if p.Tags == nil {
		p.Tags = []string{}
	}
	if p.MCPServers == nil {
		p.MCPServers = []string{}
	}
	if projectID.Valid {
		p.ProjectID = projectID.String
	}
	if configDir.Valid {
		p.ConfigDir = configDir.String
	}
	if budgetTokens.Valid {
		p.BudgetLimitTokens = &budgetTokens.Int64
	}
	if budgetCurrency.Valid {
		p.BudgetLimitCurrency = &budgetCurrency.Float64
	}
	p.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	p.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)

	return p, nil
}

// nullStr converts an empty string to sql.NullString.
func nullStr(s string) sql.NullString {
	if s == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: s, Valid: true}
}
