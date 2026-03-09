package snapshot

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"
)

// SnapshotMeta describes a stored configuration snapshot
type SnapshotMeta struct {
	ID        string `json:"id"`
	Tool      string `json:"tool"`
	Label     string `json:"label"`
	CreatedAt string `json:"createdAt"`
	Size      int    `json:"size"` // bytes of stored content
}

// snapshot is the on-disk structure
type snapshot struct {
	Meta    SnapshotMeta `json:"meta"`
	Content string       `json:"content"`
}

// Store manages configuration snapshots
type Store struct {
	baseDir string
}

// NewStore creates a snapshot store
func NewStore() (*Store, error) {
	dir, err := snapshotsBaseDir()
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create snapshots directory: %w", err)
	}
	return &Store{baseDir: dir}, nil
}

// snapshotsBaseDir returns the base directory for all snapshots
func snapshotsBaseDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}

	var base string
	switch runtime.GOOS {
	case "windows":
		appData := os.Getenv("APPDATA")
		if appData == "" {
			appData = filepath.Join(home, "AppData", "Roaming")
		}
		base = filepath.Join(appData, "lurus-switch")
	case "darwin":
		base = filepath.Join(home, "Library", "Application Support", "lurus-switch")
	default:
		base = filepath.Join(home, ".lurus-switch")
	}

	return filepath.Join(base, "snapshots"), nil
}

// toolDir returns the snapshot directory for a specific tool
func (s *Store) toolDir(tool string) (string, error) {
	if err := validateToken(tool); err != nil {
		return "", err
	}
	return filepath.Join(s.baseDir, tool), nil
}

// Take creates a new snapshot for a tool
func (s *Store) Take(tool, label, content string) error {
	dir, err := s.toolDir(tool)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create snapshot dir: %w", err)
	}

	id := fmt.Sprintf("%s-%s", time.Now().Format("20060102-150405"), sanitizeLabel(label))
	meta := SnapshotMeta{
		ID:        id,
		Tool:      tool,
		Label:     label,
		CreatedAt: time.Now().Format(time.RFC3339),
		Size:      len(content),
	}
	snap := snapshot{Meta: meta, Content: content}

	data, err := json.MarshalIndent(snap, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal snapshot: %w", err)
	}

	if err := os.WriteFile(filepath.Join(dir, id+".json"), data, 0644); err != nil {
		return err
	}

	// Keep auto-save snapshots bounded to avoid disk bloat
	if label == "auto-save" {
		s.pruneOldest(tool)
	}

	return nil
}

// List returns all snapshot metadata for a tool, sorted newest first
func (s *Store) List(tool string) ([]SnapshotMeta, error) {
	dir, err := s.toolDir(tool)
	if err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read snapshot dir: %w", err)
	}

	var metas []SnapshotMeta
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			continue
		}
		var snap snapshot
		if err := json.Unmarshal(data, &snap); err != nil {
			continue
		}
		metas = append(metas, snap.Meta)
	}

	// Sort newest first
	sort.Slice(metas, func(i, j int) bool {
		return metas[i].CreatedAt > metas[j].CreatedAt
	})

	return metas, nil
}

// Restore returns the content of a snapshot by ID
func (s *Store) Restore(tool, id string) (string, error) {
	if err := validateToken(id); err != nil {
		return "", err
	}
	dir, err := s.toolDir(tool)
	if err != nil {
		return "", err
	}

	data, err := os.ReadFile(filepath.Join(dir, id+".json"))
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("snapshot not found: %s", id)
		}
		return "", fmt.Errorf("failed to read snapshot: %w", err)
	}

	var snap snapshot
	if err := json.Unmarshal(data, &snap); err != nil {
		return "", fmt.Errorf("failed to parse snapshot: %w", err)
	}

	return snap.Content, nil
}

// Delete removes a snapshot
func (s *Store) Delete(tool, id string) error {
	if err := validateToken(id); err != nil {
		return err
	}
	dir, err := s.toolDir(tool)
	if err != nil {
		return err
	}
	path := filepath.Join(dir, id+".json")
	if err := os.Remove(path); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("snapshot not found: %s", id)
		}
		return fmt.Errorf("failed to delete snapshot: %w", err)
	}
	return nil
}

