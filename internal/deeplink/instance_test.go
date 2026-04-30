package deeplink_test

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"runtime"
	"sync"
	"testing"
	"time"

	"lurus-switch/internal/deeplink"
)

// buildURL creates a valid switch:// URL for the given type.
func buildURL(typ string) string {
	obj := map[string]string{"test": "value"}
	b, _ := json.Marshal(obj)
	enc := base64.RawURLEncoding.EncodeToString(b)
	return "switch://import?type=" + typ + "&data=" + enc
}

// TestServer_SingleInstance_RoundTrip verifies that:
//  1. The first NewServer call succeeds.
//  2. A second NewServer call returns ErrAlreadyRunning.
//  3. SendToExisting delivers the URL to the first server.
//  4. The onPayload callback receives the correct Payload.
func TestServer_SingleInstance_RoundTrip(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("named pipe test requires an interactive Windows session; run manually")
	}

	dataDir := t.TempDir()

	// Start primary instance.
	srv, err := deeplink.NewServer(dataDir)
	if err != nil {
		t.Fatalf("NewServer (first): %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var mu sync.Mutex
	var received []*deeplink.Payload

	srv.Start(ctx, func(p *deeplink.Payload) {
		mu.Lock()
		received = append(received, p)
		mu.Unlock()
	})

	// Try to start a second instance — must fail.
	_, err2 := deeplink.NewServer(dataDir)
	if err2 != deeplink.ErrAlreadyRunning {
		t.Fatalf("NewServer (second) = %v, want ErrAlreadyRunning", err2)
	}

	// Send a URL from the "second instance" to the primary.
	rawURL := buildURL("provider")
	if err := deeplink.SendToExisting(dataDir, rawURL); err != nil {
		t.Fatalf("SendToExisting: %v", err)
	}

	// Allow the goroutine to deliver the payload.
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		mu.Lock()
		n := len(received)
		mu.Unlock()
		if n > 0 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	mu.Lock()
	defer mu.Unlock()

	if len(received) == 0 {
		t.Fatal("no payload received within timeout")
	}
	p := received[0]
	if p.Type != "provider" {
		t.Errorf("Type = %q, want %q", p.Type, "provider")
	}
	if p.Raw != rawURL {
		t.Errorf("Raw = %q, want %q", p.Raw, rawURL)
	}

	// Clean up.
	if err := srv.Stop(); err != nil {
		t.Logf("srv.Stop: %v", err)
	}
}

// TestServer_Stop_ReleasesLock verifies that stopping a server allows a new
// server to bind the same channel.
func TestServer_Stop_ReleasesLock(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("named pipe test requires an interactive Windows session; run manually")
	}

	dataDir := t.TempDir()

	srv1, err := deeplink.NewServer(dataDir)
	if err != nil {
		t.Fatalf("NewServer (first): %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	srv1.Start(ctx, func(*deeplink.Payload) {})
	cancel()
	if err := srv1.Stop(); err != nil {
		t.Logf("srv1.Stop: %v", err)
	}

	// Give the OS a moment to release the file lock.
	time.Sleep(20 * time.Millisecond)

	srv2, err := deeplink.NewServer(dataDir)
	if err != nil {
		t.Fatalf("NewServer after stop: %v", err)
	}
	srv2.Stop() //nolint:errcheck
}

// TestRegister_Windows verifies that Register writes the registry on Windows.
// Skipped on non-Windows platforms.
func TestRegister_Windows(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("registry test only runs on Windows")
	}

	// Use a dummy exe path; we are only testing registry write, not launch.
	exePath := `C:\Program Files\lurus-switch\lurus-switch.exe`
	if err := deeplink.Register(exePath); err != nil {
		t.Fatalf("Register: %v", err)
	}

	// Verify by reading back the command value.
	if err := deeplink.VerifyRegisterWritten(exePath); err != nil {
		t.Errorf("VerifyRegisterWritten: %v", err)
	}

	// Clean up.
	if err := deeplink.Unregister(); err != nil {
		t.Logf("Unregister: %v", err)
	}
}
