package translator

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestRequestToOpenAI_BasicTextChat(t *testing.T) {
	req := mustReq(t, `{
		"model": "deepseek-chat",
		"max_tokens": 1024,
		"system": "You are a helpful assistant.",
		"messages": [
			{"role": "user", "content": "Hello"},
			{"role": "assistant", "content": "Hi there"},
			{"role": "user", "content": "How are you?"}
		],
		"temperature": 0.7,
		"stream": false
	}`)
	out, err := RequestToOpenAI(req)
	if err != nil {
		t.Fatal(err)
	}
	if out.Model != "deepseek-chat" {
		t.Errorf("model = %s", out.Model)
	}
	if len(out.Messages) != 4 {
		t.Fatalf("expected 4 OpenAI messages (system + 3), got %d", len(out.Messages))
	}
	if out.Messages[0].Role != "system" {
		t.Errorf("first message role = %s, want system", out.Messages[0].Role)
	}
	if string(out.Messages[0].Content) != `"You are a helpful assistant."` {
		t.Errorf("system content = %s", out.Messages[0].Content)
	}
	if *out.Temperature != 0.7 {
		t.Errorf("temp = %v", *out.Temperature)
	}
}

func TestRequestToOpenAI_AssistantToolUseBecomesToolCalls(t *testing.T) {
	req := mustReq(t, `{
		"model": "claude-sonnet-4",
		"max_tokens": 1024,
		"messages": [
			{"role": "user", "content": "What's the weather?"},
			{"role": "assistant", "content": [
				{"type": "text", "text": "I'll check."},
				{"type": "tool_use", "id": "toolu_1", "name": "get_weather", "input": {"city": "Beijing"}}
			]}
		]
	}`)
	out, _ := RequestToOpenAI(req)
	// user msg + assistant msg = 2 (no system header here)
	if len(out.Messages) != 2 {
		t.Fatalf("got %d messages", len(out.Messages))
	}
	asst := out.Messages[1]
	if asst.Role != "assistant" {
		t.Errorf("role = %s", asst.Role)
	}
	if len(asst.ToolCalls) != 1 {
		t.Fatalf("tool_calls = %d, want 1", len(asst.ToolCalls))
	}
	if asst.ToolCalls[0].Function.Name != "get_weather" {
		t.Errorf("name = %s", asst.ToolCalls[0].Function.Name)
	}
	if !strings.Contains(asst.ToolCalls[0].Function.Arguments, "Beijing") {
		t.Errorf("args lost city: %s", asst.ToolCalls[0].Function.Arguments)
	}
}

func TestRequestToOpenAI_UserToolResultBecomesToolMessage(t *testing.T) {
	req := mustReq(t, `{
		"model": "claude-sonnet-4",
		"max_tokens": 1024,
		"messages": [
			{"role": "user", "content": [
				{"type": "tool_result", "tool_use_id": "toolu_1", "content": "Sunny, 25°C"}
			]}
		]
	}`)
	out, _ := RequestToOpenAI(req)
	if len(out.Messages) != 1 {
		t.Fatalf("got %d", len(out.Messages))
	}
	msg := out.Messages[0]
	if msg.Role != "tool" {
		t.Errorf("role = %s, want tool", msg.Role)
	}
	if msg.ToolCallID != "toolu_1" {
		t.Errorf("tool_call_id = %s", msg.ToolCallID)
	}
	if !strings.Contains(string(msg.Content), "Sunny") {
		t.Errorf("content = %s", msg.Content)
	}
}

func TestRequestToOpenAI_ImageBlockEmitsStubNote(t *testing.T) {
	req := mustReq(t, `{
		"model": "deepseek-chat",
		"max_tokens": 1024,
		"messages": [
			{"role": "user", "content": [
				{"type": "text", "text": "What's in this screenshot?"},
				{"type": "image", "source": {"type": "base64", "media_type": "image/png", "data": "iVBORw0KGgo="}}
			]}
		]
	}`)
	out, err := RequestToOpenAI(req)
	if err != nil {
		t.Fatal(err)
	}
	// The user message must survive (not be swallowed) and carry both the
	// original text and a stub note naming the dropped image's media type.
	if len(out.Messages) != 1 {
		t.Fatalf("expected 1 user message, got %d", len(out.Messages))
	}
	body := string(out.Messages[0].Content)
	if !strings.Contains(body, "What's in this screenshot?") {
		t.Errorf("original text lost: %s", body)
	}
	if !strings.Contains(body, "image attached: image/png") {
		t.Errorf("stub note missing media type: %s", body)
	}
}

