package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"lurus-switch/internal/appconfig"
	"lurus-switch/internal/auth"
)

// ============================
// Auth Methods (OIDC / Zitadel PKCE)
// ============================

// GetAuthState returns the current authentication state.
func (a *App) GetAuthState() auth.AuthState {
	if a.authSession == nil {
		return auth.AuthState{IsLoggedIn: false}
	}
	return a.authSession.GetAuthState()
}

// Login initiates the PKCE login flow via system browser.
// On success, auto-provisions a gateway API token in the background.
func (a *App) Login() (auth.AuthState, error) {
	if a.authSession == nil {
		return auth.AuthState{}, fmt.Errorf("auth session not initialized")
	}

	cfg := a.getAuthConfig()
	if cfg.ClientID == "" {
		return auth.AuthState{}, fmt.Errorf("auth client_id not configured — please set it in Settings > Auth Client ID")
	}

	loginCtx, cancel := context.WithTimeout(a.ctx, 5*time.Minute)
	defer cancel()

	tokens, err := auth.LoginWithPKCE(loginCtx, cfg)
	if err != nil {
		return auth.AuthState{}, fmt.Errorf("login failed: %w", err)
	}

	if err := a.authSession.StoreTokens(tokens); err != nil {
		return auth.AuthState{}, fmt.Errorf("store tokens: %w", err)
	}

	// Fetch platform-core account snapshot (LurusID, wallet balance, VIP).
	// platform-core URL is DISTINCT from the OIDC issuer (identity.lurus.cn
	// vs auth.lurus.cn). Best-effort: a platform outage shouldn't fail
	// the OIDC login since the user IS logged in (Zitadel succeeded).
	platformURL := platformURLFromSettings()
	if pa, perr := auth.FetchPlatformAccount(a.ctx, platformURL, tokens.AccessToken); perr == nil {
		_ = a.authSession.SetPlatformAccount(pa)
		log.Printf("[auth] platform account loaded: lurus_id=%s balance=%.2f", pa.LurusID, pa.WalletBalance)
	} else {
		log.Printf("[auth] platform account fetch failed (non-fatal): %v", perr)
	}

	// Auto-provision gateway token after successful login.
	go safeGo("provision-gateway", func() { a.provisionGateway() })

	return a.authSession.GetAuthState(), nil
}

// RefreshPlatformAccount re-fetches the platform-core account snapshot
// without re-doing OIDC. Use when the UI wants up-to-date wallet
// balance after a top-up or purchase. Returns the fresh state or an
// error if no active session.
func (a *App) RefreshPlatformAccount() (auth.AuthState, error) {
	if a.authSession == nil {
		return auth.AuthState{}, fmt.Errorf("auth session not initialized")
	}
	state := a.authSession.GetAuthState()
	if !state.IsLoggedIn {
		return state, nil
	}
	accessToken := a.authSession.GetAccessToken()
	if accessToken == "" {
		return state, fmt.Errorf("no access token")
	}
	ctx, cancel := context.WithTimeout(a.ctx, 10*time.Second)
	defer cancel()
	pa, err := auth.FetchPlatformAccount(ctx, platformURLFromSettings(), accessToken)
	if err != nil {
		return state, fmt.Errorf("refresh platform account: %w", err)
	}
	_ = a.authSession.SetPlatformAccount(pa)
	return a.authSession.GetAuthState(), nil
}

// platformURLFromSettings returns the platform-core base URL — the
// app-settings override when set, else the built-in production default
// (auth.DefaultPlatformBaseURL). Centralising the lookup keeps every
// platform call site consistent.
func platformURLFromSettings() string {
	if settings, err := appconfig.LoadAppSettings(); err == nil && settings != nil && strings.TrimSpace(settings.AuthPlatformURL) != "" {
		return settings.AuthPlatformURL
	}
	return auth.DefaultPlatformBaseURL
}

// Logout clears the current session and resets the billing client.
func (a *App) Logout() error {
	// Reset the billing client so it will be re-created on next use.
	a.resetBillingClient()

	if a.authSession == nil {
		return nil
	}
	return a.authSession.Clear()
}

// authRefreshBuffer is how far ahead of expiry the token is considered
// stale. The background loop ticks at this cadence and EnsureFresh uses it
// as the pre-expiry window, so a token is renewed well before any in-flight
// request would hit a 401.
const authRefreshBuffer = 5 * time.Minute

