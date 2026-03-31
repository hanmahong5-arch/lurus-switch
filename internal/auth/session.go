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
type AuthState struct {
	IsLoggedIn      bool      `json:"is_logged_in"`
	User            *UserInfo `json:"user,omitempty"`
	ExpiresAt       string    `json:"expires_at,omitempty"`
	HasGatewayToken bool      `json:"has_gateway_token"`
}

// Session manages token lifecycle and persistence.
type Session struct {
	mu            sync.RWMutex
	tokens        *storedTokens
	user          *UserInfo
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

	// Preserve existing gateway credentials across token refreshes.
	var gwToken string
	var gwUserID int
	if s.tokens != nil {
		gwToken = s.tokens.GatewayToken
		gwUserID = s.tokens.GatewayUserID
	}

	s.tokens = &storedTokens{
		AccessToken:   resp.AccessToken,
		RefreshToken:  resp.RefreshToken,
		IDToken:       resp.IDToken,
		ExpiresAt:     time.Now().Add(time.Duration(resp.ExpiresIn) * time.Second),
		GatewayToken:  gwToken,
		GatewayUserID: gwUserID,
	}

	if resp.IDToken != "" {
		if u, err := decodeIDToken(resp.IDToken); err == nil {
			s.user = u
		}
	}

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
		ExpiresAt:       s.tokens.ExpiresAt.Format(time.RFC3339),
		HasGatewayToken: s.tokens.GatewayToken != "",
	}
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
	if tokens.IDToken != "" {
		if u, err := decodeIDToken(tokens.IDToken); err == nil {
			s.user = u
		}
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
