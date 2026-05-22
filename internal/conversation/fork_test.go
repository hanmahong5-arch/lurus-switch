package conversation

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFork_CopiesUpToAndIncludingMessage(t *testing.T) {
	dir := t.TempDir()
	parentPath := filepath.Join(dir, "parent.jsonl")
	if err := os.WriteFile(parentPath, []byte(fixtureClaude), 0o600); err != nil {
		t.Fatal(err)
	}
	parent := SessionFile{Tool: "claude", SessionID: "parent", Path: parentPath}

	res, err := Fork(parent, "a1")
	if err != nil {
		t.Fatalf("Fork: %v", err)
	}
	if res.MessagesKept != 2 {
		t.Errorf("want 2 messages kept (u1, a1), got %d", res.MessagesKept)
	}
	if res.NewSessionID == "" || res.NewPath == "" {
		t.Fatal("ForkResult missing identifiers")
	}

	data, err := os.ReadFile(res.NewPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `"uuid":"u1"`) || !strings.Contains(string(data), `"uuid":"a1"`) {
		t.Error("forked file should contain u1 and a1")
	}
	if strings.Contains(string(data), `"uuid":"a2"`) {
		t.Error("forked file should NOT contain a2 (after fork point)")
	}

	// Sidecar exists with parent + fork point.
	sc, err := readForkSidecar(res.NewPath)
	if err != nil {
		t.Fatalf("sidecar: %v", err)
	}
	if sc.ParentSessionID != "parent" || sc.ForkPointUUID != "a1" {
		t.Errorf("sidecar wrong: %+v", sc)
	}
}

func TestFork_UnknownUUIDFails(t *testing.T) {
	dir := t.TempDir()
	parentPath := filepath.Join(dir, "p.jsonl")
	_ = os.WriteFile(parentPath, []byte(fixtureClaude), 0o600)
	parent := SessionFile{Path: parentPath, SessionID: "p", Tool: "claude"}

	if _, err := Fork(parent, "does-not-exist"); err == nil {
		t.Fatal("expected error for missing UUID")
	}
	// No half-formed child should remain.
	entries, _ := os.ReadDir(dir)
	if len(entries) != 1 {
		t.Errorf("dir should still contain only the parent, got %d entries", len(entries))
	}
}
