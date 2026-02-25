package config

import (
	"encoding/json"
	"strings"
	"testing"
)

// === Constructor Tests ===

func TestNewClaudeConfig(t *testing.T) {
	cfg := NewClaudeConfig()

	if cfg == nil {
		t.Fatal("NewClaudeConfig should return non-nil config")
	}

	if cfg.Model == "" {
		t.Error("Model should have default value")
	}

	if cfg.MaxTokens <= 0 {
		t.Error("MaxTokens should have positive default value")
	}

	if cfg.Permissions.AllowBash != true {
		t.Error("AllowBash should be true by default")
	}

	if cfg.Sandbox.Type != "none" {
		t.Error("Sandbox type should be 'none' by default")
	}
}

func TestNewClaudeConfig_DefaultValues(t *testing.T) {
	cfg := NewClaudeConfig()

	// Core defaults
	if cfg.Model != "claude-sonnet-4-20250514" {
		t.Errorf("Expected model 'claude-sonnet-4-20250514', got '%s'", cfg.Model)
	}
	if cfg.MaxTokens != 8192 {
		t.Errorf("Expected maxTokens 8192, got %d", cfg.MaxTokens)
	}

	// Permissions defaults
	if !cfg.Permissions.AllowBash {
		t.Error("AllowBash should be true by default")
	}
	if !cfg.Permissions.AllowRead {
		t.Error("AllowRead should be true by default")
	}
	if !cfg.Permissions.AllowWrite {
		t.Error("AllowWrite should be true by default")
	}
	if cfg.Permissions.AllowWebFetch {
		t.Error("AllowWebFetch should be false by default")
	}

	// Sandbox defaults
	if cfg.Sandbox.Enabled {
		t.Error("Sandbox should be disabled by default")
	}
	if cfg.Sandbox.Type != "none" {
		t.Errorf("Expected sandbox type 'none', got '%s'", cfg.Sandbox.Type)
	}

	// Advanced defaults
	if cfg.Advanced.Verbose {
		t.Error("Verbose should be false by default")
	}
	if cfg.Advanced.DisableTelemetry {
		t.Error("DisableTelemetry should be false by default")
	}
	if cfg.Advanced.Timeout != 300 {
		t.Errorf("Expected timeout 300, got %d", cfg.Advanced.Timeout)
	}
}

// === JSON Serialization Tests ===

func TestClaudeConfigJSON(t *testing.T) {
	cfg := NewClaudeConfig()
	cfg.CustomInstructions = "Test instructions"

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal config: %v", err)
	}

	var decoded ClaudeConfig
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal config: %v", err)
	}

	if decoded.CustomInstructions != cfg.CustomInstructions {
		t.Error("CustomInstructions mismatch after round-trip")
	}
}

func TestClaudeConfig_JSONMarshal_AllFields(t *testing.T) {
	cfg := &ClaudeConfig{
		Model:              "claude-3-opus-20240229",
		CustomInstructions: "Be helpful and concise",
		APIKey:             "sk-ant-test-key",
		MaxTokens:          16384,
		Permissions: ClaudePermissions{
			AllowBash:           true,
			AllowRead:           true,
			AllowWrite:          false,
			AllowWebFetch:       true,
			TrustedDirectories:  []string{"/home/user", "/tmp"},
			AllowedBashCommands: []string{"git*", "bun*"},
			DeniedBashCommands:  []string{"rm -rf*"},
		},
		MCPServers: map[string]MCPServer{
			"fs": {
				Command: "mcp-fs",
				Args:    []string{"--root", "/"},
				Env:     map[string]string{"DEBUG": "true"},
			},
		},
		Sandbox: ClaudeSandbox{
			Enabled:     true,
			Type:        "docker",
			DockerImage: "ubuntu:22.04",
			Mounts: []SandboxMount{
				{Source: "/home", Destination: "/home", ReadOnly: false},
			},
		},
		Advanced: ClaudeAdvanced{
			Verbose:              true,
			DisableTelemetry:     true,
			APIEndpoint:          "https://custom.api.anthropic.com",
			Timeout:              600,
			ExperimentalFeatures: true,
		},
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal config: %v", err)
	}

	jsonStr := string(data)
	expectedFields := []string{
		"model", "customInstructions", "apiKey", "maxTokens",
		"permissions", "allowBash", "trustedDirectories",
		"mcpServers", "command", "args", "env",
		"sandbox", "dockerImage", "mounts",
		"advanced", "verbose", "apiEndpoint",
	}

	for _, field := range expectedFields {
		if !strings.Contains(jsonStr, field) {
			t.Errorf("JSON should contain field '%s'", field)
		}
	}
}

