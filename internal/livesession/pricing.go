package livesession

import "lurus-switch/internal/pricing"

// eventCost returns the USD billed for a single assistant turn. Cache
// fields are zero when the upstream JSONL didn't include them (older
// sessions, non-Claude tools), in which case this collapses cleanly to
// the input + output-only calculation.
//
// Pricing logic now lives in internal/pricing so the gateway's runtime
// cost dashboard reads from the same table as live-session estimates.
func eventCost(model string, inputTokens, outputTokens, cacheCreate, cacheRead int64) float64 {
	return pricing.Cost(model, inputTokens, outputTokens, cacheCreate, cacheRead)
}

// estimateCost is the legacy two-input signature kept for the existing
// test that doesn't need to care about caching. Prefer eventCost in new
// code paths.
func estimateCost(model string, inputTokens, outputTokens int64) float64 {
	return eventCost(model, inputTokens, outputTokens, 0, 0)
}
