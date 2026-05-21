package main

import (
	"fmt"
	"time"

	"lurus-switch/internal/capability"
	"lurus-switch/internal/provider"
)

// Audit op names for custom-provider mutations.
//
// These ops are journaled for the audit trail but intentionally have NO
// registered undo handler: the API key is redacted out of the before/after
// snapshots (secrets must never enter the journal), so an "undo delete"
// could only restore a keyless provider — a half-broken revert. Better to
// keep the op non-reversible than to silently drop the user's key.
const (
	auditOpCustomProviderSave   = "provider.custom.save"
	auditOpCustomProviderDelete = "provider.custom.delete"
)

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

// ─── Custom (user-defined) providers ────────────────────────────────────

// ListCustomProviders returns all user-defined provider endpoints.
func (a *App) ListCustomProviders() ([]provider.CustomProvider, error) {
	if a.customProviderStore == nil {
		return []provider.CustomProvider{}, nil
	}
	return a.customProviderStore.List(), nil
}

// SaveCustomProvider creates or updates a user-defined provider. A blank ID
// creates a new entry; a known ID updates in place. Gated by option.write
// and journaled for undo.
func (a *App) SaveCustomProvider(p provider.CustomProvider) (saved provider.CustomProvider, err error) {
	if a.customProviderStore == nil {
		return provider.CustomProvider{}, fmt.Errorf("custom provider store not initialized")
	}
	target := p.ID
	if target == "" {
		target = p.Name
	}
	if err = a.requireAndAudit(capability.CapOptionWrite, auditOpCustomProviderSave, target, redactCustomProvider(p)); err != nil {
		return provider.CustomProvider{}, err
	}
	// Capture before-state for undo (empty when creating).
	var before any
	if p.ID != "" {
		if prev, ok := a.customProviderStore.Get(p.ID); ok {
			before = redactCustomProvider(prev)
		}
	}
	saved, err = a.customProviderStore.Save(p)
	a.recordOutcomeFull(auditOpCustomProviderSave, saved.ID, before, redactCustomProvider(saved), err)
	return saved, err
}

// DeleteCustomProvider removes a user-defined provider by ID.
func (a *App) DeleteCustomProvider(id string) (err error) {
	if a.customProviderStore == nil {
		return fmt.Errorf("custom provider store not initialized")
	}
	if err = a.requireAndAudit(capability.CapOptionWrite, auditOpCustomProviderDelete, id, nil); err != nil {
		return err
	}
	var before any
	if prev, ok := a.customProviderStore.Get(id); ok {
		before = redactCustomProvider(prev)
	}
	err = a.customProviderStore.Delete(id)
	a.recordOutcomeFull(auditOpCustomProviderDelete, id, before, nil, err)
	return err
}

// CustomProviderTestResult is the outcome of probing a provider endpoint.
type CustomProviderTestResult struct {
	OK        bool     `json:"ok"`
	Models    []string `json:"models"`
	LatencyMs int64    `json:"latencyMs"`
	Error     string   `json:"error,omitempty"`
}

// TestCustomProvider probes a provider's /v1/models endpoint and measures
// latency. It does NOT persist anything — the form calls it before saving
// so the user can verify connectivity. Read-only, so no audit/capability gate.
func (a *App) TestCustomProvider(p provider.CustomProvider) (*CustomProviderTestResult, error) {
	start := time.Now()
	models, err := provider.FetchModels(a.ctx, p.BaseURL, p.APIKey)
	latency := time.Since(start).Milliseconds()
	if err != nil {
		return &CustomProviderTestResult{OK: false, LatencyMs: latency, Error: err.Error()}, nil
	}
	return &CustomProviderTestResult{OK: true, Models: models, LatencyMs: latency}, nil
}

// redactCustomProvider strips the API key before it enters the audit log —
// the journal must never persist secrets in before/after snapshots.
func redactCustomProvider(p provider.CustomProvider) map[string]any {
	keyState := "absent"
	if p.APIKey != "" {
		keyState = "set"
	}
	return map[string]any{
		"id":            p.ID,
		"name":          p.Name,
		"baseUrl":       p.BaseURL,
		"apiKey":        keyState,
		"defaultModels": p.DefaultModels,
		"docsUrl":       p.DocsURL,
	}
}
