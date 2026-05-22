package auth

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// UserInfo holds decoded user information from the ID token.
type UserInfo struct {
	Sub     string `json:"sub"`
	Name    string `json:"name"`
	Email   string `json:"email"`
	Picture string `json:"picture"`
}

// AuthState represents the current authentication state for the frontend.
//
// Platform carries Lurus-platform-specific identity + billing data
// (LurusID, wallet balance, VIP tier). It's nil when the platform
// /api/v1/account/me call hasn't succeeded yet (network failure, /me
// disabled in self-hosted Zitadel deployments). The OIDC User field
// is always populated when IsLoggedIn — Platform is best-effort.
type AuthState struct {
	IsLoggedIn      bool             `json:"is_logged_in"`
	User            *UserInfo        `json:"user,omitempty"`
	Platform        *PlatformAccount `json:"platform,omitempty"`
	ExpiresAt       string           `json:"expires_at,omitempty"`
	HasGatewayToken bool             `json:"has_gateway_token"`
}

// Session manages token lifecycle and persistence.
type Session struct {
	mu            sync.RWMutex
	tokens        *storedTokens
	user          *UserInfo
	platform      *PlatformAccount
	filePath      string
	encryptionKey []byte
}

type storedTokens struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	IDToken      string    `json:"id_token"`
	ExpiresAt    time.Time `json:"expires_at"`

	// Lurus gateway provisioned credentials.
	GatewayToken  string `json:"gateway_token,omitempty"`
	GatewayUserID int    `json:"gateway_user_id,omitempty"`

	// CachedUser is the /userinfo-enriched profile captured at login or
	// last refresh. Stored so a reload-from-disk doesn't have to make a
	// fresh /userinfo round-trip; id_token decode is the fallback when
	// this is nil (legacy auth.enc files written before Wave Switch-UX).
	CachedUser *UserInfo `json:"cached_user,omitempty"`

	// CachedPlatform is the /api/v1/account/me + /api/v1/wallet snapshot
	// captured at login. Refreshed on demand (FetchPlatformAccount can
	// be called any time the UI wants up-to-date balance). Nil-safe.
	CachedPlatform *PlatformAccount `json:"cached_platform,omitempty"`
}

// NewSession creates a session manager with encrypted file storage.
func NewSession() (*Session, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return nil, fmt.Errorf("get config dir: %w", err)
	}
	authDir := filepath.Join(configDir, "lurus-switch")
	if err := os.MkdirAll(authDir, 0o700); err != nil {
		return nil, fmt.Errorf("create auth dir: %w", err)
	}

	key, err := deriveEncryptionKey()
	if err != nil {
		return nil, fmt.Errorf("derive encryption key: %w", err)
	}

	s := &Session{
		filePath:      filepath.Join(authDir, "auth.enc"),
		encryptionKey: key,
	}

	_ = s.load()
	return s, nil
}

// StoreTokens persists the token response and extracts user info.
func (s *Session) StoreTokens(resp *TokenResponse) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Preserve gateway credentials + cached profile across token refreshes.
	// Refresh flows don't re-call /userinfo so we'd lose the avatar/name
	// the user sees in the panel if we dropped the cache here.
	var gwToken string
	var gwUserID int
	var cachedUser *UserInfo
	if s.tokens != nil {
		gwToken = s.tokens.GatewayToken
		gwUserID = s.tokens.GatewayUserID
		cachedUser = s.tokens.CachedUser
	}

	s.tokens = &storedTokens{
		AccessToken:   resp.AccessToken,
		RefreshToken:  resp.RefreshToken,
		IDToken:       resp.IDToken,
		ExpiresAt:     time.Now().Add(time.Duration(resp.ExpiresIn) * time.Second),
		GatewayToken:  gwToken,
		GatewayUserID: gwUserID,
		CachedUser:    cachedUser,
	}

	// Prefer /userinfo (always-populated profile) over id_token claim
	// decode (Zitadel default omits name/email/picture from id_token).
	// id_token decode remains as a fallback when /userinfo failed.
	switch {
	case resp.UserInfo != nil:
		s.user = resp.UserInfo
		s.tokens.CachedUser = resp.UserInfo
	case resp.IDToken != "":
		if u, err := decodeIDToken(resp.IDToken); err == nil {
			s.user = u
			s.tokens.CachedUser = u
		}
	}

	// Preserve any prior platform snapshot — caller refreshes via
	// FetchPlatformAccount after StoreTokens succeeds (the platform
	// call needs a populated access_token from THIS very response, so
	// it has to be invoked from outside under the new token's scope).
	s.tokens.CachedPlatform = s.platform

	return s.save()
}

// GetAuthState returns the current authentication state.
func (s *Session) GetAuthState() AuthState {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.tokens == nil || s.tokens.AccessToken == "" {
		return AuthState{IsLoggedIn: false}
	}

	return AuthState{
		IsLoggedIn:      true,
		User:            s.user,
		Platform:        s.platform,
		ExpiresAt:       s.tokens.ExpiresAt.Format(time.RFC3339),
		HasGatewayToken: s.tokens.GatewayToken != "",
	}
}

