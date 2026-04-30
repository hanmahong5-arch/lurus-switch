package agent

import (
	"os"
	"path/filepath"
	"testing"
)

func TestConfigManager_WriteReadJSON(t *testing.T) {
	dir, err := os.MkdirTemp("", "cfgmgr-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	mgr, err := NewConfigManager(dir)
	if err != nil {
		t.Fatalf("new config manager: %v", err)
	}

	type testConfig struct {
		Model  string `json:"model"`
		APIKey string `json:"apiKey"`
	}

	agentID := "agent-123"
	cfg := testConfig{Model: "claude-sonnet-4-6", APIKey: "sk-test"}

	path, err := mgr.WriteJSON(agentID, "settings.json", cfg)
	if err != nil {
		t.Fatalf("write: %v", err)
	}
	if filepath.Base(path) != "settings.json" {
		t.Errorf("path = %q, want settings.json", filepath.Base(path))
	}

	var loaded testConfig
	if err := mgr.ReadJSON(agentID, "settings.json", &loaded); err != nil {
		t.Fatalf("read: %v", err)
	}
	if loaded.Model != "claude-sonnet-4-6" {
		t.Errorf("model = %q, want %q", loaded.Model, "claude-sonnet-4-6")
	}
}

func TestConfigManager_Exists(t *testing.T) {
	dir, err := os.MkdirTemp("", "cfgmgr-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	mgr, _ := NewConfigManager(dir)

	if mgr.Exists("agent-xyz", "config.json") {
		t.Error("expected non-existent config")
	}

	mgr.WriteJSON("agent-xyz", "config.json", map[string]string{"a": "b"})

	if !mgr.Exists("agent-xyz", "config.json") {
		t.Error("expected config to exist after write")
	}
}

func TestConfigManager_Remove(t *testing.T) {
	dir, err := os.MkdirTemp("", "cfgmgr-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	mgr, _ := NewConfigManager(dir)
	mgr.WriteJSON("agent-del", "config.json", map[string]string{})

	if err := mgr.Remove("agent-del"); err != nil {
		t.Fatalf("remove: %v", err)
	}
	if mgr.Exists("agent-del", "config.json") {
		t.Error("expected config removed")
	}
}

func TestConfigManager_MultipleAgentsSameTool(t *testing.T) {
	dir, err := os.MkdirTemp("", "cfgmgr-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	mgr, _ := NewConfigManager(dir)

	// Write different configs for 3 "Claude" agents
	type cfg struct {
		Model string `json:"model"`
	}
	mgr.WriteJSON("agent-1", "settings.json", cfg{Model: "sonnet"})
	mgr.WriteJSON("agent-2", "settings.json", cfg{Model: "opus"})
	mgr.WriteJSON("agent-3", "settings.json", cfg{Model: "haiku"})

	// Verify isolation
	var c1, c2, c3 cfg
	mgr.ReadJSON("agent-1", "settings.json", &c1)
	mgr.ReadJSON("agent-2", "settings.json", &c2)
	mgr.ReadJSON("agent-3", "settings.json", &c3)

	if c1.Model != "sonnet" || c2.Model != "opus" || c3.Model != "haiku" {
		t.Errorf("configs not isolated: %q, %q, %q", c1.Model, c2.Model, c3.Model)
	}
}

func TestConfigFilename(t *testing.T) {
	cases := []struct {
		tool ToolType
		want string
	}{
		{ToolClaude, "settings.json"},
		{ToolCodex, "config.toml"},
		{ToolGemini, "settings.json"},
		{ToolOpenClaw, "openclaw.json"},
		{ToolZeroClaw, "config.toml"},
		{ToolPicoClaw, "config.json"},
		{ToolNullClaw, "config.json"},
	}
	for _, tc := range cases {
		got := ConfigFilename(tc.tool)
		if got != tc.want {
			t.Errorf("ConfigFilename(%q) = %q, want %q", tc.tool, got, tc.want)
		}
	}
}
