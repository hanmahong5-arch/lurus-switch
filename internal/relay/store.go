package relay

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

const (
	endpointsFile = "relay-endpoints.json"
	mappingFile   = "relay-mapping.json"
)

// Store persists relay endpoints and tool→relay mappings to disk.
type Store struct {
	mu         sync.RWMutex
	dataDir    string
	builtin    []RelayEndpoint
}

// NewStore creates a Store rooted at appDataDir.
func NewStore(appDataDir string) (*Store, error) {
	if err := os.MkdirAll(appDataDir, 0o755); err != nil {
		return nil, fmt.Errorf("relay store: create dir: %w", err)
	}
	s := &Store{
		dataDir: appDataDir,
		builtin: builtinEndpoints(),
	}
	return s, nil
}

// ListEndpoints returns all endpoints (builtin first, then user-defined).
// Builtin endpoints are never persisted; they are merged at read time.
func (s *Store) ListEndpoints() ([]RelayEndpoint, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	user, err := s.loadUserEndpoints()
	if err != nil {
		return nil, err
	}

	// Build a unified list: builtin first, then user custom
	all := make([]RelayEndpoint, 0, len(s.builtin)+len(user))
	all = append(all, s.builtin...)
	all = append(all, user...)
	return all, nil
}

// SaveEndpoint upserts a user-defined endpoint (builtin IDs are read-only).
func (s *Store) SaveEndpoint(ep RelayEndpoint) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if ep.ID == "" {
		buf := make([]byte, 8)
		if _, err := rand.Read(buf); err != nil {
			return fmt.Errorf("generate endpoint ID: %w", err)
		}
		ep.ID = fmt.Sprintf("relay-%x", buf)
	}

	eps, err := s.loadUserEndpoints()
	if err != nil {
		return err
	}

	found := false
	for i, e := range eps {
		if e.ID == ep.ID {
			eps[i] = ep
			found = true
			break
		}
	}
	if !found {
		eps = append(eps, ep)
	}

	return s.saveUserEndpoints(eps)
}

// DeleteEndpoint removes a user-defined endpoint by ID.
func (s *Store) DeleteEndpoint(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	eps, err := s.loadUserEndpoints()
	if err != nil {
		return err
	}

	filtered := eps[:0]
	for _, e := range eps {
		if e.ID != id {
			filtered = append(filtered, e)
		}
	}
	return s.saveUserEndpoints(filtered)
}

// GetToolMapping returns the current tool→relay-ID mapping.
func (s *Store) GetToolMapping() (ToolRelayMapping, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.loadMapping()
}

// SaveToolMapping persists the tool→relay-ID mapping.
func (s *Store) SaveToolMapping(m ToolRelayMapping) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.saveMapping(m)
}

// --- internal ---

func (s *Store) loadUserEndpoints() ([]RelayEndpoint, error) {
	path := filepath.Join(s.dataDir, endpointsFile)
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read relay endpoints: %w", err)
	}
	var eps []RelayEndpoint
	if err := json.Unmarshal(data, &eps); err != nil {
		return nil, fmt.Errorf("parse relay endpoints: %w", err)
	}
	return eps, nil
}

func (s *Store) saveUserEndpoints(eps []RelayEndpoint) error {
	data, err := json.MarshalIndent(eps, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal relay endpoints: %w", err)
	}
	path := filepath.Join(s.dataDir, endpointsFile)
	return os.WriteFile(path, data, 0o600)
}

func (s *Store) loadMapping() (ToolRelayMapping, error) {
	path := filepath.Join(s.dataDir, mappingFile)
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return make(ToolRelayMapping), nil
	}
	if err != nil {
		return nil, fmt.Errorf("read relay mapping: %w", err)
	}
	var m ToolRelayMapping
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("parse relay mapping: %w", err)
	}
	return m, nil
}

func (s *Store) saveMapping(m ToolRelayMapping) error {
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal relay mapping: %w", err)
	}
	path := filepath.Join(s.dataDir, mappingFile)
	return os.WriteFile(path, data, 0o600)
}

// builtinEndpoints returns the hard-coded Lurus official relay endpoints.
func builtinEndpoints() []RelayEndpoint {
	return []RelayEndpoint{
		{
			ID:          "lurus-newapi",
			Name:        "Lurus 官方中转站",
			Kind:        KindLurus,
			URL:         "https://newapi.lurus.cn",
			Description: "Lurus 自营 OpenAI 兼容中转，支持 30+ LLM 供应商",
		},
	}
}
