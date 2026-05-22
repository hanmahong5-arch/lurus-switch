package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

// DefaultPlatformBaseURL is the production platform-core URL that hosts
// /api/v1/account/me + /api/v1/wallet. Distinct from the OIDC issuer
// (auth.lurus.cn) which serves Zitadel — Switch was previously hitting
// the OIDC issuer for these endpoints and getting 404. Self-hosters
// override via the AuthPlatformURL app setting when running their own
// platform-core deployment.
const DefaultPlatformBaseURL = "https://identity.lurus.cn"

// PlatformAccount mirrors a subset of the platform-core GET
// /api/v1/account/me response. Wire-stable subset of
// pkg/lurusplatformclient's AccountMeResponse — keeping it inlined in
// Switch (rather than importing the SDK module) avoids pulling the
// entire lurus-platform module into the Wails build.
//
// The wire shape is documented in lurus-platform's
// pkg/lurusplatformclient/account_me.go and is additive-only on master.
type PlatformAccount struct {
	AccountID     int64   `json:"account_id"`
	LurusID       string  `json:"lurus_id"`
	DisplayName   string  `json:"display_name,omitempty"`
	Email         string  `json:"email,omitempty"`
	VIPLevel      int16   `json:"vip_level"`
	WalletBalance float64 `json:"wallet_balance"`
	WalletFrozen  float64 `json:"wallet_frozen"`
}

// FetchPlatformAccount calls GET /api/v1/account/me + /api/v1/wallet on
// the platform-core base URL (e.g. https://identity.lurus.cn — NOTE:
// distinct from the OIDC issuer auth.lurus.cn) using the supplied
// Zitadel access_token as Bearer auth. Both endpoints accept the same
// JWT (platform-core's JWT middleware validates Zitadel JWKS).
//
// We collapse the two calls into one PlatformAccount because Switch's
// UX needs both: account info for display, wallet for the balance.
// Returns nil + error on transport failure or /me non-2xx response.
// /wallet failure is silently absorbed (best-effort; the function still
// returns the account portion so the UI can render identity even when
// wallet read is broken).
//
// Empty platformBaseURL falls back to DefaultPlatformBaseURL so callers
// don't have to thread config through if the production default fits.
func FetchPlatformAccount(ctx context.Context, platformBaseURL, accessToken string) (*PlatformAccount, error) {
	if accessToken == "" {
		return nil, fmt.Errorf("platform: empty access_token")
	}
	if strings.TrimSpace(platformBaseURL) == "" {
		platformBaseURL = DefaultPlatformBaseURL
	}
	base := strings.TrimRight(platformBaseURL, "/")

	// /api/v1/account/me returns {account: {...}, vip: {...}, subscriptions: [...]}
	type meResp struct {
		Account *struct {
			ID          int64  `json:"id"`
			LurusID     string `json:"lurus_id"`
			DisplayName string `json:"display_name,omitempty"`
			Email       string `json:"email,omitempty"`
		} `json:"account"`
		VIP *struct {
			Level int16 `json:"level"`
		} `json:"vip,omitempty"`
	}
	var me meResp
	if err := platformGetJSON(ctx, base+"/api/v1/account/me", accessToken, &me); err != nil {
		return nil, fmt.Errorf("platform: /api/v1/account/me: %w", err)
	}
	if me.Account == nil {
		return nil, fmt.Errorf("platform: /api/v1/account/me returned empty account")
	}

	type walletResp struct {
		Balance float64 `json:"balance"`
		Frozen  float64 `json:"frozen"`
	}
	var w walletResp
	// Wallet failure is non-fatal — return PlatformAccount with zero
	// balance so the UI still renders identity even if /wallet is down.
	if werr := platformGetJSON(ctx, base+"/api/v1/wallet", accessToken, &w); werr != nil {
		log.Printf("[auth] platform /api/v1/wallet failed (non-fatal): %v", werr)
	}

	out := &PlatformAccount{
		AccountID:     me.Account.ID,
		LurusID:       me.Account.LurusID,
		DisplayName:   me.Account.DisplayName,
		Email:         me.Account.Email,
		WalletBalance: w.Balance,
		WalletFrozen:  w.Frozen,
	}
	if me.VIP != nil {
		out.VIPLevel = me.VIP.Level
	}
	return out, nil
}

// platformGetJSON is a focused stdlib GET that sends a Bearer header and
// decodes the JSON response. Caps body at 1 MiB; surfaces non-2xx
// responses as errors so callers don't silently get a zero-value out.
func platformGetJSON(ctx context.Context, url, bearer string, out any) error {
	reqCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+bearer)
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return fmt.Errorf("read body: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}
	if out == nil || len(body) == 0 {
		return nil
	}
	if err := json.Unmarshal(body, out); err != nil {
		return fmt.Errorf("parse JSON: %w", err)
	}
	return nil
}
