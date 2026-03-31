package main

import (
	"context"
	"fmt"
	"log"
	"os"
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

	// Auto-provision gateway token after successful login.
	go safeGo("provision-gateway", func() { a.provisionGateway() })

	return a.authSession.GetAuthState(), nil
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

// RefreshAuth refreshes the access token using the stored refresh token.
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
		log.Printf("[auth] Gateway provisioned successfully (user_id=%d, status=%s)", resp.UserID, resp.Status)
	}
}
