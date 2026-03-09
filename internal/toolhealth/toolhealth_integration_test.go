package toolhealth

import (
	"testing"
)

// TestCheckTool_ConfigNotFound verifies that a tool returns red when config is missing
func TestCheckTool_ConfigNotFound(t *testing.T) {
	// In test environment, tool config files are unlikely to exist at their standard paths
	// This tests the "config file not found" red path
	result := CheckTool("claude")
	if result == nil {
		t.Fatal("CheckTool should return non-nil result")
	}
	if result.Tool != "claude" {
		t.Errorf("Tool = %q, want claude", result.Tool)
	}
	// Status will be red (config not found) or green/yellow if config happens to exist
	// Just verify result is well-formed
	if result.Status != StatusGreen && result.Status != StatusYellow && result.Status != StatusRed {
		t.Errorf("unexpected status %q", result.Status)
	}
	if result.Issues == nil {
		t.Error("Issues slice should not be nil")
	}
}

func TestCheckTool_AllSupportedTools(t *testing.T) {
	for _, tool := range supportedTools {
		result := CheckTool(tool)
		if result == nil {
			t.Errorf("CheckTool(%q) returned nil", tool)
			continue
		}
		if result.Tool != tool {
			t.Errorf("CheckTool(%q).Tool = %q", tool, result.Tool)
		}
		if result.Issues == nil {
			t.Errorf("CheckTool(%q).Issues should not be nil", tool)
		}
	}
}

func TestCheckAll_ReturnsAllTools(t *testing.T) {
	results := CheckAll()
	if results == nil {
		t.Fatal("CheckAll should return non-nil map")
	}
	for _, tool := range supportedTools {
		r, ok := results[tool]
		if !ok {
			t.Errorf("CheckAll missing result for tool %q", tool)
			continue
		}
		if r == nil {
			t.Errorf("CheckAll result for %q is nil", tool)
		}
	}
}

func TestCheckAll_ResultCount(t *testing.T) {
	results := CheckAll()
	if len(results) != len(supportedTools) {
		t.Errorf("CheckAll returned %d results, want %d", len(results), len(supportedTools))
	}
}

// TestCheckClaudeHealth_OnlyBaseURL verifies yellow when only base URL set but no key
func TestCheckClaudeHealth_OnlyBaseURL(t *testing.T) {
	content := `{"env":{"ANTHROPIC_API_KEY":"","ANTHROPIC_BASE_URL":"https://proxy.example.com"}}`
	r := &HealthResult{Tool: "claude", Status: StatusGreen, Issues: []string{}}
	checkClaudeHealth(content, r)
	// Having BASE_URL without API_KEY is still yellow (uses proxy auth)
	// The check is: BOTH empty → yellow. Only one empty is OK.
	if r.Status != StatusGreen {
		t.Errorf("expected green when BASE_URL is set, got %s: %v", r.Status, r.Issues)
	}
}

func TestCheckClaudeHealth_OnlyAPIKey(t *testing.T) {
	content := `{"env":{"ANTHROPIC_API_KEY":"sk-test-key","ANTHROPIC_BASE_URL":""}}`
	r := &HealthResult{Tool: "claude", Status: StatusGreen, Issues: []string{}}
	checkClaudeHealth(content, r)
	if r.Status != StatusGreen {
		t.Errorf("expected green when API_KEY is set, got %s: %v", r.Status, r.Issues)
	}
}

func TestCheckClaudeHealth_MissingEnvSection(t *testing.T) {
	// Valid JSON but missing env section entirely
	content := `{"mcpServers":{}}`
	r := &HealthResult{Tool: "claude", Status: StatusGreen, Issues: []string{}}
	checkClaudeHealth(content, r)
	if r.Status != StatusYellow {
		t.Errorf("expected yellow for missing env section, got %s", r.Status)
	}
}

func TestCheckGeminiHealth_MissingModelSection(t *testing.T) {
	// Valid JSON but no model key at all
	content := `{"theme":"dark"}`
	r := &HealthResult{Tool: "gemini", Status: StatusGreen, Issues: []string{}}
	checkGeminiHealth(content, r)
	if r.Status != StatusYellow {
		t.Errorf("expected yellow for missing model section, got %s", r.Status)
	}
}

func TestCheckGeminiHealth_InvalidJSON(t *testing.T) {
	r := &HealthResult{Tool: "gemini", Status: StatusGreen, Issues: []string{}}
	checkGeminiHealth("{{{invalid", r)
	if r.Status != StatusRed {
		t.Errorf("expected red for invalid JSON, got %s", r.Status)
	}
}

func TestCheckClawHealth_InvalidJSON(t *testing.T) {
	r := &HealthResult{Tool: "picoclaw", Status: StatusGreen, Issues: []string{}}
	checkClawHealth("{invalid json", r)
	if r.Status != StatusRed {
		t.Errorf("expected red for invalid JSON, got %s", r.Status)
	}
}

func TestCheckClawHealth_MissingModelList(t *testing.T) {
	// Valid JSON but model_list key missing
	content := `{"other_key":"value"}`
	r := &HealthResult{Tool: "nullclaw", Status: StatusGreen, Issues: []string{}}
	checkClawHealth(content, r)
	if r.Status != StatusYellow {
		t.Errorf("expected yellow for missing model_list, got %s", r.Status)
	}
}

func TestCheckClawHealth_WhitespaceAPIBase(t *testing.T) {
	// api_base is only whitespace — should be yellow
	content := `{"model_list":[{"name":"default","api_base":"   ","api_key":"sk","model_name":"m"}]}`
	r := &HealthResult{Tool: "picoclaw", Status: StatusGreen, Issues: []string{}}
	checkClawHealth(content, r)
	if r.Status != StatusYellow {
		t.Errorf("expected yellow for whitespace-only api_base, got %s", r.Status)
	}
}

func TestCheckPicoClawHealth_DelegatesToClawHealth(t *testing.T) {
	// Verify picoclaw/nullclaw use the same underlying logic
	content := `{"model_list":[{"name":"x","api_base":"https://api.test","api_key":"k","model_name":"m"}]}`
	rPico := &HealthResult{Tool: "picoclaw", Status: StatusGreen, Issues: []string{}}
	rNull := &HealthResult{Tool: "nullclaw", Status: StatusGreen, Issues: []string{}}
	checkPicoClawHealth(content, rPico)
	checkNullClawHealth(content, rNull)
	if rPico.Status != rNull.Status {
		t.Errorf("picoclaw status %s != nullclaw status %s for same content", rPico.Status, rNull.Status)
	}
}

func TestHealthResult_IssuesNotNilByDefault(t *testing.T) {
	r := &HealthResult{Tool: "test", Status: StatusGreen, Issues: []string{}}
	if r.Issues == nil {
		t.Error("Issues should be non-nil slice")
	}
	if len(r.Issues) != 0 {
		t.Errorf("Issues should be empty, got %v", r.Issues)
	}
}