// maxSnapshotsPerTool is the upper bound for auto-saved snapshots per tool.
// When Take is called with label "auto-save" and this limit is reached,
// the oldest snapshot for that tool is deleted first.
const maxSnapshotsPerTool = 20

// pruneOldest deletes the oldest snapshot for a tool if count exceeds limit.
func (s *Store) pruneOldest(tool string) {
	metas, err := s.List(tool)
	if err != nil || len(metas) <= maxSnapshotsPerTool {
		return
	}
	// List is sorted newest-first; oldest is last
	oldest := metas[len(metas)-1]
	_ = s.Delete(tool, oldest.ID)
}

// ClearTool removes all snapshot files for a specific tool
func (s *Store) ClearTool(tool string) (int, error) {
	dir, err := s.toolDir(tool)
	if err != nil {
		return 0, err
	}
	files, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, fmt.Errorf("failed to read tool snapshot dir: %w", err)
	}
	count := 0
	for _, f := range files {
		if strings.HasSuffix(f.Name(), ".json") {
			if err := os.Remove(filepath.Join(dir, f.Name())); err == nil {
				count++
			}
		}
	}
	return count, nil
}

// ClearAll removes all snapshot files for all tools
func (s *Store) ClearAll() (int, error) {
	entries, err := os.ReadDir(s.baseDir)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, fmt.Errorf("failed to read snapshots directory: %w", err)
	}
	count := 0
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		toolDir := filepath.Join(s.baseDir, entry.Name())
		files, err := os.ReadDir(toolDir)
		if err != nil {
			continue
		}
		for _, f := range files {
			if strings.HasSuffix(f.Name(), ".json") {
				if err := os.Remove(filepath.Join(toolDir, f.Name())); err == nil {
					count++
				}
			}
		}
	}
	return count, nil
}

// Diff returns a simple line-diff between two snapshots (unified diff format)
func (s *Store) Diff(tool, id1, id2 string) (string, error) {
	c1, err := s.Restore(tool, id1)
	if err != nil {
		return "", fmt.Errorf("snapshot 1: %w", err)
	}
	c2, err := s.Restore(tool, id2)
	if err != nil {
		return "", fmt.Errorf("snapshot 2: %w", err)
	}

	lines1 := strings.Split(c1, "\n")
	lines2 := strings.Split(c2, "\n")

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("--- %s\n+++ %s\n", id1, id2))

	// Naive line diff (not true LCS, sufficient for config files)
	set1 := make(map[string]bool, len(lines1))
	set2 := make(map[string]bool, len(lines2))
	for _, l := range lines1 {
		set1[l] = true
	}
	for _, l := range lines2 {
		set2[l] = true
	}

	for _, l := range lines1 {
		if !set2[l] {
			sb.WriteString("- " + l + "\n")
		}
	}
	for _, l := range lines2 {
		if !set1[l] {
			sb.WriteString("+ " + l + "\n")
		}
	}

	if sb.Len() == len(fmt.Sprintf("--- %s\n+++ %s\n", id1, id2)) {
		return "(no differences)", nil
	}
	return sb.String(), nil
}

// validateToken prevents path traversal in tool/id values
func validateToken(s string) error {
	if s == "" {
		return fmt.Errorf("value must not be empty")
	}
	if strings.ContainsAny(s, `/\`) || strings.Contains(s, "..") {
		return fmt.Errorf("invalid value: %q", s)
	}
	return nil
}

// sanitizeLabel replaces spaces and special characters in snapshot labels
func sanitizeLabel(label string) string {
	replacer := strings.NewReplacer(" ", "_", "/", "_", "\\", "_", ":", "_")
	s := replacer.Replace(label)
	if len(s) > 40 {
		s = s[:40]
	}
	if s == "" {
		s = "snapshot"
	}
	return s
}
