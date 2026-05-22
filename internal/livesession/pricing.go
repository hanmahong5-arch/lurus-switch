package livesession

import "strings"

// Pricing table for cost estimation. Values are USD per 1M tokens.
//
// IMPORTANT: this estimator now counts FOUR token streams, not two —
// Claude Code uses prompt caching aggressively, and ignoring it (as the
// first cut of this code did) caused the UI to display estimates that
// were 30-60% low on long sessions. Anthropic's pricing for caching:
//
//   - input_tokens                : fresh, never-seen input — full input rate
//   - output_tokens               : full output rate
//   - cache_creation_input_tokens : input written into cache — 1.25× input
//   - cache_read_input_tokens     : input read from cache — 0.10× input
//
// We model all four. Numbers stay conservative (round up when in doubt) so
// the displayed total errs toward "warn the user" rather than "looks free".
type modelPrice struct {
	InputPerMTok         float64
	OutputPerMTok        float64
	CacheCreatePerMTok   float64 // 1.25× input
	CacheReadPerMTok     float64 // 0.10× input
}

// p builds a modelPrice from input/output rates, deriving cache rates
// from Anthropic's published 1.25× / 0.10× multipliers. Centralising the
// derivation keeps the table readable AND keeps cache rates consistent
// with the base input rate if we ever revise it.
func p(input, output float64) modelPrice {
	return modelPrice{
		InputPerMTok:       input,
		OutputPerMTok:      output,
		CacheCreatePerMTok: input * 1.25,
		CacheReadPerMTok:   input * 0.10,
	}
}

// modelPriceTable is matched by lowercase prefix — model IDs in JSONLs are
// e.g. "claude-opus-4-6", "claude-3-5-sonnet-20250625", etc., so a strict
// equality match would miss every dated variant.
var modelPriceTable = []struct {
	Prefix string
	Price  modelPrice
}{
	// Claude 4.x family (current default for Claude Code as of May 2026)
	{"claude-opus-4", p(15.00, 75.00)},
	{"claude-sonnet-4", p(3.00, 15.00)},
	{"claude-haiku-4", p(0.80, 4.00)},

	// Claude 3.x family (still in use for many older sessions)
	{"claude-3-7-sonnet", p(3.00, 15.00)},
	{"claude-3-5-sonnet", p(3.00, 15.00)},
	{"claude-3-5-haiku", p(0.80, 4.00)},
	{"claude-3-opus", p(15.00, 75.00)},
	{"claude-3-sonnet", p(3.00, 15.00)},
	{"claude-3-haiku", p(0.25, 1.25)},

	// OpenAI Codex models — coarse fallback prices for o-series / gpt-4o.
	// Switch isn't priced on OpenAI usage, so accuracy matters less here.
	{"gpt-4o", p(2.50, 10.00)},
	{"o1", p(15.00, 60.00)},
	{"o3", p(15.00, 60.00)},

	// Gemini
	{"gemini-2", p(1.25, 5.00)},
	{"gemini-1.5", p(1.25, 5.00)},
}

// fallbackPrice is used when the model id doesn't match any known prefix.
// Sonnet-tier numbers are the safest middle-ground default.
var fallbackPrice = p(3.00, 15.00)

// eventCost returns the USD billed for a single assistant turn. Cache
// fields are zero when the upstream JSONL didn't include them (older
// sessions, non-Claude tools), in which case this collapses cleanly to
// the input + output-only calculation.
func eventCost(model string, inputTokens, outputTokens, cacheCreate, cacheRead int64) float64 {
	price := lookupPrice(model)
	in := float64(inputTokens) / 1_000_000.0
	out := float64(outputTokens) / 1_000_000.0
	cc := float64(cacheCreate) / 1_000_000.0
	cr := float64(cacheRead) / 1_000_000.0
	return in*price.InputPerMTok +
		out*price.OutputPerMTok +
		cc*price.CacheCreatePerMTok +
		cr*price.CacheReadPerMTok
}

// estimateCost is the legacy two-input signature kept for the existing
// test that doesn't need to care about caching. Prefer eventCost in new
// code paths.
func estimateCost(model string, inputTokens, outputTokens int64) float64 {
	return eventCost(model, inputTokens, outputTokens, 0, 0)
}

func lookupPrice(model string) modelPrice {
	m := strings.ToLower(strings.TrimSpace(model))
	if m == "" {
		return fallbackPrice
	}
	for _, e := range modelPriceTable {
		if strings.HasPrefix(m, e.Prefix) {
			return e.Price
		}
	}
	return fallbackPrice
}
