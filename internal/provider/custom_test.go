package provider

import (
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCustomStore_SaveListDelete(t *testing.T) {
	dir := t.TempDir()
	s, err := NewCustomStore(dir)
	if err != nil {
		t.Fatal(err)
	}

	saved, err := s.Save(CustomProvider{
		Name:    "DeepSeek",
		BaseURL: "https://api.deepseek.com",
		APIKey:  "sk-secret",
	})
	if err != nil {
		t.Fatal(err)
	}
	if saved.ID == "" {
		t.Fatal("expected generated ID")
	}
	if !strings.HasPrefix(saved.ID, "custom-") {
		t.Errorf("ID %q should be prefixed custom- to avoid preset collision", saved.ID)
	}

	list := s.List()
	if len(list) != 1 || list[0].Name != "DeepSeek" {
		t.Fatalf("List = %+v", list)
	}

	if err := s.Delete(saved.ID); err != nil {
		t.Fatal(err)
	}
	if len(s.List()) != 0 {
		t.Error("expected empty after delete")
	}
}

func TestCustomStore_RejectsEmptyBaseURL(t *testing.T) {
	s, _ := NewCustomStore(t.TempDir())
	if _, err := s.Save(CustomProvider{Name: "x", BaseURL: "  "}); err == nil {
		t.Error("expected error for empty base URL")
	}
}

func TestCustomStore_DeleteMissingIsNoop(t *testing.T) {
	s, _ := NewCustomStore(t.TempDir())
	if err := s.Delete("does-not-exist"); err != nil {
		t.Errorf("delete of missing id should be no-op, got %v", err)
	}
}

func TestCustomStore_UpdatePreservesCreatedAt(t *testing.T) {
	s, _ := NewCustomStore(t.TempDir())
	saved, _ := s.Save(CustomProvider{Name: "A", BaseURL: "https://a.test"})
	orig := saved.CreatedAt

	saved.Name = "A-renamed"
	updated, err := s.Save(saved)
	if err != nil {
		t.Fatal(err)
	}
	if !updated.CreatedAt.Equal(orig) {
		t.Errorf("CreatedAt changed on update: %v -> %v", orig, updated.CreatedAt)
	}
	if updated.Name != "A-renamed" {
		t.Errorf("name not updated: %s", updated.Name)
	}
	if len(s.List()) != 1 {
		t.Errorf("update created a duplicate: %d entries", len(s.List()))
	}
}

func TestCustomStore_PersistRoundTripAndKeyObfuscation(t *testing.T) {
	dir := t.TempDir()
	s, _ := NewCustomStore(dir)
	if _, err := s.Save(CustomProvider{
		Name:    "Secret",
		BaseURL: "https://api.secret.test",
		APIKey:  "sk-plaintext-must-not-leak",
	}); err != nil {
		t.Fatal(err)
	}

	// Raw file must NOT contain the plaintext key.
	raw, err := os.ReadFile(filepath.Join(dir, customProvidersFile))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "sk-plaintext-must-not-leak") {
		t.Error("plaintext API key leaked to disk")
	}
	// But base64 of it must be present.
	enc := base64.StdEncoding.EncodeToString([]byte("sk-plaintext-must-not-leak"))
	if !strings.Contains(string(raw), enc) {
		t.Error("expected base64-obfuscated key on disk")
	}

	// Reopen and confirm the key round-trips to plaintext in memory.
	s2, err := NewCustomStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	list := s2.List()
	if len(list) != 1 || list[0].APIKey != "sk-plaintext-must-not-leak" {
		t.Fatalf("key did not round-trip: %+v", list)
	}
}

func TestCustomStore_FilePermissions(t *testing.T) {
	if os.Getenv("OS") == "Windows_NT" {
		t.Skip("POSIX file mode not meaningful on Windows")
	}
	dir := t.TempDir()
	s, _ := NewCustomStore(dir)
	_, _ = s.Save(CustomProvider{Name: "p", BaseURL: "https://p.test", APIKey: "k"})
	info, err := os.Stat(filepath.Join(dir, customProvidersFile))
	if err != nil {
		t.Fatal(err)
	}
	if perm := info.Mode().Perm(); perm != customFilePerm {
		t.Errorf("file perm = %o, want %o", perm, customFilePerm)
	}
}

func TestCustomStore_NameDefaultsToHost(t *testing.T) {
	s, _ := NewCustomStore(t.TempDir())
	saved, _ := s.Save(CustomProvider{BaseURL: "https://api.example.com/v1"})
	if saved.Name != "api.example.com" {
		t.Errorf("name = %q, want host", saved.Name)
	}
}

func TestCustomStore_CorruptFileIsError(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, customProvidersFile), []byte("{not json"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := NewCustomStore(dir); err == nil {
		t.Error("expected error opening store over corrupt file")
	}
}

func TestSlugify(t *testing.T) {
	cases := map[string]string{
		"DeepSeek":       "deepseek",
		"My  Provider!!": "my-provider",
		"  ":             "",
		"a.b.c":          "a-b-c",
	}
	for in, want := range cases {
		if got := slugify(in); got != want {
			t.Errorf("slugify(%q) = %q, want %q", in, got, want)
		}
	}
}

// Guard against accidental schema drift in the on-disk record.
func TestDiskRecord_HasNoPlaintextKeyField(t *testing.T) {
	b, _ := json.Marshal(diskRecord{APIKeyB64: "x"})
	if strings.Contains(string(b), `"apiKey"`) {
		t.Error("diskRecord must not serialize a plaintext apiKey field")
	}
}