func TestClaudeConfig_JSONMarshal_OmitEmpty(t *testing.T) {
	cfg := &ClaudeConfig{
		Model: "claude-3-opus",
	}

	data, err := json.Marshal(cfg)
	if err != nil {
		t.Fatalf("Failed to marshal config: %v", err)
	}

	jsonStr := string(data)

	// Fields with omitempty should not appear when empty
	omitEmptyFields := []string{
		"customInstructions", "apiKey", "trustedDirectories",
		"allowedBashCommands", "deniedBashCommands", "mcpServers",
	}

	for _, field := range omitEmptyFields {
		if strings.Contains(jsonStr, field) {
			t.Errorf("JSON should not contain empty field '%s'", field)
		}
	}
}

func TestClaudeConfig_JSONUnmarshal_PartialData(t *testing.T) {
	jsonData := `{
		"model": "claude-3-opus",
		"maxTokens": 4096
	}`

	var cfg ClaudeConfig
	if err := json.Unmarshal([]byte(jsonData), &cfg); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if cfg.Model != "claude-3-opus" {
		t.Errorf("Expected model 'claude-3-opus', got '%s'", cfg.Model)
	}
	if cfg.MaxTokens != 4096 {
		t.Errorf("Expected maxTokens 4096, got %d", cfg.MaxTokens)
	}
	// Other fields should be zero values
	if cfg.CustomInstructions != "" {
		t.Error("CustomInstructions should be empty")
	}
}

