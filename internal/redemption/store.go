package redemption

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"
)

// Activation is the persisted result of a successful redemption.
// All fields are sourced from the Hub /api/user/topup response plus
// fingerprint capture at the moment of redemption. Nothing in this struct
// should depend on Hub admin credentials — EndUser activation must be
// possible without an operator token on disk.
type Activation struct {
	// HubURL is the locked Hub coordinate (https://hub.acme.example).
	// Echoed back to the EndUser dashboard read-only; never user-editable.
	HubURL string `json:"hub_url"`

	// TenantSlug, when set, scopes V2 endpoints (`/api/v2/<slug>/...`).
	// Empty for legacy single-tenant Hubs that still serve V1 only.
	TenantSlug string `json:"tenant_slug,omitempty"`

	// UserToken is the bearer credential issued by Hub for subsequent
	// requests (gateway calls, /user/me, /user/heartbeat). Sent verbatim
	// in Authorization headers.
	UserToken string `json:"user_token"`

	// UserID is informational — surfaced to the EndUser dashboard so support
	// can identify the account when troubleshooting.
	UserID int `json:"user_id,omitempty"`

	// Quota and ExpiresAt mirror the redemption code's grant. Used for
	// dashboard display + early "expiring soon" warnings.
	Quota     int64     `json:"quota,omitempty"`
	ExpiresAt time.Time `json:"expires_at,omitempty"`

	// Fingerprint is the device fingerprint captured at activation time.
	// On load, the running fingerprint is compared — mismatch means the
	// activation was copied to a new machine and must be invalidated.
	Fingerprint string `json:"fingerprint"`

	// ActivatedAt is the wall-clock time of successful redemption.
	ActivatedAt time.Time `json:"activated_at"`

	// LastHeartbeat is updated by the heartbeat client on every successful
	// liveness ping. Stale > GracePeriod → degraded mode.
	LastHeartbeat time.Time `json:"last_heartbeat,omitempty"`

	// HeartbeatStatus mirrors the latest server-reported status. Empty
	// before the first heartbeat. Values: "active", "expired", "revoked".
	HeartbeatStatus string `json:"heartbeat_status,omitempty"`
}

// Status surfaces a coarse activation state to the UI without leaking
// secrets. Computed by the Store, not stored on disk — it changes when
// the wall clock advances.
type Status struct {
	Activated     bool      `json:"activated"`
	HubURL        string    `json:"hub_url,omitempty"`
	TenantSlug    string    `json:"tenant_slug,omitempty"`
	UserID        int       `json:"user_id,omitempty"`
	Quota         int64     `json:"quota,omitempty"`
	ExpiresAt     time.Time `json:"expires_at,omitempty"`
	ActivatedAt   time.Time `json:"activated_at,omitempty"`
	LastHeartbeat time.Time `json:"last_heartbeat,omitempty"`
	// State is the high-level lifecycle. UI uses this to decide whether to
	// render activation page / main page / "service expired" notice.
	State State `json:"state"`
	// StateReason is a human-friendly explanation when State is anything
	// other than StateActive. Bilingual at the binding boundary, not here.
	StateReason string `json:"state_reason,omitempty"`
}

// State enumerates the activation lifecycle.
type State string

const (
	// StateUnactivated — no activation file on disk.
	StateUnactivated State = "unactivated"
	// StateActive — token present, fingerprint matches, last heartbeat fresh.
	StateActive State = "active"
	// StateStale — token present but heartbeat is older than the grace
	// period. Hub may have lost contact; UI shows a degraded banner but
	// keeps working until heartbeat goes red or grace period elapses.
	StateStale State = "stale"
	// StateRevoked — Hub explicitly returned "revoked" or "expired" on the
	// last heartbeat. Token must not be reused.
	StateRevoked State = "revoked"
	// StateMismatch — file decrypts only with the original device's
	// fingerprint key; the running machine produced a different key, so we
	// could not even decrypt. Equivalent to "needs re-activation".
	StateMismatch State = "device_mismatch"
)

// HeartbeatGrace is how long an activation stays in StateActive after the
// last successful heartbeat. Past this, UI moves to StateStale (still
// usable, but warned). Sized at ~3x the heartbeat interval so a single
// transient network blip doesn't degrade the user's experience.
const HeartbeatGrace = 30 * time.Minute

