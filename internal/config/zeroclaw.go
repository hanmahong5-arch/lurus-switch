package config

import "lurus-switch/internal/installer"

// ZeroClawProvider defines the AI provider settings for ZeroClaw
type ZeroClawProvider struct {
	Type    string `toml:"type" json:"type"`       // "anthropic" | "openai" | "custom"
	APIKey  string `toml:"api_key" json:"api_key"`
	Model   string `toml:"model" json:"model"`
	BaseURL string `toml:"base_url" json:"base_url"`
}

// ZeroClawGateway defines the local gateway server settings
type ZeroClawGateway struct {
	Host      string `toml:"host" json:"host"`
	Port      int    `toml:"port" json:"port"`
	AuthToken string `toml:"auth_token" json:"auth_token"`
}

// ZeroClawMemory defines the memory/persistence backend settings
type ZeroClawMemory struct {
	Backend string `toml:"backend" json:"backend"` // "sqlite"
	Path    string `toml:"path" json:"path"`
}

// ZeroClawSecurity defines security-related settings
type ZeroClawSecurity struct {
	Sandbox  bool `toml:"sandbox" json:"sandbox"`
	AuditLog bool `toml:"audit_log" json:"audit_log"`
}

// ZeroClawConfig represents ZeroClaw CLI configuration (config.toml)
type ZeroClawConfig struct {
	Provider ZeroClawProvider `toml:"provider" json:"provider"`
	Gateway  ZeroClawGateway  `toml:"gateway" json:"gateway"`
	Memory   ZeroClawMemory   `toml:"memory" json:"memory"`
	Security ZeroClawSecurity `toml:"security" json:"security"`
}

// NewZeroClawConfig creates a new ZeroClaw configuration with sensible defaults
func NewZeroClawConfig() *ZeroClawConfig {
	return &ZeroClawConfig{
		Provider: ZeroClawProvider{
			Type:  "anthropic",
			Model: installer.DefaultZeroClawModel,
		},
		Gateway: ZeroClawGateway{
			Host: "127.0.0.1",
			Port: 8765,
		},
		Memory: ZeroClawMemory{
			Backend: "sqlite",
		},
		Security: ZeroClawSecurity{
			Sandbox:  false,
			AuditLog: false,
		},
	}
}
