package hotkey

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	xhotkey "golang.design/x/hotkey"
)

// ---- parseShortcut tests ----

func TestParseShortcut_ValidLetterWithModifiers(t *testing.T) {
	cases := []struct {
		input    string
		wantKey  xhotkey.Key
		wantMods int // expected modifier count
	}{
		{"Ctrl+Shift+S", xhotkey.KeyS, 2},
		{"ctrl+shift+s", xhotkey.KeyS, 2},
		{"  Ctrl + Shift + S  ", xhotkey.KeyS, 2},
		{"Ctrl+A", xhotkey.KeyA, 1},
		{"Shift+Z", xhotkey.KeyZ, 1},
		{"Alt+F4", xhotkey.KeyF4, 1},
		{"Ctrl+Alt+F12", xhotkey.KeyF12, 2},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.input, func(t *testing.T) {
			p, err := parseShortcut(tc.input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if p.key != tc.wantKey {
				t.Errorf("key: got 0x%x, want 0x%x", p.key, tc.wantKey)
			}
			if len(p.mods) != tc.wantMods {
				t.Errorf("mods count: got %d, want %d (mods=%v)", len(p.mods), tc.wantMods, p.mods)
			}
		})
	}
}

func TestParseShortcut_CmdModifier(t *testing.T) {
	// "Cmd+Shift+S" should always parse to 2 modifiers (the Cmd equivalent + Shift).
	p, err := parseShortcut("Cmd+Shift+S")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.key != xhotkey.KeyS {
		t.Errorf("key: got 0x%x, want KeyS", p.key)
	}
	if len(p.mods) != 2 {
		t.Errorf("expected 2 mods, got %d: %v", len(p.mods), p.mods)
	}
}

func TestParseShortcut_CommandOrControl(t *testing.T) {
	p, err := parseShortcut("CommandOrControl+Q")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.key != xhotkey.KeyQ {
		t.Errorf("key: got 0x%x, want KeyQ", p.key)
	}
	if len(p.mods) != 1 {
		t.Fatalf("expected 1 mod, got %d: %v", len(p.mods), p.mods)
	}
	// The returned modifier is platform-specific; just verify we got exactly one.
	// Platform-specific assertions are handled by platform_*_test.go files.
}

func TestParseShortcut_Digits(t *testing.T) {
	p, err := parseShortcut("Ctrl+0")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.key != xhotkey.Key0 {
		t.Errorf("key: got 0x%x, want Key0", p.key)
	}
}

func TestParseShortcut_FKeys(t *testing.T) {
	for _, tc := range []struct {
		input   string
		wantKey xhotkey.Key
	}{
		{"Ctrl+F1", xhotkey.KeyF1},
		{"Ctrl+F12", xhotkey.KeyF12},
	} {
		p, err := parseShortcut(tc.input)
		if err != nil {
			t.Fatalf("%s: unexpected error: %v", tc.input, err)
		}
		if p.key != tc.wantKey {
			t.Errorf("%s: key mismatch: got 0x%x want 0x%x", tc.input, p.key, tc.wantKey)
		}
	}
}

func TestParseShortcut_NamedKeys(t *testing.T) {
	cases := []struct {
		input   string
		wantKey xhotkey.Key
	}{
		{"Ctrl+Space", xhotkey.KeySpace},
		{"Ctrl+Enter", xhotkey.KeyReturn},
		{"Ctrl+Return", xhotkey.KeyReturn},
		{"Ctrl+Tab", xhotkey.KeyTab},
		{"Ctrl+Esc", xhotkey.KeyEscape},
		{"Ctrl+Escape", xhotkey.KeyEscape},
		{"Ctrl+Up", xhotkey.KeyUp},
		{"Ctrl+Down", xhotkey.KeyDown},
		{"Ctrl+Left", xhotkey.KeyLeft},
		{"Ctrl+Right", xhotkey.KeyRight},
		{"Ctrl+Delete", xhotkey.KeyDelete},
		{"Ctrl+Del", xhotkey.KeyDelete},
	}
	for _, tc := range cases {
		p, err := parseShortcut(tc.input)
		if err != nil {
			t.Fatalf("%s: unexpected error: %v", tc.input, err)
		}
		if p.key != tc.wantKey {
			t.Errorf("%s: got key 0x%x want 0x%x", tc.input, p.key, tc.wantKey)
		}
	}
}

