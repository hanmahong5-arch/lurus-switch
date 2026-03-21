package gateway

import (
	"encoding/json"
	"testing"
)

func TestShouldRectifyThinkingBudget(t *testing.T) {
	tests := []struct {
		msg  string
		want bool
	}{
		{"budget_tokens must be less than max_tokens when thinking is enabled", true},
		{"thinking.budget_tokens exceeds maximum", true},
		{"BUDGET_TOKENS invalid for THINKING mode with max_tokens", true},
		{"invalid api key", false},
		{"rate limited", false},
		{"", false},
	}
	for _, tc := range tests {
		got := ShouldRectifyThinkingBudget(tc.msg)
		if got != tc.want {
			t.Errorf("ShouldRectify(%q) = %v, want %v", tc.msg, got, tc.want)
		}
	}
}

func TestRectifyThinkingBudget_FixesOverBudget(t *testing.T) {
	body := []byte(`{
		"model": "claude-opus-4-20250514",
		"max_tokens": 100000,
		"thinking": {"type": "enabled", "budget_tokens": 50000},
		"messages": [{"role": "user", "content": "hello"}]
	}`)

	out, result := RectifyThinkingBudget(body)
	if !result.Applied {
		t.Fatal("expected rectification to be applied")
	}
	if result.OrigMaxTokens != 100000 {
		t.Errorf("origMaxTokens = %d, want 100000", result.OrigMaxTokens)
	}
	if result.OrigBudgetTokens != 50000 {
		t.Errorf("origBudgetTokens = %d, want 50000", result.OrigBudgetTokens)
	}

	var req map[string]any
	if err := json.Unmarshal(out, &req); err != nil {
		t.Fatal(err)
	}
	if mt := int64(req["max_tokens"].(float64)); mt != maxTokensValue {
		t.Errorf("max_tokens = %d, want %d", mt, maxTokensValue)
	}
	thinking := req["thinking"].(map[string]any)
	if bt := int64(thinking["budget_tokens"].(float64)); bt != maxThinkingBudget {
		t.Errorf("budget_tokens = %d, want %d", bt, maxThinkingBudget)
	}
}

func TestRectifyThinkingBudget_NoChangeNeeded(t *testing.T) {
	body := []byte(`{
		"model": "gpt-4.1",
		"max_tokens": 4096,
		"messages": [{"role": "user", "content": "hello"}]
	}`)

	_, result := RectifyThinkingBudget(body)
	if result.Applied {
		t.Error("expected no rectification for non-thinking request")
	}
}

func TestRectifyThinkingBudget_AddsMaxTokens(t *testing.T) {
	body := []byte(`{
		"model": "claude-opus-4-20250514",
		"thinking": {"type": "enabled", "budget_tokens": 10000},
		"messages": [{"role": "user", "content": "hello"}]
	}`)

	out, result := RectifyThinkingBudget(body)
	if !result.Applied {
		t.Fatal("expected rectification when max_tokens missing with thinking")
	}

	var req map[string]any
	if err := json.Unmarshal(out, &req); err != nil {
		t.Fatal(err)
	}
	mt := int64(req["max_tokens"].(float64))
	if mt != minMaxTokens {
		t.Errorf("max_tokens = %d, want %d", mt, minMaxTokens)
	}
}
