package metering

import (
	"math"
	"time"

	"lurus-switch/internal/pricing"
)

// Reconciliation compares Switch's local usage ledger against the Hub's
// server-side aggregate for the same window. This is AGGREGATE-LEVEL drift
// detection (totals over a window), not per-record matching — Switch's
// per-record correlation IDs never reach the Hub today, so a per-call join
// isn't possible without gateway passthrough + Hub-side storage (a later
// stretch). The numbers here answer "do the two sides roughly agree?", which
// is what a reseller needs to trust the money path.

// Aggregate-level tolerances for declaring the two sides reconciled. Small
// drift is expected: requests in flight at the window boundary, the Hub's
// quota rounding, retries deduped on one side but not the other.
const (
	reconcileCallTolerance      = 2    // absolute call-count slack
	reconcileTokenToleranceFrac = 0.01 // 1% of the larger token side
)

// LocalAgg is the Switch-side usage rollup over a window.
type LocalAgg struct {
	TokensIn  int64   `json:"tokensIn"`
	TokensOut int64   `json:"tokensOut"`
	Calls     int64   `json:"calls"`
	CostUSD   float64 `json:"costUSD"`
}

// HubAgg is the Hub-side rollup for the same window, sourced from the Hub's
// consume logs via the reconciliation endpoint. CostUSD is derived from the
// Hub's quota total by the caller (quota ÷ quota-per-unit).
type HubAgg struct {
	TokensIn  int64   `json:"tokensIn"`  // Σ prompt_tokens
	TokensOut int64   `json:"tokensOut"` // Σ completion_tokens
	Calls     int64   `json:"calls"`     // request_count
	CostUSD   float64 `json:"costUSD"`
}

// ReconcileReport is the drift between local and hub aggregates. Reconciled is
// false when the Hub side couldn't be fetched — the UI shows "unreconciled"
// rather than a misleading all-green, and WithinTolerance stays false.
type ReconcileReport struct {
	Local           LocalAgg `json:"local"`
	Hub             HubAgg   `json:"hub"`
	TokensInDelta   int64    `json:"tokensInDelta"`  // local − hub
	TokensOutDelta  int64    `json:"tokensOutDelta"` // local − hub
	CallsDelta      int64    `json:"callsDelta"`     // local − hub
	CostDeltaUSD    float64  `json:"costDeltaUsd"`   // local − hub
	WithinTolerance bool     `json:"withinTolerance"`
	Reconciled      bool     `json:"reconciled"`
	// Note carries a human-readable reason when Reconciled is false (Hub not
	// configured, endpoint not deployed, fetch failed). Set by the binding
	// layer, not by Reconcile (which stays pure). Empty on a clean reconcile.
	Note string `json:"note,omitempty"`
}

// LocalUsage rolls up the local ledger over [from, to]. Pure read; cost uses
// the same pricing.Cost the dashboards use.
func (s *Store) LocalUsage(from, to time.Time) LocalAgg {
	records := s.recordsInRange(from, to)
	var agg LocalAgg
	for _, r := range records {
		agg.TokensIn += r.TokensIn
		agg.TokensOut += r.TokensOut
		agg.Calls++
		agg.CostUSD += pricing.Cost(r.Model, r.TokensIn, r.TokensOut, r.CacheCreateTokens, r.CacheReadTokens)
	}
	return agg
}

// Reconcile computes the drift report. hubAvailable=false (Hub unreachable,
// not deployed yet, etc.) produces a report flagged unreconciled rather than
// pretending the two sides agree. Deterministic — no clock, no I/O.
func Reconcile(local LocalAgg, hub HubAgg, hubAvailable bool) ReconcileReport {
	rep := ReconcileReport{
		Local:          local,
		Hub:            hub,
		TokensInDelta:  local.TokensIn - hub.TokensIn,
		TokensOutDelta: local.TokensOut - hub.TokensOut,
		CallsDelta:     local.Calls - hub.Calls,
		CostDeltaUSD:   local.CostUSD - hub.CostUSD,
		Reconciled:     hubAvailable,
	}
	if hubAvailable {
		rep.WithinTolerance = abs64(rep.CallsDelta) <= reconcileCallTolerance &&
			tokensWithinTolerance(local.TokensIn, hub.TokensIn) &&
			tokensWithinTolerance(local.TokensOut, hub.TokensOut)
	}
	return rep
}

// tokensWithinTolerance reports whether two token counts agree within
// reconcileTokenToleranceFrac of the larger side. Equal counts (incl. 0,0)
// always pass.
func tokensWithinTolerance(a, b int64) bool {
	delta := abs64(a - b)
	larger := a
	if b > a {
		larger = b
	}
	tol := int64(math.Ceil(float64(larger) * reconcileTokenToleranceFrac))
	return delta <= tol
}

func abs64(v int64) int64 {
	if v < 0 {
		return -v
	}
	return v
}
