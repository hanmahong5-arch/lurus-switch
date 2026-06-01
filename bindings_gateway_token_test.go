package main

import (
	"os"
	"testing"

	"lurus-switch/internal/auth"
	"lurus-switch/internal/gateway"
	"lurus-switch/internal/proxy"
)

// newTokenTestApp builds an *App whose proxyMgr, gatewaySrv and (optionally)
// authSession point at a throwaway temp dir, so the gateway data-path token
// resolution can be exercised without touching the real user profile.
//
// Both proxy.NewProxyManager and auth.NewSession derive their storage path
// from the user config dir (APPDATA on Windows), so we redirect it to a temp
// dir for the duration of the test.
func newTokenTestApp(t *testing.T, withSession bool) *App {
	t.Helper()
	tmpDir := t.TempDir()

	for _, k := range []string{"APPDATA", "XDG_CONFIG_HOME"} {
		orig := os.Getenv(k)
		os.Setenv(k, tmpDir)
		t.Cleanup(func() { os.Setenv(k, orig) })
	}
	// macOS UserConfigDir uses $HOME/Library/Application Support; keep tests
	// hermetic there too.
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	t.Cleanup(func() { os.Setenv("HOME", origHome) })

	pm, err := proxy.NewProxyManager()
	if err != nil {
		t.Fatalf("NewProxyManager: %v", err)
	}

	svc := &services{
		proxyMgr:   pm,
		gatewaySrv: gateway.NewServer(tmpDir, nil, nil),
	}

	if withSession {
		sess, err := auth.NewSession()
		if err != nil {
			t.Fatalf("NewSession: %v", err)
		}
		svc.authSession = sess
	}

	return &App{services: svc}
}

// storeGatewayToken seeds an OIDC session that carries a provisioned gateway
// token, mirroring the post-login state (StoreTokens then SetGatewayToken).
func storeGatewayToken(t *testing.T, sess *auth.Session, token string) {
	t.Helper()
	if err := sess.StoreTokens(&auth.TokenResponse{
		AccessToken: "access-token",
		ExpiresIn:   3600,
	}); err != nil {
		t.Fatalf("StoreTokens: %v", err)
	}
	if err := sess.SetGatewayToken(token, 42); err != nil {
		t.Fatalf("SetGatewayToken: %v", err)
	}
}

// Test_syncGatewayUpstream_TokenPriority verifies the RUNNING gateway data
// path is fed the authoritative token: OIDC session gateway token wins, with a
// clean fallback to the manual proxy key (APIKey, then UserToken) when there is
// no OIDC session.
func Test_syncGatewayUpstream_TokenPriority(t *testing.T) {
	tests := []struct {
		name         string
		withSession  bool
		gatewayToken string // OIDC provisioned token (only when withSession)
		apiKey       string // proxy APIKey
		userToken    string // proxy UserToken
		wantToken    string
	}{
		{
			name:         "oidc gateway token wins over manual user token",
			withSession:  true,
			gatewayToken: "oidc-gw-token",
			userToken:    "manual-user-token",
			wantToken:    "oidc-gw-token",
		},
		{
			name:         "oidc gateway token wins over manual api key",
			withSession:  true,
			gatewayToken: "oidc-gw-token",
			apiKey:       "manual-api-key",
			userToken:    "manual-user-token",
			wantToken:    "oidc-gw-token",
		},
		{
			name:        "no oidc session falls back to manual api key",
			withSession: false,
			apiKey:      "manual-api-key",
			userToken:   "manual-user-token",
			wantToken:   "manual-api-key",
		},
		{
			name:        "no oidc session, no api key falls back to user token",
			withSession: false,
			userToken:   "manual-user-token",
			wantToken:   "manual-user-token",
		},
		{
			name:         "logged in but no gateway token provisioned falls back to manual",
			withSession:  true,
			gatewayToken: "", // session exists but no gateway token
			userToken:    "manual-user-token",
			wantToken:    "manual-user-token",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := newTokenTestApp(t, tt.withSession)

			if tt.withSession && tt.gatewayToken != "" {
				storeGatewayToken(t, app.authSession, tt.gatewayToken)
			} else if tt.withSession {
				// Session present but no gateway token: store tokens only so
				// HasGatewayToken() is false.
				if err := app.authSession.StoreTokens(&auth.TokenResponse{
					AccessToken: "access-token",
					ExpiresIn:   3600,
				}); err != nil {
					t.Fatalf("StoreTokens: %v", err)
				}
			}

			if err := app.proxyMgr.SaveSettings(&proxy.ProxySettings{
				APIEndpoint: "https://api.lurus.cn",
				APIKey:      tt.apiKey,
				UserToken:   tt.userToken,
			}); err != nil {
				t.Fatalf("SaveSettings: %v", err)
			}

			app.syncGatewayUpstream()

			got := app.gatewaySrv.GetConfig()
			if got.UserToken != tt.wantToken {
				t.Errorf("gateway UserToken = %q, want %q", got.UserToken, tt.wantToken)
			}
			if got.UpstreamURL != "https://api.lurus.cn" {
				t.Errorf("gateway UpstreamURL = %q, want %q", got.UpstreamURL, "https://api.lurus.cn")
			}
		})
	}
}

