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
