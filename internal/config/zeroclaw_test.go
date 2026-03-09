package config

import (
	"strings"
	"testing"

	"github.com/BurntSushi/toml"

	"lurus-switch/internal/installer"
)

func TestNewZeroClawConfig(t *testing.T) {
	cfg := NewZeroClawConfig()

	if cfg == nil {
		t.Fatal("NewZeroClawConfig returned nil")
	}
	if cfg.Provider.Type != "anthropic" {
		t.Errorf("expected provider.type=anthropic, got %q", cfg.Provider.Type)
	}
	if cfg.Provider.Model != installer.DefaultZeroClawModel {
		t.Errorf("expected provider.model=%q, got %q", installer.DefaultZeroClawModel, cfg.Provider.Model)
	}
	if cfg.Gateway.Port != 8765 {
		t.Errorf("expected gateway.port=8765, got %d", cfg.Gateway.Port)
	}
	if cfg.Memory.Backend != "sqlite" {
		t.Errorf("expected memory.backend=sqlite, got %q", cfg.Memory.Backend)
	}
}

func TestZeroClawConfig_TOMLRoundTrip(t *testing.T) {
	original := &ZeroClawConfig{
		Provider: ZeroClawProvider{
			Type:   "anthropic",
			APIKey: "sk-ant-test",
			Model:  "claude-sonnet-4-20250514",
		},
		Gateway: ZeroClawGateway{
			Host: "127.0.0.1",
			Port: 8765,
		},
		Memory: ZeroClawMemory{
			Backend: "sqlite",
			Path:    "/tmp/test.db",
		},
		Security: ZeroClawSecurity{
			Sandbox:  true,
			AuditLog: false,
		},
	}

	// Encode to TOML
	var buf strings.Builder
	enc := toml.NewEncoder(&buf)
	if err := enc.Encode(original); err != nil {
		t.Fatalf("TOML encode failed: %v", err)
	}
	tomlStr := buf.String()

	// Decode back
	var decoded ZeroClawConfig
	if err := toml.Unmarshal([]byte(tomlStr), &decoded); err != nil {
		t.Fatalf("TOML decode failed: %v", err)
	}

	if decoded.Provider.APIKey != original.Provider.APIKey {
		t.Errorf("provider.api_key mismatch: want %q, got %q", original.Provider.APIKey, decoded.Provider.APIKey)
	}
	if decoded.Gateway.Port != original.Gateway.Port {
		t.Errorf("gateway.port mismatch: want %d, got %d", original.Gateway.Port, decoded.Gateway.Port)
	}
	if decoded.Memory.Path != original.Memory.Path {
		t.Errorf("memory.path mismatch: want %q, got %q", original.Memory.Path, decoded.Memory.Path)
	}
	if decoded.Security.Sandbox != original.Security.Sandbox {
		t.Errorf("security.sandbox mismatch: want %v, got %v", original.Security.Sandbox, decoded.Security.Sandbox)
	}
}