// Test_SaveProxySettings_ResyncsRunningGateway verifies that editing proxy
// settings while the gateway is wired re-pushes the new upstream URL + token to
// the gateway data path (no restart required). Before the fix SaveProxySettings
// only rebuilt the billing client and left the gateway on the stale config.
func Test_SaveProxySettings_ResyncsRunningGateway(t *testing.T) {
	app := newTokenTestApp(t, false)

	// Seed an initial endpoint/token on the gateway.
	if err := app.proxyMgr.SaveSettings(&proxy.ProxySettings{
		APIEndpoint: "https://old.example.com",
		UserToken:   "old-token",
	}); err != nil {
		t.Fatalf("SaveSettings(initial): %v", err)
	}
	app.syncGatewayUpstream()
	if cfg := app.gatewaySrv.GetConfig(); cfg.UpstreamURL != "https://old.example.com" || cfg.UserToken != "old-token" {
		t.Fatalf("precondition: gateway not seeded, got %+v", cfg)
	}

	// Edit endpoint + token mid-session.
	if err := app.SaveProxySettings(&proxy.ProxySettings{
		APIEndpoint: "https://new.example.com",
		UserToken:   "new-token",
	}); err != nil {
		t.Fatalf("SaveProxySettings: %v", err)
	}

	got := app.gatewaySrv.GetConfig()
	if got.UpstreamURL != "https://new.example.com" {
		t.Errorf("after edit, gateway UpstreamURL = %q, want %q", got.UpstreamURL, "https://new.example.com")
	}
	if got.UserToken != "new-token" {
		t.Errorf("after edit, gateway UserToken = %q, want %q", got.UserToken, "new-token")
	}
}

// Test_SaveProxySettings_NoGatewayNoPanic guards the gatewaySrv == nil branch:
// SaveProxySettings must still succeed (e.g. before the gateway is built).
func Test_SaveProxySettings_NoGatewayNoPanic(t *testing.T) {
	tmpDir := t.TempDir()
	for _, k := range []string{"APPDATA", "XDG_CONFIG_HOME"} {
		orig := os.Getenv(k)
		os.Setenv(k, tmpDir)
		t.Cleanup(func() { os.Setenv(k, orig) })
	}
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	t.Cleanup(func() { os.Setenv("HOME", origHome) })

	pm, err := proxy.NewProxyManager()
	if err != nil {
		t.Fatalf("NewProxyManager: %v", err)
	}
	app := &App{services: &services{proxyMgr: pm}} // gatewaySrv intentionally nil

	if err := app.SaveProxySettings(&proxy.ProxySettings{
		APIEndpoint: "https://api.lurus.cn",
		UserToken:   "tok",
	}); err != nil {
		t.Fatalf("SaveProxySettings with nil gateway should succeed, got: %v", err)
	}
}
