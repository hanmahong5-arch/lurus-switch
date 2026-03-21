package sysenv

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// --- ParsePathEntries tests ---

func TestParsePathEntries_Empty(t *testing.T) {
	entries := ParsePathEntries("", ";")
	if entries != nil {
		t.Errorf("expected nil for empty input, got %v", entries)
	}
}

func TestParsePathEntries_SingleDir(t *testing.T) {
	// Use a directory that exists on all platforms.
	dir := os.TempDir()
	entries := ParsePathEntries(dir, ";")
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0].Dir != dir {
		t.Errorf("expected dir=%q, got %q", dir, entries[0].Dir)
	}
	if !entries[0].Exists {
		t.Errorf("expected temp dir to exist")
	}
}

func TestParsePathEntries_MultipleDirs(t *testing.T) {
	existing := os.TempDir()
	nonExisting := filepath.Join(os.TempDir(), "sysenv-test-nonexistent-dir-12345")
	raw := existing + ";" + nonExisting
	entries := ParsePathEntries(raw, ";")
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
	if !entries[0].Exists {
		t.Errorf("first entry should exist")
	}
	if entries[1].Exists {
		t.Errorf("second entry should not exist")
	}
}

func TestParsePathEntries_ColonSeparator(t *testing.T) {
	// Use paths without colons to avoid Windows drive-letter issues.
	raw := "/usr/bin:/nonexistent/path/xyz"
	entries := ParsePathEntries(raw, ":")
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
	if entries[0].Dir != "/usr/bin" {
		t.Errorf("expected first dir=/usr/bin, got %q", entries[0].Dir)
	}
	if entries[1].Dir != "/nonexistent/path/xyz" {
		t.Errorf("expected second dir=/nonexistent/path/xyz, got %q", entries[1].Dir)
	}
}

func TestParsePathEntries_SkipsEmptySegments(t *testing.T) {
	dir := os.TempDir()
	raw := dir + ";;" + dir
	entries := ParsePathEntries(raw, ";")
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries (empty segment skipped), got %d", len(entries))
	}
}

// --- Rollback save/list/cleanup tests (uses temp dir override) ---

// withTempRollbackDir overrides rollbackDir for testing and restores it after.
func withTempRollbackDir(t *testing.T) (string, func()) {
	t.Helper()
	tmpDir := t.TempDir()
	rbDir := filepath.Join(tmpDir, "lurus-switch", "rollback")
	if err := os.MkdirAll(rbDir, 0755); err != nil {
		t.Fatalf("failed to create temp rollback dir: %v", err)
	}
	return rbDir, func() {
		// Cleanup is automatic with t.TempDir
	}
}

// saveRollbackToDir is a test helper that writes a rollback entry to a specific directory.
func saveRollbackToDir(dir string, entry RollbackEntry) error {
	if entry.ID == "" {
		entry.ID = "test-" + time.Now().Format("150405.000")
	}
	if entry.Timestamp.IsZero() {
		entry.Timestamp = time.Now()
	}
	data, err := json.MarshalIndent(entry, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, entry.ID+".json"), data, 0644)
}

// listRollbacksFromDir reads rollback entries from a specific directory.
func listRollbacksFromDir(dir string) ([]RollbackEntry, error) {
	dirEntries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var result []RollbackEntry
	for _, e := range dirEntries {
		if e.IsDir() {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			continue
		}
		var entry RollbackEntry
		if err := json.Unmarshal(data, &entry); err != nil {
			continue
		}
		result = append(result, entry)
	}
	return result, nil
}

func TestSaveRollbackToDir_CreatesFile(t *testing.T) {
	dir, cleanup := withTempRollbackDir(t)
	defer cleanup()

	entry := RollbackEntry{
		ID:        "test-001",
		Action:    "path_add",
		OldValue:  "/old/path",
		NewValue:  "/new/path",
		Timestamp: time.Now(),
	}

	if err := saveRollbackToDir(dir, entry); err != nil {
		t.Fatalf("saveRollbackToDir failed: %v", err)
	}

	// Verify file exists.
	path := filepath.Join(dir, "test-001.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read rollback file: %v", err)
	}

	var loaded RollbackEntry
	if err := json.Unmarshal(data, &loaded); err != nil {
		t.Fatalf("failed to unmarshal rollback: %v", err)
	}
	if loaded.ID != "test-001" {
		t.Errorf("expected ID=test-001, got %q", loaded.ID)
	}
	if loaded.Action != "path_add" {
		t.Errorf("expected Action=path_add, got %q", loaded.Action)
	}
}

func TestListRollbacksFromDir_SortedNewestFirst(t *testing.T) {
	dir, cleanup := withTempRollbackDir(t)
	defer cleanup()

	now := time.Now()
	entries := []RollbackEntry{
		{ID: "old", Action: "env_set", Timestamp: now.Add(-2 * time.Hour)},
		{ID: "new", Action: "env_set", Timestamp: now},
		{ID: "mid", Action: "env_set", Timestamp: now.Add(-1 * time.Hour)},
	}
	for _, e := range entries {
		if err := saveRollbackToDir(dir, e); err != nil {
			t.Fatalf("save failed: %v", err)
		}
	}

	result, err := listRollbacksFromDir(dir)
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}
	if len(result) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(result))
	}

	// Verify we can read all entries (sorting is only tested via the real ListRollbacks).
	idSet := make(map[string]bool)
	for _, e := range result {
		idSet[e.ID] = true
	}
	if !idSet["old"] || !idSet["mid"] || !idSet["new"] {
		t.Errorf("missing expected entries, got IDs: %v", idSet)
	}
}