func TestRequestToOpenAI_DocumentBlock(t *testing.T) {
	req := mustReq(t, `{
		"model": "deepseek-chat",
		"max_tokens": 1024,
		"messages": [
			{"role": "user", "content": [
				{"type": "document", "source": {"type": "base64", "media_type": "application/pdf", "data": "JVBERi0="}}
			]}
		]
	}`)
	out, err := RequestToOpenAI(req)
	if err != nil {
		t.Fatal(err)
	}
	if len(out.Messages) != 1 {
		t.Fatalf("expected 1 user message even with only a document block, got %d", len(out.Messages))
	}
	if body := string(out.Messages[0].Content); !strings.Contains(body, "document attached: application/pdf") {
		t.Errorf("document stub note missing: %s", body)
	}
}

func TestRequestToOpenAI_ToolResultWithImage(t *testing.T) {
	req := mustReq(t, `{
		"model": "claude-sonnet-4",
		"max_tokens": 1024,
		"messages": [
			{"role": "user", "content": [
				{"type": "tool_result", "tool_use_id": "toolu_1", "content": [
					{"type": "text", "text": "Page loaded."},
					{"type": "image", "source": {"type": "base64", "media_type": "image/jpeg", "data": "/9j/4AAQ"}}
				]}
			]}
		]
	}`)
	out, _ := RequestToOpenAI(req)
	if len(out.Messages) != 1 {
		t.Fatalf("got %d messages", len(out.Messages))
	}
	body := string(out.Messages[0].Content)
	if !strings.Contains(body, "Page loaded.") {
		t.Errorf("tool_result text lost: %s", body)
	}
	if !strings.Contains(body, "image attached: image/jpeg") {
		t.Errorf("embedded image stub note missing from flattened tool_result: %s", body)
	}
}

func TestRequestToOpenAI_ThinkingBlockNotStubNoted(t *testing.T) {
	// thinking / redacted_thinking are model-reasoning artifacts, not user
	// content — they must be dropped silently (like the assistant path),
	// never wrapped in a "[thinking block …]" stub note that would leak
	// into the prompt sent to a text-only upstream.
	req := mustReq(t, `{
		"model": "deepseek-chat",
		"max_tokens": 1024,
		"messages": [
			{"role": "user", "content": [
				{"type": "text", "text": "Continue."},
				{"type": "thinking", "thinking": "internal reasoning"},
				{"type": "redacted_thinking", "data": "abc"}
			]}
		]
	}`)
	out, err := RequestToOpenAI(req)
	if err != nil {
		t.Fatal(err)
	}
	if len(out.Messages) != 1 {
		t.Fatalf("expected 1 user message, got %d", len(out.Messages))
	}
	body := string(out.Messages[0].Content)
	if !strings.Contains(body, "Continue.") {
		t.Errorf("user text lost: %s", body)
	}
	if strings.Contains(body, "thinking") || strings.Contains(body, "not forwarded") {
		t.Errorf("thinking block must not produce a stub note: %s", body)
	}
}

