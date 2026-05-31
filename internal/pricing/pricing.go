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

// Price captures the four-stream rate card for one model. Cache rates are
// derived from the model family's published cache multipliers off the base
// input rate; centralising the derivation keeps the table readable and the
// cache rates internally consistent.
type Price struct {
	InputPerMTok       float64
	OutputPerMTok      float64
	CacheCreatePerMTok float64
	CacheReadPerMTok   float64
}

// Cache-stream multipliers off the base input rate, by model family.
//
// Anthropic & DeepSeek: cache-create at 1.25× (Anthropic's published write
// surcharge) and cache-read at 0.10× (DeepSeek's cache-hit discount is
// comparable, ~0.1×).
//
// OpenAI & Gemini: cached input bills at 0.50× and there is no separate
// cache-create token stream — the OpenAI-compat gateway path never emits
// cacheCreate tokens (see internal/metering/types.go), so the create
// multiplier stays at 1.0× (plain input rate) purely as a safe fallback.
//
// Pinning these to named constants keeps the static table (rate/rateOpenAI)
// and the Hub-sync derivation (cacheMults) reading from one source of truth.
const (
	anthropicCacheCreateMult = 1.25
	anthropicCacheReadMult   = 0.10
	openaiCacheCreateMult    = 1.00
	openaiCacheReadMult      = 0.50
)

// rate builds an Anthropic/DeepSeek-style Price from input/output base rates.
func rate(input, output float64) Price {
	return rateWithCache(input, output, anthropicCacheCreateMult, anthropicCacheReadMult)
}

// rateOpenAI builds an OpenAI/Gemini-style Price: cached input at 0.50×, no
// cache-create surcharge. Used for the gpt-/o1/o3/gemini- table rows so an
// OpenAI cache-read bills at its real ~0.5× rate instead of Anthropic's 0.10×.
func rateOpenAI(input, output float64) Price {
	return rateWithCache(input, output, openaiCacheCreateMult, openaiCacheReadMult)
}

// rateWithCache builds a Price from base input/output rates plus explicit
// cache-create / cache-read multipliers applied to the input rate.
func rateWithCache(input, output, cacheCreateMult, cacheReadMult float64) Price {
	return Price{
		InputPerMTok:       input,
		OutputPerMTok:      output,
		CacheCreatePerMTok: input * cacheCreateMult,
		CacheReadPerMTok:   input * cacheReadMult,
	}
}

// cacheMults returns the (create, read) cache multipliers for a model family,
// matched by lowercase id prefix. Used by the Hub-sync mapper, which only
// receives input/output rates from the rate card and must reconstruct the
// cache streams the same way the static table does.
func cacheMults(model string) (createMult, readMult float64) {
	m := strings.ToLower(strings.TrimSpace(model))
	switch {
	case strings.HasPrefix(m, "gpt-"),
		strings.HasPrefix(m, "o1"),
		strings.HasPrefix(m, "o3"),
		strings.HasPrefix(m, "gemini-"):
		return openaiCacheCreateMult, openaiCacheReadMult
	default:
		return anthropicCacheCreateMult, anthropicCacheReadMult
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
	// rateOpenAI: cache-read bills at 0.50× (OpenAI's real cached-input rate),
	// not Anthropic's 0.10×.
	{"gpt-4o", rateOpenAI(2.50, 10.00)},
	{"o1", rateOpenAI(15.00, 60.00)},
	{"o3", rateOpenAI(15.00, 60.00)},

	// Gemini
	{"gemini-2", rateOpenAI(1.25, 5.00)},
	{"gemini-1.5", rateOpenAI(1.25, 5.00)},

	// DeepSeek — published USD prices as of May 2026.
	{"deepseek-chat", rate(0.27, 1.10)},
	{"deepseek-reasoner", rate(0.55, 2.19)},
	{"deepseek-v", rate(0.27, 1.10)},
}

// fallback price is sonnet-tier — safest middle ground when the model
// id doesn't match any known prefix.
var fallback = rate(3.00, 15.00)

// PriceFor looks up the rate card for a model id. A runtime override synced
// from the Hub (see source.go / sync.go) wins when present; otherwise the
// static table applies. Returns the fallback price when the model is unknown —
// callers don't need to handle a "not found" path.
func PriceFor(model string) Price {
	m := strings.ToLower(strings.TrimSpace(model))
	if m == "" {
		return fallback
	}
	if p, ok := overrides.lookup(m); ok {
		return p
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
