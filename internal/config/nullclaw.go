package config

import "lurus-switch/internal/installer"

// NullClawModel represents a model endpoint configuration for NullClaw
type NullClawModel struct {
	Name      string `json:"name"`
	APIBase   string `json:"api_base"`
	APIKey    string `json:"api_key"`
	ModelName string `json:"model_name"`
}

// NullClawAgentDefaults defines default settings for NullClaw agents
type NullClawAgentDefaults struct {
	ModelName string `json:"model_name"`
}

// NullClawAgentSettings defines agent-level configuration
type NullClawAgentSettings struct {
	Defaults NullClawAgentDefaults `json:"defaults"`
}

// NullClawConfig represents NullClaw CLI configuration (config.json)
type NullClawConfig struct {
	ModelList []NullClawModel       `json:"model_list"`
	Agents    NullClawAgentSettings `json:"agents"`
}

// NewNullClawConfig creates a new NullClaw configuration with sensible defaults
func NewNullClawConfig() *NullClawConfig {
	return &NullClawConfig{
		ModelList: []NullClawModel{
			{
				Name:      "code-switch",
				APIBase:   "",
				APIKey:    "",
				ModelName: installer.DefaultNullClawModel,
			},
		},
		Agents: NullClawAgentSettings{
			Defaults: NullClawAgentDefaults{
				ModelName: installer.DefaultNullClawModel,
			},
		},
	}
}
