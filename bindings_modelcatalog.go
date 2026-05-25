package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"lurus-switch/internal/modelcatalog"

	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

const (
	modelHealthCacheFile = "model-health-cache.json"
	modelAuthCacheFile   = "model-auth-cache.json"
	evtModelTestProgress = "model:test:progress"
	evtModelTestDone     = "model:test:done"
	evtModelAuthProgress = "model:auth:progress"
	evtModelAuthDone     = "model:auth:done"
)

// buildHealthCheckEndpoints assembles the providers to probe: configured
// relay upstreams always, plus user-defined custom providers when requested.
// Built-in presets are intentionally excluded — they're templates without
// configured keys, so probing them would just report auth failures.
func (a *App) buildHealthCheckEndpoints(includeCustom bool) []modelcatalog.ProviderEndpoint {
	var eps []modelcatalog.ProviderEndpoint
	if a.relayStore != nil {
		if relays, err := a.relayStore.ListEndpoints(); err == nil {
			for _, r := range relays {
				if r.URL == "" {
					continue
				}
				eps = append(eps, modelcatalog.ProviderEndpoint{
					ID:      "relay:" + r.ID,
					Name:    r.Name,
					BaseURL: r.URL,
					APIKey:  r.APIKey,
				})
			}
		}
	}
	if includeCustom && a.customProviderStore != nil {
		for _, c := range a.customProviderStore.List() {
			eps = append(eps, modelcatalog.ProviderEndpoint{
				ID:            "custom:" + c.ID,
				Name:          c.Name,
				BaseURL:       c.BaseURL,
				APIKey:        c.APIKey,
				DefaultModels: c.DefaultModels,
			})
		}
	}
	return eps
}

// RunModelHealthCheck probes every configured provider's /v1/models endpoint
// concurrently, streaming each result to the frontend via the
// "model:test:progress" event and a final "model:test:done". Results are
// cached to disk for GetLastHealthCheckResults.
//
// This hits each provider's /v1/models once — it verifies the endpoint is up
// and lists models, NOT that each model can actually complete a chat. The UI
// must surface that distinction.
func (a *App) RunModelHealthCheck(includeCustom bool) error {
	if a.catalogTester == nil {
		return fmt.Errorf("model tester not initialized")
	}
	endpoints := a.buildHealthCheckEndpoints(includeCustom)
	if len(endpoints) == 0 {
		// Emit an immediate "done" with no results so the UI doesn't spin.
		if a.ctx != nil {
			wailsRuntime.EventsEmit(a.ctx, evtModelTestDone, []modelcatalog.TestResult{})
		}
		return nil
	}

	go safeGo("model-health-check", func() {
		ctx := a.ctx
		if ctx == nil {
			return
		}
		results := make([]modelcatalog.TestResult, 0, len(endpoints))
		for r := range a.catalogTester.RunHealthCheck(ctx, endpoints) {
			results = append(results, r)
			wailsRuntime.EventsEmit(ctx, evtModelTestProgress, r)
		}
		a.saveHealthCheckResults(results)
		wailsRuntime.EventsEmit(ctx, evtModelTestDone, results)
	})
	return nil
}

// GetLastHealthCheckResults returns the cached results from the most recent
// run, or an empty slice if none.
func (a *App) GetLastHealthCheckResults() []modelcatalog.TestResult {
	data, err := os.ReadFile(a.healthCachePath())
	if err != nil {
		return []modelcatalog.TestResult{}
	}
	var results []modelcatalog.TestResult
	if err := json.Unmarshal(data, &results); err != nil {
		return []modelcatalog.TestResult{}
	}
	return results
}

func (a *App) healthCachePath() string {
	return filepath.Join(appDataBaseDir(), modelHealthCacheFile)
}

func (a *App) saveHealthCheckResults(results []modelcatalog.TestResult) {
	data, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		return
	}
	_ = os.WriteFile(a.healthCachePath(), data, 0o600)
}

// RunModelAuthCheck probes (endpoint × model) authenticity. Streams
// per-endpoint progress over "model:auth:progress" and a final
// "model:auth:done" with the full result set.
//
// COST WARNING — each (endpoint, model) pair triggers ONE real chat
// completion. The probe payload is minimal (prompt="ping",
// max_tokens=1) but it still consumes upstream tokens. The UI must
// gate this behind an explicit user action and show the cost.
func (a *App) RunModelAuthCheck(includeCustom bool) error {
	endpoints := a.buildHealthCheckEndpoints(includeCustom)
	if len(endpoints) == 0 {
		if a.ctx != nil {
			wailsRuntime.EventsEmit(a.ctx, evtModelAuthDone, []modelcatalog.ModelAuthResult{})
		}
		return nil
	}

	go safeGo("model-auth-check", func() {
		ctx := a.ctx
		if ctx == nil {
			return
		}
		all := make([]modelcatalog.ModelAuthResult, 0, len(endpoints))
		for _, ep := range endpoints {
			models := a.authProbeModels(ep)
			if len(models) == 0 {
				continue
			}
			perEndpoint := modelcatalog.ProbeAuthenticity(ctx, ep, models)
			for _, r := range perEndpoint {
				all = append(all, r)
				wailsRuntime.EventsEmit(ctx, evtModelAuthProgress, r)
			}
		}
		a.saveAuthCheckResults(all)
		wailsRuntime.EventsEmit(ctx, evtModelAuthDone, all)
	})
	return nil
}

// GetLastModelAuthResults returns the cached results from the most
// recent authenticity sweep, or an empty slice if none.
func (a *App) GetLastModelAuthResults() []modelcatalog.ModelAuthResult {
	data, err := os.ReadFile(a.authCachePath())
	if err != nil {
		return []modelcatalog.ModelAuthResult{}
	}
	var results []modelcatalog.ModelAuthResult
	if err := json.Unmarshal(data, &results); err != nil {
		return []modelcatalog.ModelAuthResult{}
	}
	return results
}

func (a *App) authCachePath() string {
	return filepath.Join(appDataBaseDir(), modelAuthCacheFile)
}

func (a *App) saveAuthCheckResults(results []modelcatalog.ModelAuthResult) {
	data, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		return
	}
	_ = os.WriteFile(a.authCachePath(), data, 0o600)
}

// authProbeModels picks the list of models to probe on an endpoint.
// Prefers the endpoint's own DefaultModels (set on custom providers);
// falls back to the most recent /v1/models listing from the health
// check cache so the user doesn't have to manually enumerate.
func (a *App) authProbeModels(ep modelcatalog.ProviderEndpoint) []string {
	if len(ep.DefaultModels) > 0 {
		return ep.DefaultModels
	}
	for _, r := range a.GetLastHealthCheckResults() {
		if r.ProviderID == ep.ID && len(r.Models) > 0 {
			return r.Models
		}
	}
	return nil
}