func TestParseShortcut_CaseAndSpaceTolerance(t *testing.T) {
	cases := []string{
		"ctrl+shift+s",
		"CTRL+SHIFT+S",
		"Ctrl+Shift+S",
		"  ctrl  +  shift  +  s  ",
	}
	for _, c := range cases {
		p, err := parseShortcut(c)
		if err != nil {
			t.Errorf("%q: unexpected error: %v", c, err)
			continue
		}
		if p.key != xhotkey.KeyS {
			t.Errorf("%q: wrong key: got 0x%x", c, p.key)
		}
		if len(p.mods) != 2 {
			t.Errorf("%q: wrong mod count: %d", c, len(p.mods))
		}
	}
}

func TestParseShortcut_ModifierAliases(t *testing.T) {
	aliases := []string{"Ctrl+S", "Control+S", "ctrl+s", "control+s"}
	for _, a := range aliases {
		p, err := parseShortcut(a)
		if err != nil {
			t.Errorf("%q: unexpected error: %v", a, err)
			continue
		}
		if p.key != xhotkey.KeyS {
			t.Errorf("%q: wrong key", a)
		}
		if len(p.mods) != 1 {
			t.Errorf("%q: expected 1 mod, got %d", a, len(p.mods))
		}
		if p.mods[0] != xhotkey.ModCtrl {
			t.Errorf("%q: expected ModCtrl", a)
		}
	}
}

// ---- error cases ----

func TestParseShortcut_ErrDisabled(t *testing.T) {
	cases := []string{"", "   "}
	for _, c := range cases {
		_, err := parseShortcut(c)
		if !errors.Is(err, ErrDisabled) {
			t.Errorf("%q: want ErrDisabled, got %v", c, err)
		}
	}
}

func TestParseShortcut_ErrInvalidKey(t *testing.T) {
	cases := []string{
		"InvalidKey",
		"Ctrl+InvalidKey",
		"Bogus+S",
		"Ctrl+Shift+A+B", // two non-modifier keys
	}
	for _, c := range cases {
		_, err := parseShortcut(c)
		if !errors.Is(err, ErrInvalidKey) {
			t.Errorf("%q: want ErrInvalidKey, got %v", c, err)
		}
	}
}

func TestParseShortcut_ModifierOnlyIsInvalid(t *testing.T) {
	_, err := parseShortcut("Ctrl+Shift")
	if !errors.Is(err, ErrInvalidKey) {
		t.Errorf("modifier-only shortcut: want ErrInvalidKey, got %v", err)
	}
}

// ---- config (load/save) tests ----

func TestDefaultBindings(t *testing.T) {
	b := DefaultBindings()
	if _, ok := b["quickSwitch"]; !ok {
		t.Error("DefaultBindings missing 'quickSwitch'")
	}
	if _, ok := b["showWindow"]; !ok {
		t.Error("DefaultBindings missing 'showWindow'")
	}
}

func TestLoadBindings_MissingFile_ReturnsDefaults(t *testing.T) {
	dir := t.TempDir()
	b, err := loadBindings(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defaults := DefaultBindings()
	for k, v := range defaults {
		if b[k] != v {
			t.Errorf("key %q: got %q, want %q", k, b[k], v)
		}
	}
}

func TestLoadSaveBindings_Roundtrip(t *testing.T) {
	dir := t.TempDir()
	want := Bindings{
		"quickSwitch": "Ctrl+Alt+Q",
		"showWindow":  "",
	}
	if err := saveBindings(dir, want); err != nil {
		t.Fatalf("save: %v", err)
	}
	got, err := loadBindings(dir)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("key %q: got %q, want %q", k, got[k], v)
		}
	}
}