// SetPlatformAccount records the platform-core /api/v1/account/me
// snapshot for surface display. Persists alongside tokens so the UI
// stays populated across restarts. Nil-safe — calling with nil clears
// the cached state (used after Logout or when /me returns 401).
func (s *Session) SetPlatformAccount(p *PlatformAccount) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.platform = p
	if s.tokens != nil {
		s.tokens.CachedPlatform = p
		return s.save()
	}
	return nil
}

// GetAccessToken returns the current access token.
func (s *Session) GetAccessToken() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.tokens == nil {
		return ""
	}
	return s.tokens.AccessToken
}

// GetRefreshToken returns the current refresh token.
func (s *Session) GetRefreshToken() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.tokens == nil {
		return ""
	}
	return s.tokens.RefreshToken
}

// IsExpired returns true if the access token has expired or will within the buffer.
func (s *Session) IsExpired(buffer time.Duration) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.tokens == nil {
		return true
	}
	return time.Now().Add(buffer).After(s.tokens.ExpiresAt)
}

// GetGatewayToken returns the provisioned gateway API token.
func (s *Session) GetGatewayToken() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.tokens == nil {
		return ""
	}
	return s.tokens.GatewayToken
}

// GetGatewayUserID returns the provisioned gateway user ID.
func (s *Session) GetGatewayUserID() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.tokens == nil {
		return 0
	}
	return s.tokens.GatewayUserID
}

// HasGatewayToken returns true if a gateway token has been provisioned.
func (s *Session) HasGatewayToken() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.tokens != nil && s.tokens.GatewayToken != ""
}

// SetGatewayToken stores the gateway token and user ID, then persists to disk.
func (s *Session) SetGatewayToken(token string, userID int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.tokens == nil {
		return fmt.Errorf("no active session")
	}
	s.tokens.GatewayToken = token
	s.tokens.GatewayUserID = userID
	return s.save()
}

// Clear removes all stored tokens and user info.
func (s *Session) Clear() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.tokens = nil
	s.user = nil
	s.platform = nil
	_ = os.Remove(s.filePath)
	return nil
}

func (s *Session) save() error {
	data, err := json.Marshal(s.tokens)
	if err != nil {
		return fmt.Errorf("marshal tokens: %w", err)
	}

	encrypted, err := encryptAESGCM(s.encryptionKey, data)
	if err != nil {
		return fmt.Errorf("encrypt tokens: %w", err)
	}

	return os.WriteFile(s.filePath, encrypted, 0o600)
}

func (s *Session) load() error {
	data, err := os.ReadFile(s.filePath)
	if err != nil {
		return err
	}

	decrypted, err := decryptAESGCM(s.encryptionKey, data)
	if err != nil {
		_ = os.Remove(s.filePath)
		return fmt.Errorf("decrypt tokens: %w", err)
	}

	var tokens storedTokens
	if err := json.Unmarshal(decrypted, &tokens); err != nil {
		return fmt.Errorf("unmarshal tokens: %w", err)
	}

	s.tokens = &tokens
	switch {
	case tokens.CachedUser != nil:
		s.user = tokens.CachedUser
	case tokens.IDToken != "":
		if u, err := decodeIDToken(tokens.IDToken); err == nil {
			s.user = u
		}
	}
	if tokens.CachedPlatform != nil {
		s.platform = tokens.CachedPlatform
	}

	return nil
}

// decodeIDToken extracts user info from a JWT ID token without signature validation.
// We trust the token since it came directly from our configured issuer over HTTPS.
func decodeIDToken(idToken string) (*UserInfo, error) {
	parts := strings.Split(idToken, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid JWT format")
	}

	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, fmt.Errorf("decode JWT payload: %w", err)
	}

	var claims struct {
		Sub     string `json:"sub"`
		Name    string `json:"name"`
		Email   string `json:"email"`
		Picture string `json:"picture"`
	}
	if err := json.Unmarshal(payload, &claims); err != nil {
		return nil, fmt.Errorf("unmarshal JWT claims: %w", err)
	}

	return &UserInfo{
		Sub:     claims.Sub,
		Name:    claims.Name,
		Email:   claims.Email,
		Picture: claims.Picture,
	}, nil
}

func deriveEncryptionKey() ([]byte, error) {
	hostname, _ := os.Hostname()
	u, _ := user.Current()
	username := ""
	if u != nil {
		username = u.Username
	}
	material := fmt.Sprintf("lurus-switch:auth:%s:%s", hostname, username)
	key := sha256.Sum256([]byte(material))
	return key[:], nil
}

func encryptAESGCM(key, plaintext []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, aead.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}
	return aead.Seal(nonce, nonce, plaintext, nil), nil
}

func decryptAESGCM(key, ciphertext []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonceSize := aead.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short")
	}
	nonce, encrypted := ciphertext[:nonceSize], ciphertext[nonceSize:]
	return aead.Open(nil, nonce, encrypted, nil)
}
