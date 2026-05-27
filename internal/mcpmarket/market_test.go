package mcpmarket

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"lurus-switch/internal/mcp"
)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// newMarketWithServer returns a Market wired to a test HTTP server.
func newMarketWithServer(ts *httptest.Server) *Market {
	return &Market{httpClient: ts.Client()}
}

// fakeServer creates a minimal test registry response JSON.
func fakeRegistryResponse(servers []registryServer) []byte {
	resp := registryListResponse{
		Servers: servers,
		Pagination: registryPagination{
			CurrentPage: 1,
			PageSize:    len(servers),
			TotalPages:  1,
			TotalCount:  len(servers),
		},
	}
	b, _ := json.Marshal(resp)
	return b
}

// ---------------------------------------------------------------------------
// Builtin manifest
// ---------------------------------------------------------------------------

func TestLoadBuiltin_ParsesEmbedded(t *testing.T) {
	m := NewMarket()
	servers, err := m.loadBuiltin()
	if err != nil {
		t.Fatalf("loadBuiltin error: %v", err)
	}
	if len(servers) < 10 {
		t.Errorf("expected ≥10 builtin servers, got %d", len(servers))
	}
	// Spot-check mandatory fields.
	for _, s := range servers {
		if s.ID == "" {
			t.Errorf("server missing ID: %+v", s)
		}
		if s.Name == "" {
			t.Errorf("server %s missing Name", s.ID)
		}
		if !s.Builtin {
			t.Errorf("server %s should have Builtin=true", s.ID)
		}
	}
}

func TestListServers_ReturnsBuiltinsWhenNoCacheExists(t *testing.T) {
	// Redirect cache to a temp dir that is empty.
	tmpDir := t.TempDir()
	t.Setenv("APPDATA", tmpDir)

	m := NewMarket()
	servers, err := m.ListServers()
	if err != nil {
		t.Fatalf("ListServers error: %v", err)
	}
	if len(servers) == 0 {
		t.Fatal("expected at least builtin servers")
	}
}

// ---------------------------------------------------------------------------
// Registry client (mock HTTP server)
// ---------------------------------------------------------------------------

func TestRefreshFromRegistry_PopulatesCache(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("APPDATA", tmpDir)

	raw := []registryServer{
		{
			ID:            "test-uuid-1",
			QualifiedName: "@test/server-alpha",
			DisplayName:   "Alpha",
			Description:   "Test alpha server",
			UseCount:      42,
			Verified:      true,
			Homepage:      "https://example.com/alpha",
			CreatedAt:     time.Now(),
		},
	}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(fakeRegistryResponse(raw))
	}))
	defer ts.Close()

	m := newMarketWithServer(ts)
	// Override registryBase is not possible without injection; patch via round-trip URL.
	// We test cache persistence by injecting via the real refresh path with a
	// patched httpClient that rewrites the target host.
	m.httpClient = &http.Client{
		Transport: &rewriteTransport{target: ts.URL},
	}

	ctx := context.Background()
	// Force refresh by clearing the cache file.
	if err := m.RefreshFromRegistry(ctx, ""); err != nil {
		t.Fatalf("RefreshFromRegistry error: %v", err)
	}

	// The cache should now contain the test server.
	servers, _, _, err := loadCache()
	if err != nil {
		t.Fatalf("loadCache error: %v", err)
	}
	if len(servers) == 0 {
		t.Fatal("expected cache to contain servers after refresh")
	}
	found := false
	for _, s := range servers {
		if s.QualifiedName == "@test/server-alpha" {
			found = true
		}
	}
	if !found {
		t.Error("refreshed server not found in cache")
	}
}

// rewriteTransport redirects every request to the given target origin.
type rewriteTransport struct {
	target string
}

func (r *rewriteTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req2 := req.Clone(req.Context())
	req2.URL.Host = strings.TrimPrefix(r.target, "http://")
	req2.URL.Scheme = "http"
	return http.DefaultTransport.RoundTrip(req2)
}

