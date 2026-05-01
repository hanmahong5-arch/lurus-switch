package redemption

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

// heartbeatInterval is the wake-up cadence. Five minutes balances rapid
// detection of a revoked code (worst case 5min reuse window) against load
// on a small Hub serving thousands of EndUsers.
const heartbeatInterval = 5 * time.Minute

// heartbeatTimeout caps a single heartbeat request. Longer than typical
// HTTP latency, shorter than the interval so we never overlap.
const heartbeatTimeout = 20 * time.Second

// heartbeatPath is the V2 multi-tenant heartbeat endpoint. Hub returns:
//
//	{ "status": "active" | "expired" | "revoked", "expires_at": <unix>, "quota": <int> }
//
// When TenantSlug is empty (single-tenant Hub), we fall back to the
// non-tenant path `/api/v2/switch/heartbeat`.
const (
	heartbeatPathTenant      = "/api/v2/%s/user/heartbeat"
	heartbeatPathSingleTenan = "/api/v2/switch/heartbeat"
)

// HeartbeatResponse mirrors the Hub heartbeat reply. Optional fields are
// pointers so the absence-vs-zero distinction is preserved (some Hubs may
// only echo "status").
type HeartbeatResponse struct {
	Status    string `json:"status"`
	ExpiresAt int64  `json:"expires_at,omitempty"`
	Quota     int64  `json:"quota,omitempty"`
}

// StatusEvent is what the Hub <-> Store loop emits when something changes.
// Wails consumers subscribe via "redemption:heartbeat" for UI updates.
type StatusEvent struct {
	Status    string    `json:"status"`     // last server-reported status
	State     State     `json:"state"`      // computed local state (active/stale/revoked/...)
	UpdatedAt time.Time `json:"updated_at"` // wall clock the event was produced
	Message   string    `json:"message,omitempty"`
}

// EmitFunc is how the heartbeat loop notifies the rest of the app.
// Decoupled from wails.EventsEmit so tests can capture events without a
// running Wails runtime.
type EmitFunc func(event string, payload any)

// Heartbeat is the long-running client. Start once at app startup; Stop
// in shutdown. Methods are safe for concurrent use.
type Heartbeat struct {
	store      *Store
	httpClient *http.Client
	emit       EmitFunc
	appVersion string

	mu       sync.Mutex
	cancel   context.CancelFunc
	stopOnce sync.Once
	running  bool
}

// NewHeartbeat constructs a Heartbeat tied to the given activation store.
// emit may be nil — in that case the heartbeat updates the store but does
// not notify the UI.
func NewHeartbeat(store *Store, appVersion string, emit EmitFunc) *Heartbeat {
	return &Heartbeat{
		store:      store,
		httpClient: &http.Client{Timeout: heartbeatTimeout},
		emit:       emit,
		appVersion: appVersion,
	}
}

// Start begins the heartbeat loop. Returns ErrAlreadyRunning on a second
// call. The loop fires immediately, then every heartbeatInterval.
//
// The loop exits when ctx is cancelled OR Stop is called. Either way the
// store is left in its last-known state.
func (h *Heartbeat) Start(parentCtx context.Context) error {
	h.mu.Lock()
	if h.running {
		h.mu.Unlock()
		return ErrAlreadyRunning
	}
	ctx, cancel := context.WithCancel(parentCtx)
	h.cancel = cancel
	h.running = true
	h.mu.Unlock()

	go h.loop(ctx)
	return nil
}

// Stop signals the loop to exit and blocks until the next heartbeat tick
// observes the cancellation. Idempotent.
func (h *Heartbeat) Stop() {
	h.stopOnce.Do(func() {
		h.mu.Lock()
		defer h.mu.Unlock()
		if h.cancel != nil {
			h.cancel()
		}
		h.running = false
	})
}

// ErrAlreadyRunning is returned when Start is called twice without Stop.
var ErrAlreadyRunning = errors.New("heartbeat already running")

