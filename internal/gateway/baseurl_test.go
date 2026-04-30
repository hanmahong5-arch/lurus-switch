package gateway

import "testing"

func TestNormalizeChannelBaseURL(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"groq with v1", "https://api.groq.com/openai/v1", "https://api.groq.com/openai"},
		{"groq with v1 slash", "https://api.groq.com/openai/v1/", "https://api.groq.com/openai"},
		{"groq correct", "https://api.groq.com/openai", "https://api.groq.com/openai"},
		{"openai with v1", "https://api.openai.com/v1", "https://api.openai.com"},
		{"plain domain", "https://api.example.com", "https://api.example.com"},
		{"trailing slash", "https://api.example.com/", "https://api.example.com"},
		{"newapi", "https://newapi.lurus.cn", "https://newapi.lurus.cn"},
		{"localhost with v1", "http://localhost:3000/v1", "http://localhost:3000"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NormalizeChannelBaseURL(tt.input)
			if got != tt.expected {
				t.Errorf("NormalizeChannelBaseURL(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestValidateChannelBaseURL(t *testing.T) {
	tests := []struct {
		input   string
		wantErr bool
	}{
		{"https://api.groq.com/openai/v1", true},
		{"https://api.groq.com/openai", false},
		{"https://api.openai.com/v1", true},
		{"https://newapi.lurus.cn", false},
		{"ftp://bad.example.com", true},
		{"https://generativelanguage.googleapis.com/v1beta", true}, // warning for v1beta
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := ValidateChannelBaseURL(tt.input)
			if (got != "") != tt.wantErr {
				t.Errorf("ValidateChannelBaseURL(%q) = %q, wantErr=%v", tt.input, got, tt.wantErr)
			}
		})
	}
}
