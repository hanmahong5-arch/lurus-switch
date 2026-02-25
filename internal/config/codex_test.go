package config

import (
	"encoding/json"
	"testing"

	"github.com/BurntSushi/toml"
)

// === Constructor Tests ===

func TestNewCodexConfig(t *testing.T) {
	cfg := NewCodexConfig()

	if cfg == nil {
		t.Fatal("NewCodexConfig should return non-nil config")
	}
}

func TestNewCodexConfig_DefaultValues(t *testing.T) {
	cfg := NewCodexConfig()

	// Core defaults
	if cfg.Model != "o4-mini" {
		t.Errorf("Expected model 'o4-mini', got '%s'", cfg.Model)
	}
	if cfg.ApprovalMode != "suggest" {
		t.Errorf("Expected approvalMode 'suggest', got '%s'", cfg.ApprovalMode)
	}

	// Provider defaults
	if cfg.Provider.Type != "openai" {
		t.Errorf("Expected provider type 'openai', got '%s'", cfg.Provider.Type)
	}

	// Security defaults
	if cfg.Security.NetworkAccess != "local" {
		t.Errorf("Expected network access 'local', got '%s'", cfg.Security.NetworkAccess)
	}
	if len(cfg.Security.FileAccess.AllowedDirs) != 1 || cfg.Security.FileAccess.AllowedDirs[0] != "." {
		t.Error("Expected allowed dirs to be ['.']")
	}
	if len(cfg.Security.FileAccess.DeniedPatterns) != 3 {
		t.Error("Expected 3 default denied patterns")
	}
	if !cfg.Security.CommandExecution.Enabled {
		t.Error("Command execution should be enabled by default")
	}

	// MCP defaults
	if cfg.MCP.Enabled {
		t.Error("MCP should be disabled by default")
	}

	// Sandbox defaults
	if !cfg.Sandbox.Enabled {
		t.Error("Sandbox should be enabled by default")
	}
	if cfg.Sandbox.Type != "none" {
		t.Errorf("Expected sandbox type 'none', got '%s'", cfg.Sandbox.Type)
	}

	// History defaults
	if !cfg.History.Enabled {
		t.Error("History should be enabled by default")
	}
	if cfg.History.MaxEntries != 1000 {
		t.Errorf("Expected max entries 1000, got %d", cfg.History.MaxEntries)
	}
}

// === JSON Serialization Tests ===

func TestCodexConfig_JSONMarshal_AllFields(t *testing.T) {
	cfg := &CodexConfig{
		Model:        "o4-mini",
		APIKey:       "sk-test-key",
		ApprovalMode: "auto-edit",
		Provider: CodexProvider{
			Type:            "azure",
			BaseURL:         "https://my.azure.com",
			AzureDeployment: "deployment-1",
			AzureAPIVersion: "2024-01-01",
		},
		Security: CodexSecurity{
			NetworkAccess: "full",
			FileAccess: CodexFileAccess{
				AllowedDirs:    []string{"/home", "/tmp"},
				DeniedPatterns: []string{"*.secret"},
				ReadOnlyDirs:   []string{"/etc"},
			},
			CommandExecution: CodexCommandExecution{
				Enabled:         true,
				AllowedCommands: []string{"git*", "bun*"},
				DeniedCommands:  []string{"rm -rf*"},
			},
		},
		MCP: CodexMCP{
			Enabled: true,
			Servers: []CodexMCPServer{
				{
					Name:    "test-server",
					Command: "/usr/bin/mcp",
					Args:    []string{"--port", "3000"},
					Env:     map[string]string{"DEBUG": "true"},
				},
			},
		},
		Sandbox: CodexSandbox{
			Enabled: true,
			Type:    "seatbelt",
		},
		History: CodexHistory{
			Enabled:    true,
			FilePath:   "~/.codex/history.json",
			MaxEntries: 500,
		},
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal config: %v", err)
	}

	// Verify JSON contains all fields
	jsonStr := string(data)
	expectedFields := []string{
		"model", "apiKey", "approvalMode",
		"provider", "baseUrl", "azureDeployment",
		"security", "networkAccess", "fileAccess",
		"mcp", "servers",
		"sandbox", "history",
	}

	for _, field := range expectedFields {
		if !containsString(jsonStr, field) {
			t.Errorf("JSON should contain field '%s'", field)
		}
	}
}

func TestCodexConfig_JSONUnmarshal_PartialData(t *testing.T) {
	jsonData := `{
		"model": "gpt-4",
		"approvalMode": "full-auto"
	}`

	var cfg CodexConfig
	if err := json.Unmarshal([]byte(jsonData), &cfg); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if cfg.Model != "gpt-4" {
		t.Errorf("Expected model 'gpt-4', got '%s'", cfg.Model)
	}
	if cfg.ApprovalMode != "full-auto" {
		t.Errorf("Expected approvalMode 'full-auto', got '%s'", cfg.ApprovalMode)
	}
	// Other fields should be zero values
	if cfg.Provider.Type != "" {
		t.Errorf("Expected empty provider type, got '%s'", cfg.Provider.Type)
	}
}