// Store is the singleton persisting an Activation to encrypted disk.
// All methods are safe for concurrent use.
type Store struct {
	mu       sync.RWMutex
	filePath string
	cached   *Activation // nil when not loaded yet OR no activation on disk
	loaded   bool        // true once load() has been attempted at least once
}

// activationFilename is the on-disk basename. .enc extension is a hint to
// the user that hand-editing won't work; the actual format is AES-GCM.
const activationFilename = "activation.enc"

// NewStore returns a Store rooted at the OS-appropriate app data dir.
// Creation of the parent dir is deferred to the first Save — Load doesn't
// need it, and on a fresh install we don't want to litter the FS for an
// EndUser who hasn't activated yet.
func NewStore() (*Store, error) {
	dir, err := appDataDir()
	if err != nil {
		return nil, err
	}
	return &Store{filePath: filepath.Join(dir, activationFilename)}, nil
}

// NewStoreAt is the test seam — pin the storage dir to a temp directory.
func NewStoreAt(dir string) *Store {
	return &Store{filePath: filepath.Join(dir, activationFilename)}
}

// appDataDir mirrors auth.NewSession's location so all per-user
// secrets live next to each other under the lurus-switch namespace.
// Order: %APPDATA%\lurus-switch on Windows, ~/Library/Application
// Support/lurus-switch on macOS, ~/.lurus-switch elsewhere.
func appDataDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("user home dir: %w", err)
	}
	switch runtime.GOOS {
	case "windows":
		appData := os.Getenv("APPDATA")
		if appData == "" {
			appData = filepath.Join(home, "AppData", "Roaming")
		}
		return filepath.Join(appData, "lurus-switch"), nil
	case "darwin":
		return filepath.Join(home, "Library", "Application Support", "lurus-switch"), nil
	default:
		return filepath.Join(home, ".lurus-switch"), nil
	}
}

// Load reads the activation file (if any). Returns nil, nil when no file
// exists — that's the unactivated state, not an error. Returns a non-nil
// error only when the file exists but is unreadable or corrupt.
func (s *Store) Load() (*Activation, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.loaded && s.cached != nil {
		return s.cloneLocked(), nil
	}
	s.loaded = true

	raw, err := os.ReadFile(s.filePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			s.cached = nil
			return nil, nil
		}
		return nil, fmt.Errorf("read activation: %w", err)
	}

	plain, err := decrypt(fingerprintKey(), raw)
	if err != nil {
		// Decryption failure usually means the device fingerprint changed
		// (file copied to a new machine) — caller will surface this as
		// StateMismatch. Don't delete the file; the user might restore
		// the original machine and recover.
		return nil, fmt.Errorf("decrypt activation: %w", err)
	}

	var act Activation
	if err := json.Unmarshal(plain, &act); err != nil {
		return nil, fmt.Errorf("unmarshal activation: %w", err)
	}
	s.cached = &act
	return s.cloneLocked(), nil
}

// Save writes the activation atomically: write tmp, fsync, rename.
// Atomic semantics matter because a half-written activation file would
// turn an EndUser machine into a brick on the next launch.
func (s *Store) Save(a *Activation) error {
	if a == nil {
		return errors.New("nil activation")
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := os.MkdirAll(filepath.Dir(s.filePath), 0o700); err != nil {
		return fmt.Errorf("create activation dir: %w", err)
	}

	plain, err := json.Marshal(a)
	if err != nil {
		return fmt.Errorf("marshal activation: %w", err)
	}
	enc, err := encrypt(fingerprintKey(), plain)
	if err != nil {
		return fmt.Errorf("encrypt activation: %w", err)
	}

	tmp := s.filePath + ".tmp"
	if err := os.WriteFile(tmp, enc, 0o600); err != nil {
		return fmt.Errorf("write activation tmp: %w", err)
	}
	if err := os.Rename(tmp, s.filePath); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("rename activation: %w", err)
	}

	cp := *a
	s.cached = &cp
	s.loaded = true
	return nil
}

// Clear deletes the activation file and forgets the cached state.
// Idempotent — clearing an already-cleared store is not an error.
func (s *Store) Clear() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cached = nil
	s.loaded = true
	if err := os.Remove(s.filePath); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("remove activation: %w", err)
	}
	return nil
}

