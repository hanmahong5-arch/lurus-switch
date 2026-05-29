package translator

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
)

// RequestToOpenAI translates an Anthropic POST /v1/messages payload to
// an OpenAI POST /v1/chat/completions payload. The returned struct can
// be marshalled and forwarded to any OpenAI-compatible upstream.
//
// Errors are returned only for structurally-broken Anthropic payloads
// (missing model, missing messages array). Soft mismatches like
// unknown content-block types are best-effort: we drop them and log.
func RequestToOpenAI(in *AnthropicRequest) (*OpenAIRequest, error) {
	if in == nil {
		return nil, fmt.Errorf("nil anthropic request")
	}
	if strings.TrimSpace(in.Model) == "" {
		return nil, fmt.Errorf("model is required")
	}
	if len(in.Messages) == 0 {
		return nil, fmt.Errorf("messages array is required")
	}

	out := &OpenAIRequest{
		Model:       in.Model,
		MaxTokens:   in.MaxTokens,
		Temperature: in.Temperature,
		TopP:        in.TopP,
		Stop:        in.StopSequences,
		Stream:      in.Stream,
	}
	if in.Stream {
		// Without this, most OpenAI-compatible servers don't emit the
		// final usage chunk — and we need it to populate Anthropic's
		// message_delta.usage event.
		out.StreamOptions = &StreamOptions{IncludeUsage: true}
	}

	// 1) System prompt → first system message.
	if sys := extractSystemText(in.System); sys != "" {
		raw, _ := json.Marshal(sys)
		out.Messages = append(out.Messages, OpenAIMessage{Role: "system", Content: raw})
	}

	// 2) Walk Anthropic messages; each may explode into multiple OpenAI
	//    messages (assistant tool_use blocks → assistant.tool_calls;
	//    user tool_result blocks → role=tool messages, one per result).
	for _, m := range in.Messages {
		blocks, err := normalizeContentBlocks(m.Content)
		if err != nil {
			return nil, fmt.Errorf("message %s content: %w", m.Role, err)
		}
		out.Messages = append(out.Messages, blocksToOpenAI(m.Role, blocks)...)
	}

	// 3) Tools → OpenAI function tools.
	for _, t := range in.Tools {
		out.Tools = append(out.Tools, OpenAITool{
			Type: "function",
			Function: OpenAIToolFunction{
				Name: t.Name, Description: t.Description, Parameters: t.InputSchema,
			},
		})
	}

	// 4) Tool choice mapping.
	if in.ToolChoice != nil {
		switch in.ToolChoice.Type {
		case "auto":
			out.ToolChoice = json.RawMessage(`"auto"`)
		case "any":
			// Anthropic 'any' = "must call A tool"; OpenAI's closest is
			// 'required'. Older servers may not support it; we still emit
			// since DeepSeek / OpenAI / Groq all do.
			out.ToolChoice = json.RawMessage(`"required"`)
		case "none":
			out.ToolChoice = json.RawMessage(`"none"`)
		case "tool":
			obj := map[string]any{
				"type":     "function",
				"function": map[string]any{"name": in.ToolChoice.Name},
			}
			out.ToolChoice, _ = json.Marshal(obj)
		}
	}

	return out, nil
}

// extractSystemText accepts either a JSON string or an array of system
// blocks and returns the concatenated text (Anthropic supports both).
// We strip cache_control and other fields silently — see package doc.
func extractSystemText(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return s
	}
	var blocks []ContentBlock
	if err := json.Unmarshal(raw, &blocks); err != nil {
		return ""
	}
	var b strings.Builder
	for _, blk := range blocks {
		if blk.Type == "text" && blk.Text != "" {
			if b.Len() > 0 {
				b.WriteString("\n\n")
			}
			b.WriteString(blk.Text)
		}
	}
	return b.String()
}

// normalizeContentBlocks accepts a string OR array of blocks and
// always returns a slice. String content becomes a single text block.
func normalizeContentBlocks(raw json.RawMessage) ([]ContentBlock, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	// Try string first — most common case.
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return []ContentBlock{{Type: "text", Text: s}}, nil
	}
	var blocks []ContentBlock
	if err := json.Unmarshal(raw, &blocks); err != nil {
		return nil, err
	}
	return blocks, nil
}