func TestCodexConfig_JSONRoundTrip(t *testing.T) {
	original := NewCodexConfig()
	original.APIKey = "test-key-123"
	original.Provider.BaseURL = "https://custom.api.com"

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var decoded CodexConfig
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if decoded.Model != original.Model {
		t.Error("Model mismatch after round-trip")
	}
	if decoded.APIKey != original.APIKey {
		t.Error("APIKey mismatch after round-trip")
	}
	if decoded.Provider.BaseURL != original.Provider.BaseURL {
		t.Error("Provider.BaseURL mismatch after round-trip")
	}
}

// === TOML Serialization Tests ===

func TestCodexConfig_TOMLMarshal(t *testing.T) {
	cfg := NewCodexConfig()
	cfg.Model = "o4-mini"

	var buf []byte
	var err error

	// TOML encoding
	buf, err = toml.Marshal(cfg)
	if err != nil {
		t.Fatalf("Failed to marshal to TOML: %v", err)
	}

	tomlStr := string(buf)
	if !containsString(tomlStr, "model") {
		t.Error("TOML should contain 'model' field")
	}
}

func TestCodexConfig_TOMLUnmarshal(t *testing.T) {
	tomlData := `
model = "gpt-4"
api_key = "sk-test"
approval_mode = "full-auto"

[provider]
type = "openai"

[security]
network_access = "local"

[sandbox]
enabled = true
type = "landlock"

[history]
enabled = true
max_entries = 500
`

	var cfg CodexConfig
	if _, err := toml.Decode(tomlData, &cfg); err != nil {
		t.Fatalf("Failed to decode TOML: %v", err)
	}

	if cfg.Model != "gpt-4" {
		t.Errorf("Expected model 'gpt-4', got '%s'", cfg.Model)
	}
	if cfg.APIKey != "sk-test" {
		t.Errorf("Expected api_key 'sk-test', got '%s'", cfg.APIKey)
	}
	if cfg.ApprovalMode != "full-auto" {
		t.Errorf("Expected approval_mode 'full-auto', got '%s'", cfg.ApprovalMode)
	}
	if cfg.Sandbox.Type != "landlock" {
		t.Errorf("Expected sandbox type 'landlock', got '%s'", cfg.Sandbox.Type)
	}
}

// === Provider Tests ===

func TestCodexProvider_AzureConfig(t *testing.T) {
	provider := CodexProvider{
		Type:            "azure",
		BaseURL:         "https://my-resource.openai.azure.com",
		AzureDeployment: "gpt-4-deployment",
		AzureAPIVersion: "2024-02-01",
	}

	data, err := json.Marshal(provider)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var decoded CodexProvider
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if decoded.Type != "azure" {
		t.Error("Type mismatch")
	}
	if decoded.AzureDeployment != "gpt-4-deployment" {
		t.Error("AzureDeployment mismatch")
	}
}

func TestCodexProvider_OpenRouterConfig(t *testing.T) {
	provider := CodexProvider{
		Type:    "openrouter",
		BaseURL: "https://openrouter.ai/api/v1",
	}

	data, err := json.Marshal(provider)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	if !containsString(string(data), "openrouter") {
		t.Error("JSON should contain 'openrouter'")
	}
}

func TestCodexProvider_CustomConfig(t *testing.T) {
	provider := CodexProvider{
		Type:    "custom",
		BaseURL: "https://my-llm-proxy.internal.com/v1",
	}

	data, err := json.Marshal(provider)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	if !containsString(string(data), "custom") {
		t.Error("JSON should contain 'custom'")
	}
}

// === Security Tests ===

func TestCodexSecurity_AllFields(t *testing.T) {
	security := CodexSecurity{
		NetworkAccess: "full",
		FileAccess: CodexFileAccess{
			AllowedDirs:    []string{"/home/user/code"},
			DeniedPatterns: []string{"**/.env", "**/secrets/**"},
			ReadOnlyDirs:   []string{"/etc", "/usr"},
		},
		CommandExecution: CodexCommandExecution{
			Enabled:         true,
			AllowedCommands: []string{"git*", "bun*"},
			DeniedCommands:  []string{"rm*", "sudo*"},
		},
	}

	data, err := json.Marshal(security)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var decoded CodexSecurity
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if decoded.NetworkAccess != "full" {
		t.Error("NetworkAccess mismatch")
	}
	if len(decoded.FileAccess.AllowedDirs) != 1 {
		t.Error("AllowedDirs length mismatch")
	}
	if len(decoded.CommandExecution.AllowedCommands) != 2 {
		t.Error("AllowedCommands length mismatch")
	}
}

