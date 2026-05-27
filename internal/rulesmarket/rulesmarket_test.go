package rulesmarket

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// marketEnv pins APPDATA / HOME so cache files land in a per-test tempdir.
func marketEnv(t *testing.T) {
	t.Helper()
	tmp := t.TempDir()
	t.Setenv("APPDATA", tmp)
	t.Setenv("HOME", tmp)
	t.Setenv("XDG_CONFIG_HOME", tmp)
}

// ---------------------------------------------------------------------------
// Builtin manifest
// ---------------------------------------------------------------------------

func TestLoadBuiltin_ReturnsAtLeastTenTemplates(t *testing.T) {
	m := NewMarket()
	templates, err := m.loadBuiltin()
	if err != nil {
		t.Fatalf("loadBuiltin: %v", err)
	}
	if len(templates) < 10 {
		t.Errorf("want >= 10 builtin templates, got %d", len(templates))
	}
	// Validate required fields on every template
	for _, tmpl := range templates {
		if tmpl.ID == "" || tmpl.Name == "" || tmpl.Content == "" {
			t.Errorf("template missing required field: %+v", tmpl)
		}
	}
}

func TestLoadBuiltin_NoIDCollisions(t *testing.T) {
	m := NewMarket()
	templates, _ := m.loadBuiltin()
	seen := make(map[string]bool)
	for _, tmpl := range templates {
		if seen[tmpl.ID] {
			t.Errorf("duplicate template ID: %q", tmpl.ID)
		}
		seen[tmpl.ID] = true
	}
}

func TestListTemplates_BuiltinAlwaysPresent(t *testing.T) {
	marketEnv(t)
	m := NewMarket()
	templates, err := m.ListTemplates()
	if err != nil {
		t.Fatalf("ListTemplates: %v", err)
	}
	if len(templates) < 10 {
		t.Errorf("want >= 10 templates, got %d", len(templates))
	}
}

// ---------------------------------------------------------------------------
// Cache round-trip
// ---------------------------------------------------------------------------

func TestCacheRoundTrip_SaveAndLoad(t *testing.T) {
	marketEnv(t)
	want := []RuleTemplate{
		{ID: "test-1", Name: "Test One", Category: "language", Framework: "Go",
			Content: "# Go rules", Format: FormatAgentsMD},
		{ID: "test-2", Name: "Test Two", Category: "framework", Framework: "React",
			Content: "# React rules", Format: FormatCursorRules},
	}
	if err := saveCache(want); err != nil {
		t.Fatalf("saveCache: %v", err)
	}
	got, fetchedAt, err := loadCache()
	if err != nil {
		t.Fatalf("loadCache: %v", err)
	}
	if len(got) != len(want) {
		t.Fatalf("cache len = %d, want %d", len(got), len(want))
	}
	if fetchedAt.IsZero() {
		t.Error("FetchedAt must be set after saveCache")
	}
	ids := make(map[string]bool, len(got))
	for _, tmpl := range got {
		ids[tmpl.ID] = true
	}
	for _, w := range want {
		if !ids[w.ID] {
			t.Errorf("ID %q missing from loaded cache", w.ID)
		}
	}
}

func TestLoadCache_EmptyOnMissingFile(t *testing.T) {
	marketEnv(t)
	templates, fetchedAt, err := loadCache()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(templates) != 0 {
		t.Errorf("expected empty slice, got %d templates", len(templates))
	}
	if !fetchedAt.IsZero() {
		t.Error("FetchedAt must be zero when no cache file exists")
	}
}

// ---------------------------------------------------------------------------
// Format conversion — three directions
// ---------------------------------------------------------------------------

func TestConvert_CursorRulesToAgentsMD(t *testing.T) {
	input := "## Rules\n\nBe helpful."
	got, err := Convert(input, FormatCursorRules, FormatAgentsMD)
	if err != nil {
		t.Fatalf("Convert: %v", err)
	}
	if !strings.HasPrefix(got, "# Project Rules") {
		t.Errorf("expected '# Project Rules' heading, got: %s", got[:min(40, len(got))])
	}
	if !strings.Contains(got, "Be helpful.") {
		t.Error("body content must be preserved")
	}
}

