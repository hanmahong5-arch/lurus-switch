package configapply

import (
	"errors"
	"io/fs"
	"os"
	"strings"
	"testing"
)

func TestExplainer_DiskFull(t *testing.T) {
	what, _, next := Explain(DefaultExplainers(), errors.New("write failed: no space left on device"), nil)
	if !strings.Contains(what, "磁盘空间不足") {
		t.Errorf("expected 磁盘空间不足, got %q", what)
	}
	if len(next) == 0 {
		t.Error("expected next steps")
	}
}

func TestExplainer_Permission(t *testing.T) {
	what, _, _ := Explain(DefaultExplainers(), fs.ErrPermission, nil)
	if !strings.Contains(what, "权限不足") {
		t.Errorf("expected 权限不足, got %q", what)
	}
}

func TestExplainer_FileLocked(t *testing.T) {
	what, _, _ := Explain(DefaultExplainers(),
		errors.New("rename: the process cannot access the file because it is being used by another process"),
		nil)
	if !strings.Contains(what, "被其他进程锁住") {
		t.Errorf("expected 被其他进程锁住, got %q", what)
	}
}

func TestExplainer_FileNotFound(t *testing.T) {
	what, _, _ := Explain(DefaultExplainers(), os.ErrNotExist, nil)
	if !strings.Contains(what, "不存在") {
		t.Errorf("expected 不存在, got %q", what)
	}
}

func TestExplainer_PathTooLong(t *testing.T) {
	what, _, next := Explain(DefaultExplainers(),
		errors.New("open: filename too long"), nil)
	if !strings.Contains(what, "路径") || !strings.Contains(what, "限制") {
		t.Errorf("expected path-too-long message, got %q", what)
	}
	foundDoc := false
	for _, ns := range next {
		if ns.URL != "" {
			foundDoc = true
		}
	}
	if !foundDoc {
		t.Error("expected at least one NextStep with a documentation URL")
	}
}

func TestExplainer_Generic(t *testing.T) {
	what, _, next := Explain(DefaultExplainers(), errors.New("kernel panic in module xyz"), nil)
	if !strings.Contains(what, "未分类的错误") {
		t.Errorf("expected fallback message, got %q", what)
	}
	if len(next) == 0 {
		t.Error("generic should still produce next steps")
	}
}

func TestExplainer_NilError(t *testing.T) {
	what, expected, next := Explain(DefaultExplainers(), nil, nil)
	if what != "" || expected != "" || next != nil {
		t.Error("nil error should produce empty result")
	}
}

func TestExplainer_OrderMatters(t *testing.T) {
	// PermissionDenied should match before Generic
	chain := []Explainer{PermissionDeniedExplainer{}, GenericExplainer{}}
	what, _, _ := Explain(chain, errors.New("access is denied"), nil)
	if !strings.Contains(what, "权限不足") {
		t.Errorf("expected permission-denied to win, got %q", what)
	}
}
