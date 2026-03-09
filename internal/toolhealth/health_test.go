package toolhealth

import (
	"testing"
)

func TestCheckClaudeHealth_Green(t *testing.T) {
	content := `{"env":{"ANTHROPIC_API_KEY":"sk-test","ANTHROPIC_BASE_URL":"https://api.example.com"}}`
	r := &HealthResult{Tool: "claude", Status: StatusGreen, Issues: []string{}}
	checkClaudeHealth(content, r)
	if r.Status != StatusGreen {
		t.Errorf("expected green, got %s: %v", r.Status, r.Issues)
	}
}

func TestCheckClaudeHealth_Yellow(t *testing.T) {
	content := `{"env":{"ANTHROPIC_API_KEY":"","ANTHROPIC_BASE_URL":""}}`
	r := &HealthResult{Tool: "claude", Status: StatusGreen, Issues: []string{}}
	checkClaudeHealth(content, r)
	if r.Status != StatusYellow {
		t.Errorf("expected yellow, got %s", r.Status)
	}
}

func TestCheckClaudeHealth_InvalidJSON(t *testing.T) {
	r := &HealthResult{Tool: "claude", Status: StatusGreen, Issues: []string{}}
	checkClaudeHealth("{invalid", r)
	if r.Status != StatusRed {
		t.Errorf("expected red for invalid JSON, got %s", r.Status)
	}
}

func TestCheckCodexHealth_Green(t *testing.T) {
	content := `model = "gpt-4o"
approval_policy = "on-failure"
`
	r := &HealthResult{Tool: "codex", Status: StatusGreen, Issues: []string{}}
	checkCodexHealth(content, r)
	if r.Status != StatusGreen {
		t.Errorf("expected green, got %s: %v", r.Status, r.Issues)
	}
}

func TestCheckCodexHealth_Yellow(t *testing.T) {
	content := `model = ""
approval_policy = "on-failure"
`
	r := &HealthResult{Tool: "codex", Status: StatusGreen, Issues: []string{}}
	checkCodexHealth(content, r)
	if r.Status != StatusYellow {
		t.Errorf("expected yellow for empty model, got %s", r.Status)
	}
}

func TestCheckCodexHealth_InvalidTOML(t *testing.T) {
	r := &HealthResult{Tool: "codex", Status: StatusGreen, Issues: []string{}}
	checkCodexHealth("not valid toml }{", r)
	if r.Status != StatusRed {
		t.Errorf("expected red for invalid TOML, got %s", r.Status)
	}
}

func TestCheckGeminiHealth_Green(t *testing.T) {
	content := `{"model":{"name":"gemini-2.5-flash"}}`
	r := &HealthResult{Tool: "gemini", Status: StatusGreen, Issues: []string{}}
	checkGeminiHealth(content, r)
	if r.Status != StatusGreen {
		t.Errorf("expected green, got %s: %v", r.Status, r.Issues)
	}
}

func TestCheckGeminiHealth_Yellow(t *testing.T) {
	content := `{"model":{}}`
	r := &HealthResult{Tool: "gemini", Status: StatusGreen, Issues: []string{}}
	checkGeminiHealth(content, r)
	if r.Status != StatusYellow {
		t.Errorf("expected yellow for missing model.name, got %s", r.Status)
	}
}

func TestCheckClawHealth_Green(t *testing.T) {
	content := `{"model_list":[{"name":"default","api_base":"https://api.example.com","api_key":"sk-test","model_name":"test-model"}]}`
	r := &HealthResult{Tool: "picoclaw", Status: StatusGreen, Issues: []string{}}
	checkClawHealth(content, r)
	if r.Status != StatusGreen {
		t.Errorf("expected green, got %s: %v", r.Status, r.Issues)
	}
}

func TestCheckClawHealth_Yellow(t *testing.T) {
	content := `{"model_list":[{"name":"default","api_base":"","api_key":"","model_name":"test-model"}]}`
	r := &HealthResult{Tool: "picoclaw", Status: StatusGreen, Issues: []string{}}
	checkClawHealth(content, r)
	if r.Status != StatusYellow {
		t.Errorf("expected yellow for empty api_base, got %s", r.Status)
	}
}

func TestCheckClawHealth_EmptyList(t *testing.T) {
	content := `{"model_list":[]}`
	r := &HealthResult{Tool: "nullclaw", Status: StatusGreen, Issues: []string{}}
	checkClawHealth(content, r)
	if r.Status != StatusYellow {
		t.Errorf("expected yellow for empty model_list, got %s", r.Status)
	}
}
