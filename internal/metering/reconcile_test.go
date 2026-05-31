package metering

import (
	"testing"
	"time"
)

func TestReconcile_EqualSidesWithinTolerance(t *testing.T) {
	local := LocalAgg{TokensIn: 10_000, TokensOut: 5_000, Calls: 42, CostUSD: 1.23}
	hub := HubAgg{TokensIn: 10_000, TokensOut: 5_000, Calls: 42, CostUSD: 1.23}

	rep := Reconcile(local, hub, true)
	if !rep.Reconciled {
		t.Fatal("expected Reconciled=true when hub is available")
	}
	if !rep.WithinTolerance {
		t.Errorf("equal sides should be within tolerance: %+v", rep)
	}
	if rep.TokensInDelta != 0 || rep.TokensOutDelta != 0 || rep.CallsDelta != 0 {
		t.Errorf("equal sides should have zero deltas: %+v", rep)
	}
}

func TestReconcile_DriftProducesCorrectDeltas(t *testing.T) {
	local := LocalAgg{TokensIn: 12_000, TokensOut: 6_000, Calls: 50, CostUSD: 2.00}
	hub := HubAgg{TokensIn: 10_000, TokensOut: 5_000, Calls: 40, CostUSD: 1.50}

	rep := Reconcile(local, hub, true)
	if rep.TokensInDelta != 2_000 {
		t.Errorf("TokensInDelta = %d, want 2000", rep.TokensInDelta)
	}
	if rep.TokensOutDelta != 1_000 {
		t.Errorf("TokensOutDelta = %d, want 1000", rep.TokensOutDelta)
	}
	if rep.CallsDelta != 10 {
		t.Errorf("CallsDelta = %d, want 10", rep.CallsDelta)
	}
	if rep.CostDeltaUSD < 0.49 || rep.CostDeltaUSD > 0.51 {
		t.Errorf("CostDeltaUSD = %v, want ~0.50", rep.CostDeltaUSD)
	}
	if rep.WithinTolerance {
		t.Errorf("20%% token drift + 10-call drift should NOT be within tolerance: %+v", rep)
	}
}

func TestReconcile_SmallDriftWithinTolerance(t *testing.T) {
	// 0.5% token drift and a 1-call gap stay within the default tolerances.
	local := LocalAgg{TokensIn: 100_000, TokensOut: 100_000, Calls: 101}
	hub := HubAgg{TokensIn: 100_500, TokensOut: 99_500, Calls: 100}

	rep := Reconcile(local, hub, true)
	if !rep.WithinTolerance {
		t.Errorf("sub-1%% drift should be within tolerance: %+v", rep)
	}
}

func TestReconcile_HubUnavailableMarksUnreconciled(t *testing.T) {
	local := LocalAgg{TokensIn: 5_000, TokensOut: 2_000, Calls: 9, CostUSD: 0.5}
	rep := Reconcile(local, HubAgg{}, false)

	if rep.Reconciled {
		t.Error("Reconciled should be false when hub is unavailable")
	}
	if rep.WithinTolerance {
		t.Error("WithinTolerance must stay false when unreconciled (no false all-green)")
	}
	// Deltas still reflect local minus an empty hub so the UI can show "Hub
	// reported nothing".
	if rep.TokensInDelta != 5_000 || rep.CallsDelta != 9 {
		t.Errorf("deltas vs empty hub wrong: %+v", rep)
	}
}

func TestLocalUsage_AggregatesRange(t *testing.T) {
	dir := t.TempDir()
	store, err := NewStore(dir)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}

	base := time.Date(2026, 5, 31, 10, 0, 0, 0, time.UTC)
	store.Record(Record{ID: "r1", Model: "claude-sonnet-4-6", TokensIn: 1000, TokensOut: 500, Timestamp: base})
	store.Record(Record{ID: "r2", Model: "claude-sonnet-4-6", TokensIn: 2000, TokensOut: 1000, Timestamp: base.Add(time.Hour)})
	// A record outside the window must be excluded.
	store.Record(Record{ID: "r3", Model: "claude-sonnet-4-6", TokensIn: 9999, TokensOut: 9999, Timestamp: base.AddDate(0, 0, 5)})

	agg := store.LocalUsage(base.Add(-time.Minute), base.Add(2*time.Hour))
	if agg.Calls != 2 {
		t.Errorf("Calls = %d, want 2 (r3 out of range)", agg.Calls)
	}
	if agg.TokensIn != 3000 || agg.TokensOut != 1500 {
		t.Errorf("tokens = (%d,%d), want (3000,1500)", agg.TokensIn, agg.TokensOut)
	}
	if agg.CostUSD <= 0 {
		t.Errorf("CostUSD should be positive for non-zero usage, got %v", agg.CostUSD)
	}
}
