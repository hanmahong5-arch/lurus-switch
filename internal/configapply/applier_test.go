package configapply

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/uuid"
)

func newTestApplier(t *testing.T) (*Applier, string) {
	t.Helper()
	dir := t.TempDir()
	storeDir := filepath.Join(dir, "store")
	store, err := NewStoreAt(storeDir)
	if err != nil {
		t.Fatalf("NewStoreAt: %v", err)
	}
	return NewApplier(store), dir
}

func makeUpdatePlan(targets map[string]string) *ChangePlan {
	plan := &ChangePlan{
		ID:        uuid.NewString(),
		Intent:    "test-save",
		CreatedAt: "2026-05-19T00:00:00Z",
	}
	for path, after := range targets {
		before, _ := ReadFileOrEmpty(path)
		kind := KindUpdate
		if before == "" {
			kind = KindCreate
		}
		plan.Changes = append(plan.Changes, FileChange{
			Path:   path,
			Kind:   kind,
			Before: before,
			After:  after,
			Mode:   0644,
		})
	}
	return plan
}

func TestApply_HappyPath_SingleFile(t *testing.T) {
	applier, dir := newTestApplier(t)
	target := filepath.Join(dir, "a.txt")
	plan := makeUpdatePlan(map[string]string{target: "hello\n"})

	res := applier.Apply(plan)
	if !res.Success {
		t.Fatalf("expected success, got: %+v", res)
	}
	if res.Phase != PhaseDone {
		t.Errorf("expected PhaseDone, got %s", res.Phase)
	}
	data, _ := os.ReadFile(target)
	if string(data) != "hello\n" {
		t.Errorf("got %q, want %q", data, "hello\n")
	}
	if len(res.FilesWritten) != 1 {
		t.Errorf("expected 1 written, got %d", len(res.FilesWritten))
	}
}

func TestApply_HappyPath_MultiFile(t *testing.T) {
	applier, dir := newTestApplier(t)
	a := filepath.Join(dir, "a.txt")
	b := filepath.Join(dir, "b.txt")
	plan := makeUpdatePlan(map[string]string{a: "1", b: "2"})

	res := applier.Apply(plan)
	if !res.Success {
		t.Fatalf("expected success, got: %+v", res)
	}
	if len(res.FilesWritten) != 2 {
		t.Errorf("expected 2 written, got %d", len(res.FilesWritten))
	}
}

func TestApply_RollbackOnWriteFailure(t *testing.T) {
	applier, dir := newTestApplier(t)
	a := filepath.Join(dir, "a.txt")
	if err := os.WriteFile(a, []byte("original-a"), 0644); err != nil {
		t.Fatal(err)
	}
	// Make parentAsFile a regular file so MkdirAll(parentAsFile/...) fails on
	// every OS (the path component is occupied by a non-directory). Portable
	// way to force a write failure without OS-specific permission tricks.
	parentAsFile := filepath.Join(dir, "is-file")
	if err := os.WriteFile(parentAsFile, []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}
	bad := filepath.Join(parentAsFile, "child.txt")

	plan := makeUpdatePlan(map[string]string{a: "new-a", bad: "new-b"})

	res := applier.Apply(plan)
	if res.Success {
		t.Fatalf("expected failure, got success: %+v", res)
	}
	data, _ := os.ReadFile(a)
	if string(data) != "original-a" {
		t.Errorf("rollback failed: a.txt = %q, want %q", data, "original-a")
	}
	if !res.RollbackDone && len(res.FilesRolled) == 0 {
		t.Errorf("expected rollback marker, got RollbackDone=%v FilesRolled=%v",
			res.RollbackDone, res.FilesRolled)
	}
	if res.WhatHappened == "" {
		t.Error("expected WhatHappened to be populated by explainer")
	}
}

func TestApply_NilPlan(t *testing.T) {
	applier, _ := newTestApplier(t)
	res := applier.Apply(nil)
	if res.Success {
		t.Error("nil plan should fail")
	}
	if !strings.Contains(res.RawError, "nil plan") {
		t.Errorf("expected nil plan error, got %q", res.RawError)
	}
}

func TestApply_RelativePathRejected(t *testing.T) {
	applier, _ := newTestApplier(t)
	plan := &ChangePlan{
		ID:     uuid.NewString(),
		Intent: "test",
		Changes: []FileChange{{
			Path:   "relative/path.txt",
			Kind:   KindCreate,
			After:  "x",
			Mode:   0644,
		}},
	}
	res := applier.Apply(plan)
	if res.Success {
		t.Error("relative path should be rejected")
	}
	if !strings.Contains(res.RawError, "not absolute") {
		t.Errorf("expected 'not absolute' in error, got %q", res.RawError)
	}
}

func TestApply_EmptyPlan(t *testing.T) {
	applier, _ := newTestApplier(t)
	plan := &ChangePlan{
		ID:     uuid.NewString(),
		Intent: "test",
	}
	res := applier.Apply(plan)
	if !res.Success {
		t.Errorf("empty plan should succeed (no-op), got: %+v", res)
	}
	if !strings.Contains(res.WhatHappened, "无改动") {
		t.Errorf("expected '无改动' note, got %q", res.WhatHappened)
	}
}

func TestRegistry_PlanExecution(t *testing.T) {
	reg := NewRegistry()
	reg.Register(SaveSingleFilePlanner{
		IntentName: "save-test",
		DescribeFn: func(p map[string]any) string { return "test save" },
	})

	dir := t.TempDir()
	target := filepath.Join(dir, "out.txt")

	plan, err := reg.Plan("save-test", map[string]any{
		"path":    target,
		"content": "hello",
	})
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	if plan.ID == "" {
		t.Error("expected plan ID populated")
	}
	if plan.Description != "test save" {
		t.Errorf("Description = %q", plan.Description)
	}
	if len(plan.Changes) != 1 {
		t.Fatalf("expected 1 change, got %d", len(plan.Changes))
	}
	if plan.Changes[0].After != "hello" {
		t.Errorf("After = %q", plan.Changes[0].After)
	}
	if plan.Changes[0].Kind != KindCreate {
		t.Errorf("Kind = %s, want create (file did not exist)", plan.Changes[0].Kind)
	}
}

func TestRegistry_UnknownIntent(t *testing.T) {
	reg := NewRegistry()
	_, err := reg.Plan("never-registered", nil)
	if err == nil {
		t.Error("expected error for unknown intent")
	}
}
