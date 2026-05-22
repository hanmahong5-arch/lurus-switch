package main

import (
	"fmt"

	"lurus-switch/internal/relay"
	"lurus-switch/internal/tray"
)

// appRelayProvider adapts the relay store + router into the
// tray.RelayProvider contract so the systray menu can populate the
// "Switch Relay →" submenu and the tooltip's relay-state hint.
type appRelayProvider struct {
	app *App
}

func (p *appRelayProvider) ListEntries() []tray.RelayMenuEntry {
	if p == nil || p.app == nil || p.app.relayStore == nil {
		return nil
	}
	eps, err := p.app.relayStore.ListEndpoints()
	if err != nil {
		return nil
	}
	var states map[string]relay.CircuitState
	if p.app.relayRouter != nil {
		states = p.app.relayRouter.Breaker().Snapshot()
	}
	out := make([]tray.RelayMenuEntry, 0, len(eps))
	for _, ep := range eps {
		title := ep.Name
		if title == "" {
			title = ep.ID
		}
		state := ""
		if st, ok := states[ep.ID]; ok {
			state = string(st.Status)
		}
		out = append(out, tray.RelayMenuEntry{ID: ep.ID, Title: title, State: state})
	}
	return out
}

func (p *appRelayProvider) CurrentPickSummary() string {
	if p == nil || p.app == nil || p.app.relayRouter == nil {
		return ""
	}
	res, err := p.app.relayRouter.Pick("", relay.PickHint{})
	if err != nil {
		return "Relay: no healthy endpoint"
	}
	if res.Endpoint.LatencyMs > 0 {
		return fmt.Sprintf("Relay: %s (%dms)", res.Endpoint.Name, res.Endpoint.LatencyMs)
	}
	return "Relay: " + res.Endpoint.Name
}
