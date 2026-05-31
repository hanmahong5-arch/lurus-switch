package pricing

import (
	"math"
	"testing"
)

func approxEq(a, b float64) bool {
	return math.Abs(a-b) < 1e-9
}

func TestPriceFor_KnownPrefixes(t *testing.T) {
	cases := []struct {
		model        string
		wantInput    float64
		wantOutput   float64
	}{
		{"claude-opus-4-7", 15.00, 75.00},
		{"claude-sonnet-4-6", 3.00, 15.00},
		{"claude-haiku-4-5-20251001", 0.80, 4.00},
		{"claude-3-5-sonnet-20250625", 3.00, 15.00},
		{"gpt-4o-mini", 2.50, 10.00},
		{"o1-preview", 15.00, 60.00},
		{"deepseek-chat", 0.27, 1.10},
	}
	for _, c := range cases {
		p := PriceFor(c.model)
		if !approxEq(p.InputPerMTok, c.wantInput) {
			t.Errorf("%s input=%v, want %v", c.model, p.InputPerMTok, c.wantInput)
		}
		if !approxEq(p.OutputPerMTok, c.wantOutput) {
			t.Errorf("%s output=%v, want %v", c.model, p.OutputPerMTok, c.wantOutput)
		}
	}
}

func TestPriceFor_UnknownFallsBackToSonnetTier(t *testing.T) {
	p := PriceFor("unknown-future-model-9000")
	if !approxEq(p.InputPerMTok, 3.00) || !approxEq(p.OutputPerMTok, 15.00) {
		t.Errorf("unknown should fall back to sonnet-tier, got %+v", p)
	}
	// Empty string should also fall back.
	if PriceFor("") != PriceFor("unknown-future-model") {
		t.Errorf("empty model should match the same fallback path")
	}
}

func TestCost_InputOutputBasics(t *testing.T) {
	// 1M in + 1M out of sonnet → 3 + 15 = 18 USD.
	got := Cost("claude-sonnet-4-6", 1_000_000, 1_000_000, 0, 0)
	if !approxEq(got, 18.0) {
		t.Errorf("sonnet 1M+1M = %v, want 18", got)
	}
	// 1M in + 1M out of opus → 15 + 75 = 90.
	got = Cost("claude-opus-4-7", 1_000_000, 1_000_000, 0, 0)
	if !approxEq(got, 90.0) {
		t.Errorf("opus 1M+1M = %v, want 90", got)
	}
	// Zero tokens → zero cost regardless of model.
	if got := Cost("anything", 0, 0, 0, 0); got != 0 {
		t.Errorf("zero tokens should cost 0, got %v", got)
	}
}

func TestCost_OpenAICacheRateHalf(t *testing.T) {
	// gpt-4o input is $2.50/MTok. An OpenAI cache-read bills at 0.50× input
	// (real OpenAI cached-input rate), NOT Anthropic's 0.10×. 800 cache-read
	// tokens → 800/1e6 × (2.50 × 0.50) = 0.001 USD.
	got := Cost("gpt-4o", 0, 0, 0, 800)
	want := float64(800) / 1e6 * (2.50 * 0.50)
	if !approxEq(got, want) {
		t.Errorf("gpt-4o cache-read 800 = %v, want %v (0.50× input, not 0.10×)", got, want)
	}
	// Regression guard: the old behaviour (0.10×) would have produced 0.0002.
	if approxEq(got, float64(800)/1e6*(2.50*0.10)) {
		t.Errorf("gpt-4o cache-read still using Anthropic 0.10× rate: %v", got)
	}

	// Claude stays at 0.10× — same 800 cache-read tokens on sonnet ($3.00/MTok)
	// → 800/1e6 × (3.00 × 0.10).
	gotClaude := Cost("claude-sonnet-4-6", 0, 0, 0, 800)
	wantClaude := float64(800) / 1e6 * (3.00 * 0.10)
	if !approxEq(gotClaude, wantClaude) {
		t.Errorf("claude cache-read 800 = %v, want %v (0.10× input)", gotClaude, wantClaude)
	}

	// Gemini follows the OpenAI family too (0.50×).
	gotGemini := Cost("gemini-2-flash", 0, 0, 0, 1_000_000)
	wantGemini := 1.25 * 0.50 // gemini-2 input $1.25/MTok × 0.50
	if !approxEq(gotGemini, wantGemini) {
		t.Errorf("gemini-2 cache-read 1M = %v, want %v (0.50× input)", gotGemini, wantGemini)
	}
}

func TestPriceFor_OpenAIFamilyCacheMultipliers(t *testing.T) {
	// gpt-4o: cache-read 0.50× input, cache-create 1.0× input (no surcharge).
	p := PriceFor("gpt-4o-mini")
	if !approxEq(p.CacheReadPerMTok, 2.50*0.50) {
		t.Errorf("gpt-4o cache-read rate = %v, want %v", p.CacheReadPerMTok, 2.50*0.50)
	}
	if !approxEq(p.CacheCreatePerMTok, 2.50*1.00) {
		t.Errorf("gpt-4o cache-create rate = %v, want %v", p.CacheCreatePerMTok, 2.50*1.00)
	}
	// claude keeps Anthropic multipliers.
	c := PriceFor("claude-opus-4-7")
	if !approxEq(c.CacheReadPerMTok, 15.00*0.10) {
		t.Errorf("claude cache-read rate = %v, want %v", c.CacheReadPerMTok, 15.00*0.10)
	}
	if !approxEq(c.CacheCreatePerMTok, 15.00*1.25) {
		t.Errorf("claude cache-create rate = %v, want %v", c.CacheCreatePerMTok, 15.00*1.25)
	}
}

func TestCost_CacheStreamsApplyCorrectMultipliers(t *testing.T) {
	// 1M cache_create at sonnet rate (3.00 × 1.25 = 3.75).
	cc := Cost("claude-sonnet-4-6", 0, 0, 1_000_000, 0)
	if !approxEq(cc, 3.75) {
		t.Errorf("cache_create 1M = %v, want 3.75", cc)
	}
	// 1M cache_read at sonnet rate (3.00 × 0.10 = 0.30).
	cr := Cost("claude-sonnet-4-6", 0, 0, 0, 1_000_000)
	if !approxEq(cr, 0.30) {
		t.Errorf("cache_read 1M = %v, want 0.30", cr)
	}
	// All four streams together — sum should be linear.
	total := Cost("claude-sonnet-4-6", 100, 50, 200, 500)
	expected :=
		float64(100)/1e6*3.00 +
			float64(50)/1e6*15.00 +
			float64(200)/1e6*3.75 +
			float64(500)/1e6*0.30
	if !approxEq(total, expected) {
		t.Errorf("four-stream sum = %v, want %v", total, expected)
	}
}