func TestClaudeConfig_JSONRoundTrip_MCPServers(t *testing.T) {
	original := NewClaudeConfig()
	original.MCPServers = map[string]MCPServer{
		"fs": {
			Command: "mcp-server-fs",
			Args:    []string{"--root", "/home/user", "--verbose"},
			Env:     map[string]string{"DEBUG": "true", "LOG_LEVEL": "info"},
		},
		"git": {
			Command: "mcp-server-git",
			Args:    []string{"--repo", "."},
		},
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var decoded ClaudeConfig
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if len(decoded.MCPServers) != 2 {
		t.Errorf("Expected 2 MCP servers, got %d", len(decoded.MCPServers))
	}

	if fs, ok := decoded.MCPServers["fs"]; !ok {
		t.Error("fs server should exist")
	} else {
		if fs.Command != "mcp-server-fs" {
			t.Error("fs command mismatch")
		}
		if len(fs.Args) != 3 {
			t.Error("fs args length mismatch")
		}
		if fs.Env["DEBUG"] != "true" {
			t.Error("fs env DEBUG mismatch")
		}
	}
}

func TestClaudeConfig_JSONRoundTrip_SandboxMounts(t *testing.T) {
	original := NewClaudeConfig()
	original.Sandbox.Enabled = true
	original.Sandbox.Type = "docker"
	original.Sandbox.Mounts = []SandboxMount{
		{Source: "/home/user/code", Destination: "/code", ReadOnly: false},
		{Source: "/etc/config", Destination: "/config", ReadOnly: true},
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var decoded ClaudeConfig
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if len(decoded.Sandbox.Mounts) != 2 {
		t.Errorf("Expected 2 mounts, got %d", len(decoded.Sandbox.Mounts))
	}

	if decoded.Sandbox.Mounts[0].Source != "/home/user/code" {
		t.Error("First mount source mismatch")
	}
	if decoded.Sandbox.Mounts[1].ReadOnly != true {
		t.Error("Second mount readOnly mismatch")
	}
}

// === Permissions Tests ===

func TestClaudePermissions_Defaults(t *testing.T) {
	cfg := NewClaudeConfig()

	if !cfg.Permissions.AllowBash {
		t.Error("AllowBash should be true by default")
	}
	if !cfg.Permissions.AllowRead {
		t.Error("AllowRead should be true by default")
	}
	if !cfg.Permissions.AllowWrite {
		t.Error("AllowWrite should be true by default")
	}
	if cfg.Permissions.AllowWebFetch {
		t.Error("AllowWebFetch should be false by default")
	}
}

func TestClaudePermissions_Serialization(t *testing.T) {
	perms := ClaudePermissions{
		AllowBash:           true,
		AllowRead:           true,
		AllowWrite:          false,
		AllowWebFetch:       true,
		TrustedDirectories:  []string{"/home", "/tmp", "/var/log"},
		AllowedBashCommands: []string{"git*", "bun*"},
		DeniedBashCommands:  []string{"rm*", "sudo*", "chmod*"},
	}

	data, err := json.Marshal(perms)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var decoded ClaudePermissions
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if decoded.AllowBash != true {
		t.Error("AllowBash mismatch")
	}
	if decoded.AllowWrite != false {
		t.Error("AllowWrite mismatch")
	}
	if len(decoded.TrustedDirectories) != 3 {
		t.Error("TrustedDirectories length mismatch")
	}
	if len(decoded.AllowedBashCommands) != 2 {
		t.Error("AllowedBashCommands length mismatch")
	}
}

// === MCP Server Tests ===

func TestMCPServer_Serialization(t *testing.T) {
	server := MCPServer{
		Command: "/usr/local/bin/mcp-server",
		Args:    []string{"--port", "8080", "--verbose"},
		Env: map[string]string{
			"API_KEY": "secret123",
			"DEBUG":   "true",
		},
	}

	data, err := json.Marshal(server)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var decoded MCPServer
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if decoded.Command != server.Command {
		t.Error("Command mismatch")
	}
	if len(decoded.Args) != 3 {
		t.Error("Args length mismatch")
	}
	if decoded.Env["API_KEY"] != "secret123" {
		t.Error("Env API_KEY mismatch")
	}
}

func TestMCPServer_EmptyOptionalFields(t *testing.T) {
	server := MCPServer{
		Command: "simple-server",
	}

	data, err := json.Marshal(server)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	// Args and Env should be omitted
	jsonStr := string(data)
	if strings.Contains(jsonStr, "args") {
		t.Error("Empty args should be omitted")
	}
	if strings.Contains(jsonStr, "env") {
		t.Error("Empty env should be omitted")
	}
}

// === Sandbox Mount Tests ===

func TestSandboxMount_Serialization(t *testing.T) {
	mount := SandboxMount{
		Source:      "/home/user/code",
		Destination: "/code",
		ReadOnly:    true,
	}

	data, err := json.Marshal(mount)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var decoded SandboxMount
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if decoded.Source != mount.Source {
		t.Error("Source mismatch")
	}
	if decoded.Destination != mount.Destination {
		t.Error("Destination mismatch")
	}
	if decoded.ReadOnly != mount.ReadOnly {
		t.Error("ReadOnly mismatch")
	}
}

func TestSandboxMount_ReadWriteDefault(t *testing.T) {
	mount := SandboxMount{
		Source:      "/tmp",
		Destination: "/tmp",
		// ReadOnly not specified, should default to false
	}

	data, err := json.Marshal(mount)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var decoded SandboxMount
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if decoded.ReadOnly != false {
		t.Error("ReadOnly should default to false")
	}
}

// === Advanced Settings Tests ===

func TestClaudeAdvanced_Serialization(t *testing.T) {
	advanced := ClaudeAdvanced{
		Verbose:              true,
		DisableTelemetry:     true,
		APIEndpoint:          "https://custom.api.anthropic.com",
		Timeout:              600,
		ExperimentalFeatures: true,
	}

	data, err := json.Marshal(advanced)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var decoded ClaudeAdvanced
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if decoded.Verbose != true {
		t.Error("Verbose mismatch")
	}
	if decoded.DisableTelemetry != true {
		t.Error("DisableTelemetry mismatch")
	}
	if decoded.APIEndpoint != "https://custom.api.anthropic.com" {
		t.Error("APIEndpoint mismatch")
	}
	if decoded.Timeout != 600 {
		t.Error("Timeout mismatch")
	}
	if decoded.ExperimentalFeatures != true {
		t.Error("ExperimentalFeatures mismatch")
	}
}

func TestClaudeAdvanced_DefaultEndpoint(t *testing.T) {
	cfg := NewClaudeConfig()

	// Default should have empty APIEndpoint (uses default)
	if cfg.Advanced.APIEndpoint != "" {
		t.Error("APIEndpoint should be empty by default")
	}
}

// === Sandbox Tests ===

func TestClaudeSandbox_DockerConfig(t *testing.T) {
	sandbox := ClaudeSandbox{
		Enabled:     true,
		Type:        "docker",
		DockerImage: "ubuntu:22.04",
		Mounts: []SandboxMount{
			{Source: "/home", Destination: "/home", ReadOnly: false},
		},
	}

	data, err := json.Marshal(sandbox)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var decoded ClaudeSandbox
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if decoded.Type != "docker" {
		t.Error("Type mismatch")
	}
	if decoded.DockerImage != "ubuntu:22.04" {
		t.Error("DockerImage mismatch")
	}
	if len(decoded.Mounts) != 1 {
		t.Error("Mounts length mismatch")
	}
}

func TestClaudeSandbox_WSLConfig(t *testing.T) {
	sandbox := ClaudeSandbox{
		Enabled: true,
		Type:    "wsl",
	}

	data, err := json.Marshal(sandbox)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var decoded ClaudeSandbox
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if decoded.Type != "wsl" {
		t.Error("Type mismatch")
	}
}
