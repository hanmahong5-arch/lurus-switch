package redemption

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// TestDeviceFingerprint_Stable_AndShape verifies the fingerprint is
// deterministic across calls within a single process, and conforms to
// the documented length / hex shape. Cross-machine variation is exercised
// implicitly — if a different machine returns the same fingerprint we'd
// see flakes here once a binary moves between CI runners.
func TestDeviceFingerprint_Stable_AndShape(t *testing.T) {
	a := DeviceFingerprint()
	b := DeviceFingerprint()
	if a != b {
		t.Fatalf("fingerprint not stable: %q vs %q", a, b)
	}
	if len(a) != fingerprintLength {
		t.Fatalf("fingerprint length = %d, want %d", len(a), fingerprintLength)
	}
	for _, c := range a {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
			t.Fatalf("fingerprint has non-hex char: %q", a)
		}
	}
}

// TestStore_SaveLoad_RoundTrip ensures a saved Activation round-trips
// through encrypt/decrypt without field loss.
func TestStore_SaveLoad_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	s := NewStoreAt(dir)

	want := &Activation{
		HubURL:      "https://hub.acme.example",
		TenantSlug:  "acme",
		UserToken:   "tok-12345",
		UserID:      42,
		Quota:       1_000_000,
		ExpiresAt:   time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC),
		Fingerprint: DeviceFingerprint(),
		ActivatedAt: time.Date(2026, 5, 1, 10, 0, 0, 0, time.UTC),
	}
	if err := s.Save(want); err != nil {
		t.Fatalf("Save: %v", err)
	}

	// Force a fresh Store to bypass the in-memory cache.
	s2 := NewStoreAt(dir)
	got, err := s2.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got == nil {
		t.Fatal("Load returned nil after Save")
	}
	if got.UserToken != want.UserToken || got.UserID != want.UserID || got.Quota != want.Quota {
		t.Errorf("token/user/quota mismatch: got %+v, want %+v", got, want)
	}
	if !got.ExpiresAt.Equal(want.ExpiresAt) {
		t.Errorf("ExpiresAt mismatch: got %v, want %v", got.ExpiresAt, want.ExpiresAt)
	}
}

// TestStore_Load_NoFile_IsNotError captures the contract that an
// unactivated install returns (nil, nil), not an error — the activation
// page relies on this to render without a panic.
func TestStore_Load_NoFile_IsNotError(t *testing.T) {
	s := NewStoreAt(t.TempDir())
	got, err := s.Load()
	if err != nil {
		t.Fatalf("Load on empty dir returned error: %v", err)
	}
	if got != nil {
		t.Fatalf("Load on empty dir returned %+v, want nil", got)
	}
}

// TestStore_Status_Lifecycle walks the full state machine to lock in the
// transition rules — these are the contract the EndUser activation gate
// enforces in App.tsx.
func TestStore_Status_Lifecycle(t *testing.T) {
	s := NewStoreAt(t.TempDir())

	// Unactivated.
	if got := s.Status(time.Now()); got.State != StateUnactivated {
		t.Errorf("empty store state = %v, want %v", got.State, StateUnactivated)
	}

	// Save → Active (no heartbeat yet).
	now := time.Now().UTC()
	act := &Activation{
		HubURL:      "https://hub.example",
		UserToken:   "tok",
		Fingerprint: DeviceFingerprint(),
		ActivatedAt: now,
		ExpiresAt:   now.Add(30 * 24 * time.Hour),
	}
	if err := s.Save(act); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if got := s.Status(now); got.State != StateActive {
		t.Errorf("fresh save state = %v, want %v", got.State, StateActive)
	}

	// Heartbeat: stale.
	if err := s.UpdateHeartbeat(now.Add(-2*HeartbeatGrace), "active"); err != nil {
		t.Fatalf("UpdateHeartbeat: %v", err)
	}
	if got := s.Status(now); got.State != StateStale {
		t.Errorf("stale heartbeat state = %v, want %v", got.State, StateStale)
	}

	// Heartbeat: revoked → state moves to Revoked even with fresh timestamp.
	if err := s.UpdateHeartbeat(now, "revoked"); err != nil {
		t.Fatalf("UpdateHeartbeat: %v", err)
	}
	if got := s.Status(now); got.State != StateRevoked {
		t.Errorf("revoked status state = %v, want %v", got.State, StateRevoked)
	}

	// Clear → Unactivated again.
	if err := s.Clear(); err != nil {
		t.Fatalf("Clear: %v", err)
	}
	if got := s.Status(now); got.State != StateUnactivated {
		t.Errorf("post-clear state = %v, want %v", got.State, StateUnactivated)
	}
}

