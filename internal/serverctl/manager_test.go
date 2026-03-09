package serverctl

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestManager_NewManager(t *testing.T) {
	dir := t.TempDir()
	m := NewManager(dir)

	if m.cfg.Port != defaultPort {
		t.Errorf("expected default port %d, got %d", defaultPort, m.cfg.Port)
	}
	if m.cfg.AutoStart != false {
		t.Error("expected AutoStart to be false by default")
	}
	if m.appDataDir != dir {
		t.Errorf("expected appDataDir %q, got %q", dir, m.appDataDir)
	}
	expectedServerDir := filepath.Join(dir, serverSubDir)
	if m.serverDir != expectedServerDir {
		t.Errorf("expected serverDir %q, got %q", expectedServerDir, m.serverDir)
	}
}

func TestManager_Status_NotRunning(t *testing.T) {
	dir := t.TempDir()
	m := NewManager(dir)

	status := m.Status()

	if status.Running {
		t.Error("expected Running to be false for a new manager")
	}
	if status.Port != defaultPort {
		t.Errorf("expected port %d, got %d", defaultPort, status.Port)
	}
	if status.URL != "" {
		t.Errorf("expected empty URL, got %q", status.URL)
	}
	if status.Uptime != 0 {
		t.Errorf("expected uptime 0, got %d", status.Uptime)
	}
}

func TestManager_GetConfig_Default(t *testing.T) {
	dir := t.TempDir()
	m := NewManager(dir)

	cfg := m.GetConfig()

	if cfg.Port != defaultPort {
		t.Errorf("expected port %d, got %d", defaultPort, cfg.Port)
	}
	if cfg.AutoStart != false {
		t.Error("expected AutoStart to be false")
	}
	if cfg.SessionSecret == "" {
		t.Error("expected SessionSecret to be non-empty")
	}
	if len(cfg.SessionSecret) != secretLength {
		t.Errorf("expected SessionSecret length %d, got %d", secretLength, len(cfg.SessionSecret))
	}
}

func TestManager_SaveConfig_Persists(t *testing.T) {
	dir := t.TempDir()
	m := NewManager(dir)

	newCfg := m.GetConfig()
	newCfg.Port = 8080
	if err := m.SaveConfig(newCfg); err != nil {
		t.Fatalf("SaveConfig failed: %v", err)
	}

	// Create a new manager from the same directory to verify persistence.
	m2 := NewManager(dir)
	cfg2 := m2.GetConfig()

	if cfg2.Port != 8080 {
		t.Errorf("expected persisted port 8080, got %d", cfg2.Port)
	}
}

func TestManager_SaveConfig_PreservesToken(t *testing.T) {
	dir := t.TempDir()
	m := NewManager(dir)

	// Manually set AdminToken by writing config directly.
	m.mu.Lock()
	m.cfg.AdminToken = "test-token-abc123"
	_ = m.saveConfig(m.cfg)
	m.mu.Unlock()

	// SaveConfig with a new port; AdminToken should be preserved.
	newCfg := m.GetConfig()
	newCfg.Port = 9999
	newCfg.AdminToken = "" // Caller might clear it; SaveConfig should preserve.
	if err := m.SaveConfig(newCfg); err != nil {
		t.Fatalf("SaveConfig failed: %v", err)
	}

	got := m.GetConfig()
	if got.AdminToken != "test-token-abc123" {
		t.Errorf("expected AdminToken %q, got %q", "test-token-abc123", got.AdminToken)
	}
	if got.Port != 9999 {
		t.Errorf("expected port 9999, got %d", got.Port)
	}
}

func TestManager_GetAdminToken_Empty(t *testing.T) {
	dir := t.TempDir()
	m := NewManager(dir)

	token := m.GetAdminToken()
	if token != "" {
		t.Errorf("expected empty admin token, got %q", token)
	}
}

func TestManager_GetURL_NotRunning(t *testing.T) {
	dir := t.TempDir()
	m := NewManager(dir)

	url := m.GetURL()
	if url != "" {
		t.Errorf("expected empty URL for non-running server, got %q", url)
	}
}

