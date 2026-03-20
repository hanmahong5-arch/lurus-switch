package appreg

import (
	"os"
	"testing"
)

func TestRegistry_RegisterAndLookup(t *testing.T) {
	dir := t.TempDir()
	reg, err := NewRegistry(dir)
	if err != nil {
		t.Fatalf("NewRegistry: %v", err)
	}

	// Should have builtins.
	apps := reg.List()
	if len(apps) == 0 {
		t.Fatal("expected builtin apps, got 0")
	}

	// Register a user app.
	app, err := reg.Register("My Script", "script", "A Python analysis script")
	if err != nil {
		t.Fatalf("Register: %v", err)
	}
	if app.Token == "" {
		t.Fatal("expected non-empty token")
	}
	if app.Kind != KindUser {
		t.Fatalf("expected KindUser, got %v", app.Kind)
	}

	// Lookup by token.
	foundID := reg.LookupByToken(app.Token)
	if foundID != app.ID {
		t.Fatalf("LookupByToken: expected %q, got %q", app.ID, foundID)
	}

	// Invalid token returns empty.
	if id := reg.LookupByToken("sk-switch-invalid"); id != "" {
		t.Fatalf("expected empty for invalid token, got %q", id)
	}
}

func TestRegistry_ResetToken(t *testing.T) {
	dir := t.TempDir()
	reg, err := NewRegistry(dir)
	if err != nil {
		t.Fatalf("NewRegistry: %v", err)
	}

	app, err := reg.Register("Test App", "", "")
	if err != nil {
		t.Fatalf("Register: %v", err)
	}
	oldToken := app.Token

	newToken, err := reg.ResetToken(app.ID)
	if err != nil {
		t.Fatalf("ResetToken: %v", err)
	}
	if newToken == oldToken {
		t.Fatal("expected new token to differ from old")
	}

	// Old token should no longer work.
	if id := reg.LookupByToken(oldToken); id != "" {
		t.Fatal("old token should be invalidated")
	}
	// New token should work.
	if id := reg.LookupByToken(newToken); id != app.ID {
		t.Fatalf("new token lookup: expected %q, got %q", app.ID, id)
	}
}

func TestRegistry_DeleteUserApp(t *testing.T) {
	dir := t.TempDir()
	reg, err := NewRegistry(dir)
	if err != nil {
		t.Fatalf("NewRegistry: %v", err)
	}

	app, err := reg.Register("Temp App", "", "")
	if err != nil {
		t.Fatalf("Register: %v", err)
	}

	if err := reg.Delete(app.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	if a := reg.Get(app.ID); a != nil {
		t.Fatal("expected app to be deleted")
	}
}

func TestRegistry_CannotDeleteBuiltin(t *testing.T) {
	dir := t.TempDir()
	reg, err := NewRegistry(dir)
	if err != nil {
		t.Fatalf("NewRegistry: %v", err)
	}

	err = reg.Delete("claude")
	if err == nil {
		t.Fatal("expected error deleting builtin app")
	}
}

func TestRegistry_Persistence(t *testing.T) {
	dir := t.TempDir()

	// Create registry and register an app.
	reg1, err := NewRegistry(dir)
	if err != nil {
		t.Fatalf("NewRegistry: %v", err)
	}
	app, err := reg1.Register("Persisted App", "", "")
	if err != nil {
		t.Fatalf("Register: %v", err)
	}

	// Create a new registry from the same dir — should load persisted data.
	reg2, err := NewRegistry(dir)
	if err != nil {
		t.Fatalf("NewRegistry (reload): %v", err)
	}

	found := reg2.Get(app.ID)
	if found == nil {
		t.Fatal("expected persisted app to be found")
	}
	if found.Name != "Persisted App" {
		t.Fatalf("expected name %q, got %q", "Persisted App", found.Name)
	}
	if id := reg2.LookupByToken(app.Token); id != app.ID {
		t.Fatalf("token lookup after reload: expected %q, got %q", app.ID, id)
	}
}

func TestRegistry_BuiltinCount(t *testing.T) {
	dir := t.TempDir()
	reg, err := NewRegistry(dir)
	if err != nil {
		t.Fatalf("NewRegistry: %v", err)
	}

	expected := len(builtinTools())
	apps := reg.List()
	builtinCount := 0
	for _, a := range apps {
		if a.Kind == KindBuiltin {
			builtinCount++
		}
	}
	if builtinCount != expected {
		t.Fatalf("expected %d builtin apps, got %d", expected, builtinCount)
	}
}

func TestRegistry_RegisterEmptyNameFails(t *testing.T) {
	dir := t.TempDir()
	reg, _ := NewRegistry(dir)
	_, err := reg.Register("", "", "")
	if err == nil {
		t.Fatal("expected error for empty name")
	}
}

func TestGenerateToken(t *testing.T) {
	token, err := generateToken()
	if err != nil {
		t.Fatalf("generateToken: %v", err)
	}
	if len(token) != len(tokenPrefix)+tokenRandomBytes*2 {
		t.Fatalf("unexpected token length: %d", len(token))
	}
	if token[:len(tokenPrefix)] != tokenPrefix {
		t.Fatalf("token missing prefix: %s", token)
	}

	// Tokens should be unique.
	token2, _ := generateToken()
	if token == token2 {
		t.Fatal("tokens should be unique")
	}

	// Clean up any test artifacts.
	os.RemoveAll(t.TempDir())
}
