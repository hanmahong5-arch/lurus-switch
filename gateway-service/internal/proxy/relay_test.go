package proxy

import (
	"testing"

	"github.com/pocketzworld/lurus-common/models"
)

func TestPlatformConstants(t *testing.T) {
	if PlatformClaude != "claude" {
		t.Errorf("PlatformClaude should be 'claude', got '%s'", PlatformClaude)
	}
	if PlatformCodex != "codex" {
		t.Errorf("PlatformCodex should be 'codex', got '%s'", PlatformCodex)
	}
	if PlatformGemini != "gemini" {
		t.Errorf("PlatformGemini should be 'gemini', got '%s'", PlatformGemini)
	}
}

func TestRelayService_IsModelSupported(t *testing.T) {
	rs := &RelayService{}

	tests := []struct {
		name     string
		provider *models.Provider
		model    string
		expected bool
	}{
		{
			name:     "No models specified - all supported",
			provider: &models.Provider{SupportedModels: nil},
			model:    "any-model",
			expected: true,
		},
		{
			name:     "Empty models map - all supported",
			provider: &models.Provider{SupportedModels: map[string]bool{}},
			model:    "any-model",
			expected: true,
		},
		{
			name: "Exact match - supported",
			provider: &models.Provider{
				SupportedModels: map[string]bool{
					"claude-3-opus": true,
				},
			},
			model:    "claude-3-opus",
			expected: true,
		},
		{
			name: "Exact match - not supported",
			provider: &models.Provider{
				SupportedModels: map[string]bool{
					"claude-3-opus": true,
				},
			},
			model:    "claude-3-sonnet",
			expected: false,
		},
		{
			name: "Wildcard match - supported",
			provider: &models.Provider{
				SupportedModels: map[string]bool{
					"claude-*": true,
				},
			},
			model:    "claude-3-opus",
			expected: true,
		},
		{
			name: "Wildcard no match",
			provider: &models.Provider{
				SupportedModels: map[string]bool{
					"claude-*": true,
				},
			},
			model:    "gpt-4",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := rs.isModelSupported(tt.provider, tt.model)
			if result != tt.expected {
				t.Errorf("isModelSupported(%s) = %v, want %v", tt.model, result, tt.expected)
			}
		})
	}
}

func TestRelayService_GetEffectiveModel(t *testing.T) {
	rs := &RelayService{}

	tests := []struct {
		name     string
		provider *models.Provider
		model    string
		expected string
	}{
		{
			name:     "No mapping - return original",
			provider: &models.Provider{ModelMapping: nil},
			model:    "claude-3-opus",
			expected: "claude-3-opus",
		},
		{
			name:     "Empty mapping - return original",
			provider: &models.Provider{ModelMapping: map[string]string{}},
			model:    "claude-3-opus",
			expected: "claude-3-opus",
		},
		{
			name: "Exact mapping",
			provider: &models.Provider{
				ModelMapping: map[string]string{
					"gpt-4": "openai/gpt-4",
				},
			},
			model:    "gpt-4",
			expected: "openai/gpt-4",
		},
		{
			name: "Wildcard mapping with wildcard replacement",
			provider: &models.Provider{
				ModelMapping: map[string]string{
					"claude-*": "anthropic/claude-*",
				},
			},
			model:    "claude-3-opus",
			expected: "anthropic/claude-3-opus",
		},
		{
			name: "Wildcard mapping with fixed replacement",
			provider: &models.Provider{
				ModelMapping: map[string]string{
					"claude-*": "anthropic/claude",
				},
			},
			model:    "claude-3-opus",
			expected: "anthropic/claude",
		},
		{
			name: "No matching mapping - return original",
			provider: &models.Provider{
				ModelMapping: map[string]string{
					"gpt-*": "openai/gpt-*",
				},
			},
			model:    "claude-3-opus",
			expected: "claude-3-opus",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := rs.getEffectiveModel(tt.provider, tt.model)
			if result != tt.expected {
				t.Errorf("getEffectiveModel(%s) = %s, want %s", tt.model, result, tt.expected)
			}
		})
	}
}

func TestRelayService_FilterProviders(t *testing.T) {
	rs := &RelayService{}

	providers := []*models.Provider{
		{Name: "P1", Enabled: true, APIURL: "http://a", APIKey: "k1", SupportedModels: map[string]bool{"claude-*": true}},
		{Name: "P2", Enabled: true, APIURL: "http://b", APIKey: "k2", SupportedModels: map[string]bool{"gpt-*": true}},
		{Name: "P3", Enabled: false, APIURL: "http://c", APIKey: "k3", SupportedModels: map[string]bool{"claude-*": true}},
		{Name: "P4", Enabled: true, APIURL: "", APIKey: "k4", SupportedModels: map[string]bool{"claude-*": true}},
		{Name: "P5", Enabled: true, APIURL: "http://e", APIKey: "", SupportedModels: map[string]bool{"claude-*": true}},
	}

	// Filter for claude model
	result := rs.filterProviders(providers, "claude-3-opus")
	if len(result) != 1 {
		t.Errorf("Expected 1 provider for claude, got %d", len(result))
	}
	if result[0].Name != "P1" {
		t.Errorf("Expected P1, got %s", result[0].Name)
	}

	// Filter for gpt model
	result = rs.filterProviders(providers, "gpt-4")
	if len(result) != 1 {
		t.Errorf("Expected 1 provider for gpt, got %d", len(result))
	}
	if result[0].Name != "P2" {
		t.Errorf("Expected P2, got %s", result[0].Name)
	}
}

