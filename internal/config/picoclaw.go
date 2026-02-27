package config

import "lurus-switch/internal/installer"

// PicoClawModel represents a model endpoint configuration for PicoClaw
type PicoClawModel struct {
	Name      string `json:"name"`
	APIBase   string `json:"api_base"`
	APIKey    string `json:"api_key"`
	ModelName string `json:"model_name"`
}

// PicoClawAgentDefaults defines default settings for PicoClaw agents
type PicoClawAgentDefaults struct {
	ModelName string `json:"model_name"`
}

// PicoClawAgentSettings defines agent-level configuration
type PicoClawAgentSettings struct {
	Defaults PicoClawAgentDefaults `json:"defaults"`
}

// PicoClawConfig represents PicoClaw CLI configuration (config.json)
type PicoClawConfig struct {
	ModelList []PicoClawModel       `json:"model_list"`
	Agents    PicoClawAgentSettings `json:"agents"`
}

// NewPicoClawConfig creates a new PicoClaw configuration with sensible defaults
func NewPicoClawConfig() *PicoClawConfig {
	return &PicoClawConfig{
		ModelList: []PicoClawModel{
			{
				Name:      "default",
				APIBase:   "",
				APIKey:    "",
				ModelName: installer.DefaultPicoClawModel,
			},
		},
		Agents: PicoClawAgentSettings{
			Defaults: PicoClawAgentDefaults{
				ModelName: installer.DefaultPicoClawModel,
			},
		},
	}
}
