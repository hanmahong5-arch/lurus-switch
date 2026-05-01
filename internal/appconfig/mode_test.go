package appconfig

import "testing"

func TestAppMode_Valid(t *testing.T) {
	cases := []struct {
		mode AppMode
		want bool
	}{
		{ModePersonal, true},
		{ModeReseller, true},
		{ModeEndUser, true},
		{ModeUnset, false},
		{AppMode("user"), false},     // legacy raw, must go through migrateLegacyMode first
		{AppMode("promoter"), false}, // legacy raw
		{AppMode("admin"), false},
		{AppMode(""), false},
	}
	for _, c := range cases {
		if got := c.mode.Valid(); got != c.want {
			t.Errorf("AppMode(%q).Valid() = %v, want %v", c.mode, got, c.want)
		}
	}
}

func TestMigrateLegacyMode(t *testing.T) {
	cases := []struct {
		in   string
		want AppMode
	}{
		{"user", ModePersonal},
		{"USER", ModePersonal},
		{" user ", ModePersonal},
		{"promoter", ModeReseller},
		{"PROMOTER", ModeReseller},
		{"personal", ModePersonal},
		{"reseller", ModeReseller},
		{"enduser", ModeEndUser},
		{"", ModeUnset},
		{"junk", AppMode("junk")},
	}
	for _, c := range cases {
		if got := migrateLegacyMode(c.in); got != c.want {
			t.Errorf("migrateLegacyMode(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestNormalizeMode(t *testing.T) {
	cases := []struct {
		in     string
		want   AppMode
		wantOK bool
	}{
		{"personal", ModePersonal, true},
		{"user", ModePersonal, true},        // legacy migration succeeds
		{"promoter", ModeReseller, true},    // legacy migration succeeds
		{"reseller", ModeReseller, true},
		{"enduser", ModeEndUser, true},
		{"", ModeUnset, true},               // empty is valid (=unset, prompt user)
		{"garbage", ModePersonal, false},    // invalid coerced to personal w/ ok=false
	}
	for _, c := range cases {
		got, ok := normalizeMode(c.in)
		if got != c.want || ok != c.wantOK {
			t.Errorf("normalizeMode(%q) = (%q, %v), want (%q, %v)", c.in, got, ok, c.want, c.wantOK)
		}
	}
}

func TestCanTransition(t *testing.T) {
	cases := []struct {
		name    string
		current AppMode
		next    AppMode
		locked  bool
		wantErr bool
	}{
		{"personal->reseller unlocked", ModePersonal, ModeReseller, false, false},
		{"reseller->enduser unlocked", ModeReseller, ModeEndUser, false, false},
		{"enduser->personal locked blocks", ModeEndUser, ModePersonal, true, true},
		{"enduser->enduser locked allowed (no-op)", ModeEndUser, ModeEndUser, true, false},
		{"invalid target rejected", ModePersonal, AppMode("admin"), false, true},
	}
	for _, c := range cases {
		err := CanTransition(c.current, c.next, c.locked)
		if (err != nil) != c.wantErr {
			t.Errorf("%s: CanTransition err=%v wantErr=%v", c.name, err, c.wantErr)
		}
	}
}

func TestSaveAppSettings_LegacyAppModeMigration(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("APPDATA", tmp)
	t.Setenv("HOME", tmp)
	t.Setenv("XDG_CONFIG_HOME", tmp)

	s := DefaultAppSettings()
	s.AppMode = "user" // legacy v0.1.0 value
	if err := SaveAppSettings(s); err != nil {
		t.Fatalf("save legacy: %v", err)
	}
	if s.AppMode != string(ModePersonal) {
		t.Errorf("expected legacy 'user' to migrate to %q, got %q", ModePersonal, s.AppMode)
	}

	loaded, err := LoadAppSettings()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if loaded.AppMode != string(ModePersonal) {
		t.Errorf("after load, AppMode = %q, want %q", loaded.AppMode, ModePersonal)
	}
}

func TestSaveAppSettings_LockEnforcement(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("APPDATA", tmp)
	t.Setenv("HOME", tmp)
	t.Setenv("XDG_CONFIG_HOME", tmp)

	// Step 1: write a locked EndUser config (simulating white-label package).
	locked := DefaultAppSettings()
	locked.AppMode = string(ModeEndUser)
	locked.LockedHubURL = "https://acme.example/"
	if err := SaveAppSettings(locked); err != nil {
		t.Fatalf("save locked: %v", err)
	}

	// Step 2: attempt to switch to personal — must fail.
	attempt := DefaultAppSettings()
	attempt.AppMode = string(ModePersonal)
	if err := SaveAppSettings(attempt); err == nil {
		t.Error("expected lock to block transition to personal, got nil error")
	}

	// Step 3: blanking LockedHubURL while staying in EndUser must be ignored —
	// the FS-side URL is preserved.
	attempt2 := DefaultAppSettings()
	attempt2.AppMode = string(ModeEndUser)
	attempt2.LockedHubURL = ""
	if err := SaveAppSettings(attempt2); err != nil {
		t.Fatalf("same-mode save: %v", err)
	}
	loaded, _ := LoadAppSettings()
	if loaded.LockedHubURL != "https://acme.example/" {
		t.Errorf("LockedHubURL was wiped: got %q", loaded.LockedHubURL)
	}
}

func TestSaveAppSettings_RejectsInvalidMode(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("APPDATA", tmp)
	t.Setenv("HOME", tmp)
	t.Setenv("XDG_CONFIG_HOME", tmp)

	s := DefaultAppSettings()
	s.AppMode = "rogue-value"
	if err := SaveAppSettings(s); err == nil {
		t.Error("expected error for unrecognized mode, got nil")
	}
}

func TestIsModeLocked(t *testing.T) {
	cases := []struct {
		name string
		s    *AppSettings
		want bool
	}{
		{"nil settings", nil, false},
		{"unset mode", &AppSettings{AppMode: string(ModeUnset)}, false},
		{"personal", &AppSettings{AppMode: string(ModePersonal)}, false},
		{"enduser without url", &AppSettings{AppMode: string(ModeEndUser)}, false},
		{"enduser with whitespace url", &AppSettings{AppMode: string(ModeEndUser), LockedHubURL: "   "}, false},
		{"enduser locked", &AppSettings{AppMode: string(ModeEndUser), LockedHubURL: "https://acme.example/"}, true},
	}
	for _, c := range cases {
		if got := IsModeLocked(c.s); got != c.want {
			t.Errorf("%s: IsModeLocked = %v, want %v", c.name, got, c.want)
		}
	}
}
