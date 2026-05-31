package pricing

import (
	"strings"
	"sync"
	"time"
)

// overrideStore is a runtime rate-card overlay synced from the Hub
// (FetchRateCard → Override). When populated it is the authoritative source
// for the models it names; PriceFor consults it before the static table.
//
// Single source of truth with a fail-safe: a failed or empty sync leaves the
// overlay untouched/cleared so the static table stays authoritative — a bad
// Hub response can never zero out pricing. Matching mirrors the static table's
// longest-prefix semantics so "claude-opus-4-6" resolves against a
// "claude-opus-4" override entry.
type overrideStore struct {
	mu        sync.RWMutex
	byPrefix  map[string]Price // lowercase prefix → price
	updatedAt time.Time        // when Override last replaced the overlay
}

// overrides is the package-level overlay. Empty until SyncPricing runs.
var overrides = &overrideStore{}

// Override atomically replaces the entire runtime overlay. Passing nil or an
// empty map clears it, reverting every lookup to the static table. Prefixes
// are lowercased and trimmed; blank prefixes are dropped. now supplies the
// "updated at" stamp (injectable for tests); a nil now falls back to
// time.Now.
func Override(card map[string]Price, now func() time.Time) {
	cleaned := make(map[string]Price, len(card))
	for prefix, price := range card {
		p := strings.ToLower(strings.TrimSpace(prefix))
		if p == "" {
			continue
		}
		cleaned[p] = price
	}

	ts := time.Now()
	if now != nil {
		ts = now()
	}

	overrides.mu.Lock()
	defer overrides.mu.Unlock()
	if len(cleaned) == 0 {
		overrides.byPrefix = nil
	} else {
		overrides.byPrefix = cleaned
	}
	overrides.updatedAt = ts
}

// lookup returns the overlay price for model using longest-prefix matching.
// The model id is expected pre-lowercased (PriceFor does this). ok is false
// when the overlay is empty or no prefix matches.
func (o *overrideStore) lookup(model string) (Price, bool) {
	o.mu.RLock()
	defer o.mu.RUnlock()
	if len(o.byPrefix) == 0 {
		return Price{}, false
	}
	var (
		best    Price
		bestLen = -1
	)
	for prefix, price := range o.byPrefix {
		if strings.HasPrefix(model, prefix) && len(prefix) > bestLen {
			best = price
			bestLen = len(prefix)
		}
	}
	if bestLen < 0 {
		return Price{}, false
	}
	return best, true
}

// OverrideCount reports the number of models currently overlaid. Zero means
// the static table is fully authoritative. Exposed for the SyncPricing
// binding (so the UI can show "N models synced") and tests.
func OverrideCount() int {
	overrides.mu.RLock()
	defer overrides.mu.RUnlock()
	return len(overrides.byPrefix)
}

// LastOverrideAt returns when the overlay was last replaced, zero if never.
// Callers can gate refetches against a TTL without imposing a background
// timer on this leaf package.
func LastOverrideAt() time.Time {
	overrides.mu.RLock()
	defer overrides.mu.RUnlock()
	return overrides.updatedAt
}