// TestStore_Status_ExpiresAt covers the "ExpiresAt past now" case —
// independent of heartbeat status, an expired activation must be treated
// as revoked so the UI prompts for a new code.
func TestStore_Status_ExpiresAt(t *testing.T) {
	s := NewStoreAt(t.TempDir())
	now := time.Now().UTC()
	act := &Activation{
		HubURL:      "https://hub.example",
		UserToken:   "tok",
		Fingerprint: DeviceFingerprint(),
		ActivatedAt: now.Add(-48 * time.Hour),
		ExpiresAt:   now.Add(-1 * time.Hour),
	}
	if err := s.Save(act); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if got := s.Status(now); got.State != StateRevoked {
		t.Errorf("expired activation state = %v, want %v", got.State, StateRevoked)
	}
}

// TestRedeem_Success_PopulatesActivation drives the happy path against a
// stub Hub that mirrors the V2 envelope shape.
func TestRedeem_Success_PopulatesActivation(t *testing.T) {
	hub := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != redeemEndpoint {
			t.Errorf("unexpected path %q", r.URL.Path)
		}
		if r.Header.Get("X-Device-Fingerprint") == "" {
			t.Error("missing X-Device-Fingerprint header")
		}
		var req RedeemRequest
		_ = json.NewDecoder(r.Body).Decode(&req)
		if req.Code != "ABC123" {
			t.Errorf("code = %q, want ABC123", req.Code)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"success": true,
			"message": "ok",
			"data": map[string]any{
				"user_token":  "tok-secret",
				"user_id":     7,
				"quota":       int64(500_000),
				"expires_at":  time.Now().Add(24 * time.Hour).Unix(),
				"tenant_slug": "acme",
			},
		})
	}))
	defer hub.Close()

	r := NewRedeemer("test")
	act, err := r.Redeem(context.Background(), hub.URL, "ABC123")
	if err != nil {
		t.Fatalf("Redeem: %v", err)
	}
	if act.UserToken != "tok-secret" || act.UserID != 7 || act.Quota != 500_000 {
		t.Errorf("activation fields wrong: %+v", act)
	}
	if act.TenantSlug != "acme" {
		t.Errorf("tenant slug = %q, want acme", act.TenantSlug)
	}
	if act.Fingerprint == "" {
		t.Error("fingerprint not captured")
	}
}

// TestRedeem_404_MapsToEndpointAbsent locks in the user-friendly error for
// Hubs that haven't deployed the redeem endpoint yet — without this, the
// EndUser sees a generic "code not found" and assumes their code is bad.
func TestRedeem_404_MapsToEndpointAbsent(t *testing.T) {
	hub := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.NotFound(w, nil)
	}))
	defer hub.Close()

	r := NewRedeemer("test")
	_, err := r.Redeem(context.Background(), hub.URL, "ABC")
	re, ok := IsRedeemError(err)
	if !ok {
		t.Fatalf("Redeem error type = %T, want *RedeemError", err)
	}
	if re.Kind != ErrEndpointAbsent {
		t.Errorf("kind = %v, want %v", re.Kind, ErrEndpointAbsent)
	}
}