func TestRequestToOpenAI_ToolChoiceVariants(t *testing.T) {
	// Compare JSON semantically since Go's encoding/json doesn't
	// guarantee map key order — `{"type":"function","function":{...}}`
	// and `{"function":{...},"type":"function"}` are the same value.
	cases := map[string]string{
		`{"type":"auto"}`:                     `"auto"`,
		`{"type":"any"}`:                      `"required"`,
		`{"type":"none"}`:                     `"none"`,
		`{"type":"tool","name":"get_weather"}`: `{"type":"function","function":{"name":"get_weather"}}`,
	}
	for input, expectedJSON := range cases {
		body := `{
			"model": "x", "max_tokens": 1, "messages": [{"role":"user","content":"hi"}],
			"tool_choice": ` + input + `
		}`
		req := mustReq(t, body)
		out, _ := RequestToOpenAI(req)
		var got, want any
		_ = json.Unmarshal(out.ToolChoice, &got)
		_ = json.Unmarshal([]byte(expectedJSON), &want)
		gotJSON, _ := json.Marshal(got)
		wantJSON, _ := json.Marshal(want)
		if string(gotJSON) != string(wantJSON) {
			t.Errorf("input %s → got %s, want %s", input, gotJSON, wantJSON)
		}
	}
}

func TestRequestToOpenAI_StreamForcesIncludeUsage(t *testing.T) {
	req := mustReq(t, `{
		"model": "x", "max_tokens": 1,
		"messages": [{"role":"user","content":"hi"}],
		"stream": true
	}`)
	out, _ := RequestToOpenAI(req)
	if out.StreamOptions == nil || !out.StreamOptions.IncludeUsage {
		t.Error("stream=true should set stream_options.include_usage=true")
	}
}

func TestRequestToOpenAI_RejectsMissingModel(t *testing.T) {
	req := mustReq(t, `{"max_tokens": 1, "messages": [{"role":"user","content":"hi"}]}`)
	_, err := RequestToOpenAI(req)
	if err == nil {
		t.Error("expected error for missing model")
	}
}

func TestResponseToAnthropic_TextOnly(t *testing.T) {
	resp := &OpenAIResponse{
		ID: "chatcmpl-abc", Choices: []OpenAIChoice{{
			Message:      OpenAIMessage{Role: "assistant", Content: json.RawMessage(`"Hello there"`)},
			FinishReason: "stop",
		}},
		Usage: OpenAIUsage{PromptTokens: 10, CompletionTokens: 3},
	}
	out := ResponseToAnthropic(resp, "claude-sonnet-4")
	if out.Role != "assistant" {
		t.Errorf("role = %s", out.Role)
	}
	if len(out.Content) != 1 {
		t.Fatalf("content blocks = %d", len(out.Content))
	}
	if out.Content[0].Type != "text" || out.Content[0].Text != "Hello there" {
		t.Errorf("got %+v", out.Content[0])
	}
	if out.StopReason != "end_turn" {
		t.Errorf("stop = %s", out.StopReason)
	}
	if out.Usage.InputTokens != 10 || out.Usage.OutputTokens != 3 {
		t.Errorf("usage = %+v", out.Usage)
	}
}

func TestResponseToAnthropic_ToolCalls(t *testing.T) {
	resp := &OpenAIResponse{
		ID: "chatcmpl-x", Choices: []OpenAIChoice{{
			Message: OpenAIMessage{
				Role:    "assistant",
				Content: json.RawMessage(`null`),
				ToolCalls: []OpenAIToolCall{{
					ID:   "call_1",
					Type: "function",
					Function: OpenAIFunctionCall{
						Name:      "get_weather",
						Arguments: `{"city":"Tokyo"}`,
					},
				}},
			},
			FinishReason: "tool_calls",
		}},
	}
	out := ResponseToAnthropic(resp, "claude-x")
	// Should have one tool_use content block (no text since content was null)
	hasToolUse := false
	for _, c := range out.Content {
		if c.Type == "tool_use" && c.Name == "get_weather" {
			hasToolUse = true
			if !strings.Contains(string(c.Input), "Tokyo") {
				t.Errorf("input lost city: %s", c.Input)
			}
		}
	}
	if !hasToolUse {
		t.Errorf("no tool_use block in %+v", out.Content)
	}
	if out.StopReason != "tool_use" {
		t.Errorf("stop = %s, want tool_use", out.StopReason)
	}
}

