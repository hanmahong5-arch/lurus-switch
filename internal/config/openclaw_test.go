package config

import (
	"encoding/json"
	"testing"

	"lurus-switch/internal/installer"
)

func TestNewOpenClawConfig(t *testing.T) {
	cfg := NewOpenClawConfig()

	if cfg == nil {
		t.Fatal("NewOpenClawConfig returned nil")
	}
	if cfg.Gateway.Port != 18789 {
		t.Errorf("expected gateway.port=18789, got %d", cfg.Gateway.Port)
	}
	if cfg.Provider.Type != "anthropic" {
		t.Errorf("expected provider.type=anthropic, got %q", cfg.Provider.Type)
	}
	if cfg.Provider.Model != installer.DefaultOpenClawModel {
		t.Errorf("expected provider.model=%q, got %q", installer.DefaultOpenClawModel, cfg.Provider.Model)
	}
	if cfg.Channels.DMPolicy != "all" {
		t.Errorf("expected channels.dm_policy=all, got %q", cfg.Channels.DMPolicy)
	}
	if cfg.Skills.Enabled == nil {
		t.Error("expected skills.enabled to be non-nil slice")
	}
}

func TestOpenClawConfig_JSONRoundTrip(t *testing.T) {
	original := &OpenClawConfig{
		Gateway: OpenClawGateway{
			Port:      18789,
			AuthToken: "token-abc",
		},
		Provider: OpenClawProvider{
			Type:   "anthropic",
			APIKey: "sk-ant-xyz",
			Model:  "claude-sonnet-4-20250514",
		},
		Channels: OpenClawChannels{
			DMPolicy: "allowlist",
		},
		Skills: OpenClawSkills{
			Enabled: []string{"web-search", "code-exec"},
		},
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("JSON marshal failed: %v", err)
	}

	var decoded OpenClawConfig
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("JSON unmarshal failed: %v", err)
	}

	if decoded.Gateway.AuthToken != original.Gateway.AuthToken {
		t.Errorf("gateway.auth_token mismatch: want %q, got %q", original.Gateway.AuthToken, decoded.Gateway.AuthToken)
	}
	if decoded.Provider.APIKey != original.Provider.APIKey {
		t.Errorf("provider.api_key mismatch: want %q, got %q", original.Provider.APIKey, decoded.Provider.APIKey)
	}
	if decoded.Channels.DMPolicy != original.Channels.DMPolicy {
		t.Errorf("channels.dm_policy mismatch: want %q, got %q", original.Channels.DMPolicy, decoded.Channels.DMPolicy)
	}
	if len(decoded.Skills.Enabled) != len(original.Skills.Enabled) {
		t.Errorf("skills.enabled length mismatch: want %d, got %d", len(original.Skills.Enabled), len(decoded.Skills.Enabled))
	}
}
