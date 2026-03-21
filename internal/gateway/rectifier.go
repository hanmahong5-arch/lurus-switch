package gateway

// Thinking Budget Rectifier — auto-fixes Anthropic API errors related to
// thinking budget constraints. When the upstream returns an error mentioning
// "budget_tokens" + "thinking", this module rewrites the request body to
// comply with the provider's budget limits and retries.
//
// Inspired by CC-Switch's thinking_budget_rectifier.rs.

import (
	"encoding/json"
	"strings"
)

const (
	maxThinkingBudget = 32000
	maxTokensValue    = 64000
	minMaxTokens      = maxThinkingBudget + 1
)

// RectifyResult describes what the rectifier changed, if anything.
type RectifyResult struct {
	Applied          bool   `json:"applied"`
	OrigMaxTokens    int64  `json:"origMaxTokens,omitempty"`
	NewMaxTokens     int64  `json:"newMaxTokens,omitempty"`
	OrigBudgetTokens int64  `json:"origBudgetTokens,omitempty"`
	NewBudgetTokens  int64  `json:"newBudgetTokens,omitempty"`
	Reason           string `json:"reason,omitempty"`
}

// ShouldRectifyThinkingBudget checks whether an error response indicates
// a thinking budget constraint violation that we can auto-fix.
func ShouldRectifyThinkingBudget(errorMsg string) bool {
	lower := strings.ToLower(errorMsg)
	return strings.Contains(lower, "budget_tokens") &&
		(strings.Contains(lower, "thinking") || strings.Contains(lower, "max_tokens"))
}

// RectifyThinkingBudget rewrites a request body to fix budget constraint issues.
// Returns the modified body and a result describing the changes.
func RectifyThinkingBudget(body []byte) ([]byte, RectifyResult) {
	var req map[string]any
	if err := json.Unmarshal(body, &req); err != nil {
		return body, RectifyResult{}
	}

	result := RectifyResult{}
	changed := false

	// Fix max_tokens
	if mt, ok := req["max_tokens"]; ok {
		if mtf, ok := mt.(float64); ok {
			result.OrigMaxTokens = int64(mtf)
			if mtf > float64(maxTokensValue) {
				req["max_tokens"] = maxTokensValue
				result.NewMaxTokens = maxTokensValue
				changed = true
			}
		}
	}

	// Fix thinking.budget_tokens
	if thinking, ok := req["thinking"]; ok {
		if tm, ok := thinking.(map[string]any); ok {
			if bt, ok := tm["budget_tokens"]; ok {
				if btf, ok := bt.(float64); ok {
					result.OrigBudgetTokens = int64(btf)
					if btf > float64(maxThinkingBudget) {
						tm["budget_tokens"] = maxThinkingBudget
						result.NewBudgetTokens = maxThinkingBudget
						changed = true
					}
				}
			}

			// Ensure max_tokens > budget_tokens
			if _, ok := req["max_tokens"]; !ok {
				req["max_tokens"] = minMaxTokens
				result.NewMaxTokens = minMaxTokens
				changed = true
			} else if mtf, ok := req["max_tokens"].(float64); ok && mtf <= float64(maxThinkingBudget) {
				req["max_tokens"] = minMaxTokens
				result.NewMaxTokens = minMaxTokens
				changed = true
			} else if mti, ok := req["max_tokens"].(int); ok && int64(mti) <= maxThinkingBudget {
				req["max_tokens"] = minMaxTokens
				result.NewMaxTokens = minMaxTokens
				changed = true
			}
		}
	}

	if !changed {
		return body, RectifyResult{}
	}

	result.Applied = true
	result.Reason = "auto-fixed thinking budget constraints"
	out, err := json.Marshal(req)
	if err != nil {
		return body, RectifyResult{}
	}
	return out, result
}
