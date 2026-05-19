package configapply

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"
)

// Store keeps plans and apply-transaction records on disk so a crashed Switch
// can resume rollback. Plans live in memory + JSON; transactions persist
// pre-content for files touched in case of mid-write crash.
type Store struct {
	dir   string
	mu    sync.RWMutex
	plans map[string]*ChangePlan
}

func NewStore() (*Store, error) {
	dir, err := storeDir()
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("create store dir: %w", err)
	}
	return &Store{dir: dir, plans: map[string]*ChangePlan{}}, nil
}

func NewStoreAt(dir string) (*Store, error) {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("create store dir: %w", err)
	}
	return &Store{dir: dir, plans: map[string]*ChangePlan{}}, nil
}

func storeDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	switch runtime.GOOS {
	case "windows":
		appData := os.Getenv("APPDATA")
		if appData == "" {
			appData = filepath.Join(home, "AppData", "Roaming")
		}
		return filepath.Join(appData, "lurus-switch", "configapply"), nil
	case "darwin":
		return filepath.Join(home, "Library", "Application Support", "lurus-switch", "configapply"), nil
	default:
		return filepath.Join(home, ".lurus-switch", "configapply"), nil
	}
}

func (s *Store) PutPlan(plan *ChangePlan) error {
	if plan == nil || plan.ID == "" {
		return fmt.Errorf("invalid plan")
	}
	s.mu.Lock()
	s.plans[plan.ID] = plan
	s.mu.Unlock()
	return nil
}

func (s *Store) GetPlan(id string) (*ChangePlan, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	p, ok := s.plans[id]
	return p, ok
}

func (s *Store) DeletePlan(id string) {
	s.mu.Lock()
	delete(s.plans, id)
	s.mu.Unlock()
}

// Transaction records the pre-state of a multi-file write. Persisted before
// PhaseWrite begins so a crash mid-write can still roll back on next startup.
type Transaction struct {
	ID           string             `json:"id"`
	PlanID       string             `json:"planID"`
	StartedAt    string             `json:"startedAt"`
	CompletedAt  string             `json:"completedAt,omitempty"`
	State        string             `json:"state"`
	PreContents  map[string]string  `json:"preContents"`
	PreExisted   map[string]bool    `json:"preExisted"`
}

func (s *Store) BeginTransaction(plan *ChangePlan) (*Transaction, error) {
	tx := &Transaction{
		ID:          fmt.Sprintf("tx-%s", time.Now().UTC().Format("20060102-150405.000")),
		PlanID:      plan.ID,
		StartedAt:   time.Now().UTC().Format(time.RFC3339),
		State:       "started",
		PreContents: map[string]string{},
		PreExisted:  map[string]bool{},
	}
	for _, ch := range plan.Changes {
		before, err := ReadFileOrEmpty(ch.Path)
		if err != nil {
			return nil, fmt.Errorf("read before %s: %w", ch.Path, err)
		}
		tx.PreContents[ch.Path] = before
		_, statErr := os.Stat(ch.Path)
		tx.PreExisted[ch.Path] = statErr == nil
	}
	if err := s.writeTxFile(tx); err != nil {
		return nil, fmt.Errorf("persist tx: %w", err)
	}
	return tx, nil
}

func (s *Store) CompleteTransaction(tx *Transaction, success bool) error {
	if tx == nil {
		return fmt.Errorf("nil tx")
	}
	tx.CompletedAt = time.Now().UTC().Format(time.RFC3339)
	if success {
		tx.State = "applied"
	} else {
		tx.State = "rolledback"
	}
	return s.writeTxFile(tx)
}

func (s *Store) writeTxFile(tx *Transaction) error {
	data, err := json.MarshalIndent(tx, "", "  ")
	if err != nil {
		return err
	}
	path := filepath.Join(s.dir, tx.ID+".json")
	return WriteAtomic(path, data, 0644)
}

// ListPendingTransactions returns transactions whose state is still "started".
// Switch calls this at startup to recover from crashes mid-write.
func (s *Store) ListPendingTransactions() ([]*Transaction, error) {
	entries, err := os.ReadDir(s.dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var out []*Transaction
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".json" {
			continue
		}
		data, err := os.ReadFile(filepath.Join(s.dir, e.Name()))
		if err != nil {
			continue
		}
		var tx Transaction
		if err := json.Unmarshal(data, &tx); err != nil {
			continue
		}
		if tx.State == "started" {
			out = append(out, &tx)
		}
	}
	return out, nil
}

// PurgeOld removes transactions older than maxAge. Run at startup to bound disk.
func (s *Store) PurgeOld(maxAge time.Duration) (int, error) {
	entries, err := os.ReadDir(s.dir)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, err
	}
	cutoff := time.Now().Add(-maxAge)
	var purged int
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".json" {
			continue
		}
		path := filepath.Join(s.dir, e.Name())
		info, err := os.Stat(path)
		if err != nil {
			continue
		}
		if info.ModTime().Before(cutoff) {
			if err := os.Remove(path); err == nil {
				purged++
			}
		}
	}
	return purged, nil
}
