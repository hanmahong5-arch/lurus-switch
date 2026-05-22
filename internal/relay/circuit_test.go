package relay

import (
	"testing"
	"time"
)

func TestCircuit_OpensAfterThreshold(t *testing.T) {
	now := time.Date(2026, 5, 11, 12, 0, 0, 0, time.UTC)
	clock := func() time.Time { return now }
	b := NewCircuitBreakerForTest(3, 10*time.Second, clock)

	if !b.Allow("a") {
		t.Fatal("fresh breaker should allow")
	}
	b.RecordFailure("a", "boom")
	b.RecordFailure("a", "boom")
	if !b.Allow("a") {
		t.Fatal("under threshold should still allow")
	}
	b.RecordFailure("a", "boom")
	if b.Allow("a") {
		t.Fatal("at threshold should open")
	}

	// Advance past cooldown — half-open lets one through.
	now = now.Add(11 * time.Second)
	if !b.Allow("a") {
		t.Fatal("after cooldown should half-open")
	}
	b.RecordSuccess("a")
	if got := b.Snapshot()["a"].Status; got != StatusClosed {
		t.Fatalf("after success want closed, got %s", got)
	}
}

func TestCircuit_ResetClearsState(t *testing.T) {
	b := NewCircuitBreaker()
	b.RecordFailure("x", "oops")
	b.Reset("x")
	if _, ok := b.Snapshot()["x"]; ok {
		t.Fatal("Reset should remove the endpoint state")
	}
}
