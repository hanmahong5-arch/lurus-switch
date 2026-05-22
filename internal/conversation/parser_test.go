package conversation

import (
	"strings"
	"testing"
)

const fixtureClaude = `{"type":"user","uuid":"u1","timestamp":"2026-05-01T10:00:00Z","message":{"role":"user","content":"hi there"}}
{"type":"assistant","uuid":"a1","parentUuid":"u1","timestamp":"2026-05-01T10:00:01Z","message":{"role":"assistant","model":"claude-opus-4-7","content":[{"type":"text","text":"hello"}],"usage":{"input_tokens":10,"output_tokens":5}}}
{"type":"assistant","uuid":"a2","timestamp":"2026-05-01T10:00:02Z","message":{"role":"assistant","model":"claude-opus-4-7","content":[{"type":"tool_use","name":"bash","input":{"cmd":"ls"}}]}}
{"type":"tool_result","uuid":"r1","timestamp":"2026-05-01T10:00:03Z","content":[{"type":"text","text":"file.txt"}]}
not json at all — should be skipped
{"type":"summary","uuid":"s1","timestamp":"2026-05-01T10:00:04Z"}
`

func TestParse_HappyPath(t *testing.T) {
	events, err := Parse(strings.NewReader(fixtureClaude))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(events) != 5 {
		t.Fatalf("want 5 events (malformed line dropped), got %d", len(events))
	}
	if events[0].Type != EventUser {
		t.Errorf("event 0: want user, got %s", events[0].Type)
	}
	if events[0].Content != "hi there" {
		t.Errorf("event 0 content: %q", events[0].Content)
	}
	if events[1].Type != EventAssistant {
		t.Errorf("event 1: want assistant, got %s", events[1].Type)
	}
	if events[1].InputTokens != 10 || events[1].OutputTokens != 5 {
		t.Errorf("event 1 tokens: in=%d out=%d", events[1].InputTokens, events[1].OutputTokens)
	}
	if events[2].Type != EventToolUse {
		t.Errorf("event 2: want tool_use, got %s", events[2].Type)
	}
	if events[2].ToolName != "bash" {
		t.Errorf("event 2 tool name: %q", events[2].ToolName)
	}
	if events[3].Type != EventToolResult {
		t.Errorf("event 3: want tool_result, got %s", events[3].Type)
	}
	if events[4].Type != EventMeta {
		t.Errorf("event 4 (summary): want meta, got %s", events[4].Type)
	}
}

func TestParse_UnknownKeysPreserved(t *testing.T) {
	line := `{"type":"user","uuid":"u1","timestamp":"2026-05-01T10:00:00Z","message":{"role":"user","content":"x"},"futureField":{"nested":true}}`
	events, err := Parse(strings.NewReader(line))
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 {
		t.Fatalf("want 1, got %d", len(events))
	}
	if !strings.Contains(string(events[0].Raw), "futureField") {
		t.Error("Raw should preserve unknown keys for forward-compat")
	}
}

func TestFindEventContext(t *testing.T) {
	events, _ := Parse(strings.NewReader(fixtureClaude))
	slice, idx, ok := FindEventContext(events, "a1", 1)
	if !ok {
		t.Fatal("expected to find a1")
	}
	if len(slice) != 3 {
		t.Fatalf("want 3 events around a1, got %d", len(slice))
	}
	if slice[idx].MessageUUID != "a1" {
		t.Errorf("center should be a1, got %s", slice[idx].MessageUUID)
	}
}
