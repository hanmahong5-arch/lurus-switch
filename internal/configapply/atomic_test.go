package configapply

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestWriteAtomic_NewFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "new.txt")
	if err := WriteAtomic(path, []byte("hello\n"), 0644); err != nil {
		t.Fatalf("WriteAtomic: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if string(data) != "hello\n" {
		t.Errorf("got %q, want %q", data, "hello\n")
	}
}

func TestWriteAtomic_Overwrite(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "exist.txt")
	if err := os.WriteFile(path, []byte("old"), 0644); err != nil {
		t.Fatalf("seed: %v", err)
	}
	if err := WriteAtomic(path, []byte("new"), 0644); err != nil {
		t.Fatalf("WriteAtomic: %v", err)
	}
	data, _ := os.ReadFile(path)
	if string(data) != "new" {
		t.Errorf("got %q, want %q", data, "new")
	}
}

func TestWriteAtomic_NoTempLeak(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "f.txt")
	if err := WriteAtomic(path, []byte("x"), 0644); err != nil {
		t.Fatalf("WriteAtomic: %v", err)
	}
	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), ".tmp-configapply-") {
			t.Errorf("leaked temp file: %s", e.Name())
		}
	}
}

func TestReadFileOrEmpty_Missing(t *testing.T) {
	got, err := ReadFileOrEmpty(filepath.Join(t.TempDir(), "nope"))
	if err != nil {
		t.Fatalf("ReadFileOrEmpty: %v", err)
	}
	if got != "" {
		t.Errorf("expected empty string, got %q", got)
	}
}

func TestReadFileOrEmpty_Exists(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "f")
	if err := os.WriteFile(path, []byte("body"), 0644); err != nil {
		t.Fatal(err)
	}
	got, err := ReadFileOrEmpty(path)
	if err != nil {
		t.Fatalf("ReadFileOrEmpty: %v", err)
	}
	if got != "body" {
		t.Errorf("got %q, want %q", got, "body")
	}
}

func TestFileSizeMatches(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "f")
	if err := os.WriteFile(path, []byte("12345"), 0644); err != nil {
		t.Fatal(err)
	}
	ok, err := FileSizeMatches(path, 5)
	if err != nil {
		t.Fatalf("FileSizeMatches: %v", err)
	}
	if !ok {
		t.Error("expected size match")
	}
	ok, _ = FileSizeMatches(path, 99)
	if ok {
		t.Error("expected size mismatch")
	}
}

// TestWriteAtomic_ModePreservation verifies that WriteAtomic honours the caller-
// supplied file mode so that secret-bearing configs (0600) stay 0600 and
// settings files (0644) stay 0644 after a write.
//
// On Windows the kernel ignores the Unix permission bits and ACLs govern
// access instead; we check the mode only on non-Windows platforms and merely
// verify that the file exists and has the right content on Windows.
func TestWriteAtomic_ModePreservation(t *testing.T) {
	tests := []struct {
		name    string
		content string
		mode    os.FileMode
	}{
		{name: "secret_0600", content: `{"api_key":"sk-secret"}`, mode: 0o600},
		{name: "settings_0644", content: `{"setting":"value"}`, mode: 0o644},
		{name: "overwrite_keeps_mode", content: "line1\n", mode: 0o600},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "cfg.json")

			// Pre-seed a file with a different mode to ensure Chmod takes effect
			// even for the overwrite case.
			_ = os.WriteFile(path, []byte("old"), 0o644)

			if err := WriteAtomic(path, []byte(tt.content), tt.mode); err != nil {
				t.Fatalf("WriteAtomic: %v", err)
			}

			// Content must be correct.
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read: %v", err)
			}
			if string(data) != tt.content {
				t.Errorf("content = %q, want %q", data, tt.content)
			}

			// Mode check — skipped on Windows where permission bits are not
			// meaningful (ACLs govern access there).
			info, err := os.Stat(path)
			if err != nil {
				t.Fatalf("stat: %v", err)
			}
			got := info.Mode().Perm()
			if runtime.GOOS != "windows" && got != tt.mode {
				t.Errorf("mode = %04o, want %04o", got, tt.mode)
			}

			// No temp files must be left behind.
			entries, _ := os.ReadDir(dir)
			for _, e := range entries {
				if strings.HasPrefix(e.Name(), ".tmp-configapply-") {
					t.Errorf("leaked temp file: %s", e.Name())
				}
			}
		})
	}
}
