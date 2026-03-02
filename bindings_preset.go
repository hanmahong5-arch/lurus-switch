package main

import (
	"lurus-switch/internal/config"
	"lurus-switch/internal/preset"
)

// ============================
// Preset Methods (S3.4)
// ============================

// GetClaudePresets returns all available Claude configuration presets.
func (a *App) GetClaudePresets() []preset.Preset {
	return preset.ClaudePresets()
}

// ApplyClaudePreset returns a ClaudeConfig pre-filled with the given preset values.
// The caller is expected to validate and save the returned config.
func (a *App) ApplyClaudePreset(id string) (*config.ClaudeConfig, error) {
	return preset.ApplyClaudePreset(id)
}

// GetCodexPresets returns all available Codex configuration presets.
func (a *App) GetCodexPresets() []preset.Preset {
	return preset.CodexPresets()
}

// ApplyCodexPreset returns a CodexConfig pre-filled with the given preset values.
func (a *App) ApplyCodexPreset(id string) (*config.CodexConfig, error) {
	return preset.ApplyCodexPreset(id)
}

// GetGeminiPresets returns all available Gemini configuration presets.
func (a *App) GetGeminiPresets() []preset.Preset {
	return preset.GeminiPresets()
}

// ApplyGeminiPreset returns a GeminiConfig pre-filled with the given preset values.
func (a *App) ApplyGeminiPreset(id string) (*config.GeminiConfig, error) {
	return preset.ApplyGeminiPreset(id)
}
