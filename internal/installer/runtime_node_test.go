package installer

import (
	"context"
	"testing"
)

func TestNewNodeRuntime(t *testing.T) {
	rt := NewNodeRuntime()
	if rt == nil {
		t.Fatal("NewNodeRuntime should return non-nil")
	}
	if rt.nodePath != "" {
		t.Error("nodePath should be empty initially")
	}
}

func TestNodeRuntime_GetPath_InitiallyEmpty(t *testing.T) {
	rt := NewNodeRuntime()
	if rt.GetPath() != "" {
		t.Error("GetPath should be empty before FindNode is called")
	}
}

func TestNodeRuntime_IsInstalled_NoPanic(t *testing.T) {
	rt := NewNodeRuntime()
	// Should not panic regardless of environment
	_ = rt.IsInstalled()
}

func TestNodeRuntime_FindNode_CachesResult(t *testing.T) {
	rt := NewNodeRuntime()
	if !rt.IsInstalled() {
		t.Skip("node not installed, skipping cache test")
	}

	path1, err := rt.FindNode()
	if err != nil {
		t.Fatalf("FindNode failed: %v", err)
	}
	path2, err := rt.FindNode()
	if err != nil {
		t.Fatalf("FindNode second call failed: %v", err)
	}
	if path1 != path2 {
		t.Errorf("FindNode should return cached result: %q vs %q", path1, path2)
	}
}

func TestNodeRuntime_MeetsMinVersion_NoNode(t *testing.T) {
	rt := &NodeRuntime{nodePath: "/nonexistent/node"}
	ctx := context.Background()
	if rt.MeetsMinVersion(ctx, 22) {
		t.Error("MeetsMinVersion should return false when node is not accessible")
	}
}
