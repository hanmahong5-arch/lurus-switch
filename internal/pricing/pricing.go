// Package pricing centralises the USD-per-model price tables used by
// both the live-session cost estimator and the runtime metering
// dashboard. Promoted out of internal/livesession (W3.3) so the
// gateway's CostUSD aggregation reads from the same source of truth.
//
// Prices are denominated in USD per 1M tokens for each of four streams
// (fresh input, output, cache-create, cache-read) and follow the
// public Anthropic / OpenAI / Google rate cards as of May 2026.
// Numbers stay conservative (round up when in doubt) so the UI errs
// toward "warn the user" rather than "looks free". Update the table
// when official rates move — there's no live ratio sync.
package pricing

import "strings"

// Price captures the four-stream rate card for one model. Cache rates
// are derived from Anthropic's published 1.25× / 0.10× multipliers
// off the base input rate; centralising the derivation keeps the
// table readable and the cache rates internally consistent.
type Price struct {
	InputPerMTok       float64
	OutputPerMTok      float64
	CacheCreatePerMTok float64
	CacheReadPerMTok   float64
}

// rate builds a Price from input/output base rates.
func rate(input, output float64) Price {
	return Price{
		InputPerMTok:       input,
		OutputPerMTok:      output,
		CacheCreatePerMTok: input * 1.25,
		CacheReadPerMTok:   input * 0.10,
	}
}

// table is matched by lowercase prefix — model IDs in the wild are
// e.g. "claude-opus-4-6", "claude-3-5-sonnet-20250625", so strict
// equality would miss every dated variant.
var table = []struct {
	Prefix string
	Price  Price
}{
	// Claude 4.x family (current default for Claude Code as of May 2026)
	{"claude-opus-4", rate(15.00, 75.00)},
	{"claude-sonnet-4", rate(3.00, 15.00)},
	{"claude-haiku-4", rate(0.80, 4.00)},

	// Claude 3.x family (still in use for many older sessions)
	{"claude-3-7-sonnet", rate(3.00, 15.00)},
	{"claude-3-5-sonnet", rate(3.00, 15.00)},
	{"claude-3-5-haiku", rate(0.80, 4.00)},
	{"claude-3-opus", rate(15.00, 75.00)},
	{"claude-3-sonnet", rate(3.00, 15.00)},
	{"claude-3-haiku", rate(0.25, 1.25)},

	// OpenAI Codex models — coarse fallback prices for o-series / gpt-4o.
	{"gpt-4o", rate(2.50, 10.00)},
	{"o1", rate(15.00, 60.00)},
	{"o3", rate(15.00, 60.00)},

	// Gemini
	{"gemini-2", rate(1.25, 5.00)},
	{"gemini-1.5", rate(1.25, 5.00)},

	// DeepSeek — published USD prices as of May 2026.
	{"deepseek-chat", rate(0.27, 1.10)},
	{"deepseek-reasoner", rate(0.55, 2.19)},
	{"deepseek-v", rate(0.27, 1.10)},
}

// fallback price is sonnet-tier — safest middle ground when the model
// id doesn't match any known prefix.
var fallback = rate(3.00, 15.00)

// PriceFor looks up the rate card for a model id. Returns the fallback
// price when the model is unknown — callers don't need to handle a
// "not found" path.
func PriceFor(model string) Price {
	m := strings.ToLower(strings.TrimSpace(model))
	if m == "" {
		return fallback
	}
	for _, e := range table {
		if strings.HasPrefix(m, e.Prefix) {
			return e.Price
		}
	}
	return fallback
}

// Cost returns the USD billed for a given token mix on a given model.
// Cache fields are zero when the upstream JSONL / response doesn't
// include them (older sessions, non-Claude tools) — the formula
// collapses cleanly to input + output only.
func Cost(model string, tokensIn, tokensOut, cacheCreate, cacheRead int64) float64 {
	p := PriceFor(model)
	in := float64(tokensIn) / 1_000_000.0
	out := float64(tokensOut) / 1_000_000.0
	cc := float64(cacheCreate) / 1_000_000.0
	cr := float64(cacheRead) / 1_000_000.0
	return in*p.InputPerMTok +
		out*p.OutputPerMTok +
		cc*p.CacheCreatePerMTok +
		cr*p.CacheReadPerMTok
}