func TestStreamTranslator_TextDeltaFlow(t *testing.T) {
	upstream := `data: {"choices":[{"delta":{"role":"assistant"}}]}

data: {"choices":[{"delta":{"content":"Hello"}}]}

data: {"choices":[{"delta":{"content":", world"}}]}

data: {"choices":[{"delta":{},"finish_reason":"stop"}]}

data: {"choices":[],"usage":{"prompt_tokens":12,"completion_tokens":4}}

data: [DONE]
`
	var buf bytes.Buffer
	tr := NewStreamTranslator("msg_x", "deepseek-chat", 0)
	if err := tr.Run(strings.NewReader(upstream), &buf, nil); err != nil {
		t.Fatal(err)
	}
	out := buf.String()

	mustHave(t, out, "event: message_start")
	mustHave(t, out, `"id":"msg_x"`)
	mustHave(t, out, "event: content_block_start")
	mustHave(t, out, `"type":"text"`)
	mustHave(t, out, "event: content_block_delta")
	mustHave(t, out, `"text":"Hello"`)
	mustHave(t, out, `"text":", world"`)
	mustHave(t, out, "event: content_block_stop")
	mustHave(t, out, "event: message_delta")
	mustHave(t, out, `"stop_reason":"end_turn"`)
	mustHave(t, out, `"output_tokens":4`)
	mustHave(t, out, "event: message_stop")

	// Usage() must surface the upstream's final usage chunk so the gateway
	// can meter streaming traffic and feed the budget wall.
	in, outTok := tr.Usage()
	if in != 12 || outTok != 4 {
		t.Fatalf("Usage() = (%d, %d), want (12, 4)", in, outTok)
	}
}

func TestStreamTranslator_UsageZeroWithoutUpstreamChunk(t *testing.T) {
	// Upstream never sends a usage chunk — Usage() falls back to the seed
	// input count and zero output (no fabricated tokens).
	upstream := `data: {"choices":[{"delta":{"content":"hi"}}]}

data: {"choices":[{"delta":{},"finish_reason":"stop"}]}

data: [DONE]
`
	var buf bytes.Buffer
	tr := NewStreamTranslator("msg_z", "deepseek-chat", 3)
	if err := tr.Run(strings.NewReader(upstream), &buf, nil); err != nil {
		t.Fatal(err)
	}
	in, outTok := tr.Usage()
	if in != 3 || outTok != 0 {
		t.Fatalf("Usage() = (%d, %d), want (3, 0)", in, outTok)
	}
}

func TestStreamTranslator_ToolCallFlow(t *testing.T) {
	upstream := `data: {"choices":[{"delta":{"tool_calls":[{"index":0,"id":"call_1","type":"function","function":{"name":"get_weather","arguments":""}}]}}]}

data: {"choices":[{"delta":{"tool_calls":[{"index":0,"function":{"arguments":"{\"city\""}}]}}]}

data: {"choices":[{"delta":{"tool_calls":[{"index":0,"function":{"arguments":":\"Tokyo\"}"}}]}}]}

data: {"choices":[{"delta":{},"finish_reason":"tool_calls"}],"usage":{"prompt_tokens":15,"completion_tokens":8}}

data: [DONE]
`
	var buf bytes.Buffer
	tr := NewStreamTranslator("msg_t", "deepseek-chat", 0)
	if err := tr.Run(strings.NewReader(upstream), &buf, nil); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	mustHave(t, out, `"type":"tool_use"`)
	mustHave(t, out, `"name":"get_weather"`)
	mustHave(t, out, `"id":"call_1"`)
	mustHave(t, out, `"type":"input_json_delta"`)
	mustHave(t, out, `"partial_json":"{\"city\""`)
	mustHave(t, out, `"partial_json":":\"Tokyo\"}"`)
	mustHave(t, out, `"stop_reason":"tool_use"`)
}

// ─── Helpers ──────────────────────────────────────────────────────

func mustReq(t *testing.T, raw string) *AnthropicRequest {
	t.Helper()
	var req AnthropicRequest
	if err := json.Unmarshal([]byte(raw), &req); err != nil {
		t.Fatal(err)
	}
	return &req
}

func mustHave(t *testing.T, out, sub string) {
	t.Helper()
	if !strings.Contains(out, sub) {
		t.Errorf("output missing %q\nfull output:\n%s", sub, out)
	}
}
