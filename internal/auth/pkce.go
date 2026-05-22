package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/pkg/browser"
)

const (
	callbackPort = 31416
	callbackPath = "/auth/callback"
)

// productionClientID is the Zitadel OIDC app id for switch-desktop on
// auth.lurus.cn. Registered 2026-05-12 as a NATIVE app + PKCE + loopback
// (devMode=true so http://localhost is permitted). Self-hosters who run
// their own Zitadel can override via the AuthClientID app setting; the
// user-supplied value in app-settings.json wins over this default.
const productionClientID = "372642785217478778"

// OIDCConfig holds the OIDC provider configuration.
type OIDCConfig struct {
	Issuer      string
	ClientID    string
	RedirectURI string
	Scopes      []string
}

// DefaultConfig returns the default OIDC configuration for Lurus.
// ClientID defaults to the production switch-desktop app on auth.lurus.cn;
// users on a self-hosted Zitadel override it via Settings > Auth Client ID.
func DefaultConfig() OIDCConfig {
	return OIDCConfig{
		Issuer:      "https://auth.lurus.cn",
		ClientID:    productionClientID,
		RedirectURI: fmt.Sprintf("http://localhost:%d%s", callbackPort, callbackPath),
		Scopes:      []string{"openid", "profile", "email", "offline_access"},
	}
}

// TokenResponse holds the tokens returned by the OIDC provider.
//
// UserInfo is populated by exchangeCode after a successful token exchange:
// Zitadel's id_token only carries the sub claim by default, so name / email
// / picture come from a follow-up call to the standard OIDC /userinfo
// endpoint. The field is unexported on the wire (json:"-") because it isn't
// part of the OIDC token response — it's a derived enrichment we attach
// before handing the response back to the caller.
type TokenResponse struct {
	AccessToken  string    `json:"access_token"`
	TokenType    string    `json:"token_type"`
	ExpiresIn    int       `json:"expires_in"`
	RefreshToken string    `json:"refresh_token,omitempty"`
	IDToken      string    `json:"id_token,omitempty"`
	Scope        string    `json:"scope,omitempty"`
	UserInfo     *UserInfo `json:"-"`
}

func generateCodeVerifier() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate random bytes: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func generateCodeChallenge(verifier string) string {
	h := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(h[:])
}

func generateState() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate state: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

