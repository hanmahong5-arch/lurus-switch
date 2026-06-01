package auth

import (
	"context"
	"errors"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// newTestSession builds a Session backed by a temp-dir auth.enc with a
// frozen clock so expiry math is deterministic.
func newTestSession(t *testing.T, now func() time.Time) *Session {
	t.Helper()
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i)
	}
	return &Session{
		filePath:      filepath.Join(t.TempDir(), "auth.enc"),
		encryptionKey: key,
		now:           now,
	}
}

func TestSession_IsExpired_UsesInjectedClock(t *testing.T) {
	base := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
	cases := []struct {
		name      string
		expiresAt time.Time
		buffer    time.Duration
		now       time.Time
		want      bool
	}{
		{"valid well ahead", base.Add(time.Hour), time.Minute, base, false},
		{"already past", base.Add(-time.Minute), 0, base, true},
		{"within buffer", base.Add(30 * time.Second), time.Minute, base, true},
		{"exactly at edge of buffer", base.Add(time.Minute), time.Minute, base, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := newTestSession(t, func() time.Time { return tc.now })
			s.tokens = &storedTokens{AccessToken: "a", ExpiresAt: tc.expiresAt}
			if got := s.IsExpired(tc.buffer); got != tc.want {
				t.Fatalf("IsExpired(%v) = %v, want %v", tc.buffer, got, tc.want)
			}
		})
	}
}

func TestSession_IsExpired_NilTokensIsExpired(t *testing.T) {
	s := newTestSession(t, time.Now)
	if !s.IsExpired(0) {
		t.Fatal("nil tokens must report expired")
	}
}

func TestSession_EnsureFresh_SkipsWhenStillValid(t *testing.T) {
	base := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
	s := newTestSession(t, func() time.Time { return base })
	s.tokens = &storedTokens{
		AccessToken:  "still-good",
		RefreshToken: "r",
		ExpiresAt:    base.Add(time.Hour),
	}
	var calls int32
	refresh := func(ctx context.Context, rt string) (*TokenResponse, error) {
		atomic.AddInt32(&calls, 1)
		return &TokenResponse{AccessToken: "new"}, nil
	}
	if err := s.EnsureFresh(context.Background(), time.Minute, refresh); err != nil {
		t.Fatalf("EnsureFresh returned error: %v", err)
	}
	if got := atomic.LoadInt32(&calls); got != 0 {
		t.Fatalf("refresh called %d times, want 0 (token still valid)", got)
	}
	if s.GetAccessToken() != "still-good" {
		t.Fatalf("access token mutated: %q", s.GetAccessToken())
	}
}

func TestSession_EnsureFresh_RefreshesWhenNearExpiry(t *testing.T) {
	base := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
	s := newTestSession(t, func() time.Time { return base })
	s.tokens = &storedTokens{
		AccessToken:  "old",
		RefreshToken: "refresh-tok",
		ExpiresAt:    base.Add(30 * time.Second), // within the 1m buffer
	}
	var gotRT string
	refresh := func(ctx context.Context, rt string) (*TokenResponse, error) {
		gotRT = rt
		return &TokenResponse{AccessToken: "fresh", RefreshToken: "rotated", ExpiresIn: 3600}, nil
	}
	if err := s.EnsureFresh(context.Background(), time.Minute, refresh); err != nil {
		t.Fatalf("EnsureFresh returned error: %v", err)
	}
	if gotRT != "refresh-tok" {
		t.Fatalf("refresh got refresh-token %q, want %q", gotRT, "refresh-tok")
	}
	if s.GetAccessToken() != "fresh" {
		t.Fatalf("access token = %q, want fresh", s.GetAccessToken())
	}
	if s.GetRefreshToken() != "rotated" {
		t.Fatalf("refresh token = %q, want rotated", s.GetRefreshToken())
	}
}

func TestSession_EnsureFresh_NoRefreshTokenReturnsErr(t *testing.T) {
	base := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
	s := newTestSession(t, func() time.Time { return base })
	s.tokens = &storedTokens{
		AccessToken: "expired",
		ExpiresAt:   base.Add(-time.Hour),
	}
	called := false
	refresh := func(ctx context.Context, rt string) (*TokenResponse, error) {
		called = true
		return nil, nil
	}
	err := s.EnsureFresh(context.Background(), time.Minute, refresh)
	if err == nil {
		t.Fatal("expected error when no refresh token present")
	}
	if called {
		t.Fatal("refresh must not be called without a refresh token")
	}
}

func TestSession_EnsureFresh_FailureNeverClearsSession(t *testing.T) {
	base := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
	s := newTestSession(t, func() time.Time { return base })
	s.tokens = &storedTokens{
		AccessToken:  "expired",
		RefreshToken: "r",
		ExpiresAt:    base.Add(-time.Hour),
	}
	wantErr := errors.New("network down")
	refresh := func(ctx context.Context, rt string) (*TokenResponse, error) {
		return nil, wantErr
	}
	err := s.EnsureFresh(context.Background(), time.Minute, refresh)
	if !errors.Is(err, wantErr) {
		t.Fatalf("EnsureFresh err = %v, want %v", err, wantErr)
	}
	// Session must be intact for the caller to decide what to do.
	if s.GetAccessToken() != "expired" {
		t.Fatalf("session was mutated on refresh failure: %q", s.GetAccessToken())
	}
	if !s.GetAuthState().IsLoggedIn {
		t.Fatal("EnsureFresh must NOT clear the session on refresh failure")
	}
}

func TestSession_EnsureFresh_SingleFlightCoalescesConcurrentRefresh(t *testing.T) {
	base := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
	s := newTestSession(t, func() time.Time { return base })
	s.tokens = &storedTokens{
		AccessToken:  "old",
		RefreshToken: "r",
		ExpiresAt:    base.Add(-time.Hour), // expired
	}

	var calls int32
	release := make(chan struct{})
	entered := make(chan struct{}, 1)
	refresh := func(ctx context.Context, rt string) (*TokenResponse, error) {
		atomic.AddInt32(&calls, 1)
		select {
		case entered <- struct{}{}:
		default:
		}
		<-release // hold the in-flight refresh until both goroutines have raced
		return &TokenResponse{AccessToken: "fresh", RefreshToken: "r2", ExpiresIn: 3600}, nil
	}

	const n = 8
	var wg sync.WaitGroup
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func() {
			defer wg.Done()
			_ = s.EnsureFresh(context.Background(), time.Minute, refresh)
		}()
	}
	<-entered      // first refresh is now in-flight
	time.Sleep(20 * time.Millisecond) // let the others pile up on the single-flight gate
	close(release) // let the in-flight refresh complete
	wg.Wait()

	// The in-flight refresh updates the token; once fresh, the queued callers
	// re-check expiry under the guard and skip a second network round-trip.
	if got := atomic.LoadInt32(&calls); got != 1 {
		t.Fatalf("refresh called %d times, want exactly 1 (single-flight)", got)
	}
	if s.GetAccessToken() != "fresh" {
		t.Fatalf("access token = %q, want fresh", s.GetAccessToken())
	}
}
