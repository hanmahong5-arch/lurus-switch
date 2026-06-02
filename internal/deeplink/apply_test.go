package deeplink

import (
	"encoding/json"
	"strings"
	"testing"

	"lurus-switch/internal/mcp"
	"lurus-switch/internal/promptlib"
)

// -- stub stores --------------------------------------------------------

type fakeMCPStore struct {
	saved []mcp.MCPPreset
}

func (s *fakeMCPStore) SavePreset(p mcp.MCPPreset) error {
	s.saved = append(s.saved, p)
	return nil
}

type fakePromptStore struct {
	saved []promptlib.Prompt
}

func (s *fakePromptStore) SavePrompt(p promptlib.Prompt) error {
	s.saved = append(s.saved, p)
	return nil
}

// -- helpers ------------------------------------------------------------

func mustMarshal(t *testing.T, v any) json.RawMessage {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
	return json.RawMessage(b)
}

func makePayload(t *testing.T, typ string, v any) Payload {
	t.Helper()
	return Payload{Type: typ, Data: mustMarshal(t, v), Raw: "switch://import?type=" + typ}
}

// -- tests --------------------------------------------------------------

// TestApply_MCP_LandsPreset verifies that a valid mcp payload is decoded and
// saved to the MCP store.
func TestApply_MCP_LandsPreset(t *testing.T) {
	store := &fakeMCPStore{}
	p := makePayload(t, "mcp", map[string]any{
		"name":        "My Tool",
		"description": "A test tool",
		"server": map[string]any{
			"name":    "my-tool",
			"type":    "stdio",
			"command": "npx",
			"args":    []string{"-y", "my-tool"},
		},
		"tags": []string{"test"},
	})

	summary, err := Apply(p, store, nil)
	if err != nil {
		t.Fatalf("Apply mcp: %v", err)
	}
	if !strings.Contains(summary, "My Tool") {
		t.Errorf("summary = %q, want to contain %q", summary, "My Tool")
	}
	if len(store.saved) != 1 {
		t.Fatalf("saved count = %d, want 1", len(store.saved))
	}
	if store.saved[0].Name != "My Tool" {
		t.Errorf("Name = %q, want %q", store.saved[0].Name, "My Tool")
	}
	if store.saved[0].Server.Command != "npx" {
		t.Errorf("Server.Command = %q, want %q", store.saved[0].Server.Command, "npx")
	}
}

// TestApply_Prompt_LandsPrompt verifies that a valid prompt payload is decoded
// and saved to the prompt store.
func TestApply_Prompt_LandsPrompt(t *testing.T) {
	store := &fakePromptStore{}
	p := makePayload(t, "prompt", map[string]any{
		"name":        "Shared Prompt",
		"category":    "coding",
		"content":     "You are a helpful assistant.",
		"targetTools": []string{"all"},
		"tags":        []string{"shared"},
	})

	summary, err := Apply(p, nil, store)
	if err != nil {
		t.Fatalf("Apply prompt: %v", err)
	}
	if !strings.Contains(summary, "Shared Prompt") {
		t.Errorf("summary = %q, want to contain %q", summary, "Shared Prompt")
	}
	if len(store.saved) != 1 {
		t.Fatalf("saved count = %d, want 1", len(store.saved))
	}
	if store.saved[0].Name != "Shared Prompt" {
		t.Errorf("Name = %q, want %q", store.saved[0].Name, "Shared Prompt")
	}
	if store.saved[0].Content != "You are a helpful assistant." {
		t.Errorf("Content = %q, want match", store.saved[0].Content)
	}
}

// TestApply_UnknownType_Errors verifies that unknown types (including "skill"
// and arbitrary values) return a clear "not yet supported" error.
func TestApply_UnknownType_Errors(t *testing.T) {
	cases := []string{"skill", "unknown", "foo", ""}
	store := &fakeMCPStore{}
	pStore := &fakePromptStore{}

	for _, typ := range cases {
		t.Run(typ, func(t *testing.T) {
			p := Payload{Type: typ, Data: mustMarshal(t, map[string]any{"x": 1})}
			_, err := Apply(p, store, pStore)
			if err == nil {
				t.Fatalf("Apply(%q) should have errored", typ)
			}
			if !strings.Contains(err.Error(), "not yet supported") {
				t.Errorf("error %q does not mention 'not yet supported'", err.Error())
			}
		})
	}
}

// TestApply_MCP_MissingName_Errors verifies field validation before saving.
func TestApply_MCP_MissingName_Errors(t *testing.T) {
	store := &fakeMCPStore{}
	p := makePayload(t, "mcp", map[string]any{
		"description": "no name",
		"server":      map[string]any{"name": "x", "type": "stdio"},
	})
	_, err := Apply(p, store, nil)
	if err == nil {
		t.Fatal("expected error for missing name")
	}
	if !strings.Contains(err.Error(), "name") {
		t.Errorf("error %q should mention 'name'", err.Error())
	}
	if len(store.saved) != 0 {
		t.Error("nothing should have been saved")
	}
}

// TestApply_MCP_MissingServerType_Errors ensures server.type is required.
func TestApply_MCP_MissingServerType_Errors(t *testing.T) {
	store := &fakeMCPStore{}
	p := makePayload(t, "mcp", map[string]any{
		"name":   "no-type",
		"server": map[string]any{"name": "x"},
	})
	_, err := Apply(p, store, nil)
	if err == nil {
		t.Fatal("expected error for missing server.type")
	}
	if !strings.Contains(err.Error(), "server.type") {
		t.Errorf("error %q should mention 'server.type'", err.Error())
	}
}

// TestApply_Prompt_MissingContent_Errors ensures content is required.
func TestApply_Prompt_MissingContent_Errors(t *testing.T) {
	store := &fakePromptStore{}
	p := makePayload(t, "prompt", map[string]any{
		"name":     "no content",
		"category": "coding",
	})
	_, err := Apply(p, nil, store)
	if err == nil {
		t.Fatal("expected error for missing content")
	}
	if !strings.Contains(err.Error(), "content") {
		t.Errorf("error %q should mention 'content'", err.Error())
	}
	if len(store.saved) != 0 {
		t.Error("nothing should have been saved")
	}
}

// TestApply_MCP_NilStore_Errors ensures nil store is rejected with a clear
// error rather than a nil-pointer panic.
func TestApply_MCP_NilStore_Errors(t *testing.T) {
	p := makePayload(t, "mcp", map[string]any{
		"name":   "ok",
		"server": map[string]any{"name": "x", "type": "stdio"},
	})
	_, err := Apply(p, nil, nil)
	if err == nil {
		t.Fatal("expected error for nil mcp store")
	}
	if !strings.Contains(err.Error(), "not initialized") {
		t.Errorf("error %q should mention 'not initialized'", err.Error())
	}
}

// TestApply_Prompt_NilStore_Errors ensures nil store is rejected cleanly.
func TestApply_Prompt_NilStore_Errors(t *testing.T) {
	p := makePayload(t, "prompt", map[string]any{
		"name":    "ok",
		"content": "something",
	})
	_, err := Apply(p, nil, nil)
	if err == nil {
		t.Fatal("expected error for nil prompt store")
	}
	if !strings.Contains(err.Error(), "not initialized") {
		t.Errorf("error %q should mention 'not initialized'", err.Error())
	}
}