// refreshFunc builds an auth.RefreshFunc bound to the current OIDC config.
// Shared by the manual RefreshAuth binding and the automatic loop so both
// exchange the refresh_token through the exact same path.
func (a *App) refreshFunc() auth.RefreshFunc {
	cfg := a.getAuthConfig()
	return func(ctx context.Context, refreshToken string) (*auth.TokenResponse, error) {
		return auth.RefreshAccessToken(ctx, cfg, refreshToken)
	}
}

// RefreshAuth refreshes the access token using the stored refresh token.
// Manual entry point (account popover). Unlike the automatic loop, a
// refresh FAILURE here clears the session so the user is dropped to the
// re-login screen — but only because the absence of a refresh token or a
// hard refresh error means the session can no longer be kept alive.
func (a *App) RefreshAuth() (auth.AuthState, error) {
	if a.authSession == nil {
		return auth.AuthState{}, fmt.Errorf("auth session not initialized")
	}

	refreshToken := a.authSession.GetRefreshToken()
	if refreshToken == "" {
		return auth.AuthState{IsLoggedIn: false}, nil
	}

	cfg := a.getAuthConfig()
	refreshCtx, cancel := context.WithTimeout(a.ctx, 30*time.Second)
	defer cancel()

	tokens, err := auth.RefreshAccessToken(refreshCtx, cfg, refreshToken)
	if err != nil {
		// Refresh failed — clear session so user can re-login.
		a.authSession.Clear() //nolint:errcheck
		return auth.AuthState{IsLoggedIn: false}, nil
	}

	if err := a.authSession.StoreTokens(tokens); err != nil {
		return auth.AuthState{}, fmt.Errorf("store refreshed tokens: %w", err)
	}

	return a.authSession.GetAuthState(), nil
}

// startAuthRefresh performs a best-effort refresh on startup (when the
// stored token is near/past expiry) and then launches a background timer
// that keeps the session live without manual action. No-op when there is
// no OIDC session (manual-proxy users have no refresh token, so EnsureFresh
// returns ErrNoRefreshToken and we simply don't start the loop).
func (a *App) startAuthRefresh(ctx context.Context) {
	if a.authSession == nil {
		return
	}
	if a.authSession.GetRefreshToken() == "" {
		return // non-OIDC login: nothing to keep fresh.
	}

	a.stopAuthRefresh = make(chan struct{})

	// Best-effort startup refresh, off the UI thread. A failure here is
	// non-fatal: the existing token (if any) stays usable, and the loop
	// will retry on its cadence. Never clears the session.
	go safeGo("auth-refresh-startup", func() {
		a.ensureAuthFresh(ctx)
		a.runAuthRefreshLoop(ctx)
	})
}

// ensureAuthFresh runs one pre-expiry refresh attempt through the session's
// single-flight guard. On failure it logs and leaves the session intact —
// the automatic path NEVER forces re-login (that decision belongs to the
// manual RefreshAuth / actual API 401 handling). A missing refresh token is
// treated as "nothing to do", not an error to surface.
func (a *App) ensureAuthFresh(ctx context.Context) {
	if a.authSession == nil {
		return
	}
	refreshCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	err := a.authSession.EnsureFresh(refreshCtx, authRefreshBuffer, a.refreshFunc())
	switch {
	case err == nil:
		return
	case errors.Is(err, auth.ErrNoRefreshToken):
		return // not an OIDC session — nothing to refresh.
	default:
		// Keep the session as-is; a transient network failure must not
		// drop a valid login. The loop will retry next tick.
		log.Printf("[auth] background token refresh failed (non-fatal): %v", err)
	}
}

// runAuthRefreshLoop ticks at the refresh buffer cadence, renewing the
// token before it expires. Exits on ctx cancellation or stopAuthRefresh.
func (a *App) runAuthRefreshLoop(ctx context.Context) {
	ticker := time.NewTicker(authRefreshBuffer)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-a.stopAuthRefresh:
			return
		case <-ticker.C:
			a.ensureAuthFresh(ctx)
		}
	}
}

// stopAuthRefreshLoop signals the background refresh loop to exit. Safe to
// call when the loop never started (nil channel) and idempotent.
func (a *App) stopAuthRefreshLoop() {
	if a.stopAuthRefresh == nil {
		return
	}
	a.stopAuthRefreshOnce.Do(func() { close(a.stopAuthRefresh) })
}

