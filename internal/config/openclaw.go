package config

import "lurus-switch/internal/installer"

// OpenClawGateway defines the local gateway server settings for OpenClaw
type OpenClawGateway struct {
	Port      int    `json:"port"`
	AuthToken string `json:"auth_token"`
}

// OpenClawProvider defines the AI provider settings for OpenClaw
type OpenClawProvider struct {
	Type   string `json:"type"`    // "anthropic" | "openai" | "custom"
	APIKey string `json:"api_key"`
	Model  string `json:"model"`
}

// OpenClawChannels defines channel-level access policies
type OpenClawChannels struct {
	DMPolicy string `json:"dm_policy"` // "all" | "none" | "allowlist"
}

// OpenClawSkills defines which skills are enabled
type OpenClawSkills struct {
	Enabled []string `json:"enabled"`
}

// OpenClawConfig represents OpenClaw CLI configuration (openclaw.json)
type OpenClawConfig struct {
	Gateway  OpenClawGateway  `json:"gateway"`
	Provider OpenClawProvider `json:"provider"`
	Channels OpenClawChannels `json:"channels"`
	Skills   OpenClawSkills   `json:"skills"`
}

// NewOpenClawConfig creates a new OpenClaw configuration with sensible defaults
func NewOpenClawConfig() *OpenClawConfig {
	return &OpenClawConfig{
		Gateway: OpenClawGateway{
			Port: 18789,
		},
		Provider: OpenClawProvider{
			Type:  "anthropic",
			Model: installer.DefaultOpenClawModel,
		},
		Channels: OpenClawChannels{
			DMPolicy: "all",
		},
		Skills: OpenClawSkills{
			Enabled: []string{},
		},
	}
}