// Tick runs a single heartbeat synchronously. Exposed for tests and for
// the binding layer's "force heartbeat now" debug method.
func (h *Heartbeat) Tick(ctx context.Context) error {
	return h.tickOnce(ctx)
}

func (h *Heartbeat) loop(ctx context.Context) {
	// Fire once immediately so the UI doesn't show "never heartbeated" for
	// the first 5 minutes after boot.
	_ = h.tickOnce(ctx)

	t := time.NewTicker(heartbeatInterval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			_ = h.tickOnce(ctx)
		}
	}
}

func (h *Heartbeat) tickOnce(ctx context.Context) error {
	act, err := h.store.Load()
	if err != nil {
		// Decryption failure — most likely device fingerprint changed.
		// Emit so UI can switch to the activation page.
		h.notify(StatusEvent{
			Status:    "device_mismatch",
			State:     StateMismatch,
			UpdatedAt: time.Now().UTC(),
			Message:   err.Error(),
		})
		return err
	}
	if act == nil {
		return nil // Not activated; nothing to heartbeat.
	}

	resp, err := h.callHub(ctx, act)
	if err != nil {
		// Network failure — don't flip state on a single miss; the grace
		// period handles transient outages. Just emit so any "connecting…"
		// indicator can update.
		h.notify(StatusEvent{
			Status:    "transient",
			State:     h.store.Status(time.Now()).State,
			UpdatedAt: time.Now().UTC(),
			Message:   err.Error(),
		})
		return err
	}

	// Persist the new heartbeat status so Status() reflects it after restart.
	now := time.Now().UTC()
	if err := h.store.UpdateHeartbeat(now, resp.Status); err != nil {
		return err
	}

	h.notify(StatusEvent{
		Status:    resp.Status,
		State:     h.store.Status(now).State,
		UpdatedAt: now,
	})
	return nil
}

func (h *Heartbeat) callHub(ctx context.Context, act *Activation) (*HeartbeatResponse, error) {
	hubURL := strings.TrimRight(act.HubURL, "/")
	path := heartbeatPathSingleTenan
	if act.TenantSlug != "" {
		path = fmt.Sprintf(heartbeatPathTenant, act.TenantSlug)
	}

	body, _ := json.Marshal(map[string]any{
		"fingerprint": act.Fingerprint,
		"app_version": h.appVersion,
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, hubURL+path, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("heartbeat build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", act.UserToken)
	req.Header.Set("X-Device-Fingerprint", act.Fingerprint)

	resp, err := h.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("heartbeat: %w", err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)

	// 401 → token revoked; record it as a hard revocation rather than
	// transient since the Hub explicitly rejected our credentials.
	if resp.StatusCode == http.StatusUnauthorized {
		return &HeartbeatResponse{Status: "revoked"}, nil
	}
	// 404 — old Hub without the heartbeat endpoint. Treat as a soft
	// "active" so the EndUser keeps working. The grace period is enough
	// of a safety net.
	if resp.StatusCode == http.StatusNotFound {
		return &HeartbeatResponse{Status: "active"}, nil
	}

	var env struct {
		Success bool            `json:"success"`
		Message string          `json:"message"`
		Data    json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(raw, &env); err != nil {
		return nil, fmt.Errorf("heartbeat: non-JSON response (HTTP %d)", resp.StatusCode)
	}
	if !env.Success {
		return nil, fmt.Errorf("heartbeat rejected: %s", env.Message)
	}
	var hb HeartbeatResponse
	if len(env.Data) > 0 && string(env.Data) != "null" {
		if err := json.Unmarshal(env.Data, &hb); err != nil {
			return nil, fmt.Errorf("heartbeat decode: %w", err)
		}
	}
	if hb.Status == "" {
		hb.Status = "active"
	}
	return &hb, nil
}

func (h *Heartbeat) notify(ev StatusEvent) {
	if h.emit == nil {
		return
	}
	h.emit("redemption:heartbeat", ev)
}
