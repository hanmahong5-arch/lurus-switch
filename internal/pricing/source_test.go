package pricing

import (
	"testing"
	"time"
)

// resetOverrides clears the package overlay so tests don't leak into each
// other (the overlay is package-global state).
func resetOverrides(t *testing.T) {
	t.Helper()
	Override(nil, nil)
	t.Cleanup(func() { Override(nil, nil) })
}

func TestOverride_HitWinsOverStaticTable(t *testing.T) {
	resetOverrides(t)

	// Static gpt-4o input is $2.50/MTok. Overlay it with a different rate.
	Override(map[string]Price{
		"gpt-4o": rateOpenAI(9.99, 19.99),
	}, nil)

	p := PriceFor("gpt-4o-mini") // prefix-matches the "gpt-4o" overlay entry
	if !approxEq(p.InputPerMTok, 9.99) {
		t.Errorf("overlay miss: input = %v, want 9.99", p.InputPerMTok)
	}
	if OverrideCount() != 1 {
		t.Errorf("OverrideCount = %d, want 1", OverrideCount())
	}
}

func TestOverride_EmptyOrClearedFallsBackToStatic(t *testing.T) {
	resetOverrides(t)

	// Populate then clear with an empty map → static table authoritative again.
	Override(map[string]Price{"gpt-4o": rateOpenAI(9.99, 19.99)}, nil)
	Override(map[string]Price{}, nil)

	p := PriceFor("gpt-4o")
	if !approxEq(p.InputPerMTok, 2.50) {
		t.Errorf("after clear, input = %v, want static 2.50", p.InputPerMTok)
	}
	if OverrideCount() != 0 {
		t.Errorf("OverrideCount = %d, want 0 after clear", OverrideCount())
	}

	// nil clears too.
	Override(map[string]Price{"gpt-4o": rateOpenAI(1, 1)}, nil)
	Override(nil, nil)
	if OverrideCount() != 0 {
		t.Errorf("nil Override should clear overlay, count = %d", OverrideCount())
	}
}

func TestOverride_LongestPrefixWins(t *testing.T) {
	resetOverrides(t)

	Override(map[string]Price{
		"claude":         rate(1, 1),
		"claude-opus-4":  rate(2, 2),
		"claude-opus-4-": rate(3, 3),
	}, nil)

	// "claude-opus-4-7" should bind to the longest matching prefix.
	p := PriceFor("claude-opus-4-7")
	if !approxEq(p.InputPerMTok, 3) {
		t.Errorf("longest-prefix lookup: input = %v, want 3", p.InputPerMTok)
	}
	// "claude-sonnet" only matches the broad "claude" entry.
	p = PriceFor("claude-sonnet-4")
	if !approxEq(p.InputPerMTok, 1) {
		t.Errorf("broad-prefix lookup: input = %v, want 1", p.InputPerMTok)
	}
}

func TestOverride_BlankPrefixesDropped(t *testing.T) {
	resetOverrides(t)
	Override(map[string]Price{
		"":      rate(5, 5),
		"  ":    rate(6, 6),
		"gpt-4o": rateOpenAI(7, 7),
	}, nil)
	if OverrideCount() != 1 {
		t.Errorf("blank prefixes should be dropped, count = %d, want 1", OverrideCount())
	}
}

func TestOverride_UpdatedAtStampInjectable(t *testing.T) {
	resetOverrides(t)
	fixed := time.Date(2026, 5, 31, 12, 0, 0, 0, time.UTC)
	Override(map[string]Price{"gpt-4o": rateOpenAI(1, 1)}, func() time.Time { return fixed })
	if got := LastOverrideAt(); !got.Equal(fixed) {
		t.Errorf("LastOverrideAt = %v, want %v", got, fixed)
	}
}