func TestConvert_AgentsMDToClaudeMD(t *testing.T) {
	input := "# Project Rules\n\nUse context."
	got, err := Convert(input, FormatAgentsMD, FormatClaudeMD)
	if err != nil {
		t.Fatalf("Convert: %v", err)
	}
	if !strings.HasPrefix(got, "# CLAUDE.md") {
		t.Errorf("expected '# CLAUDE.md' heading, got: %s", got[:min(40, len(got))])
	}
	if !strings.Contains(got, "Use context.") {
		t.Error("body content must be preserved")
	}
	// Old heading must not appear
	if strings.Contains(got, "# Project Rules") {
		t.Error("old '# Project Rules' heading must be stripped")
	}
}

func TestConvert_ClaudeMDToCursorRules(t *testing.T) {
	input := "# CLAUDE.md\n\nAvoid globals."
	got, err := Convert(input, FormatClaudeMD, FormatCursorRules)
	if err != nil {
		t.Fatalf("Convert: %v", err)
	}
	// cursorrules does not mandate a heading — CLAUDE.md heading must be removed
	if strings.HasPrefix(got, "# CLAUDE.md") {
		t.Errorf("CLAUDE.md heading should be stripped for cursorrules output: %s", got[:min(40, len(got))])
	}
	if !strings.Contains(got, "Avoid globals.") {
		t.Error("body content must be preserved")
	}
}

func TestConvert_SameFormat_IsNoop(t *testing.T) {
	input := "# CLAUDE.md\n\nSome rules."
	got, err := Convert(input, FormatClaudeMD, FormatClaudeMD)
	if err != nil {
		t.Fatalf("Convert same-format: %v", err)
	}
	if got != input {
		t.Errorf("same-format Convert must return input unchanged; got %q", got)
	}
}

func TestConvert_EmptyContent_Errors(t *testing.T) {
	_, err := Convert("", FormatCursorRules, FormatAgentsMD)
	if err == nil {
		t.Error("expected error for empty content")
	}
}

// ---------------------------------------------------------------------------
// WriteRuleToProject — file creation and append / overwrite semantics
// ---------------------------------------------------------------------------

func TestWriteRuleToProject_CreatesNewFile(t *testing.T) {
	marketEnv(t)
	dir := t.TempDir()
	m := NewMarket()
	tmpl := RuleTemplate{
		ID: "go-rules", Name: "Go Rules", Format: FormatCursorRules,
		Content: "## Basics\n\nAlways handle errors.",
	}
	result, err := m.WriteRuleToProject(context.Background(), dir, tmpl, FormatAgentsMD, false)
	if err != nil {
		t.Fatalf("WriteRuleToProject: %v", err)
	}
	if result.Appended || result.Skipped {
		t.Errorf("new file should not be appended/skipped: %+v", result)
	}
	data, _ := os.ReadFile(filepath.Join(dir, "AGENTS.md"))
	if !strings.Contains(string(data), "# Project Rules") {
		t.Errorf("expected AGENTS.md heading in file; got: %s", data)
	}
}

func TestWriteRuleToProject_AppendWhenFileExists(t *testing.T) {
	marketEnv(t)
	dir := t.TempDir()
	existing := filepath.Join(dir, "AGENTS.md")
	if err := os.WriteFile(existing, []byte("# Project Rules\n\nExisting rule.\n"), 0644); err != nil {
		t.Fatal(err)
	}
	m := NewMarket()
	tmpl := RuleTemplate{
		ID: "new-rule", Name: "New Rule", Format: FormatCursorRules,
		Content: "New rule content.",
	}
	result, err := m.WriteRuleToProject(context.Background(), dir, tmpl, FormatAgentsMD, false)
	if err != nil {
		t.Fatalf("WriteRuleToProject: %v", err)
	}
	if !result.Appended {
		t.Error("expected Appended = true when file already exists")
	}
	data, _ := os.ReadFile(existing)
	if !strings.Contains(string(data), "Existing rule.") {
		t.Error("existing content must be preserved")
	}
	if !strings.Contains(string(data), "New rule content.") {
		t.Error("new content must be appended")
	}
}

