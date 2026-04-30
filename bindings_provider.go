package main

import "lurus-switch/internal/provider"

// GetProviderPresets returns all built-in provider presets (20+ providers).
func (a *App) GetProviderPresets() []provider.Preset {
	return provider.Presets()
}

// GetProviderPresetsByCategory returns presets filtered by category.
// Categories: "official", "china", "proxy", "cloud", "self-hosted"
func (a *App) GetProviderPresetsByCategory(category string) []provider.Preset {
	return provider.PresetsByCategory(category)
}

// GetProviderPreset returns a single provider preset by ID, or nil.
func (a *App) GetProviderPreset(id string) *provider.Preset {
	return provider.PresetByID(id)
}

// FetchProviderModels queries an OpenAI-compatible /v1/models endpoint to discover
// which model IDs the provider currently exposes. Used by the ProxyConfigPanel to
// auto-populate a model dropdown after the user sets baseURL + apiKey.
// Returns sorted, deduplicated model IDs. Empty apiKey is allowed (for local runtimes).
func (a *App) FetchProviderModels(baseURL, apiKey string) ([]string, error) {
	return provider.FetchModels(a.ctx, baseURL, apiKey)
}