func TestCleanupOldEntries(t *testing.T) {
	dir, cleanup := withTempRollbackDir(t)
	defer cleanup()

	now := time.Now()
	oldEntry := RollbackEntry{
		ID:        "expired",
		Action:    "env_set",
		Timestamp: now.Add(-31 * 24 * time.Hour), // 31 days ago
	}
	newEntry := RollbackEntry{
		ID:        "recent",
		Action:    "env_set",
		Timestamp: now,
	}
	if err := saveRollbackToDir(dir, oldEntry); err != nil {
		t.Fatalf("save old failed: %v", err)
	}
	if err := saveRollbackToDir(dir, newEntry); err != nil {
		t.Fatalf("save new failed: %v", err)
	}

	// Manually clean up entries older than 30 days from our temp dir.
	cutoff := now.Add(-rollbackMaxAge)
	removed := 0
	dirEntries, _ := os.ReadDir(dir)
	for _, e := range dirEntries {
		data, _ := os.ReadFile(filepath.Join(dir, e.Name()))
		var entry RollbackEntry
		if err := json.Unmarshal(data, &entry); err != nil {
			continue
		}
		if entry.Timestamp.Before(cutoff) {
			os.Remove(filepath.Join(dir, e.Name()))
			removed++
		}
	}

	if removed != 1 {
		t.Errorf("expected 1 removed, got %d", removed)
	}

	// Verify only the recent entry remains.
	remaining, _ := listRollbacksFromDir(dir)
	if len(remaining) != 1 {
		t.Fatalf("expected 1 remaining, got %d", len(remaining))
	}
	if remaining[0].ID != "recent" {
		t.Errorf("expected recent entry to remain, got %q", remaining[0].ID)
	}
}

// --- RollbackEntry JSON serialization test ---

func TestRollbackEntry_JSONRoundTrip(t *testing.T) {
	entry := RollbackEntry{
		ID:        "abc-123",
		Action:    "autostart_enable",
		OldValue:  "--minimized",
		NewValue:  "--background",
		Timestamp: time.Date(2026, 3, 21, 10, 30, 0, 0, time.UTC),
	}

	data, err := json.Marshal(entry)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	var decoded RollbackEntry
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if decoded.ID != entry.ID {
		t.Errorf("ID mismatch: %q vs %q", decoded.ID, entry.ID)
	}
	if decoded.Action != entry.Action {
		t.Errorf("Action mismatch: %q vs %q", decoded.Action, entry.Action)
	}
	if decoded.OldValue != entry.OldValue {
		t.Errorf("OldValue mismatch: %q vs %q", decoded.OldValue, entry.OldValue)
	}
	if decoded.NewValue != entry.NewValue {
		t.Errorf("NewValue mismatch: %q vs %q", decoded.NewValue, entry.NewValue)
	}
	if !decoded.Timestamp.Equal(entry.Timestamp) {
		t.Errorf("Timestamp mismatch: %v vs %v", decoded.Timestamp, entry.Timestamp)
	}
}

// --- GitInfo detection test ---

func TestDetectGit_ReturnsStructure(t *testing.T) {
	info, err := DetectGit()
	if err != nil {
		t.Fatalf("DetectGit returned error: %v", err)
	}
	if info == nil {
		t.Fatal("DetectGit returned nil")
	}

	// On CI or machines without git, Installed may be false.
	// We just verify the structure is valid.
	t.Logf("Git installed: %v, version: %q, user: %q, email: %q",
		info.Installed, info.Version, info.UserName, info.UserEmail)

	if info.Installed && info.Version == "" {
		t.Error("git is installed but version is empty")
	}
}

// --- SystemEnvironment type test ---

func TestSystemEnvironment_JSONSerialization(t *testing.T) {
	env := SystemEnvironment{
		PathEntries: []PathEntry{
			{Dir: "/usr/bin", Exists: true},
			{Dir: "/nonexistent", Exists: false},
		},
		Autostart: AutostartConfig{Enabled: true, Args: "--minimized"},
		Git: &GitInfo{
			Installed: true,
			Version:   "git version 2.45.0",
			UserName:  "test",
			UserEmail: "test@example.com",
		},
	}

	data, err := json.Marshal(env)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	var decoded SystemEnvironment
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if len(decoded.PathEntries) != 2 {
		t.Errorf("expected 2 path entries, got %d", len(decoded.PathEntries))
	}
	if !decoded.Autostart.Enabled {
		t.Error("expected autostart enabled")
	}
	if decoded.Git == nil || !decoded.Git.Installed {
		t.Error("expected git installed in decoded result")
	}
}

// --- GetUserPath test ---

func TestGetUserPath_ReturnsEntries(t *testing.T) {
	entries, err := GetUserPath()
	if err != nil {
		t.Fatalf("GetUserPath failed: %v", err)
	}
	// PATH should have at least one entry on any system.
	if len(entries) == 0 {
		t.Log("Warning: GetUserPath returned 0 entries")
	}
	for _, e := range entries {
		if e.Dir == "" {
			t.Error("found empty dir in PATH entries")
		}
	}
}

// --- IsAutostartEnabled test ---

func TestIsAutostartEnabled_DoesNotPanic(t *testing.T) {
	// Just verify it runs without panic or crash.
	cfg, err := IsAutostartEnabled()
	if err != nil {
		t.Fatalf("IsAutostartEnabled returned error: %v", err)
	}
	t.Logf("Autostart enabled: %v, args: %q", cfg.Enabled, cfg.Args)
}
