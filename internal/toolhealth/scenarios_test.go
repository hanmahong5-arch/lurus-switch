package toolhealth

import (
	"sync"
	"testing"
)

// ============================================================
// Helpers
// ============================================================

// newResult returns a fresh HealthResult in green state for a given tool.
func newResult(tool string) *HealthResult {
	return &HealthResult{Tool: tool, Status: StatusGreen, Issues: []string{}}
}

// ============================================================
// Scenario: Claude config — all reachable states
// ============================================================

// TestScenario_ClaudeConfig_AllHealthStates exercises every reachable health
// state for the Claude tool config in a single logical sequence — simulating a
// user who goes through setup: no key → proxy only → key only → both set.
func TestScenario_ClaudeConfig_AllHealthStates(t *testing.T) {
	cases := []struct {
		desc       string
		content    string
		wantStatus HealthStatus
	}{
		{
			desc:       "invalid JSON → red",
			content:    `{not json}`,
			wantStatus: StatusRed,
		},
		{
			desc:       "both key and baseURL empty → yellow",
			content:    `{"env":{"ANTHROPIC_API_KEY":"","ANTHROPIC_BASE_URL":""}}`,
			wantStatus: StatusYellow,
		},
		{
			desc:       "env section missing entirely → yellow",
			content:    `{"mcpServers":{}}`,
			wantStatus: StatusYellow,
		},
		{
			desc:       "only API key set → green",
			content:    `{"env":{"ANTHROPIC_API_KEY":"sk-test","ANTHROPIC_BASE_URL":""}}`,
			wantStatus: StatusGreen,
		},
		{
			desc:       "only BASE_URL set (proxy auth) → green",
			content:    `{"env":{"ANTHROPIC_API_KEY":"","ANTHROPIC_BASE_URL":"https://proxy.example.com"}}`,
			wantStatus: StatusGreen,
		},
		{
			desc:       "both set → green",
			content:    `{"env":{"ANTHROPIC_API_KEY":"sk-test","ANTHROPIC_BASE_URL":"https://proxy.example.com"}}`,
			wantStatus: StatusGreen,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			r := newResult("claude")
			checkClaudeHealth(tc.content, r)
			if r.Status != tc.wantStatus {
				t.Errorf("got %s, want %s (issues: %v)", r.Status, tc.wantStatus, r.Issues)
			}
		})
	}
}

// ============================================================
// Scenario: Gemini config — all reachable states
// ============================================================

func TestScenario_GeminiConfig_AllHealthStates(t *testing.T) {
	cases := []struct {
		desc       string
		content    string
		wantStatus HealthStatus
	}{
		{"invalid JSON → red", `{bad json`, StatusRed},
		{"model section missing → yellow", `{"theme":"dark"}`, StatusYellow},
		{"model object present but name empty → yellow", `{"model":{}}`, StatusYellow},
		{"model.name set → green", `{"model":{"name":"gemini-2.5-flash"}}`, StatusGreen},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			r := newResult("gemini")
			checkGeminiHealth(tc.content, r)
			if r.Status != tc.wantStatus {
				t.Errorf("got %s, want %s (issues: %v)", r.Status, tc.wantStatus, r.Issues)
			}
		})
	}
}

// ============================================================
// Scenario: Codex config — all reachable states
// ============================================================

func TestScenario_CodexConfig_AllHealthStates(t *testing.T) {
	cases := []struct {
		desc       string
		content    string
		wantStatus HealthStatus
	}{
		{"invalid TOML → red", `}{not toml`, StatusRed},
		{"model key missing → yellow", `approval_policy = "on-failure"`, StatusYellow},
		{"model empty string → yellow", `model = ""` + "\n", StatusYellow},
		{"model set → green", `model = "gpt-4o"` + "\n", StatusGreen},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			r := newResult("codex")
			checkCodexHealth(tc.content, r)
			if r.Status != tc.wantStatus {
				t.Errorf("got %s, want %s (issues: %v)", r.Status, tc.wantStatus, r.Issues)
			}
		})
	}
}

