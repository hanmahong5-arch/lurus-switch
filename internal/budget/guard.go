// Package budget enforces a hard token-spend ceiling at the Switch
// gateway. Defends against the 2025-2026 horror story where a runaway
// Claude Code session burned $1,600 in tokens overnight (#token-burn
// thread on r/ClaudeCode). Two limits are tracked:
//
//   - Daily   — total tokens routed through this Switch instance today.
//   - Session — tokens since the user last clicked "reset session".
//
// When either limit is hit, Check() returns Allowed=false and the
// gateway responds 429 with a friendly "you've hit your spend wall"
// payload. The user lifts the wall by raising the limit or clicking
// reset — never silently, so they're aware the cap was reached.
package budget

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"lurus-switch/internal/metering"
)

// Config is persisted to disk as JSON. 0 means "no limit on this axis".
type Config struct {
	Enabled       bool  `json:"enabled"`
	DailyTokens   int64 `json:"dailyTokens"`
	SessionTokens int64 `json:"sessionTokens"`
	// SoftWarnPct triggers a non-blocking warning event when usage
	// crosses this percentage of either limit. 0 disables the warning.
	SoftWarnPct int `json:"softWarnPct"`
}

func DefaultConfig() Config {
	return Config{
		Enabled:       false, // off by default — opt-in
		DailyTokens:   0,
		SessionTokens: 0,
		SoftWarnPct:   80,
	}
}

// Verdict tells the gateway whether to forward the request.
type Verdict struct {
	Allowed   bool   `json:"allowed"`
	Reason    string `json:"reason,omitempty"`
	LimitKind string `json:"limitKind,omitempty"` // "daily" | "session"
	Used      int64  `json:"used,omitempty"`
	Limit     int64  `json:"limit,omitempty"`
}

// Status is what the UI consumes to render gauges.
type Status struct {
	Enabled       bool      `json:"enabled"`
	DailyTokens   int64     `json:"dailyTokens"`   // configured limit
	SessionTokens int64     `json:"sessionTokens"` // configured limit
	DailyUsed     int64     `json:"dailyUsed"`     // current total
	SessionUsed   int64     `json:"sessionUsed"`   // current total
	DailyPct      int       `json:"dailyPct"`      // 0..100, 0 if no limit
	SessionPct    int       `json:"sessionPct"`    // 0..100
	SessionStart  time.Time `json:"sessionStart"`
	SoftWarnPct   int       `json:"softWarnPct"`
	HitDaily      bool      `json:"hitDaily"`
	HitSession    bool      `json:"hitSession"`
	WarnDaily     bool      `json:"warnDaily"`
	WarnSession   bool      `json:"warnSession"`
}

// Guard is the live in-process budget enforcer. The session counter is
// process-lifetime; the daily counter delegates to the metering store
// (which is the authoritative source for "tokens today"), keeping us
// from double-counting.
type Guard struct {
	mu           sync.RWMutex
	cfg          Config
	cfgPath      string
	sessionUsed  atomic.Int64
	sessionStart time.Time
	today        func() metering.DailySummary
}

// New loads the persisted config (if any) and initialises a Guard.
// If `today` is nil, daily-limit checks are bypassed (useful for tests
// without a metering store).
func New(cfgPath string, today func() metering.DailySummary) (*Guard, error) {
	g := &Guard{
		cfgPath:      cfgPath,
		sessionStart: time.Now(),
		today:        today,
	}
	g.cfg = DefaultConfig()
	if cfgPath != "" {
		if data, err := os.ReadFile(cfgPath); err == nil {
			var c Config
			if json.Unmarshal(data, &c) == nil {
				g.cfg = c
			}
		}
	}
	return g, nil
}

func (g *Guard) GetConfig() Config {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.cfg
}

// SetConfig validates and persists. Negative limits are clamped to 0
// (= unlimited) so the UI can't accidentally lock users out.
func (g *Guard) SetConfig(c Config) error {
	if c.DailyTokens < 0 {
		c.DailyTokens = 0
	}
	if c.SessionTokens < 0 {
		c.SessionTokens = 0
	}
	if c.SoftWarnPct < 0 || c.SoftWarnPct > 100 {
		c.SoftWarnPct = 80
	}
	g.mu.Lock()
	g.cfg = c
	g.mu.Unlock()
	if g.cfgPath == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(g.cfgPath), 0o755); err != nil {
		return err
	}
	body, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(g.cfgPath, body, 0o644)
}