// blocksToOpenAI converts the unified-block slice for a single
// Anthropic message into the (potentially multiple) OpenAI messages.
//
// Cases:
//   - assistant + text block(s)           → 1 message with text content
//   - assistant + text + tool_use blocks  → 1 message: text content + tool_calls
//   - user + text block(s)                → 1 message
//   - user + tool_result blocks           → N tool-role messages (one per result)
//   - mixed user (text + tool_result)     → tool messages first, then a user
//     message for the residual text. (OpenAI doesn't allow tool messages to
//     interleave with text inside a single message, so we split.)
func blocksToOpenAI(role string, blocks []ContentBlock) []OpenAIMessage {
	if len(blocks) == 0 {
		return nil
	}
	switch role {
	case "assistant":
		return assistantBlocksToOpenAI(blocks)
	case "user":
		return userBlocksToOpenAI(blocks)
	default:
		// Unknown roles fall through as user messages with concat text.
		return userBlocksToOpenAI(blocks)
	}
}

func assistantBlocksToOpenAI(blocks []ContentBlock) []OpenAIMessage {
	var textParts []string
	var toolCalls []OpenAIToolCall
	for _, b := range blocks {
		switch b.Type {
		case "text":
			if b.Text != "" {
				textParts = append(textParts, b.Text)
			}
		case "tool_use":
			args := string(b.Input)
			if args == "" {
				args = "{}"
			}
			toolCalls = append(toolCalls, OpenAIToolCall{
				ID:   b.ID,
				Type: "function",
				Function: OpenAIFunctionCall{
					Name: b.Name, Arguments: args,
				},
			})
		}
		// Unknown / thinking blocks: ignored — no OpenAI-side analog.
	}
	msg := OpenAIMessage{Role: "assistant"}
	if len(textParts) > 0 {
		c, _ := json.Marshal(strings.Join(textParts, "\n\n"))
		msg.Content = c
	} else {
		// OpenAI requires content key but allows null/empty when tool_calls
		// is set. Use empty string for max compatibility.
		msg.Content = json.RawMessage(`""`)
	}
	if len(toolCalls) > 0 {
		msg.ToolCalls = toolCalls
	}
	return []OpenAIMessage{msg}
}

func userBlocksToOpenAI(blocks []ContentBlock) []OpenAIMessage {
	var out []OpenAIMessage
	var textParts []string
	for _, b := range blocks {
		switch b.Type {
		case "text":
			if b.Text != "" {
				textParts = append(textParts, b.Text)
			}
		case "tool_result":
			// Flatten the tool_result content (string OR array of blocks)
			// into one string — OpenAI's role=tool message body is a string.
			body := flattenToolResultContent(b.Content)
			if b.IsError && body != "" {
				body = "[ERROR] " + body
			}
			c, _ := json.Marshal(body)
			out = append(out, OpenAIMessage{
				Role:       "tool",
				ToolCallID: b.ToolUseID,
				Content:    c,
			})
		default:
			// image / document / unknown: emit a stub text note so the
			// model at least knows non-text content was present, rather
			// than silently dropping it (the old behaviour, which left
			// the upstream answering as if no screenshot was attached).
			// Actually forwarding the bytes as OpenAI image_url parts is
			// gated on a per-channel "vision capable" bit — a separate
			// feature; blind base64 to a text-only model 400s.
			if note := stubNoteForBlock(b); note != "" {
				textParts = append(textParts, note)
			}
		}
	}
	if len(textParts) > 0 {
		c, _ := json.Marshal(strings.Join(textParts, "\n\n"))
		out = append(out, OpenAIMessage{Role: "user", Content: c})
	}
	return out
}

func flattenToolResultContent(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var s string
	if json.Unmarshal(raw, &s) == nil {
		return s
	}
	var blocks []ContentBlock
	if json.Unmarshal(raw, &blocks) != nil {
		return string(raw)
	}
	var b strings.Builder
	for _, blk := range blocks {
		line := blk.Text
		if blk.Type != "text" {
			// Tool results sometimes embed image blocks (e.g. a screenshot
			// from a browser tool). Note their presence instead of dropping.
			line = stubNoteForBlock(blk)
		}
		if line == "" {
			continue
		}
		if b.Len() > 0 {
			b.WriteString("\n")
		}
		b.WriteString(line)
	}
	return b.String()
}