// ============================================================
// Scenario: Claw config — all reachable states
// ============================================================

func TestScenario_ClawConfig_AllHealthStates(t *testing.T) {
	cases := []struct {
		desc       string
		content    string
		wantStatus HealthStatus
	}{
		{"invalid JSON → red", `{{{`, StatusRed},
		{"model_list missing → yellow", `{"other":"value"}`, StatusYellow},
		{"model_list empty array → yellow", `{"model_list":[]}`, StatusYellow},
		{"api_base empty string → yellow", `{"model_list":[{"name":"x","api_base":"","api_key":"k","model_name":"m"}]}`, StatusYellow},
		{"api_base only whitespace → yellow", `{"model_list":[{"name":"x","api_base":"   ","api_key":"k","model_name":"m"}]}`, StatusYellow},
		{"api_base set → green", `{"model_list":[{"name":"x","api_base":"https://api.test","api_key":"k","model_name":"m"}]}`, StatusGreen},
	}

	for _, tc := range cases {
		for _, tool := range []string{"picoclaw", "nullclaw"} {
			tool := tool // capture
			tc := tc    // capture
			t.Run(tool+"/"+tc.desc, func(t *testing.T) {
				r := newResult(tool)
				checkClawHealth(tc.content, r)
				if r.Status != tc.wantStatus {
					t.Errorf("got %s, want %s (issues: %v)", r.Status, tc.wantStatus, r.Issues)
				}
			})
		}
	}
}

// ============================================================
// Scenario: User transitions config from broken to working
// ============================================================

// TestScenario_UserFixesBrokenConfig_StatusTransitionsToGreen simulates the
// sequence: config is broken (red) → user edits and fixes it → green.
// This is the "undo a mistake" flow.
func TestScenario_UserFixesBrokenConfig_StatusTransitionsToGreen(t *testing.T) {
	// Step 1: broken config — user saved invalid JSON
	r1 := newResult("claude")
	checkClaudeHealth(`{broken`, r1)
	if r1.Status != StatusRed {
		t.Fatalf("expected red for broken config, got %s", r1.Status)
	}

	// Step 2: user edits config via the app — missing API key (incomplete fix)
	r2 := newResult("claude")
	checkClaudeHealth(`{"env":{"ANTHROPIC_API_KEY":"","ANTHROPIC_BASE_URL":""}}`, r2)
	if r2.Status != StatusYellow {
		t.Fatalf("expected yellow for empty keys, got %s", r2.Status)
	}

	// Step 3: user enters API key — full fix
	r3 := newResult("claude")
	checkClaudeHealth(`{"env":{"ANTHROPIC_API_KEY":"sk-real-key","ANTHROPIC_BASE_URL":""}}`, r3)
	if r3.Status != StatusGreen {
		t.Fatalf("expected green after fix, got %s (issues: %v)", r3.Status, r3.Issues)
	}
}

// ============================================================
// Scenario: CheckAll is idempotent — calling twice returns same results
// ============================================================

// TestScenario_CheckAll_Idempotent verifies that calling CheckAll multiple
// times in sequence (e.g., user refreshes the dashboard repeatedly) returns
// consistent results. The health checks must not have side effects.
func TestScenario_CheckAll_Idempotent(t *testing.T) {
	// Without real tool configs on disk, all tools return red (file not found).
	// We just verify that two calls return the same status set.
	results1 := CheckAll()
	results2 := CheckAll()

	for tool, r1 := range results1 {
		r2, ok := results2[tool]
		if !ok {
			t.Errorf("second CheckAll missing tool %q", tool)
			continue
		}
		if r1.Status != r2.Status {
			t.Errorf("tool %q: first status %s, second %s (not idempotent)", tool, r1.Status, r2.Status)
		}
	}
}

// ============================================================
// Scenario: CheckAll covers exactly the supported tools
// ============================================================