// LoginWithPKCE performs the full PKCE authorization code flow.
// Opens the system browser, listens for callback, exchanges code for tokens.
func LoginWithPKCE(ctx context.Context, cfg OIDCConfig) (*TokenResponse, error) {
	if cfg.ClientID == "" {
		return nil, fmt.Errorf("auth client_id not configured — please set it in Settings")
	}

	codeVerifier, err := generateCodeVerifier()
	if err != nil {
		return nil, err
	}
	codeChallenge := generateCodeChallenge(codeVerifier)

	state, err := generateState()
	if err != nil {
		return nil, err
	}

	authURL := fmt.Sprintf("%s/oauth/v2/authorize", strings.TrimRight(cfg.Issuer, "/"))
	params := url.Values{
		"client_id":             {cfg.ClientID},
		"redirect_uri":          {cfg.RedirectURI},
		"response_type":         {"code"},
		"scope":                 {strings.Join(cfg.Scopes, " ")},
		"code_challenge":        {codeChallenge},
		"code_challenge_method": {"S256"},
		"state":                 {state},
	}

	fullURL := authURL + "?" + params.Encode()

	codeCh := make(chan string, 1)
	errCh := make(chan error, 1)

	listener, err := net.Listen("tcp", fmt.Sprintf("localhost:%d", callbackPort))
	if err != nil {
		return nil, fmt.Errorf("start callback server on port %d: %w", callbackPort, err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc(callbackPath, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("state") != state {
			errCh <- fmt.Errorf("state mismatch: possible CSRF attack")
			http.Error(w, "State mismatch", http.StatusBadRequest)
			return
		}

		if errMsg := r.URL.Query().Get("error"); errMsg != "" {
			desc := r.URL.Query().Get("error_description")
			errCh <- fmt.Errorf("authorization error: %s — %s", errMsg, desc)
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			fmt.Fprintf(w, authFailHTML, errMsg, desc)
			return
		}

		code := r.URL.Query().Get("code")
		if code == "" {
			errCh <- fmt.Errorf("no authorization code in callback")
			http.Error(w, "Missing code", http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, authSuccessHTML)
		codeCh <- code
	})

	server := &http.Server{Handler: mux}

	go func() {
		if sErr := server.Serve(listener); sErr != nil && sErr != http.ErrServerClosed {
			errCh <- fmt.Errorf("callback server: %w", sErr)
		}
	}()

	defer func() {
		shutCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		server.Shutdown(shutCtx) //nolint:errcheck
	}()

	if err := browser.OpenURL(fullURL); err != nil {
		return nil, fmt.Errorf("open browser: %w", err)
	}

	select {
	case code := <-codeCh:
		return exchangeCode(ctx, cfg, code, codeVerifier)
	case err := <-errCh:
		return nil, err
	case <-ctx.Done():
		return nil, fmt.Errorf("login timeout or cancelled")
	}
}

func exchangeCode(ctx context.Context, cfg OIDCConfig, code, codeVerifier string) (*TokenResponse, error) {
	tokenURL := fmt.Sprintf("%s/oauth/v2/token", strings.TrimRight(cfg.Issuer, "/"))

	data := url.Values{
		"grant_type":    {"authorization_code"},
		"client_id":     {cfg.ClientID},
		"code":          {code},
		"redirect_uri":  {cfg.RedirectURI},
		"code_verifier": {codeVerifier},
	}

	req, err := http.NewRequestWithContext(ctx, "POST", tokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("build token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("token exchange: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read token response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token exchange failed (HTTP %d): %s", resp.StatusCode, string(body))
	}

	var tokenResp TokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return nil, fmt.Errorf("parse token response: %w", err)
	}

	// Fetch /userinfo so the session has a populated UserInfo regardless of
	// whether Zitadel embedded the profile claims in the id_token (it
	// doesn't by default — only sub is guaranteed). Best-effort: a failure
	// here is logged and tokens still flow through, the session will then
	// fall back to id_token claim decoding.
	if u, uErr := fetchUserInfo(ctx, cfg, tokenResp.AccessToken); uErr == nil {
		tokenResp.UserInfo = u
	}

	return &tokenResp, nil
}

// fetchUserInfo calls the OIDC /userinfo endpoint with the access_token and
// returns the user profile claims. Endpoint path is the OIDC spec one
// (configurable in cfg.Issuer); 4xx/5xx is treated as an error so callers
// can decide whether to fall back to id_token decoding.
func fetchUserInfo(ctx context.Context, cfg OIDCConfig, accessToken string) (*UserInfo, error) {
	if accessToken == "" {
		return nil, fmt.Errorf("userinfo: empty access_token")
	}
	url := strings.TrimRight(cfg.Issuer, "/") + "/oidc/v1/userinfo"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("userinfo: build request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("userinfo: request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("userinfo: read body: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("userinfo: HTTP %d: %s", resp.StatusCode, string(body))
	}

	var info UserInfo
	if err := json.Unmarshal(body, &info); err != nil {
		return nil, fmt.Errorf("userinfo: parse: %w", err)
	}
	return &info, nil
}

// RefreshAccessToken uses a refresh token to obtain a new access token.
func RefreshAccessToken(ctx context.Context, cfg OIDCConfig, refreshToken string) (*TokenResponse, error) {
	tokenURL := fmt.Sprintf("%s/oauth/v2/token", strings.TrimRight(cfg.Issuer, "/"))

	data := url.Values{
		"grant_type":    {"refresh_token"},
		"client_id":     {cfg.ClientID},
		"refresh_token": {refreshToken},
	}

	req, err := http.NewRequestWithContext(ctx, "POST", tokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("build refresh request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("refresh token: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read refresh response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token refresh failed (HTTP %d): %s", resp.StatusCode, string(body))
	}

	var tokenResp TokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return nil, fmt.Errorf("parse refresh response: %w", err)
	}

	return &tokenResp, nil
}

const authSuccessHTML = `<!DOCTYPE html><html><head><meta charset="utf-8"><title>Login</title>
<style>body{font-family:system-ui;display:flex;justify-content:center;align-items:center;height:100vh;margin:0;background:#f8fafc}
.card{text-align:center;padding:3rem;border-radius:12px;background:#fff;box-shadow:0 2px 8px rgba(0,0,0,.1)}
h1{color:#16a34a;font-size:1.5rem}p{color:#64748b}</style></head>
<body><div class="card"><h1>&#10003; Login Successful</h1><p>You may close this page and return to Lurus Switch.</p></div></body></html>`

const authFailHTML = `<!DOCTYPE html><html><head><meta charset="utf-8"><title>Login Failed</title>
<style>body{font-family:system-ui;display:flex;justify-content:center;align-items:center;height:100vh;margin:0;background:#f8fafc}
.card{text-align:center;padding:3rem;border-radius:12px;background:#fff;box-shadow:0 2px 8px rgba(0,0,0,.1)}
h1{color:#dc2626;font-size:1.5rem}p{color:#64748b}</style></head>
<body><div class="card"><h1>&#10007; Login Failed</h1><p>%s: %s</p><p>Please close this page and try again.</p></div></body></html>`
