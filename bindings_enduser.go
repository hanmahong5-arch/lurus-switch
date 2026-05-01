package main

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"lurus-switch/internal/appconfig"
	"lurus-switch/internal/redemption"

	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// ============================
// EndUser Activation (S-Xc.3 + S-Xc.4 + S-Xc.5) — redemption + heartbeat surface.
// ============================
//
// Frontend lifecycle:
//
//   1. App boot reads GetEndUserStatus(). If state == "unactivated", the
//      EndUserActivationPage is shown.
//   2. User submits a code → ActivateRedemption(code). On success, App.tsx
//      transitions to EndUserMainPage and the heartbeat goroutine starts.
//   3. The heartbeat emits "redemption:heartbeat" events whose payload
//      mirrors redemption.StatusEvent. The frontend listens to these and
//      bounces back to the activation page when state flips to revoked or
//      device_mismatch.
//   4. ClearActivation() (Settings page "重置激活") wipes the local file
//      and stops the heartbeat — user can re-activate with a new code.
//
// Hub URL is sourced exclusively from AppSettings.LockedHubURL (white-label
// builds) or AppSettings.Reseller.HubURL (debugging on a Reseller machine).
// We refuse to redeem against an arbitrary URL passed from the frontend —
// that would bypass the white-label lock.

// ActivationStatus mirrors redemption.Status with the additional fields
// the EndUser dashboard needs (brand display name, hub URL is already
// sourced from settings on the frontend).
type ActivationStatus struct {
	State         string    `json:"state"`
	StateReason   string    `json:"stateReason,omitempty"`
	Activated     bool      `json:"activated"`
	HubURL        string    `json:"hubUrl,omitempty"`
	TenantSlug    string    `json:"tenantSlug,omitempty"`
	UserID        int       `json:"userId,omitempty"`
	Quota         int64     `json:"quota,omitempty"`
	ExpiresAt     time.Time `json:"expiresAt,omitempty"`
	ActivatedAt   time.Time `json:"activatedAt,omitempty"`
	LastHeartbeat time.Time `json:"lastHeartbeat,omitempty"`
	Fingerprint   string    `json:"fingerprint,omitempty"`
}

// ActivationResult is the response returned to ActivateRedemption — same
// shape as ActivationStatus but a separate type so Wails generates a
// distinct TypeScript export (frontend can import each independently).
type ActivationResult = ActivationStatus

// GetDeviceFingerprint is exposed primarily for the activation page —
// users can copy-paste this when contacting support to confirm the right
// device is bound.
func (a *App) GetDeviceFingerprint() string {
	return redemption.DeviceFingerprint()
}

// GetEndUserStatus returns the current activation lifecycle state.
// Always succeeds — an unreadable activation file becomes StateMismatch,
// not an error.
func (a *App) GetEndUserStatus() (*ActivationStatus, error) {
	if a.redemptionStore == nil {
		return &ActivationStatus{State: string(redemption.StateUnactivated)}, nil
	}
	st := a.redemptionStore.Status(time.Now())
	return convertStatus(st), nil
}

// ActivateRedemption exchanges a code for a Hub-issued user token and
// persists the activation. The Hub URL is read from app settings — the
// caller does NOT supply it, since white-label builds must be locked
// to the embedded URL.
//
// Returns the resulting ActivationStatus on success. On a typed
// RedeemError, returns a Go error with a user-safe message and the kind
// embedded as a JSON suffix `[kind=...]` so the UI can choose its toast
// copy without parsing the message text.
func (a *App) ActivateRedemption(code string) (*ActivationResult, error) {
	if a.redemptionStore == nil || a.redeemer == nil {
		return nil, errors.New("redemption subsystem unavailable")
	}

	hubURL, err := resolveEndUserHubURL()
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(a.ctx, 30*time.Second)
	defer cancel()

	act, err := a.redeemer.Redeem(ctx, hubURL, code)
	if err != nil {
		if re, ok := redemption.IsRedeemError(err); ok {
			return nil, fmt.Errorf("%s [kind=%s]", re.Message, re.Kind)
		}
		return nil, err
	}
	if err := a.redemptionStore.Save(act); err != nil {
		return nil, fmt.Errorf("save activation: %w", err)
	}

	// Re-evaluate the heartbeat lifecycle now that we have a token. The
	// loop is a no-op when no activation is on disk; restarting it kicks
	// off an immediate first-tick.
	a.restartHeartbeatLocked()

	st := a.redemptionStore.Status(time.Now())
	return convertStatus(st), nil
}

// ClearActivation wipes the activation file and stops the heartbeat.
// Used by the EndUser settings "重置激活" button (Personal/Reseller modes
// don't surface this — there's nothing to clear).
func (a *App) ClearActivation() error {
	if a.redemptionStore == nil {
		return nil
	}
	if a.heartbeat != nil {
		a.heartbeat.Stop()
	}
	return a.redemptionStore.Clear()
}

// HeartbeatNow forces an immediate liveness probe — exposed primarily for
// QA / "I just changed something on the Hub, refresh" flows. Safe to call
// when the loop isn't running (no-op).
func (a *App) HeartbeatNow() error {
	if a.heartbeat == nil {
		return errors.New("heartbeat not initialized")
	}
	ctx, cancel := context.WithTimeout(a.ctx, 30*time.Second)
	defer cancel()
	return a.heartbeat.Tick(ctx)
}

// resolveEndUserHubURL picks the Hub coordinate the activation must
// target. Priority:
//
//  1. AppSettings.LockedHubURL — set by white-label packager; immutable.
//  2. AppSettings.Reseller.HubURL — for testing the EndUser flow on a
//     Reseller's own machine before exporting a white-label build.
//
// Empty result → user-facing error pointing to the white-label builder.
func resolveEndUserHubURL() (string, error) {
	s, err := appconfig.LoadAppSettings()
	if err != nil {
		return "", fmt.Errorf("load app settings: %w", err)
	}
	if locked := strings.TrimSpace(s.LockedHubURL); locked != "" {
		return locked, nil
	}
	if hub := strings.TrimSpace(s.Reseller.HubURL); hub != "" {
		return hub, nil
	}
	return "", errors.New("Hub URL 未配置：请使用经销商提供的白标安装包，或在 Reseller 模式下配置 Hub。")
}

// convertStatus maps the internal redemption.Status (snake-cased Go
// fields) to the JSON-friendly ActivationStatus surfaced to the frontend.
func convertStatus(st redemption.Status) *ActivationStatus {
	return &ActivationStatus{
		State:         string(st.State),
		StateReason:   st.StateReason,
		Activated:     st.Activated,
		HubURL:        st.HubURL,
		TenantSlug:    st.TenantSlug,
		UserID:        st.UserID,
		Quota:         st.Quota,
		ExpiresAt:     st.ExpiresAt,
		ActivatedAt:   st.ActivatedAt,
		LastHeartbeat: st.LastHeartbeat,
	}
}

// restartHeartbeatLocked stops the current heartbeat (if any) and starts
// a new one bound to a.ctx. Called after an activation flips state or
// after a clear.
func (a *App) restartHeartbeatLocked() {
	if a.heartbeat != nil {
		a.heartbeat.Stop()
	}
	emit := func(event string, payload any) {
		if a.ctx == nil {
			return
		}
		wailsRuntime.EventsEmit(a.ctx, event, payload)
	}
	a.heartbeat = redemption.NewHeartbeat(a.redemptionStore, AppVersion, emit)
	if a.ctx != nil {
		_ = a.heartbeat.Start(a.ctx)
	}
}