func TestWriteRuleToProject_OverwriteWhenRequested(t *testing.T) {
	marketEnv(t)
	dir := t.TempDir()
	existing := filepath.Join(dir, "AGENTS.md")
	if err := os.WriteFile(existing, []byte("Old content.\n"), 0644); err != nil {
		t.Fatal(err)
	}
	m := NewMarket()
	tmpl := RuleTemplate{
		ID: "replace-rule", Name: "Replace", Format: FormatAgentsMD,
		Content: "# Project Rules\n\nNew only.",
	}
	result, err := m.WriteRuleToProject(context.Background(), dir, tmpl, FormatAgentsMD, true)
	if err != nil {
		t.Fatalf("WriteRuleToProject: %v", err)
	}
	if result.Appended || result.Skipped {
		t.Errorf("overwrite should not set Appended/Skipped: %+v", result)
	}
	data, _ := os.ReadFile(existing)
	if strings.Contains(string(data), "Old content.") {
		t.Error("old content must be replaced on overwrite")
	}
}

func TestWriteRuleToProject_IdempotentWhenContentIdentical(t *testing.T) {
	marketEnv(t)
	dir := t.TempDir()
	content := "# Project Rules\n\nSame content.\n"
	existing := filepath.Join(dir, "AGENTS.md")
	if err := os.WriteFile(existing, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	m := NewMarket()
	tmpl := RuleTemplate{
		ID: "idem", Name: "Idem", Format: FormatAgentsMD,
		Content: "# Project Rules\n\nSame content.",
	}
	result, err := m.WriteRuleToProject(context.Background(), dir, tmpl, FormatAgentsMD, false)
	if err != nil {
		t.Fatalf("WriteRuleToProject: %v", err)
	}
	if !result.Skipped {
		t.Error("expected Skipped = true when content is already present")
	}
}

func TestWriteRuleToProject_EmptyProjectDir_Errors(t *testing.T) {
	m := NewMarket()
	tmpl := RuleTemplate{ID: "x", Content: "content"}
	_, err := m.WriteRuleToProject(context.Background(), "", tmpl, FormatAgentsMD, false)
	if err == nil {
		t.Error("expected error for empty projectDir")
	}
}

// ---------------------------------------------------------------------------
// Remote refresh (mocked HTTP server)
// ---------------------------------------------------------------------------

func TestRefreshFromRemote_ParsesManifestAndCaches(t *testing.T) {
	marketEnv(t)
	remote := []RuleTemplate{
		{ID: "remote-1", Name: "Remote One", Category: "language", Framework: "Kotlin",
			Content: "## Kotlin rules\n\nUse coroutines.", Format: FormatCursorRules},
	}
	data, _ := json.Marshal(remote)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(data)
	}))
	defer srv.Close()

	m := NewMarket()
	if err := m.RefreshFromRemote(context.Background(), srv.URL); err != nil {
		t.Fatalf("RefreshFromRemote: %v", err)
	}

	cached, _, _ := loadCache()
	found := false
	for _, c := range cached {
		if c.ID == "remote-1" {
			found = true
		}
	}
	if !found {
		t.Error("remote template should be present in cache after refresh")
	}
}

func TestRefreshFromRemote_EmptyURL_IsNoop(t *testing.T) {
	marketEnv(t)
	m := NewMarket()
	if err := m.RefreshFromRemote(context.Background(), ""); err != nil {
		t.Errorf("empty URL must not return error: %v", err)
	}
}

// ---------------------------------------------------------------------------
// TargetFileName helper
// ---------------------------------------------------------------------------

func TestTargetFileName_AllFormats(t *testing.T) {
	cases := []struct{ format Format; want string }{
		{FormatAgentsMD, "AGENTS.md"},
		{FormatClaudeMD, "CLAUDE.md"},
		{FormatCursorRules, ".cursorrules"},
		{FormatWindsurf, ".windsurfrules"},
	}
	for _, tc := range cases {
		got := TargetFileName(tc.format)
		if got != tc.want {
			t.Errorf("TargetFileName(%q) = %q, want %q", tc.format, got, tc.want)
		}
	}
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