// UpdateHeartbeat patches the on-disk activation with fresh heartbeat
// metadata. Loads first if not yet cached; returns an error if no
// activation is present (callers should not heartbeat before Save).
func (s *Store) UpdateHeartbeat(at time.Time, status string) error {
	return s.UpdateHeartbeatQuota(at, status, 0, time.Time{})
}

// UpdateHeartbeatQuota is the quota-aware sibling of UpdateHeartbeat. In
// addition to the liveness timestamp + status, it refreshes the displayed
// grant from the Hub's heartbeat reply so the EndUser dashboard never shows
// a stale activation-time number after the reseller tops up or the balance
// burns down.
//
// Conservative merge rules keep the file backward-compatible and avoid
// clobbering a valid grant with an absent field:
//
//   - quota:     persisted only when > 0. The heartbeat envelope uses a
//     non-pointer int with `omitempty`, so an absent quota arrives as 0 and
//     is indistinguishable from a genuine zero balance — we prefer keeping
//     the last known positive grant over zeroing it on a thin reply.
//   - expiresAt: persisted only when strictly newer than the stored value.
//     A renewal extends the window; a heartbeat that omits expires_at (zero)
//     never shortens it.
//
// Passing quota<=0 and a zero expiresAt makes this behave exactly like
// UpdateHeartbeat — the timestamp/status-only path callers relied on before.
func (s *Store) UpdateHeartbeatQuota(at time.Time, status string, quota int64, expiresAt time.Time) error {
	act, err := s.Load()
	if err != nil {
		return err
	}
	if act == nil {
		return errors.New("no activation to update")
	}
	act.LastHeartbeat = at
	act.HeartbeatStatus = status
	if quota > 0 {
		act.Quota = quota
	}
	if !expiresAt.IsZero() && expiresAt.After(act.ExpiresAt) {
		act.ExpiresAt = expiresAt
	}
	return s.Save(act)
}

// Status computes the current lifecycle state from the on-disk activation
// (loading if needed) plus the wall clock.
//
// Decision tree:
//
//   - no file               → StateUnactivated
//   - file but decrypt fails → StateMismatch
//   - server says "revoked"  → StateRevoked
//   - server says "expired"  → StateRevoked
//   - last heartbeat > grace → StateStale
//   - otherwise              → StateActive
func (s *Store) Status(now time.Time) Status {
	act, err := s.Load()
	if err != nil {
		return Status{State: StateMismatch, StateReason: err.Error()}
	}
	if act == nil {
		return Status{State: StateUnactivated}
	}
	st := Status{
		Activated:     true,
		HubURL:        act.HubURL,
		TenantSlug:    act.TenantSlug,
		UserID:        act.UserID,
		Quota:         act.Quota,
		ExpiresAt:     act.ExpiresAt,
		ActivatedAt:   act.ActivatedAt,
		LastHeartbeat: act.LastHeartbeat,
		State:         StateActive,
	}
	switch act.HeartbeatStatus {
	case "revoked":
		st.State = StateRevoked
		st.StateReason = "Hub 已撤销该激活码"
		return st
	case "expired":
		st.State = StateRevoked
		st.StateReason = "激活码已过期"
		return st
	}
	if !act.ExpiresAt.IsZero() && now.After(act.ExpiresAt) {
		st.State = StateRevoked
		st.StateReason = "激活码已过期"
		return st
	}
	if !act.LastHeartbeat.IsZero() && now.Sub(act.LastHeartbeat) > HeartbeatGrace {
		st.State = StateStale
		st.StateReason = fmt.Sprintf("Hub 连接中断超过 %s", HeartbeatGrace)
	}
	return st
}

func (s *Store) cloneLocked() *Activation {
	if s.cached == nil {
		return nil
	}
	cp := *s.cached
	return &cp
}

// encrypt AES-256-GCM. The nonce is prepended to the ciphertext.
func encrypt(key, plaintext []byte) ([]byte, error) {
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

func decrypt(key, ciphertext []byte) ([]byte, error) {
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
		return nil, errors.New("ciphertext too short")
	}
	nonce, enc := ciphertext[:nonceSize], ciphertext[nonceSize:]
	return aead.Open(nil, nonce, enc, nil)
}