// getAuthConfig builds an OIDCConfig from app settings with defaults.
func (a *App) getAuthConfig() auth.OIDCConfig {
	cfg := auth.DefaultConfig()

	settings, err := appconfig.LoadAppSettings()
	if err == nil && settings != nil {
		if settings.AuthClientID != "" {
			cfg.ClientID = settings.AuthClientID
		}
		if settings.AuthIssuer != "" {
			cfg.Issuer = settings.AuthIssuer
		}
	}

	return cfg
}

// provisionGateway provisions a lurus-api gateway token for the authenticated user.
// Called asynchronously after login succeeds. Reads the internal API key from env.
func (a *App) provisionGateway() {
	if a.authSession == nil {
		return
	}

	state := a.authSession.GetAuthState()
	if !state.IsLoggedIn || state.User == nil {
		return
	}

	// Determine the gateway API base URL.
	gwURL := "https://api.lurus.cn"
	if a.proxyMgr != nil {
		if ep := a.proxyMgr.GetSettings().APIEndpoint; ep != "" {
			gwURL = ep
		}
	}

	// Use the internal provisioning key from environment.
	internalKey := os.Getenv("LURUS_SWITCH_INTERNAL_KEY")
	if internalKey == "" {
		log.Printf("[auth] LURUS_SWITCH_INTERNAL_KEY not set, skipping gateway provisioning")
		return
	}

	ctx, cancel := context.WithTimeout(a.ctx, 30*time.Second)
	defer cancel()

	resp, err := auth.Provision(ctx, gwURL, internalKey, state.User.Sub, state.User.Email, state.User.Name)
	if err != nil {
		log.Printf("[auth] Gateway provisioning failed: %v", err)
		return
	}

	if resp.TokenKey != "" {
		if err := a.authSession.SetGatewayToken(resp.TokenKey, resp.UserID); err != nil {
			log.Printf("[auth] Failed to save gateway token: %v", err)
			return
		}
		// Reset billing client so it picks up the new gateway token.
		a.resetBillingClient()
		// If the gateway is already running when the user logs in, push the
		// freshly provisioned OIDC token onto the live data path now instead
		// of waiting for the next StartGateway/SaveProxySettings — syncGatewayUpstream
		// honors "OIDC gateway token > manual proxy token".
		if a.gatewaySrv != nil {
			a.syncGatewayUpstream()
		}
		log.Printf("[auth] Gateway provisioned successfully (user_id=%d, status=%s)", resp.UserID, resp.Status)
	}

	// SSO bridge: bind the OIDC subject to the org-chart Employee
	// record so the Enterprise admin can answer "who logged in just
	// now" and route requests by department. Best-effort — missing
	// orgsync, missing employee, or already-bound subject all silently
	// no-op (the audit log gets the principal regardless).
	a.bindSSOSubject(state.User)
}

// bindSSOSubject finds the org-chart Employee whose email matches the
// OIDC user info and patches their SSOSubject if not already set. The
// orgsync store enforces immutability after first bind, so this is
// safe to call on every login.
func (a *App) bindSSOSubject(u *auth.UserInfo) {
	if u == nil || u.Sub == "" || u.Email == "" {
		return
	}
	if a.services == nil {
		return
	}
	store, err := a.orgsyncStore()
	if err != nil || store == nil {
		// Personal-mode installs skip orgsync entirely; that's not an
		// error, so log only on real failures.
		if err != nil {
			log.Printf("[auth] orgsync unavailable, skipping SSO bind: %v", err)
		}
		return
	}
	emp := store.FindEmployeeByEmail(u.Email)
	if emp == nil {
		// Unknown employee — Enterprise admin hasn't enrolled them yet.
		// Don't auto-create here; the SCIM/manual flow owns enrollment.
		log.Printf("[auth] SSO bind: no Employee record for %q (admin must add to org chart)", u.Email)
		return
	}
	if emp.SSOSubject == u.Sub {
		return // already bound, no-op
	}
	patch := *emp
	patch.SSOSubject = u.Sub
	if _, err := store.UpdateEmployee(patch); err != nil {
		log.Printf("[auth] SSO bind: UpdateEmployee for %s failed: %v", u.Email, err)
		return
	}
	log.Printf("[auth] SSO bind: linked %s → employee %s", u.Email, emp.ID)
}