// stubNoteForBlock returns a human-readable placeholder for a content
// block Switch can't forward to an OpenAI-compatible upstream (image,
// document, or any unknown type). Returns "" for blocks handled
// elsewhere (text / tool_use / tool_result) or with no type.
func stubNoteForBlock(b ContentBlock) string {
	switch b.Type {
	case "", "text", "tool_use", "tool_result":
		return ""
	case "image":
		return fmt.Sprintf("[image attached: %s — not forwarded to this upstream]", mediaTypeOrUnknown(b.Source))
	case "document":
		return fmt.Sprintf("[document attached: %s — not forwarded to this upstream]", mediaTypeOrUnknown(b.Source))
	default:
		return fmt.Sprintf("[%s block — not forwarded to this upstream]", b.Type)
	}
}

func mediaTypeOrUnknown(s *ContentBlockSource) string {
	if s != nil && s.MediaType != "" {
		return s.MediaType
	}
	return "unknown type"
}

// ─── Response (non-stream) ─────────────────────────────────────────

// ResponseToAnthropic wraps an OpenAI ChatCompletion in the shape
// Claude Code expects to read back. Only choice[0] is honored; OpenAI
// rarely returns more than one and Anthropic Messages has no analog.
func ResponseToAnthropic(in *OpenAIResponse, model string) *AnthropicResponse {
	out := &AnthropicResponse{
		ID:    "msg_" + idFromOpenAI(in.ID),
		Type:  "message",
		Role:  "assistant",
		Model: model,
		Usage: AnthropicUsage{
			InputTokens:  in.Usage.PromptTokens,
			OutputTokens: in.Usage.CompletionTokens,
		},
	}
	if len(in.Choices) == 0 {
		out.StopReason = "end_turn"
		return out
	}
	c := in.Choices[0]

	if textContent := readContentString(c.Message.Content); textContent != "" {
		out.Content = append(out.Content, ContentBlock{Type: "text", Text: textContent})
	}
	for _, tc := range c.Message.ToolCalls {
		out.Content = append(out.Content, ContentBlock{
			Type:  "tool_use",
			ID:    tc.ID,
			Name:  tc.Function.Name,
			Input: json.RawMessage(orDefault(tc.Function.Arguments, "{}")),
		})
	}
	if len(out.Content) == 0 {
		// Anthropic clients tolerate an empty content array, but Claude
		// Code's parser is happier with at least one block. Push an empty
		// text block.
		out.Content = []ContentBlock{{Type: "text", Text: ""}}
	}
	out.StopReason = mapFinishReason(c.FinishReason, len(c.Message.ToolCalls) > 0)
	return out
}

// readContentString unwraps a JSON-encoded string content field, or
// returns "" if it's null/empty/not-a-string. (OpenAI responses can
// nominally have content as null when tool_calls is present.)
func readContentString(raw json.RawMessage) string {
	if len(raw) == 0 || string(raw) == "null" {
		return ""
	}
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return s
	}
	return ""
}

func mapFinishReason(reason string, hadToolCall bool) string {
	switch reason {
	case "stop":
		return "end_turn"
	case "length":
		return "max_tokens"
	case "tool_calls", "function_call":
		return "tool_use"
	case "content_filter":
		return "end_turn"
	}
	if hadToolCall {
		return "tool_use"
	}
	return "end_turn"
}

func orDefault(s, fallback string) string {
	if strings.TrimSpace(s) == "" {
		return fallback
	}
	return s
}

// idFromOpenAI normalizes the upstream id ("chatcmpl-…") into the
// 24-char alnum slug Anthropic uses. If the upstream ID is missing,
// generate a fresh random.
func idFromOpenAI(id string) string {
	id = strings.TrimPrefix(id, "chatcmpl-")
	id = strings.TrimPrefix(id, "chatcmpl_")
	if len(id) >= 16 {
		return id
	}
	var b [12]byte
	if _, err := rand.Read(b[:]); err == nil {
		return hex.EncodeToString(b[:])
	}
	return "0123456789abcdef0123"
}