// TestScenario_CheckAll_CoversSupportedToolsExactly verifies that CheckAll
// returns one result per supported tool — no more, no fewer. This catches regressions
// if a tool is added to supportedTools without corresponding health check logic.
func TestScenario_CheckAll_CoversSupportedToolsExactly(t *testing.T) {
	results := CheckAll()

	if len(results) != len(supportedTools) {
		t.Errorf("CheckAll returned %d results, want %d", len(results), len(supportedTools))
	}

	for _, tool := range supportedTools {
		r, ok := results[tool]
		if !ok {
			t.Errorf("CheckAll missing result for tool %q", tool)
			continue
		}
		if r == nil {
			t.Errorf("CheckAll result for %q is nil", tool)
			continue
		}
		if r.Tool != tool {
			t.Errorf("result.Tool = %q, want %q", r.Tool, tool)
		}
		if r.Issues == nil {
			t.Errorf("result.Issues for %q must be non-nil slice", tool)
		}
	}
}

// ============================================================
// Scenario: Concurrent health checks (dashboard refreshes rapidly)
// ============================================================

// TestScenario_ConcurrentCheckAll_NoPanic simulates the dashboard calling
// CheckAll simultaneously from multiple goroutines (e.g., auto-refresh while
// user manually clicks refresh). Must not panic or race.
func TestScenario_ConcurrentCheckAll_NoPanic(t *testing.T) {
	var wg sync.WaitGroup
	const goroutines = 5

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			results := CheckAll()
			_ = results
		}()
	}

	wg.Wait()
}

// ============================================================
// Scenario: Issues slice is always initialized (never nil)
// ============================================================

// TestScenario_HealthResult_IssuesAlwaysInitialized verifies that the Issues
// field is never nil regardless of tool status. A nil slice causes a JSON
// marshal difference (null vs []) that breaks the frontend.
func TestScenario_HealthResult_IssuesAlwaysInitialized(t *testing.T) {
	// checkClaudeHealth on green path — must not set Issues to nil
	r := newResult("claude")
	checkClaudeHealth(`{"env":{"ANTHROPIC_API_KEY":"sk-test"}}`, r)
	if r.Issues == nil {
		t.Error("Issues should be non-nil even when green")
	}

	// CheckTool on missing config — must have non-nil Issues
	result := CheckTool("claude")
	if result.Issues == nil {
		t.Error("CheckTool Issues should be non-nil (config not found case)")
	}
}

// ============================================================
// Scenario: Unknown tool name — graceful handling
// ============================================================

// TestScenario_UnknownToolName_NoPanic verifies that CheckTool with an
// unexpected tool name (e.g., typo from frontend) does not panic.
// It should attempt a config read and return some result.
func TestScenario_UnknownToolName_NoPanic(t *testing.T) {
	result := CheckTool("nonexistent-tool-xyz")
	if result == nil {
		t.Fatal("CheckTool with unknown name returned nil")
	}
	// Must have a tool name set
	if result.Tool != "nonexistent-tool-xyz" {
		t.Errorf("result.Tool = %q, want nonexistent-tool-xyz", result.Tool)
	}
}

// ============================================================
// Scenario: PicoClaw and NullClaw use identical validation logic
// ============================================================

// TestScenario_PicoClawNullClaw_SameContentSameStatus verifies that the two
// "Claw" tools produce identical health status for identical config content.
// This is important for UI consistency when showing tool health side by side.
func TestScenario_PicoClawNullClaw_SameContentSameStatus(t *testing.T) {
	contents := []string{
		`{{{`,
		`{"other":"value"}`,
		`{"model_list":[]}`,
		`{"model_list":[{"name":"x","api_base":"","api_key":"k","model_name":"m"}]}`,
		`{"model_list":[{"name":"x","api_base":"https://api.test","api_key":"k","model_name":"m"}]}`,
	}

	for _, content := range contents {
		rPico := newResult("picoclaw")
		rNull := newResult("nullclaw")
		checkPicoClawHealth(content, rPico)
		checkNullClawHealth(content, rNull)

		if rPico.Status != rNull.Status {
			t.Errorf("content=%q: picoclaw=%s, nullclaw=%s (should be equal)",
				content, rPico.Status, rNull.Status)
		}
	}
}
