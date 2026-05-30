package budget

import (
	"path/filepath"
	"sync"
	"testing"

	"lurus-switch/internal/metering"
)

func TestGuard_DisabledByDefaultAllowsEverything(t *testing.T) {
	g, _ := New("", nil)
	g.RecordUsage(1_000_000, 1_000_000) // a lot
	if v := g.Check(); !v.Allowed {
		t.Errorf("disabled guard should allow; got %+v", v)
	}
}

func TestGuard_SessionLimitBlocks(t *testing.T) {
	g, _ := New("", nil)
	_ = g.SetConfig(Config{Enabled: true, SessionTokens: 1000})
	g.RecordUsage(600, 500) // 1100 > 1000
	v := g.Check()
	if v.Allowed {
		t.Error("should block at session cap")
	}
	if v.LimitKind != "session" {
		t.Errorf("limitKind=%s, want session", v.LimitKind)
	}
}

func TestGuard_DailyLimitBlocks(t *testing.T) {
	today := func() metering.DailySummary {
		return metering.DailySummary{TokensIn: 800, TokensOut: 250}
	}
	g, _ := New("", today)
	_ = g.SetConfig(Config{Enabled: true, DailyTokens: 1000})
	v := g.Check()
	if v.Allowed {
		t.Error("should block at daily cap")
	}
	if v.LimitKind != "daily" {
		t.Errorf("limitKind=%s, want daily", v.LimitKind)
	}
}

func TestGuard_ResetSessionClearsCounter(t *testing.T) {
	g, _ := New("", nil)
	_ = g.SetConfig(Config{Enabled: true, SessionTokens: 1000})
	g.RecordUsage(1500, 0)
	if g.Check().Allowed {
		t.Fatal("expected block before reset")
	}
	g.ResetSession()
	if !g.Check().Allowed {
		t.Error("expected allow after reset")
	}
}

func TestGuard_PersistsToDisk(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "budget.json")
	g, _ := New(cfgPath, nil)
	if err := g.SetConfig(Config{Enabled: true, SessionTokens: 12345, SoftWarnPct: 75}); err != nil {
		t.Fatal(err)
	}
	g2, _ := New(cfgPath, nil)
	if !g2.GetConfig().Enabled {
		t.Error("enabled lost across reload")
	}
	if g2.GetConfig().SessionTokens != 12345 {
		t.Errorf("session=%d, want 12345", g2.GetConfig().SessionTokens)
	}
}

func TestGuard_StatusReportsPctAndWarning(t *testing.T) {
	today := func() metering.DailySummary {
		return metering.DailySummary{TokensIn: 800, TokensOut: 0}
	}
	g, _ := New("", today)
	_ = g.SetConfig(Config{Enabled: true, DailyTokens: 1000, SoftWarnPct: 70})
	st := g.Status()
	if st.DailyPct != 80 {
		t.Errorf("dailyPct=%d, want 80", st.DailyPct)
	}
	if !st.WarnDaily || st.HitDaily {
		t.Errorf("expected WarnDaily=true HitDaily=false; got %+v", st)
	}
}

// TestGuard_ConcurrentCheckRecordNoRace runs many goroutines that interleave
// Check() and RecordUsage() against the session cap. Under `go test -race`
// this surfaces any torn read between the counter read in Check and the
// counter mutation in RecordUsage / ResetSession.
//
// Honesty note (global CLAUDE.md §4.1⑥): this proves the SESSION axis is
// data-race free and that once usage crosses the cap, Check eventually and
// consistently blocks. It does NOT prove zero overshoot — the cap is
// post-hoc (the request's real token count is only known after the upstream
// responds), so a request whose RecordUsage lands while several Checks are
// in flight can still graze the cap by one request. Atomicity here means
// "no two concurrent callers race a half-applied increment", not "the wall
// is a hard pre-charge". Eliminating overshoot needs a pre-flight
// reservation wired through the gateway hot path (left to a follow-up).
func TestGuard_ConcurrentCheckRecordNoRace(t *testing.T) {
	g, _ := New("", nil)
	_ = g.SetConfig(Config{Enabled: true, SessionTokens: 1000})

	const workers = 32
	const perWorker = 50
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < perWorker; j++ {
				_ = g.Check()
				g.RecordUsage(1, 1)
			}
		}()
	}
	wg.Wait()

	// After 32*50*2 = 3200 tokens recorded against a 1000 cap, Check must
	// block — and the verdict must report the session axis.
	v := g.Check()
	if v.Allowed {
		t.Errorf("session cap should be hit after concurrent load; got %+v", v)
	}
	if v.LimitKind != "session" {
		t.Errorf("limitKind=%s, want session", v.LimitKind)
	}
	if got := g.Status().SessionUsed; got != int64(workers*perWorker*2) {
		t.Errorf("sessionUsed=%d, want %d (no lost increments)", got, workers*perWorker*2)
	}
}

func TestGuard_NegativeConfigClampedToZero(t *testing.T) {
	g, _ := New("", nil)
	_ = g.SetConfig(Config{Enabled: true, DailyTokens: -100, SessionTokens: -50, SoftWarnPct: 999})
	c := g.GetConfig()
	if c.DailyTokens != 0 || c.SessionTokens != 0 {
		t.Error("negative limits not clamped")
	}
	if c.SoftWarnPct != 80 {
		t.Errorf("softWarnPct=%d, want 80 (default after clamp)", c.SoftWarnPct)
	}
}
