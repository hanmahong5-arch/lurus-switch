package budget

import (
	"path/filepath"
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
