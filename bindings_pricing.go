package main

import (
	"net/http"
	"time"

	"lurus-switch/internal/pricing"
)

// ============================
// Pricing sync bindings (Wave 1 W1.1)
// ============================
//
// Switch's cost dashboards read from internal/pricing, whose static table is a
// conservative fallback. SyncPricing pulls the operator's live rate card from
// the Hub's public /api/v2/switch/pricing endpoint and overlays it, so the
// dashboard reflects the operator's real ratios. A failed sync is non-fatal:
// the overlay is left untouched and the static table stays authoritative.

// pricingSyncTimeout bounds the rate-card fetch.
const pricingSyncTimeout = 15 * time.Second

// PricingSyncResult reports the outcome of a SyncPricing call to the frontend.
type PricingSyncResult struct {
	ModelsSynced int    `json:"modelsSynced"`
	SyncedAt     string `json:"syncedAt"` // RFC3339, empty if never
	Source       string `json:"source"`   // Hub base URL used
}

// SyncPricing fetches the Hub rate card and overlays it onto the pricing table.
// Safe to call at startup and on demand. Returns an error when the Hub URL is
// unconfigured or the fetch fails — callers treat it as a soft warning (the
// static table still serves cost estimates).
func (a *App) SyncPricing() (*PricingSyncResult, error) {
	base, err := a.publicHubBaseURL()
	if err != nil {
		return nil, err
	}

	// Timeout-only client: nil Transport routes through the patched default
	// transport, so a configured BYO upstream proxy still applies.
	client := &http.Client{Timeout: pricingSyncTimeout}

	card, err := pricing.FetchRateCard(a.hubCtx(), client, base)
	if err != nil {
		return nil, err
	}

	pricing.Override(card, time.Now)

	return &PricingSyncResult{
		ModelsSynced: pricing.OverrideCount(),
		SyncedAt:     pricing.LastOverrideAt().Format(time.RFC3339),
		Source:       base,
	}, nil
}

// GetPricingSyncStatus reports the current overlay state without re-fetching.
// ModelsSynced==0 means the static table is fully authoritative.
func (a *App) GetPricingSyncStatus() *PricingSyncResult {
	syncedAt := ""
	if at := pricing.LastOverrideAt(); !at.IsZero() {
		syncedAt = at.Format(time.RFC3339)
	}
	return &PricingSyncResult{
		ModelsSynced: pricing.OverrideCount(),
		SyncedAt:     syncedAt,
	}
}