func TestCodexFileAccess_Patterns(t *testing.T) {
	fileAccess := CodexFileAccess{
		AllowedDirs:    []string{".", "./src", "./tests"},
		DeniedPatterns: []string{"**/.env*", "**/*.key", "**/secrets/**", "**/.git/**"},
		ReadOnlyDirs:   []string{"./node_modules", "./vendor"},
	}

	data, err := json.Marshal(fileAccess)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var decoded CodexFileAccess
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if len(decoded.AllowedDirs) != 3 {
		t.Errorf("Expected 3 allowed dirs, got %d", len(decoded.AllowedDirs))
	}
	if len(decoded.DeniedPatterns) != 4 {
		t.Errorf("Expected 4 denied patterns, got %d", len(decoded.DeniedPatterns))
	}
}

func TestCodexCommandExecution_Settings(t *testing.T) {
	cmdExec := CodexCommandExecution{
		Enabled:         false,
		AllowedCommands: []string{},
		DeniedCommands:  []string{"*"}, // Deny all
	}

	data, err := json.Marshal(cmdExec)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var decoded CodexCommandExecution
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if decoded.Enabled {
		t.Error("Enabled should be false")
	}
	if len(decoded.DeniedCommands) != 1 {
		t.Error("DeniedCommands length mismatch")
	}
}

// === MCP Tests ===

func TestCodexMCP_ServersArray(t *testing.T) {
	mcp := CodexMCP{
		Enabled: true,
		Servers: []CodexMCPServer{
			{
				Name:    "fs-server",
				Command: "mcp-server-fs",
				Args:    []string{"--root", "/home/user"},
			},
			{
				Name:    "git-server",
				Command: "mcp-server-git",
				Args:    []string{"--repo", "."},
				Env:     map[string]string{"GIT_DIR": ".git"},
			},
		},
	}

	data, err := json.Marshal(mcp)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var decoded CodexMCP
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if !decoded.Enabled {
		t.Error("MCP should be enabled")
	}
	if len(decoded.Servers) != 2 {
		t.Errorf("Expected 2 servers, got %d", len(decoded.Servers))
	}
	if decoded.Servers[1].Name != "git-server" {
		t.Error("Second server name mismatch")
	}
}

func TestCodexMCPServer_Serialization(t *testing.T) {
	server := CodexMCPServer{
		Name:    "test-mcp",
		Command: "/usr/local/bin/mcp-server",
		Args:    []string{"--verbose", "--port", "8080"},
		Env: map[string]string{
			"API_KEY": "secret123",
			"DEBUG":   "true",
		},
	}

	data, err := json.Marshal(server)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var decoded CodexMCPServer
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if decoded.Name != "test-mcp" {
		t.Error("Name mismatch")
	}
	if decoded.Command != "/usr/local/bin/mcp-server" {
		t.Error("Command mismatch")
	}
	if len(decoded.Args) != 3 {
		t.Error("Args length mismatch")
	}
	if decoded.Env["API_KEY"] != "secret123" {
		t.Error("Env API_KEY mismatch")
	}
}

// === Sandbox Tests ===

func TestCodexSandbox_Types(t *testing.T) {
	sandboxTypes := []struct {
		sandboxType string
		description string
	}{
		{"seatbelt", "macOS sandbox"},
		{"landlock", "Linux sandbox"},
		{"none", "No sandbox"},
	}

	for _, tc := range sandboxTypes {
		t.Run(tc.sandboxType, func(t *testing.T) {
			sandbox := CodexSandbox{
				Enabled: true,
				Type:    tc.sandboxType,
			}

			data, err := json.Marshal(sandbox)
			if err != nil {
				t.Fatalf("Failed to marshal: %v", err)
			}

			var decoded CodexSandbox
			if err := json.Unmarshal(data, &decoded); err != nil {
				t.Fatalf("Failed to unmarshal: %v", err)
			}

			if decoded.Type != tc.sandboxType {
				t.Errorf("Expected type '%s', got '%s'", tc.sandboxType, decoded.Type)
			}
		})
	}
}

// === History Tests ===

func TestCodexHistory_Settings(t *testing.T) {
	history := CodexHistory{
		Enabled:    true,
		FilePath:   "~/.config/codex/history.json",
		MaxEntries: 2000,
	}

	data, err := json.Marshal(history)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var decoded CodexHistory
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if !decoded.Enabled {
		t.Error("History should be enabled")
	}
	if decoded.FilePath != "~/.config/codex/history.json" {
		t.Error("FilePath mismatch")
	}
	if decoded.MaxEntries != 2000 {
		t.Error("MaxEntries mismatch")
	}
}

func TestCodexHistory_Disabled(t *testing.T) {
	history := CodexHistory{
		Enabled: false,
	}

	data, err := json.Marshal(history)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var decoded CodexHistory
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if decoded.Enabled {
		t.Error("History should be disabled")
	}
}

// === Helper Functions ===

func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstring(s, substr))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
