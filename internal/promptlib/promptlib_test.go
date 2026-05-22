package promptlib

import (
	"strings"
	"testing"
)

// promptEnv pins the prompt-library base dir to a per-test tempdir via
// APPDATA / HOME / XDG_CONFIG_HOME, matching the repo-wide pattern in
// internal/config/store_test.go.
func promptEnv(t *testing.T) {
	t.Helper()
	tmp := t.TempDir()
	t.Setenv("APPDATA", tmp)
	t.Setenv("HOME", tmp)
	t.Setenv("XDG_CONFIG_HOME", tmp)
}

// TestGetBuiltinPrompts_BaselineShape locks in the count and required
// descriptor fields for the bundled prompt library — the PromptLibrary
// drawer in the UI renders directly off this slice.
func TestGetBuiltinPrompts_BaselineShape(t *testing.T) {
	prompts := GetBuiltinPrompts()
	if len(prompts) < 6 {
		t.Fatalf("GetBuiltinPrompts returned %d, want >= 6", len(prompts))
	}
	seen := make(map[string]bool, len(prompts))
	for _, p := range prompts {
		if p.ID == "" || p.Name == "" || p.Content == "" {
			t.Errorf("builtin prompt missing required field: %+v", p)
		}
		if seen[p.ID] {
			t.Errorf("duplicate builtin prompt ID: %q", p.ID)
		}
		seen[p.ID] = true
	}
	// The code-review prompt is the canonical demo shown on first launch.
	if !seen["builtin-code-review"] {
		t.Error("builtin-code-review prompt missing")
	}
}

// TestNewStore_CreatesDir captures the contract that NewStore is
// idempotent and returns a usable Store on a clean machine.
func TestNewStore_CreatesDir(t *testing.T) {
	promptEnv(t)
	s, err := NewStore()
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	if s == nil || s.dir == "" {
		t.Fatal("Store nil or empty dir")
	}
}

// TestStore_SaveGetList_RoundTrip exercises the full lifecycle of a user
// prompt — guards against a future field rename silently dropping data.
func TestStore_SaveGetList_RoundTrip(t *testing.T) {
	promptEnv(t)
	s, err := NewStore()
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}

	want := Prompt{
		Name:        "Smoke",
		Category:    "coding",
		Tags:        []string{"smoke"},
		Content:     "you are a tester",
		TargetTools: []string{"claude"},
	}
	if err := s.SavePrompt(want); err != nil {
		t.Fatalf("SavePrompt: %v", err)
	}

	list, err := s.ListPrompts("")
	if err != nil {
		t.Fatalf("ListPrompts: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("list len = %d, want 1", len(list))
	}
	got := list[0]
	if got.Name != want.Name || got.Content != want.Content {
		t.Errorf("name/content lost: %+v", got)
	}
	if got.ID == "" {
		t.Error("ID not auto-generated on Save")
	}
	if got.CreatedAt == "" || got.UpdatedAt == "" {
		t.Errorf("timestamps not set: created=%q updated=%q", got.CreatedAt, got.UpdatedAt)
	}

	// Get by ID round-trips
	fetched, err := s.GetPrompt(got.ID)
	if err != nil {
		t.Fatalf("GetPrompt: %v", err)
	}
	if fetched.Content != want.Content {
		t.Errorf("GetPrompt content mismatch: %q vs %q", fetched.Content, want.Content)
	}

	// Category filter excludes non-matching prompts
	other, _ := s.ListPrompts("writing")
	if len(other) != 0 {
		t.Errorf("filter mismatch returned %d, want 0", len(other))
	}
}

// TestStore_DeletePrompt_RemovesEntry — the UI's trash icon depends on
// Delete actually clearing the JSON file from disk.
func TestStore_DeletePrompt_RemovesEntry(t *testing.T) {
	promptEnv(t)
	s, _ := NewStore()
	_ = s.SavePrompt(Prompt{Name: "to-delete", Content: "x"})
	list, _ := s.ListPrompts("")
	if len(list) != 1 {
		t.Fatalf("expected 1 prompt, got %d", len(list))
	}
	if err := s.DeletePrompt(list[0].ID); err != nil {
		t.Fatalf("DeletePrompt: %v", err)
	}
	after, _ := s.ListPrompts("")
	if len(after) != 0 {
		t.Errorf("after Delete, %d remain", len(after))
	}
	// GetPrompt on a deleted ID must return an error, not a zero-value Prompt.
	if _, err := s.GetPrompt(list[0].ID); err == nil {
		t.Error("GetPrompt on deleted ID should error")
	}
}

// TestValidateID_RejectsBadInput pins the path-traversal guard — the ID
// is used as a filename so "../" must be rejected at every entry point.
func TestValidateID_RejectsBadInput(t *testing.T) {
	cases := []struct {
		in      string
		wantErr bool
	}{
		{"normal-id", false},
		{"prompt-123", false},
		{"", true},
		{"../bad", true},
		{`back\slash`, true},
		{"has/slash", true},
		{"..", true},
	}
	for _, tc := range cases {
		t.Run(tc.in, func(t *testing.T) {
			err := validateID(tc.in)
			if (err != nil) != tc.wantErr {
				t.Errorf("validateID(%q) err = %v, wantErr %v", tc.in, err, tc.wantErr)
			}
		})
	}
}

// TestStore_ExportImport_RoundTrip locks in the JSON-array export format
// the user uses to back up their prompt library between machines.
//
// Note: SavePrompt auto-generates IDs from time.Now().UnixMilli(); two
// rapid back-to-back saves can collide, so the test sets explicit IDs.
// That's also closer to the real flow — the UI passes an ID when editing.
func TestStore_ExportImport_RoundTrip(t *testing.T) {
	promptEnv(t)
	s, _ := NewStore()
	if err := s.SavePrompt(Prompt{ID: "p-one", Name: "one", Content: "alpha", Category: "coding"}); err != nil {
		t.Fatalf("Save 1: %v", err)
	}
	if err := s.SavePrompt(Prompt{ID: "p-two", Name: "two", Content: "beta", Category: "writing"}); err != nil {
		t.Fatalf("Save 2: %v", err)
	}

	export, err := s.ExportAll()
	if err != nil {
		t.Fatalf("ExportAll: %v", err)
	}
	if !strings.Contains(export, "alpha") || !strings.Contains(export, "beta") {
		t.Errorf("export missing payload: %s", export)
	}

	// Import into a fresh store
	promptEnv(t)
	s2, _ := NewStore()
	n, err := s2.ImportFromJSON(export)
	if err != nil {
		t.Fatalf("ImportFromJSON: %v", err)
	}
	if n != 2 {
		t.Errorf("import count = %d, want 2", n)
	}
	list, _ := s2.ListPrompts("")
	if len(list) != 2 {
		t.Errorf("imported list len = %d, want 2", len(list))
	}
}