func TestGenerateSecret(t *testing.T) {
	secret := generateSecret()

	if len(secret) != secretLength {
		t.Errorf("expected secret length %d, got %d", secretLength, len(secret))
	}

	for i, c := range secret {
		if !strings.ContainsRune(secretAlphabet, c) {
			t.Errorf("character at index %d (%c) is not in secretAlphabet", i, c)
		}
	}

	// Verify all characters are alphanumeric (a second pass for robustness).
	for _, c := range secret {
		if (c < 'a' || c > 'z') && (c < 'A' || c > 'Z') && (c < '0' || c > '9') {
			t.Errorf("unexpected character %c in secret", c)
		}
	}
}

func TestBinaryName_Platform(t *testing.T) {
	name := binaryName()

	if runtime.GOOS == "windows" {
		if name != binaryNameWindows {
			t.Errorf("on windows expected %q, got %q", binaryNameWindows, name)
		}
	} else {
		if name != binaryNameUnix {
			t.Errorf("on non-windows expected %q, got %q", binaryNameUnix, name)
		}
	}
}

func TestDetectBinary_NotFound(t *testing.T) {
	dir := t.TempDir()

	// Create the server subdirectory but do not place any binary in it.
	serverDir := filepath.Join(dir, serverSubDir)
	if err := os.MkdirAll(serverDir, 0o755); err != nil {
		t.Fatalf("failed to create server dir: %v", err)
	}

	result := detectBinary(dir)
	// detectBinary also checks next to the running executable, so it might
	// find a binary there in some CI environments. We only assert that it
	// does NOT return the path inside our temp dir.
	expected := filepath.Join(serverDir, binaryName())
	if result == expected {
		t.Errorf("detectBinary should not find binary at %q in empty temp dir", expected)
	}
}

func TestDefaultBinaryPath(t *testing.T) {
	dir := t.TempDir()

	got := defaultBinaryPath(dir)
	expected := filepath.Join(dir, serverSubDir, binaryName())

	if got != expected {
		t.Errorf("expected %q, got %q", expected, got)
	}

	// Verify the path ends with the correct binary name.
	base := filepath.Base(got)
	if runtime.GOOS == "windows" {
		if base != binaryNameWindows {
			t.Errorf("expected binary name %q, got %q", binaryNameWindows, base)
		}
	} else {
		if base != binaryNameUnix {
			t.Errorf("expected binary name %q, got %q", binaryNameUnix, base)
		}
	}
}

func TestPlatformSuffix(t *testing.T) {
	suffix := platformSuffix()

	if suffix == "" {
		t.Fatal("platformSuffix returned empty string")
	}

	// Must contain a dash separating OS and architecture.
	if !strings.Contains(suffix, "-") {
		t.Errorf("expected suffix to contain '-', got %q", suffix)
	}

	parts := strings.SplitN(suffix, "-", 2)
	osName := parts[0]
	archPart := parts[1]

	// OS part must match runtime.GOOS.
	if osName != runtime.GOOS {
		t.Errorf("expected OS %q in suffix, got %q", runtime.GOOS, osName)
	}

	// On Windows, suffix should end with ".exe".
	if runtime.GOOS == "windows" {
		if !strings.HasSuffix(suffix, ".exe") {
			t.Errorf("expected .exe suffix on windows, got %q", suffix)
		}
		// archPart should be like "amd64.exe" or "arm64.exe"
		archOnly := strings.TrimSuffix(archPart, ".exe")
		if archOnly != runtime.GOARCH {
			t.Errorf("expected arch %q, got %q", runtime.GOARCH, archOnly)
		}
	} else {
		if strings.HasSuffix(suffix, ".exe") {
			t.Errorf("unexpected .exe suffix on non-windows: %q", suffix)
		}
		if archPart != runtime.GOARCH {
			t.Errorf("expected arch %q, got %q", runtime.GOARCH, archPart)
		}
	}
}

func TestManager_SaveConfig_WritesValidJSON(t *testing.T) {
	dir := t.TempDir()
	m := NewManager(dir)

	cfg := m.GetConfig()
	cfg.Port = 12345
	if err := m.SaveConfig(cfg); err != nil {
		t.Fatalf("SaveConfig failed: %v", err)
	}

	// Read the raw file and verify it is valid JSON.
	configPath := filepath.Join(dir, serverSubDir, configFileName)
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("failed to read config file: %v", err)
	}

	var parsed ServerConfig
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("config file is not valid JSON: %v", err)
	}
	if parsed.Port != 12345 {
		t.Errorf("expected port 12345 in file, got %d", parsed.Port)
	}
}
