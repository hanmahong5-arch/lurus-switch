package snapshot

import (
	"strings"
	"testing"
)

// snapshotEnv redirects the snapshot base dir to a per-test tempdir by
// pinning APPDATA / HOME / XDG_CONFIG_HOME — the same pattern the rest of
// the repo uses (see internal/config/store_test.go).
func snapshotEnv(t *testing.T) {
	t.Helper()
	tmp := t.TempDir()
	t.Setenv("APPDATA", tmp)
	t.Setenv("HOME", tmp)
	t.Setenv("XDG_CONFIG_HOME", tmp)
}

// TestNewStore_CreatesBaseDir captures the contract that NewStore returns a
// usable Store even on a clean machine — the snapshot UI relies on this to
// avoid an empty-state crash.
func TestNewStore_CreatesBaseDir(t *testing.T) {
	snapshotEnv(t)
	s, err := NewStore()
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	if s == nil {
		t.Fatal("NewStore returned nil store")
	}
	if s.baseDir == "" {
		t.Error("baseDir not set after NewStore")
	}
}

// TestStore_TakeListRestore_RoundTrip locks in the on-disk format — if a
// future refactor renames a JSON field, the snapshot won't round-trip and
// this test fails before any user notices.
func TestStore_TakeListRestore_RoundTrip(t *testing.T) {
	snapshotEnv(t)
	s, err := NewStore()
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}

	const content = `{"model":"claude-sonnet-4-20250514"}`
	if err := s.Take("claude", "before-edit", content); err != nil {
		t.Fatalf("Take: %v", err)
	}

	metas, err := s.List("claude")
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(metas) != 1 {
		t.Fatalf("List returned %d items, want 1", len(metas))
	}
	if metas[0].Tool != "claude" || metas[0].Label != "before-edit" {
		t.Errorf("meta mismatch: %+v", metas[0])
	}
	if metas[0].Size != len(content) {
		t.Errorf("size = %d, want %d", metas[0].Size, len(content))
	}

	restored, err := s.Restore("claude", metas[0].ID)
	if err != nil {
		t.Fatalf("Restore: %v", err)
	}
	if restored != content {
		t.Errorf("Restore content = %q, want %q", restored, content)
	}
}

// TestStore_Delete_RemovesEntry ensures Delete actually clears the on-disk
// file (a hard requirement for the "Clear all snapshots" UI affordance).
func TestStore_Delete_RemovesEntry(t *testing.T) {
	snapshotEnv(t)
	s, _ := NewStore()
	_ = s.Take("claude", "x", "hello")

	metas, _ := s.List("claude")
	if len(metas) == 0 {
		t.Fatal("nothing to delete")
	}
	if err := s.Delete("claude", metas[0].ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	after, _ := s.List("claude")
	if len(after) != 0 {
		t.Errorf("after Delete, %d items remain; want 0", len(after))
	}
}

// TestValidateToken_RejectsPathTraversal locks in the file-system safety
// contract — the snapshot loader runs unsandboxed, so a tool ID containing
// "../" would let a malicious config escape the snapshots directory.
func TestValidateToken_RejectsPathTraversal(t *testing.T) {
	cases := []struct {
		in      string
		wantErr bool
	}{
		{"claude", false},
		{"codex_2", false},
		{"", true},
		{"../etc", true},
		{`tool\name`, true},
		{"tool/name", true},
		{"..", true},
	}
	for _, tc := range cases {
		t.Run(tc.in, func(t *testing.T) {
			err := validateToken(tc.in)
			if (err != nil) != tc.wantErr {
				t.Errorf("validateToken(%q) err = %v, wantErr %v", tc.in, err, tc.wantErr)
			}
		})
	}
}

// TestSanitizeLabel_RemovesSeparators captures the trim+replace behaviour
// — snapshot IDs are filenames, so spaces and slashes must collapse to
// underscores or restore on Windows breaks.
func TestSanitizeLabel_RemovesSeparators(t *testing.T) {
	got := sanitizeLabel("hello world / safe:edit")
	if strings.ContainsAny(got, ` /\:`) {
		t.Errorf("sanitizeLabel leaked separator: %q", got)
	}
	if got := sanitizeLabel(""); got == "" {
		t.Error("sanitizeLabel(\"\") returned empty, want fallback")
	}
}
