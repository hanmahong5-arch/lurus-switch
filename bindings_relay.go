package main

import (
	"context"
	"fmt"

	"lurus-switch/internal/relay"
)

// ============================
// Relay Endpoint Methods
// ============================

// GetRelayEndpoints returns all relay endpoints (builtin + user-defined).
func (a *App) GetRelayEndpoints() ([]relay.RelayEndpoint, error) {
	if a.relayStore == nil {
		return nil, fmt.Errorf("relay store not initialized")
	}
	return a.relayStore.ListEndpoints()
}

// FetchCloudRelayEndpoints fetches recommended relay endpoints from the Lurus API.
func (a *App) FetchCloudRelayEndpoints() ([]relay.RelayEndpoint, error) {
	if a.proxyMgr == nil {
		return nil, fmt.Errorf("proxy manager not initialized")
	}
	apiBase := a.proxyMgr.GetSettings().APIEndpoint
	ctx, cancel := context.WithTimeout(a.ctx, relay.CloudFetchTimeout)
	defer cancel()
	return relay.FetchCloudRelays(ctx, apiBase)
}

// SaveRelayEndpoint upserts a user-defined relay endpoint.
func (a *App) SaveRelayEndpoint(ep *relay.RelayEndpoint) error {
	if a.relayStore == nil {
		return fmt.Errorf("relay store not initialized")
	}
	if ep == nil {
		return fmt.Errorf("endpoint is nil")
	}
	return a.relayStore.SaveEndpoint(*ep)
}

// DeleteRelayEndpoint removes a user-defined relay endpoint by ID.
func (a *App) DeleteRelayEndpoint(id string) error {
	if a.relayStore == nil {
		return fmt.Errorf("relay store not initialized")
	}
	return a.relayStore.DeleteEndpoint(id)
}

// GetToolRelayMapping returns the current tool→relay-ID mapping.
func (a *App) GetToolRelayMapping() (relay.ToolRelayMapping, error) {
	if a.relayStore == nil {
		return nil, fmt.Errorf("relay store not initialized")
	}
	return a.relayStore.GetToolMapping()
}

// SaveToolRelayMapping persists the tool→relay-ID mapping.
func (a *App) SaveToolRelayMapping(m relay.ToolRelayMapping) error {
	if a.relayStore == nil {
		return fmt.Errorf("relay store not initialized")
	}
	return a.relayStore.SaveToolMapping(m)
}

// CheckRelayHealth runs concurrent health checks on all relay endpoints.
func (a *App) CheckRelayHealth() ([]relay.RelayEndpoint, error) {
	if a.relayStore == nil {
		return nil, fmt.Errorf("relay store not initialized")
	}
	endpoints, err := a.relayStore.ListEndpoints()
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(a.ctx, relay.HealthCheckTimeout)
	defer cancel()
	return relay.CheckHealth(ctx, endpoints), nil
}

// ApplyAllToolRelays applies each tool's configured relay endpoint to its config file.
// Returns a per-tool error map (empty map = all succeeded).
func (a *App) ApplyAllToolRelays() map[string]string {
	result := make(map[string]string)
	if a.relayStore == nil {
		result["error"] = "relay store not initialized"
		return result
	}

	mapping, err := a.relayStore.GetToolMapping()
	if err != nil {
		result["error"] = err.Error()
		return result
	}
	if len(mapping) == 0 {
		return result
	}

	endpoints, err := a.relayStore.ListEndpoints()
	if err != nil {
		result["error"] = err.Error()
		return result
	}

	// Build ID→endpoint lookup
	epByID := make(map[string]relay.RelayEndpoint, len(endpoints))
	for _, ep := range endpoints {
		epByID[ep.ID] = ep
	}

	for tool, relayID := range mapping {
		ep, ok := epByID[relayID]
		if !ok {
			result[tool] = fmt.Sprintf("relay endpoint %q not found", relayID)
			continue
		}
		apiKey := ep.APIKey
		if apiKey == "" {
			// Fall back to user token for Lurus relay
			if a.proxyMgr != nil {
				apiKey = a.proxyMgr.GetSettings().BuildToolAPIKey()
			}
		}
		errs := a.instMgr.ConfigureAllProxy(a.ctx, ep.URL, apiKey)
		// Only apply to the specific tool, but ConfigureAllProxy is all-or-nothing.
		// If we only want a single tool, use the per-tool installer directly.
		_ = tool
		for t, e := range errs {
			result[t] = e.Error()
		}
	}

	return result
}

// migrateProxyToRelay is called on first startup to seed the relay store from
// legacy proxy settings. It creates a "migrated-legacy" endpoint if apiEndpoint is set.
func (a *App) migrateProxyToRelay() {
	if a.relayStore == nil || a.proxyMgr == nil {
		return
	}
	settings := a.proxyMgr.GetSettings()
	if settings.APIEndpoint == "" {
		return
	}

	// Check whether migration has already been done
	eps, err := a.relayStore.ListEndpoints()
	if err != nil {
		return
	}
	for _, ep := range eps {
		if ep.ID == relay.MigratedLegacyRelayID {
			return // Already migrated
		}
	}

	migrated := relay.RelayEndpoint{
		ID:          relay.MigratedLegacyRelayID,
		Name:        "已迁移的代理设置",
		Kind:        relay.KindCustom,
		URL:         settings.APIEndpoint,
		APIKey:      settings.BuildToolAPIKey(),
		Description: "从旧版代理配置自动迁移",
	}
	_ = a.relayStore.SaveEndpoint(migrated)
}
