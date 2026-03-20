package installer

import "testing"

func TestToolEndpoint_BareURL(t *testing.T) {
	base := "https://api.lurus.cn"
	tests := []struct {
		tool string
		want string
	}{
		{ToolClaude, "https://api.lurus.cn"},
		{ToolGemini, "https://api.lurus.cn"},
		{ToolZeroClaw, "https://api.lurus.cn"},
		{ToolOpenClaw, "https://api.lurus.cn"},
		{ToolCodex, "https://api.lurus.cn/v1"},
		{ToolPicoClaw, "https://api.lurus.cn/v1"},
		{ToolNullClaw, "https://api.lurus.cn/v1"},
	}
	for _, tt := range tests {
		got := ToolEndpoint(tt.tool, base)
		if got != tt.want {
			t.Errorf("ToolEndpoint(%q, %q) = %q, want %q", tt.tool, base, got, tt.want)
		}
	}
}

func TestToolEndpoint_WithV1Suffix(t *testing.T) {
	base := "https://api.lurus.cn/v1"
	tests := []struct {
		tool string
		want string
	}{
		{ToolClaude, "https://api.lurus.cn"},
		{ToolGemini, "https://api.lurus.cn"},
		{ToolZeroClaw, "https://api.lurus.cn"},
		{ToolOpenClaw, "https://api.lurus.cn"},
		{ToolCodex, "https://api.lurus.cn/v1"},
		{ToolPicoClaw, "https://api.lurus.cn/v1"},
		{ToolNullClaw, "https://api.lurus.cn/v1"},
	}
	for _, tt := range tests {
		got := ToolEndpoint(tt.tool, base)
		if got != tt.want {
			t.Errorf("ToolEndpoint(%q, %q) = %q, want %q", tt.tool, base, got, tt.want)
		}
	}
}

func TestToolEndpoint_TrailingSlash(t *testing.T) {
	got := ToolEndpoint(ToolClaude, "https://api.lurus.cn/")
	if got != "https://api.lurus.cn" {
		t.Errorf("ToolEndpoint(claude, trailing slash) = %q, want bare domain", got)
	}
	got = ToolEndpoint(ToolCodex, "https://api.lurus.cn/")
	if got != "https://api.lurus.cn/v1" {
		t.Errorf("ToolEndpoint(codex, trailing slash) = %q, want with /v1", got)
	}
}
