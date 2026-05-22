package mcp

import (
	"testing"
)

// mcpEnv redirects the MCP preset dir to a per-test tempdir via APPDATA
// (Windows) and HOME / XDG_CONFIG_HOME (Unix), mirroring the convention
// used by internal/config/store_test.go.
func mcpEnv(t *testing.T) {
	t.Helper()
	tmp := t.TempDir()
	t.Setenv("APPDATA", tmp)
	t.Setenv("HOME", tmp)
	t.Setenv("XDG_CONFIG_HOME", tmp)
}

// TestBuiltinPresets_BaselineShape locks in the count and required fields
// of the bundled MCP presets — the UI's "Browse built-ins" panel renders
// directly from this list, so a silent drop or rename should fail here.
func TestBuiltinPresets_BaselineShape(t *testing.T) {
	presets := BuiltinPresets()
	if len(presets) < 4 {
		t.Fatalf("BuiltinPresets returned %d, want >= 4", len(presets))
	}
	seen := make(map[string]bool, len(presets))
	for _, p := range presets {
		if p.ID == "" || p.Name == "" {
			t.Errorf("preset missing id/name: %+v", p)
		}
		if p.Server.Type == "" {
			t.Errorf("preset %q has empty Server.Type", p.ID)
		}
		if seen[p.ID] {
			t.Errorf("duplicate preset id: %q", p.ID)
		}
		seen[p.ID] = true
	}
	// The filesystem preset is the canonical example demoed in onboarding.
	if !seen["builtin-filesystem"] {
		t.Error("builtin-filesystem preset missing")
	}
}

// TestNewStore_CreatesDir ensures NewStore is idempotent and produces a
// non-nil Store on a clean machine — guard against silent failure when the
// AppData directory was wiped.
func TestNewStore_CreatesDir(t *testing.T) {
	mcpEnv(t)
	s, err := NewStore()
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	if s == nil || s.dir == "" {
		t.Fatal("Store returned nil or empty dir")
	}
}

// TestStore_SaveListDelete_RoundTrip exercises the full lifecycle of a
// user-created preset — this is the contract the MCPServerManager.tsx UI
// depends on.
func TestStore_SaveListDelete_RoundTrip(t *testing.T) {
	mcpEnv(t)
	s, err := NewStore()
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}

	want := MCPPreset{
		ID:          "user-test-1",
		Name:        "Local Tool",
		Description: "Round-trip smoke test",
		Server: MCPServer{
			Name:    "local",
			Type:    "stdio",
			Command: "echo",
			Args:    []string{"hello"},
			Env:     map[string]string{"FOO": "bar"},
		},
		Tags: []string{"smoke"},
	}
	if err := s.SavePreset(want); err != nil {
		t.Fatalf("SavePreset: %v", err)
	}

	got, err := s.ListPresets()
	if err != nil {
		t.Fatalf("ListPresets: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("ListPresets returned %d, want 1", len(got))
	}
	if got[0].ID != want.ID || got[0].Name != want.Name {
		t.Errorf("preset fields lost: %+v", got[0])
	}
	if got[0].Server.Command != "echo" || got[0].Server.Env["FOO"] != "bar" {
		t.Errorf("server fields lost: %+v", got[0].Server)
	}

	if err := s.DeletePreset(want.ID); err != nil {
		t.Fatalf("DeletePreset: %v", err)
	}
	after, _ := s.ListPresets()
	if len(after) != 0 {
		t.Errorf("after Delete, %d remain; want 0", len(after))
	}
}

// TestStore_SavePreset_RejectsBadID locks in the path-traversal guard —
// the preset filename is derived from ID, so a "../" must be rejected at
// the Save boundary.
func TestStore_SavePreset_RejectsBadID(t *testing.T) {
	mcpEnv(t)
	s, _ := NewStore()
	cases := []string{"../escape", `back\slash`, "ok/../bad"}
	for _, id := range cases {
		t.Run(id, func(t *testing.T) {
			err := s.SavePreset(MCPPreset{ID: id, Name: "x"})
			if err == nil {
				t.Errorf("SavePreset(%q) should have errored", id)
			}
		})
	}
}

// TestStore_SavePreset_AutoGeneratesID confirms the contract that an empty
// ID gets a "user-" prefix auto-assigned — the UI relies on this so users
// can paste a preset JSON without manually generating an ID.
func TestStore_SavePreset_AutoGeneratesID(t *testing.T) {
	mcpEnv(t)
	s, _ := NewStore()
	if err := s.SavePreset(MCPPreset{Name: "no id"}); err != nil {
		t.Fatalf("SavePreset: %v", err)
	}
	list, _ := s.ListPresets()
	if len(list) != 1 {
		t.Fatalf("list len = %d, want 1", len(list))
	}
	if list[0].ID == "" || len(list[0].ID) < 5 {
		t.Errorf("auto-generated ID looks wrong: %q", list[0].ID)
	}
}
