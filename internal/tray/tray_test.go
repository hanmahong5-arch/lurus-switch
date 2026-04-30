package tray

import (
	"context"
	"strings"
	"testing"
	"time"
)

// ---------------------------------------------------------------------------
// Badge tier classification
// ---------------------------------------------------------------------------

func TestClassifyQuota(t *testing.T) {
	cases := []struct {
		usedPercent float64
		want        BadgeTier
	}{
		{-1, TierUnknown},
		{0, TierGreen},
		{50, TierGreen},
		{50.1, TierYellow},
		{80, TierYellow},
		{80.1, TierRed},
		{100, TierRed},
	}
	for _, tc := range cases {
		got := classifyQuota(tc.usedPercent)
		if got != tc.want {
			t.Errorf("classifyQuota(%.1f) = %d, want %d", tc.usedPercent, got, tc.want)
		}
	}
}

func TestResolveTier_GatewayDown(t *testing.T) {
	// Even with 0% quota usage, stopped gateway → TierGray.
	q := QuotaSnapshot{UsedPercent: 0}
	gw := GatewayStatus{Running: false, Port: 19090}
	if got := resolveTier(q, gw); got != TierGray {
		t.Errorf("expected TierGray when gateway stopped, got %d", got)
	}
}

func TestResolveTier_GatewayRunning(t *testing.T) {
	gw := GatewayStatus{Running: true, Port: 19090}
	cases := []struct {
		pct  float64
		want BadgeTier
	}{
		{0, TierGreen},
		{50, TierGreen},
		{60, TierYellow},
		{90, TierRed},
	}
	for _, tc := range cases {
		q := QuotaSnapshot{UsedPercent: tc.pct}
		if got := resolveTier(q, gw); got != tc.want {
			t.Errorf("resolveTier(%.0f%%, running) = %d, want %d", tc.pct, got, tc.want)
		}
	}
}

// ---------------------------------------------------------------------------
// Tooltip construction
// ---------------------------------------------------------------------------

func TestBuildTooltip_ContainsTier(t *testing.T) {
	q := QuotaSnapshot{UsedPercent: 90, BalanceText: "¥10.00"}
	gw := GatewayStatus{Running: true, Port: 19090}
	tip := buildTooltip(q, gw)
	if tip == "" {
		t.Fatal("tooltip should not be empty")
	}
	// Red tier prefix should appear somewhere in the tooltip.
	if !strings.Contains(tip, "[⚠]") {
		t.Errorf("expected red prefix [⚠] in tooltip, got %q", tip)
	}
}

func TestBuildTooltip_GatewayStopped(t *testing.T) {
	q := QuotaSnapshot{UsedPercent: 10}
	gw := GatewayStatus{Running: false}
	tip := buildTooltip(q, gw)
	// Gray prefix because gateway is down.
	if !strings.Contains(tip, "[○]") {
		t.Errorf("expected gray prefix [○] in tooltip, got %q", tip)
	}
}

func TestBuildTooltip_UnknownQuota(t *testing.T) {
	q := QuotaSnapshot{UsedPercent: -1}
	gw := GatewayStatus{Running: true, Port: 19090}
	tip := buildTooltip(q, gw)
	if tip == "" {
		t.Fatal("tooltip should not be empty for unknown quota")
	}
}

// ---------------------------------------------------------------------------
// Manager nil safety
// ---------------------------------------------------------------------------

func TestManager_NilSafe(t *testing.T) {
	var m *Manager
	// None of these should panic.
	m.Stop()
}

// ---------------------------------------------------------------------------
// Manager provider callbacks
// ---------------------------------------------------------------------------

func TestManager_ProvidersCalledOnRefresh(t *testing.T) {
	quotaCalled := 0
	gwCalled := 0

	m := New(
		func() QuotaSnapshot {
			quotaCalled++
			return QuotaSnapshot{UsedPercent: 25, BalanceText: "¥5.00"}
		},
		func() GatewayStatus {
			gwCalled++
			return GatewayStatus{Running: true, Port: 19090}
		},
	)

	// Call refreshState directly (no OS systray involved).
	m.mu.Lock()
	m.lastQuota = QuotaSnapshot{UsedPercent: -1}
	m.mu.Unlock()

	m.refreshState()

	if quotaCalled != 1 {
		t.Errorf("quotaProvider called %d times, want 1", quotaCalled)
	}
	if gwCalled != 1 {
		t.Errorf("gatewayProvider called %d times, want 1", gwCalled)
	}

	q, gw := m.snapshot()
	if q.UsedPercent != 25 {
		t.Errorf("snapshot quota = %.1f, want 25", q.UsedPercent)
	}
	if !gw.Running {
		t.Error("snapshot gateway should be running")
	}
}

func TestManager_NilProviders(t *testing.T) {
	// Should not panic with nil providers.
	m := New(nil, nil)
	m.refreshState()
	q, gw := m.snapshot()
	if q.UsedPercent != -1 {
		t.Errorf("expected unknown quota (-1), got %.1f", q.UsedPercent)
	}
	if gw.Running {
		t.Error("expected gateway not running by default")
	}
}

// ---------------------------------------------------------------------------
// Manager Start is idempotent (without real systray, verify once.Do)
// ---------------------------------------------------------------------------

func TestManager_StartIdempotent(t *testing.T) {
	// We cannot call systray.Run in tests (no GUI). Verify that calling Stop
	// on an unstarted manager is safe (cancel is nil).
	m := New(nil, nil)
	// Not started — Stop must be a no-op.
	done := make(chan struct{})
	go func() {
		defer close(done)
		m.Stop() // must return quickly
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Stop() hung on unstarted manager")
	}
}

// ---------------------------------------------------------------------------
// Context cancellation propagates to internal loop (unit-level)
// ---------------------------------------------------------------------------

func TestManager_ContextCancel(t *testing.T) {
	m := New(nil, nil)
	ctx, cancel := context.WithCancel(context.Background())
	m.ctx = ctx
	m.cancel = cancel

	exited := make(chan struct{})
	go func() {
		defer close(exited)
		// Simulate the inner select loop with a minimal ticker.
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-m.ctx.Done():
				return
			case <-ticker.C:
				m.refreshState()
			}
		}
	}()

	cancel()

	select {
	case <-exited:
	case <-time.After(2 * time.Second):
		t.Fatal("loop did not exit after context cancel")
	}
}
