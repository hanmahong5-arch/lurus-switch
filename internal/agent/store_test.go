package agent

import (
	"os"
	"testing"

	"lurus-switch/internal/db"
)

func setupTestDB(t *testing.T) (*db.DB, func()) {
	t.Helper()
	dir, err := os.MkdirTemp("", "agent-test-*")
	if err != nil {
		t.Fatalf("create temp dir: %v", err)
	}
	database, err := db.Open(dir)
	if err != nil {
		os.RemoveAll(dir)
		t.Fatalf("open db: %v", err)
	}
	return database, func() {
		database.Close()
		os.RemoveAll(dir)
	}
}

func TestCreate_Success(t *testing.T) {
	database, cleanup := setupTestDB(t)
	defer cleanup()
	store := NewStore(database)

	p, err := store.Create(CreateParams{
		Name:     "Frontend Reviewer",
		Icon:     "👀",
		Tags:     []string{"review", "frontend"},
		ToolType: ToolClaude,
		ModelID:  "claude-sonnet-4-6",
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if p.ID == "" {
		t.Error("expected non-empty ID")
	}
	if p.Name != "Frontend Reviewer" {
		t.Errorf("name = %q, want %q", p.Name, "Frontend Reviewer")
	}
	if p.Icon != "👀" {
		t.Errorf("icon = %q, want %q", p.Icon, "👀")
	}
	if len(p.Tags) != 2 || p.Tags[0] != "review" {
		t.Errorf("tags = %v, want [review, frontend]", p.Tags)
	}
	if p.ToolType != ToolClaude {
		t.Errorf("toolType = %q, want %q", p.ToolType, ToolClaude)
	}
	if p.Status != StatusCreated {
		t.Errorf("status = %q, want %q", p.Status, StatusCreated)
	}
	if p.BudgetPeriod != BudgetMonthly {
		t.Errorf("budgetPeriod = %q, want %q", p.BudgetPeriod, BudgetMonthly)
	}
}

func TestCreate_Validation(t *testing.T) {
	database, cleanup := setupTestDB(t)
	defer cleanup()
	store := NewStore(database)

	// Missing name
	_, err := store.Create(CreateParams{ToolType: ToolClaude, ModelID: "m"})
	if err == nil {
		t.Error("expected error for empty name")
	}

	// Invalid tool type
	_, err = store.Create(CreateParams{Name: "x", ToolType: "invalid", ModelID: "m"})
	if err == nil {
		t.Error("expected error for invalid tool type")
	}

	// Missing model
	_, err = store.Create(CreateParams{Name: "x", ToolType: ToolClaude})
	if err == nil {
		t.Error("expected error for empty model ID")
	}
}

func TestGet_NotFound(t *testing.T) {
	database, cleanup := setupTestDB(t)
	defer cleanup()
	store := NewStore(database)

	_, err := store.Get("nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent agent")
	}
}

func TestList_Empty(t *testing.T) {
	database, cleanup := setupTestDB(t)
	defer cleanup()
	store := NewStore(database)

	agents, err := store.List(nil)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(agents) != 0 {
		t.Errorf("expected 0 agents, got %d", len(agents))
	}
}

func TestList_WithFilters(t *testing.T) {
	database, cleanup := setupTestDB(t)
	defer cleanup()
	store := NewStore(database)

	// Create 3 agents with different tools
	store.Create(CreateParams{Name: "A1", ToolType: ToolClaude, ModelID: "m1", Tags: []string{"dev"}})
	store.Create(CreateParams{Name: "A2", ToolType: ToolCodex, ModelID: "m2", Tags: []string{"dev"}})
	store.Create(CreateParams{Name: "A3", ToolType: ToolClaude, ModelID: "m3", Tags: []string{"review"}})

	// Filter by tool type
	tt := ToolClaude
	agents, err := store.List(&ListFilter{ToolType: &tt})
	if err != nil {
		t.Fatalf("list by tool: %v", err)
	}
	if len(agents) != 2 {
		t.Errorf("expected 2 claude agents, got %d", len(agents))
	}

	// Filter by tag
	tag := "dev"
	agents, err = store.List(&ListFilter{Tag: &tag})
	if err != nil {
		t.Fatalf("list by tag: %v", err)
	}
	if len(agents) != 2 {
		t.Errorf("expected 2 agents with tag 'dev', got %d", len(agents))
	}

	// Filter by status
	status := StatusCreated
	agents, err = store.List(&ListFilter{Status: &status})
	if err != nil {
		t.Fatalf("list by status: %v", err)
	}
	if len(agents) != 3 {
		t.Errorf("expected 3 created agents, got %d", len(agents))
	}
}

func TestUpdate(t *testing.T) {
	database, cleanup := setupTestDB(t)
	defer cleanup()
	store := NewStore(database)

	p, _ := store.Create(CreateParams{
		Name:     "Original",
		ToolType: ToolClaude,
		ModelID:  "m1",
	})

	newName := "Updated"
	newModel := "m2"
	newTags := []string{"new-tag"}
	updated, err := store.Update(p.ID, UpdateParams{
		Name:    &newName,
		ModelID: &newModel,
		Tags:    newTags,
	})
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if updated.Name != "Updated" {
		t.Errorf("name = %q, want %q", updated.Name, "Updated")
	}
	if updated.ModelID != "m2" {
		t.Errorf("modelId = %q, want %q", updated.ModelID, "m2")
	}
	if len(updated.Tags) != 1 || updated.Tags[0] != "new-tag" {
		t.Errorf("tags = %v, want [new-tag]", updated.Tags)
	}
}

func TestSetStatus(t *testing.T) {
	database, cleanup := setupTestDB(t)
	defer cleanup()
	store := NewStore(database)

	p, _ := store.Create(CreateParams{
		Name:     "Test",
		ToolType: ToolClaude,
		ModelID:  "m1",
	})

	if err := store.SetStatus(p.ID, StatusRunning); err != nil {
		t.Fatalf("set status: %v", err)
	}

	got, _ := store.Get(p.ID)
	if got.Status != StatusRunning {
		t.Errorf("status = %q, want %q", got.Status, StatusRunning)
	}
}

func TestDelete(t *testing.T) {
	database, cleanup := setupTestDB(t)
	defer cleanup()
	store := NewStore(database)

	p, _ := store.Create(CreateParams{
		Name:     "ToDelete",
		ToolType: ToolClaude,
		ModelID:  "m1",
	})

	if err := store.Delete(p.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}

	_, err := store.Get(p.ID)
	if err == nil {
		t.Error("expected error after delete")
	}
}

func TestDelete_NotFound(t *testing.T) {
	database, cleanup := setupTestDB(t)
	defer cleanup()
	store := NewStore(database)

	err := store.Delete("nonexistent")
	if err == nil {
		t.Error("expected error for deleting nonexistent agent")
	}
}

func TestCount(t *testing.T) {
	database, cleanup := setupTestDB(t)
	defer cleanup()
	store := NewStore(database)

	store.Create(CreateParams{Name: "A1", ToolType: ToolClaude, ModelID: "m1"})
	store.Create(CreateParams{Name: "A2", ToolType: ToolCodex, ModelID: "m2"})

	n, err := store.Count()
	if err != nil {
		t.Fatalf("count: %v", err)
	}
	if n != 2 {
		t.Errorf("count = %d, want 2", n)
	}
}

func TestCountByStatus(t *testing.T) {
	database, cleanup := setupTestDB(t)
	defer cleanup()
	store := NewStore(database)

	p1, _ := store.Create(CreateParams{Name: "A1", ToolType: ToolClaude, ModelID: "m1"})
	store.Create(CreateParams{Name: "A2", ToolType: ToolCodex, ModelID: "m2"})
	store.SetStatus(p1.ID, StatusRunning)

	counts, err := store.CountByStatus()
	if err != nil {
		t.Fatalf("count by status: %v", err)
	}
	if counts[StatusRunning] != 1 {
		t.Errorf("running = %d, want 1", counts[StatusRunning])
	}
	if counts[StatusCreated] != 1 {
		t.Errorf("created = %d, want 1", counts[StatusCreated])
	}
}

func TestBudgetFields(t *testing.T) {
	database, cleanup := setupTestDB(t)
	defer cleanup()
	store := NewStore(database)

	tokens := int64(100000)
	currency := 10.0
	p, err := store.Create(CreateParams{
		Name:                "Budget Agent",
		ToolType:            ToolClaude,
		ModelID:             "m1",
		BudgetLimitTokens:   &tokens,
		BudgetLimitCurrency: &currency,
		BudgetPeriod:        BudgetDaily,
		BudgetPolicy:        PolicyDegrade,
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if p.BudgetLimitTokens == nil || *p.BudgetLimitTokens != 100000 {
		t.Errorf("budget tokens = %v, want 100000", p.BudgetLimitTokens)
	}
	if p.BudgetLimitCurrency == nil || *p.BudgetLimitCurrency != 10.0 {
		t.Errorf("budget currency = %v, want 10.0", p.BudgetLimitCurrency)
	}
	if p.BudgetPeriod != BudgetDaily {
		t.Errorf("budget period = %q, want %q", p.BudgetPeriod, BudgetDaily)
	}
	if p.BudgetPolicy != PolicyDegrade {
		t.Errorf("budget policy = %q, want %q", p.BudgetPolicy, PolicyDegrade)
	}
}

func TestMultipleAgentsSameTool(t *testing.T) {
	database, cleanup := setupTestDB(t)
	defer cleanup()
	store := NewStore(database)

	// Create 3 Claude agents with different names — the core multi-instance capability
	a1, _ := store.Create(CreateParams{Name: "Claude-Frontend", ToolType: ToolClaude, ModelID: "claude-sonnet-4-6"})
	a2, _ := store.Create(CreateParams{Name: "Claude-Backend", ToolType: ToolClaude, ModelID: "claude-opus-4-6"})
	a3, _ := store.Create(CreateParams{Name: "Claude-Review", ToolType: ToolClaude, ModelID: "claude-haiku-4-5"})

	tt := ToolClaude
	agents, err := store.List(&ListFilter{ToolType: &tt})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(agents) != 3 {
		t.Fatalf("expected 3 claude agents, got %d", len(agents))
	}

	// Verify they have different IDs and models
	ids := map[string]bool{a1.ID: true, a2.ID: true, a3.ID: true}
	if len(ids) != 3 {
		t.Error("expected 3 unique IDs")
	}
	if a1.ModelID == a2.ModelID {
		t.Error("expected different models for different agents")
	}
}