func TestRefreshFromRegistry_NetworkErrorIsSwallowed(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("APPDATA", tmpDir)

	m := &Market{
		httpClient: &http.Client{
			Transport: &failTransport{},
			Timeout:   1 * time.Second,
		},
	}
	ctx := context.Background()
	// Should return nil (not an error) when the network is unreachable.
	if err := m.RefreshFromRegistry(ctx, "query"); err != nil {
		t.Errorf("expected nil on network failure, got: %v", err)
	}
}

type failTransport struct{}

func (ft *failTransport) RoundTrip(*http.Request) (*http.Response, error) {
	return nil, &netError{msg: "connection refused"}
}

type netError struct{ msg string }

func (e *netError) Error() string   { return e.msg }
func (e *netError) Timeout() bool   { return false }
func (e *netError) Temporary() bool { return false }

// ---------------------------------------------------------------------------
// Cross-CLI installation
// ---------------------------------------------------------------------------

func TestInstallToTools_WritesClaudeCodeConfig(t *testing.T) {
	home := t.TempDir()
	// Create the .claude directory.
	claudeDir := filepath.Join(home, ".claude")
	if err := os.MkdirAll(claudeDir, 0755); err != nil {
		t.Fatal(err)
	}

	server := MarketServer{
		ID:            "github",
		QualifiedName: "@modelcontextprotocol/server-github",
		Name:          "GitHub",
		Description:   "GitHub MCP server",
		Category:      "vcs",
		Builtin:       true,
	}
	userConfig := map[string]string{
		"GITHUB_PERSONAL_ACCESS_TOKEN": "ghp_test",
	}

	m := NewMarket()
	// Patch tool config path via env override (USERPROFILE / HOME fallback).
	t.Setenv("USERPROFILE", home)
	t.Setenv("HOME", home)

	report, err := m.InstallToTools(context.Background(), server, userConfig, []TargetTool{ToolClaudeCode})
	if err != nil {
		t.Fatalf("InstallToTools error: %v", err)
	}
	if len(report.Statuses) != 1 {
		t.Fatalf("expected 1 status, got %d", len(report.Statuses))
	}
	st := report.Statuses[0]
	if !st.OK {
		t.Errorf("expected OK=true, got error: %s", st.Error)
	}

	// Verify the settings file was written with mcpServers.
	data, err := os.ReadFile(filepath.Join(claudeDir, "settings.json"))
	if err != nil {
		t.Fatalf("read settings.json: %v", err)
	}
	var root map[string]any
	if err := json.Unmarshal(data, &root); err != nil {
		t.Fatalf("parse settings.json: %v", err)
	}
	mcpServers, ok := root["mcpServers"].(map[string]any)
	if !ok || len(mcpServers) == 0 {
		t.Errorf("mcpServers not found in settings.json: %v", root)
	}
}

func TestInstallToTools_WritesMultipleTools(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)

	// Pre-create all tool config dirs.
	for _, sub := range []string{".claude", ".cursor", ".gemini"} {
		if err := os.MkdirAll(filepath.Join(home, sub), 0755); err != nil {
			t.Fatal(err)
		}
	}

	server := MarketServer{
		ID:            "memory",
		QualifiedName: "@modelcontextprotocol/server-memory",
		Name:          "Memory",
		Category:      "memory",
	}

	m := NewMarket()
	targets := []TargetTool{ToolClaudeCode, ToolCursor, ToolGemini}
	report, err := m.InstallToTools(context.Background(), server, nil, targets)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(report.Statuses) != len(targets) {
		t.Fatalf("expected %d statuses, got %d", len(targets), len(report.Statuses))
	}
	for _, st := range report.Statuses {
		if !st.OK {
			t.Errorf("tool %s failed: %s", st.Tool, st.Error)
		}
	}
}

