package provider

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	customProvidersFile = "custom-providers.json"
	customFilePerm      = 0o600
)

// CustomProvider is a user-defined OpenAI-compatible endpoint. Unlike the
// built-in presets it is persisted to disk and editable at runtime, so an
// enterprise can point Switch at a private deployment without a code change.
//
// APIKey is stored base64-obfuscated on disk (NOT encryption — see Store).
// In memory and across the Wails boundary it is the plaintext key.
type CustomProvider struct {
	ID            string            `json:"id"`
	Name          string            `json:"name"`
	BaseURL       string            `json:"baseUrl"`
	APIKey        string            `json:"apiKey"`
	DefaultModels []string          `json:"defaultModels"`
	Headers       map[string]string `json:"headers,omitempty"`
	DocsURL       string            `json:"docsUrl,omitempty"`
	Description   string            `json:"description,omitempty"`
	CreatedAt     time.Time         `json:"createdAt"`
}

// CustomStore persists user-defined providers in a single JSON file. N is
// expected to be small (<20) so we keep the whole set in memory and rewrite
// the file on every mutation — no need for an index.
type CustomStore struct {
	mu    sync.RWMutex
	path  string
	cache map[string]CustomProvider
}

// NewCustomStore opens (or initializes) the custom-providers file under
// appDataDir. A missing file is not an error — it means "no custom
// providers yet".
func NewCustomStore(appDataDir string) (*CustomStore, error) {
	if appDataDir == "" {
		return nil, fmt.Errorf("appDataDir is required")
	}
	s := &CustomStore{
		path:  filepath.Join(appDataDir, customProvidersFile),
		cache: make(map[string]CustomProvider),
	}
	if err := s.load(); err != nil {
		return nil, err
	}
	return s, nil
}

// List returns all custom providers sorted by CreatedAt (oldest first).
func (s *CustomStore) List() []CustomProvider {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]CustomProvider, 0, len(s.cache))
	for _, p := range s.cache {
		out = append(out, p)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].CreatedAt.Before(out[j].CreatedAt)
	})
	return out
}

// Get returns a single provider by ID.
func (s *CustomStore) Get(id string) (CustomProvider, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	p, ok := s.cache[id]
	return p, ok
}

// Save creates or updates a provider. A blank ID is treated as "create"
// and gets a generated ID. BaseURL is required; Name defaults to the host.
// Returns the stored provider (with its assigned ID + CreatedAt).
func (s *CustomStore) Save(p CustomProvider) (CustomProvider, error) {
	p.Name = strings.TrimSpace(p.Name)
	p.BaseURL = strings.TrimSpace(p.BaseURL)
	if p.BaseURL == "" {
		return CustomProvider{}, fmt.Errorf("base URL is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if p.ID == "" {
		p.ID = newCustomID(p.Name, p.BaseURL)
		if _, exists := s.cache[p.ID]; exists {
			return CustomProvider{}, fmt.Errorf("a custom provider with id %q already exists", p.ID)
		}
		p.CreatedAt = time.Now()
	} else if existing, ok := s.cache[p.ID]; ok {
		// Update: preserve original CreatedAt.
		p.CreatedAt = existing.CreatedAt
	} else {
		// Caller supplied an ID for a provider that doesn't exist — accept
		// it as a create with that exact ID (import/restore path).
		p.CreatedAt = time.Now()
	}
	if p.Name == "" {
		p.Name = hostFromURL(p.BaseURL)
	}

	s.cache[p.ID] = p
	if err := s.persist(); err != nil {
		return CustomProvider{}, err
	}
	return p, nil
}

// Delete removes a provider by ID. Deleting a missing ID is a no-op (the
// desired end state — provider absent — already holds).
func (s *CustomStore) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.cache[id]; !ok {
		return nil
	}
	delete(s.cache, id)
	return s.persist()
}

// --- persistence ---------------------------------------------------------

// diskRecord mirrors CustomProvider but with the API key base64-obfuscated.
// This is intentionally NOT encryption — it stops a casual `cat` / backup
// sync from leaking the key in plaintext, paired with 0600 file perms. OS
// keyring integration is tracked for a later wave.
type diskRecord struct {
	ID            string            `json:"id"`
	Name          string            `json:"name"`
	BaseURL       string            `json:"baseUrl"`
	APIKeyB64     string            `json:"apiKeyB64"`
	DefaultModels []string          `json:"defaultModels"`
	Headers       map[string]string `json:"headers,omitempty"`
	DocsURL       string            `json:"docsUrl,omitempty"`
	Description   string            `json:"description,omitempty"`
	CreatedAt     time.Time         `json:"createdAt"`
}

func (s *CustomStore) load() error {
	data, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // fresh install
		}
		return fmt.Errorf("read custom providers: %w", err)
	}
	var records []diskRecord
	if err := json.Unmarshal(data, &records); err != nil {
		return fmt.Errorf("parse custom providers: %w", err)
	}
	for _, r := range records {
		key := ""
		if r.APIKeyB64 != "" {
			if dec, derr := base64.StdEncoding.DecodeString(r.APIKeyB64); derr == nil {
				key = string(dec)
			}
		}
		s.cache[r.ID] = CustomProvider{
			ID:            r.ID,
			Name:          r.Name,
			BaseURL:       r.BaseURL,
			APIKey:        key,
			DefaultModels: r.DefaultModels,
			Headers:       r.Headers,
			DocsURL:       r.DocsURL,
			Description:   r.Description,
			CreatedAt:     r.CreatedAt,
		}
	}
	return nil
}

// persist rewrites the whole file. Caller must hold s.mu.
func (s *CustomStore) persist() error {
	records := make([]diskRecord, 0, len(s.cache))
	for _, p := range s.cache {
		records = append(records, diskRecord{
			ID:            p.ID,
			Name:          p.Name,
			BaseURL:       p.BaseURL,
			APIKeyB64:     base64.StdEncoding.EncodeToString([]byte(p.APIKey)),
			DefaultModels: p.DefaultModels,
			Headers:       p.Headers,
			DocsURL:       p.DocsURL,
			Description:   p.Description,
			CreatedAt:     p.CreatedAt,
		})
	}
	sort.Slice(records, func(i, j int) bool {
		return records[i].CreatedAt.Before(records[j].CreatedAt)
	})
	data, err := json.MarshalIndent(records, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal custom providers: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return fmt.Errorf("create app data dir: %w", err)
	}
	if err := os.WriteFile(s.path, data, customFilePerm); err != nil {
		return fmt.Errorf("write custom providers: %w", err)
	}
	return nil
}

// --- id helpers ----------------------------------------------------------

// newCustomID derives a stable, readable ID from the name (or host),
// prefixed so it can never collide with a built-in preset ID.
func newCustomID(name, baseURL string) string {
	seed := name
	if seed == "" {
		seed = hostFromURL(baseURL)
	}
	slug := slugify(seed)
	if slug == "" {
		slug = "provider"
	}
	return fmt.Sprintf("custom-%s-%d", slug, time.Now().UnixNano()%1_000_000)
}

func slugify(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	var b strings.Builder
	prevDash := false
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
			prevDash = false
		default:
			if !prevDash && b.Len() > 0 {
				b.WriteByte('-')
				prevDash = true
			}
		}
	}
	return strings.Trim(b.String(), "-")
}

func hostFromURL(u string) string {
	s := u
	if i := strings.Index(s, "://"); i >= 0 {
		s = s[i+3:]
	}
	if i := strings.IndexAny(s, "/:"); i >= 0 {
		s = s[:i]
	}
	return s
}