func TestLoadBindings_CorruptFile_FallsBackToDefaults(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "hotkey.json")
	if err := os.WriteFile(p, []byte("not valid json }{"), 0644); err != nil {
		t.Fatal(err)
	}
	b, err := loadBindings(dir)
	// err is non-nil but bindings should be defaults (not nil/empty).
	if err == nil {
		t.Error("expected non-nil error on corrupt file")
	}
	if len(b) == 0 {
		t.Error("expected default bindings on corrupt file, got empty")
	}
}

func TestLoadBindings_BackfillsMissingKeys(t *testing.T) {
	dir := t.TempDir()
	// Persist a bindings file that only has quickSwitch (missing showWindow).
	partial := Bindings{"quickSwitch": "Ctrl+Shift+S"}
	if err := saveBindings(dir, partial); err != nil {
		t.Fatal(err)
	}
	b, err := loadBindings(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := b["showWindow"]; !ok {
		t.Error("loadBindings should back-fill missing 'showWindow' from defaults")
	}
}

func TestSaveBindings_CreatesDirectory(t *testing.T) {
	base := t.TempDir()
	dir := filepath.Join(base, "nested", "config")
	if err := saveBindings(dir, DefaultBindings()); err != nil {
		t.Fatalf("saveBindings should create dir hierarchy: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "hotkey.json")); err != nil {
		t.Errorf("hotkey.json not found after save: %v", err)
	}
}

// ---- Manager nil-safety tests ----

func TestManager_NilReceiver_NoOp(t *testing.T) {
	var m *Manager
	// None of these should panic.
	errs := m.Start(context.Background())
	if errs != nil {
		t.Error("nil Manager.Start should return nil")
	}
	m.Stop()
	b := m.GetBindings()
	if len(b) == 0 {
		t.Error("nil Manager.GetBindings should return defaults")
	}
	_ = m.UpdateBinding("quickSwitch", "Ctrl+Shift+S")
}

// ---- Manager UpdateBinding persistence test (no actual OS registration) ----

func TestManager_UpdateBinding_PersistsToFile(t *testing.T) {
	dir := t.TempDir()
	// Create manager but do NOT call Start (avoids real OS hotkey registration).
	m := New(dir, nil)
	// Manually set up bindings without registering with the OS.
	m.bindings = DefaultBindings()

	newShortcut := "Ctrl+Alt+P"
	if err := m.UpdateBinding("quickSwitch", newShortcut); err != nil {
		// On Windows in a test environment the registration may fail; that's OK —
		// we only care that the file was persisted.
		// If err is a RegistrationError, the file should still have been written.
		var regErr RegistrationError
		if !errors.As(err, &regErr) {
			t.Fatalf("unexpected non-registration error: %v", err)
		}
	}

	// Verify file on disk.
	b, err := loadBindings(dir)
	if err != nil {
		t.Fatalf("loadBindings: %v", err)
	}
	if b["quickSwitch"] != newShortcut {
		t.Errorf("persisted shortcut: got %q, want %q", b["quickSwitch"], newShortcut)
	}
}

func TestManager_UpdateBinding_DisableBinding(t *testing.T) {
	dir := t.TempDir()
	m := New(dir, nil)
	m.bindings = DefaultBindings()

	if err := m.UpdateBinding("showWindow", ""); err != nil {
		t.Fatalf("disabling binding should not error: %v", err)
	}
	b, err := loadBindings(dir)
	if err != nil {
		t.Fatalf("loadBindings: %v", err)
	}
	if b["showWindow"] != "" {
		t.Errorf("disabled binding should persist as empty string, got %q", b["showWindow"])
	}
}
