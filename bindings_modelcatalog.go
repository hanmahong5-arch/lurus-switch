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
	evtModelTestProgress = "model:test:progress"
	evtModelTestDone     = "model:test:done"
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
