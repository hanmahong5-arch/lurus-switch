package configapply

import (
	"os"
	"path/filepath"
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