// TestRedeem_HubFailure_ClassifiesByMessage exercises the message
// classifier so a future Hub copy edit doesn't silently lump every error
// into "code not found".
func TestRedeem_HubFailure_ClassifiesByMessage(t *testing.T) {
	cases := []struct {
		name    string
		message string
		want    RedeemErrorKind
	}{
		{"used", "兑换码已使用", ErrCodeUsed},
		{"used-en", "code already redeemed", ErrCodeUsed},
		{"expired", "兑换码已过期", ErrCodeExpired},
		{"disabled", "兑换码已禁用", ErrCodeDisabled},
		{"not_found", "兑换码不存在", ErrCodeNotFound},
		{"invalid", "invalid code", ErrCodeNotFound},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := classifyRedeemFailure(http.StatusBadRequest, tc.message)
			re, ok := IsRedeemError(err)
			if !ok {
				t.Fatalf("not a RedeemError: %T", err)
			}
			if re.Kind != tc.want {
				t.Errorf("message %q → kind %v, want %v", tc.message, re.Kind, tc.want)
			}
		})
	}
}

// TestRedeem_InvalidInput_NeverHitsNetwork ensures we short-circuit empty
// fields before issuing an HTTP request — preserves the user's quota of
// "real" attempts when the Hub rate-limits.
func TestRedeem_InvalidInput_NeverHitsNetwork(t *testing.T) {
	called := false
	hub := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))
	defer hub.Close()

	r := NewRedeemer("test")
	if _, err := r.Redeem(context.Background(), "", "ABC"); err == nil {
		t.Error("empty hub URL should error")
	}
	if _, err := r.Redeem(context.Background(), hub.URL, ""); err == nil {
		t.Error("empty code should error")
	}
	if called {
		t.Error("Hub was hit despite invalid input")
	}
}

// TestHeartbeat_Tick_UpdatesStore drives a single tick through the
// heartbeat client against a stub Hub, asserting both the store mutation
// and the emitted event.
func TestHeartbeat_Tick_UpdatesStore(t *testing.T) {
	hub := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "tok" {
			t.Error("missing Authorization header")
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"success": true,
			"data":    map[string]any{"status": "active"},
		})
	}))
	defer hub.Close()

	store := NewStoreAt(t.TempDir())
	if err := store.Save(&Activation{
		HubURL:      hub.URL,
		UserToken:   "tok",
		Fingerprint: DeviceFingerprint(),
		ActivatedAt: time.Now().UTC().Add(-time.Hour),
	}); err != nil {
		t.Fatalf("Save: %v", err)
	}

	var got StatusEvent
	hb := NewHeartbeat(store, "test", func(_ string, payload any) {
		if ev, ok := payload.(StatusEvent); ok {
			got = ev
		}
	})
	if err := hb.Tick(context.Background()); err != nil {
		t.Fatalf("Tick: %v", err)
	}
	if got.Status != "active" {
		t.Errorf("event status = %q, want active", got.Status)
	}
	loaded, _ := store.Load()
	if loaded.HeartbeatStatus != "active" || loaded.LastHeartbeat.IsZero() {
		t.Errorf("store not updated: %+v", loaded)
	}
}

// TestHeartbeat_401_Revokes locks in the contract that an unauthorized
// heartbeat permanently flags the activation as revoked — the EndUser
// must be bounced to the activation page on the next status query.
func TestHeartbeat_401_Revokes(t *testing.T) {
	hub := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
	}))
	defer hub.Close()

	store := NewStoreAt(t.TempDir())
	if err := store.Save(&Activation{
		HubURL:      hub.URL,
		UserToken:   "tok",
		Fingerprint: DeviceFingerprint(),
		ActivatedAt: time.Now().UTC(),
	}); err != nil {
		t.Fatalf("Save: %v", err)
	}

	hb := NewHeartbeat(store, "test", nil)
	if err := hb.Tick(context.Background()); err != nil {
		t.Fatalf("Tick: %v", err)
	}
	if got := store.Status(time.Now()); got.State != StateRevoked {
		t.Errorf("post-401 state = %v, want %v", got.State, StateRevoked)
	}
}