func TestInstallToTools_EmptyTargetTools_ReturnsError(t *testing.T) {
	m := NewMarket()
	_, err := m.InstallToTools(context.Background(), MarketServer{ID: "x"}, nil, nil)
	if err == nil {
		t.Error("expected error for empty targetTools")
	}
}

// ---------------------------------------------------------------------------
// Preset save
// ---------------------------------------------------------------------------

func TestSaveAsPreset_PersistsPreset(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("APPDATA", tmpDir)

	store, err := mcp.NewStore()
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}

	server := MarketServer{
		ID:            "fetch",
		QualifiedName: "@modelcontextprotocol/server-fetch",
		Name:          "Fetch",
		Description:   "Fetch web content",
		Category:      "web",
	}
	m := NewMarket()
	preset, err := m.SaveAsPreset(store, server, nil)
	if err != nil {
		t.Fatalf("SaveAsPreset error: %v", err)
	}
	if preset.Name != server.Name {
		t.Errorf("preset name: got %q, want %q", preset.Name, server.Name)
	}

	// Verify it was persisted.
	presets, err := store.ListPresets()
	if err != nil {
		t.Fatalf("ListPresets error: %v", err)
	}
	if len(presets) == 0 {
		t.Error("expected at least one preset after save")
	}
}

func TestSaveAsPreset_NilStore_ReturnsError(t *testing.T) {
	m := NewMarket()
	_, err := m.SaveAsPreset(nil, MarketServer{ID: "x"}, nil)
	if err == nil {
		t.Error("expected error for nil store")
	}
}

// ---------------------------------------------------------------------------
// GetServer
// ---------------------------------------------------------------------------

func TestGetServer_FoundByQualifiedName(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("APPDATA", tmpDir)

	m := NewMarket()
	server, err := m.GetServer(context.Background(), "@modelcontextprotocol/server-github")
	if err != nil {
		t.Fatalf("GetServer error: %v", err)
	}
	if server.Name != "GitHub" {
		t.Errorf("unexpected server name: %s", server.Name)
	}
}

func TestGetServer_NotFound_ReturnsError(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("APPDATA", tmpDir)

	m := NewMarket()
	_, err := m.GetServer(context.Background(), "this-does-not-exist")
	if err == nil {
		t.Error("expected error for unknown server")
	}
}

// ---------------------------------------------------------------------------
// Builtin fallback manifest — all required servers present
// ---------------------------------------------------------------------------

func TestBuiltinManifest_ContainsRequiredServers(t *testing.T) {
	required := []string{"github", "filesystem", "postgres", "sqlite", "playwright",
		"fetch", "brave-search", "slack", "linear", "notion"}

	m := NewMarket()
	servers, err := m.loadBuiltin()
	if err != nil {
		t.Fatalf("loadBuiltin: %v", err)
	}
	byID := make(map[string]bool, len(servers))
	for _, s := range servers {
		byID[s.ID] = true
	}
	for _, id := range required {
		if !byID[id] {
			t.Errorf("builtin manifest missing required server: %s", id)
		}
	}
}

// ---------------------------------------------------------------------------
// patchMCPConfig idempotency
// ---------------------------------------------------------------------------

func TestPatchMCPConfig_Idempotent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "settings.json")

	server := mcp.MCPServer{
		Name:    "test-server",
		Type:    "stdio",
		Command: "npx",
		Args:    []string{"-y", "@test/server"},
	}

	// Write once.
	if err := patchMCPConfig(path, server); err != nil {
		t.Fatalf("first patchMCPConfig: %v", err)
	}
	data1, _ := os.ReadFile(path)

	// Write again — result must be equal.
	if err := patchMCPConfig(path, server); err != nil {
		t.Fatalf("second patchMCPConfig: %v", err)
	}
	data2, _ := os.ReadFile(path)

	if string(data1) != string(data2) {
		t.Errorf("patchMCPConfig is not idempotent:\nbefore: %s\nafter:  %s", data1, data2)
	}
}