// ResetSession zeroes the session counter and re-stamps the start time.
// The counter is mutated under g.mu so Check() observes the reset and the
// start-time restamp as one atomic step (no window where the counter is
// zeroed but Check still sees the old value).
func (g *Guard) ResetSession() {
	g.mu.Lock()
	g.sessionUsed.Store(0)
	g.sessionStart = time.Now()
	g.mu.Unlock()
}

// RecordUsage is called by the gateway AFTER a successful upstream
// response, with the actual token count from the response body.
func (g *Guard) RecordUsage(in, out int64) {
	if in < 0 {
		in = 0
	}
	if out < 0 {
		out = 0
	}
	// Mutate the session counter under g.mu so a concurrent Check() sees
	// either the pre- or post-increment value as a consistent snapshot,
	// never a torn read. The counter stays atomic.Int64 so Status() (which
	// only reads) needs no lock.
	g.mu.Lock()
	g.sessionUsed.Add(in + out)
	g.mu.Unlock()
}

// Check is called BEFORE forwarding. Returns Allowed=false when either
// limit is already exceeded.
//
// Concurrency contract (the session axis is now atomic, the daily axis is
// not): the session counter read + threshold comparison happen under a
// single critical section, so N concurrent callers observe a consistent
// snapshot rather than racing a half-applied RecordUsage. We still do not
// predict the size of the upcoming request, so a request whose body lands
// while usage sits just under the cap may graze it by one — but two
// concurrent callers can no longer BOTH see "under limit" against a counter
// one of them is mid-incrementing. The daily axis delegates to the metering
// store (an external source we cannot atomically reserve against), so it
// stays advisory; that is documented at the call site below.
func (g *Guard) Check() Verdict {
	cfg := g.GetConfig()
	if !cfg.Enabled {
		return Verdict{Allowed: true}
	}
	if cfg.DailyTokens > 0 && g.today != nil {
		// Daily axis is advisory: g.today() reads the metering store, an
		// independent owner of "tokens today". We cannot reserve against it
		// atomically, so concurrent requests may each pass this gate before
		// the store reflects their usage.
		s := g.today()
		used := s.TokensIn + s.TokensOut
		if used >= cfg.DailyTokens {
			return Verdict{
				Allowed: false, LimitKind: "daily", Used: used, Limit: cfg.DailyTokens,
				Reason: fmt.Sprintf("daily token cap reached: %d / %d", used, cfg.DailyTokens),
			}
		}
	}
	if cfg.SessionTokens > 0 {
		// Session axis is atomic: take the guard lock so the read and the
		// comparison are a single snapshot consistent with RecordUsage /
		// ResetSession (both of which mutate under this same discipline).
		g.mu.Lock()
		used := g.sessionUsed.Load()
		hit := used >= cfg.SessionTokens
		g.mu.Unlock()
		if hit {
			return Verdict{
				Allowed: false, LimitKind: "session", Used: used, Limit: cfg.SessionTokens,
				Reason: fmt.Sprintf("session token cap reached: %d / %d", used, cfg.SessionTokens),
			}
		}
	}
	return Verdict{Allowed: true}
}

func (g *Guard) Status() Status {
	cfg := g.GetConfig()
	g.mu.RLock()
	start := g.sessionStart
	g.mu.RUnlock()

	st := Status{
		Enabled:       cfg.Enabled,
		DailyTokens:   cfg.DailyTokens,
		SessionTokens: cfg.SessionTokens,
		SessionStart:  start,
		SoftWarnPct:   cfg.SoftWarnPct,
		SessionUsed:   g.sessionUsed.Load(),
	}
	if g.today != nil {
		s := g.today()
		st.DailyUsed = s.TokensIn + s.TokensOut
	}
	if cfg.DailyTokens > 0 {
		st.DailyPct = pctClamped(st.DailyUsed, cfg.DailyTokens)
		st.HitDaily = st.DailyUsed >= cfg.DailyTokens
		st.WarnDaily = !st.HitDaily && cfg.SoftWarnPct > 0 && st.DailyPct >= cfg.SoftWarnPct
	}
	if cfg.SessionTokens > 0 {
		st.SessionPct = pctClamped(st.SessionUsed, cfg.SessionTokens)
		st.HitSession = st.SessionUsed >= cfg.SessionTokens
		st.WarnSession = !st.HitSession && cfg.SoftWarnPct > 0 && st.SessionPct >= cfg.SoftWarnPct
	}
	return st
}

func pctClamped(used, limit int64) int {
	if limit <= 0 {
		return 0
	}
	p := int(used * 100 / limit)
	if p < 0 {
		return 0
	}
	if p > 100 {
		return 100
	}
	return p
}