func TestRelayService_ReplaceModel(t *testing.T) {
	rs := &RelayService{}

	body := []byte(`{"model":"claude-3-opus","messages":[]}`)
	newBody := rs.replaceModel(body, "anthropic/claude-3-opus")

	expected := `{"messages":[],"model":"anthropic/claude-3-opus"}`
	if string(newBody) != expected {
		// JSON marshaling may change order, just check the model was replaced
		if !containsModel(newBody, "anthropic/claude-3-opus") {
			t.Errorf("Model not replaced correctly")
		}
	}
}

func containsModel(body []byte, model string) bool {
	return contains(string(body), `"model":"` + model + `"`) ||
	       contains(string(body), `"model": "` + model + `"`)
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && containsAt(s, substr)))
}

func containsAt(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestClassifyHTTPError(t *testing.T) {
	tests := []struct {
		statusCode int
		expected   string
	}{
		{401, "auth_error"},
		{403, "auth_error"},
		{429, "rate_limit"},
		{400, "client_error"},
		{404, "client_error"},
		{422, "client_error"},
		{500, "server_error"},
		{502, "server_error"},
		{503, "server_error"},
		{200, "unknown_error"},
	}

	for _, tt := range tests {
		result := classifyHTTPError(tt.statusCode)
		if result != tt.expected {
			t.Errorf("classifyHTTPError(%d) = %s, want %s", tt.statusCode, result, tt.expected)
		}
	}
}

func TestGenerateTraceID(t *testing.T) {
	id1 := generateTraceID()

	if id1 == "" {
		t.Error("generateTraceID returned empty string")
	}
	// IDs are based on nanoseconds, should be numeric
	if len(id1) < 10 {
		t.Errorf("generateTraceID should be at least 10 chars, got %d", len(id1))
	}
}

func TestGenerateID(t *testing.T) {
	id1 := generateID()

	if id1 == "" {
		t.Error("generateID returned empty string")
	}
	if len(id1) < 4 || id1[:4] != "req_" {
		t.Errorf("generateID should start with 'req_', got %s", id1)
	}
	// Should contain nanosecond timestamp
	if len(id1) < 15 {
		t.Errorf("generateID should be at least 15 chars, got %d", len(id1))
	}
}

func TestRequestLog(t *testing.T) {
	log := &RequestLog{
		ID:           "req-1",
		TraceID:      "trace-1",
		UserID:       "user-1",
		Platform:     "claude",
		Model:        "claude-3-opus",
		Provider:     "anthropic",
		IsStream:     true,
		HTTPCode:     200,
		InputTokens:  1000,
		OutputTokens: 500,
	}

	if log.ID != "req-1" {
		t.Errorf("Expected ID 'req-1', got '%s'", log.ID)
	}
	if log.InputTokens+log.OutputTokens != 1500 {
		t.Error("Token count incorrect")
	}
}

func TestParseSSETokenUsage(t *testing.T) {
	rs := &RelayService{}
	reqLog := &RequestLog{}

	// Test valid usage data
	line := []byte(`data: {"usage":{"input_tokens":100,"output_tokens":50,"cache_read_input_tokens":10}}`)
	rs.parseSSETokenUsage(line, reqLog)

	if reqLog.InputTokens != 100 {
		t.Errorf("Expected input_tokens 100, got %d", reqLog.InputTokens)
	}
	if reqLog.OutputTokens != 50 {
		t.Errorf("Expected output_tokens 50, got %d", reqLog.OutputTokens)
	}
	if reqLog.CacheReadTokens != 10 {
		t.Errorf("Expected cache_read_tokens 10, got %d", reqLog.CacheReadTokens)
	}

	// Test [DONE] message
	reqLog2 := &RequestLog{}
	line2 := []byte("data: [DONE]")
	rs.parseSSETokenUsage(line2, reqLog2)
	// Should not crash, no changes

	// Test non-data line
	reqLog3 := &RequestLog{}
	line3 := []byte("event: message")
	rs.parseSSETokenUsage(line3, reqLog3)
	// Should not crash, no changes
}

func TestParseNormalTokenUsage(t *testing.T) {
	rs := &RelayService{}
	reqLog := &RequestLog{}

	// Test Claude format
	body := []byte(`{"usage":{"input_tokens":200,"output_tokens":100},"stop_reason":"end_turn"}`)
	rs.parseNormalTokenUsage(body, reqLog)

	if reqLog.InputTokens != 200 {
		t.Errorf("Expected input_tokens 200, got %d", reqLog.InputTokens)
	}
	if reqLog.OutputTokens != 100 {
		t.Errorf("Expected output_tokens 100, got %d", reqLog.OutputTokens)
	}
	if reqLog.FinishReason != "end_turn" {
		t.Errorf("Expected finish_reason 'end_turn', got '%s'", reqLog.FinishReason)
	}
}
