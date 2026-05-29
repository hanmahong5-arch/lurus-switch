// Package translator converts between Anthropic Messages API
// (POST /v1/messages) and OpenAI Chat Completions API
// (POST /v1/chat/completions). Lets Claude Code talk to any
// OpenAI-compatible upstream — DeepSeek, Groq, OpenRouter, Ollama,
// the user's home server, anything.
//
// Scope of this MVP:
//   - Text + tool_use / tool_result content blocks
//   - System messages (string and array)
//   - Streaming SSE in both directions
//   - Standard params: model / max_tokens / temperature / top_p /
//     stop_sequences / tools / tool_choice
//
// Out of scope (fall through unmodified or rejected with a clear
// error so we don't pretend we handled them):
//   - Image / document content blocks (multimodal) — the bytes aren't
//     forwarded (the upstream may be text-only), but we no longer drop
//     them silently: a stub text note marks where non-text content was
//     so the model isn't left answering as if nothing was attached.
//   - Prompt caching cache_control hints — we strip them; upstream
//     is unlikely to honor Anthropic-specific hints anyway.
//   - Server tools (web_search, computer_use, code_execution).
//   - Extended thinking blocks — left in passthrough; downstream
//     OpenAI servers will ignore the unknown content type field.
package translator

import "encoding/json"

// ─── Anthropic schema ─────────────────────────────────────────────

// AnthropicRequest is the inbound POST /v1/messages body. Fields we
// care about translating are typed; the rest go through json.RawMessage
// so we can preserve future additions without recompiling.
type AnthropicRequest struct {
	Model         string             `json:"model"`
	MaxTokens     int                `json:"max_tokens"`
	System        json.RawMessage    `json:"system,omitempty"` // string OR []SystemBlock
	Messages      []AnthropicMessage `json:"messages"`
	Tools         []AnthropicTool    `json:"tools,omitempty"`
	ToolChoice    *AnthropicToolChoice `json:"tool_choice,omitempty"`
	Temperature   *float64           `json:"temperature,omitempty"`
	TopP          *float64           `json:"top_p,omitempty"`
	TopK          *int               `json:"top_k,omitempty"`
	StopSequences []string           `json:"stop_sequences,omitempty"`
	Stream        bool               `json:"stream,omitempty"`
	Metadata      json.RawMessage    `json:"metadata,omitempty"`
}

// AnthropicMessage role is "user" or "assistant". Content is either a
// plain string or an array of typed blocks. We always normalize to
// blocks during translation so callers see a single shape.
type AnthropicMessage struct {
	Role    string          `json:"role"`
	Content json.RawMessage `json:"content"`
}

// ContentBlock is the unified shape after we normalize string content
// into a [{type:text,text:s}] array. Type tells which fields apply.
type ContentBlock struct {
	Type string `json:"type"`

	// type=text
	Text string `json:"text,omitempty"`

	// type=tool_use
	ID    string          `json:"id,omitempty"`
	Name  string          `json:"name,omitempty"`
	Input json.RawMessage `json:"input,omitempty"` // arbitrary JSON

	// type=tool_result
	ToolUseID string          `json:"tool_use_id,omitempty"`
	IsError   bool            `json:"is_error,omitempty"`
	// Content of tool_result can itself be a string or array of blocks;
	// we keep it raw and let the consumer flatten.
	Content json.RawMessage `json:"content,omitempty"`

	// type=image | document. We don't forward the bytes (the upstream may
	// not be vision-capable), but we read media_type so we can emit a
	// human-readable stub note instead of silently dropping the block.
	Source *ContentBlockSource `json:"source,omitempty"`
}

// ContentBlockSource is the `source` object of an image / document block.
// Only the fields needed for a stub note are typed; the base64 payload
// (data / url) is intentionally ignored — see userBlocksToOpenAI.
type ContentBlockSource struct {
	Type      string `json:"type"`                 // "base64" | "url"
	MediaType string `json:"media_type,omitempty"` // e.g. "image/png"
}

type AnthropicTool struct {
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	InputSchema json.RawMessage `json:"input_schema"`
}

// AnthropicToolChoice is one of:
//   { "type": "auto" }
//   { "type": "any" }
//   { "type": "tool", "name": "<name>" }
//   { "type": "none" }
type AnthropicToolChoice struct {
	Type string `json:"type"`
	Name string `json:"name,omitempty"`
}

// AnthropicResponse is what we emit back to the client when not
// streaming.
type AnthropicResponse struct {
	ID           string         `json:"id"`
	Type         string         `json:"type"` // always "message"
	Role         string         `json:"role"` // always "assistant"
	Model        string         `json:"model"`
	Content      []ContentBlock `json:"content"`
	StopReason   string         `json:"stop_reason"` // end_turn|max_tokens|stop_sequence|tool_use
	StopSequence *string        `json:"stop_sequence"`
	Usage        AnthropicUsage `json:"usage"`
}

type AnthropicUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
	// CacheCreation/Read are Anthropic-only; we leave them at 0 since
	// no OpenAI-compat server reports cache reads in the same shape.
	CacheCreationInputTokens int `json:"cache_creation_input_tokens,omitempty"`
	CacheReadInputTokens     int `json:"cache_read_input_tokens,omitempty"`
}

// AnthropicError is the error envelope shape Claude Code expects so
// we don't break its error-handling code paths.
type AnthropicError struct {
	Type  string             `json:"type"`
	Error AnthropicErrorBody `json:"error"`
}

type AnthropicErrorBody struct {
	Type    string `json:"type"`    // invalid_request_error | api_error | etc
	Message string `json:"message"`
}

// ─── OpenAI schema ────────────────────────────────────────────────

type OpenAIRequest struct {
	Model            string          `json:"model"`
	Messages         []OpenAIMessage `json:"messages"`
	Tools            []OpenAITool    `json:"tools,omitempty"`
	ToolChoice       json.RawMessage `json:"tool_choice,omitempty"` // string OR object
	Temperature      *float64        `json:"temperature,omitempty"`
	TopP             *float64        `json:"top_p,omitempty"`
	MaxTokens        int             `json:"max_tokens,omitempty"`
	Stop             []string        `json:"stop,omitempty"`
	Stream           bool            `json:"stream,omitempty"`
	StreamOptions    *StreamOptions  `json:"stream_options,omitempty"`
}

type StreamOptions struct {
	IncludeUsage bool `json:"include_usage"`
}

type OpenAIMessage struct {
	Role       string         `json:"role"` // system | user | assistant | tool
	Content    json.RawMessage `json:"content,omitempty"` // string or null
	ToolCalls  []OpenAIToolCall `json:"tool_calls,omitempty"`
	ToolCallID string         `json:"tool_call_id,omitempty"` // role=tool
	Name       string         `json:"name,omitempty"`
}

type OpenAIToolCall struct {
	ID       string             `json:"id"`
	Type     string             `json:"type"` // "function"
	Function OpenAIFunctionCall `json:"function"`
}

type OpenAIFunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"` // JSON string per OpenAI spec
}

type OpenAITool struct {
	Type     string             `json:"type"` // "function"
	Function OpenAIToolFunction `json:"function"`
}

type OpenAIToolFunction struct {
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	Parameters  json.RawMessage `json:"parameters"`
}

// OpenAIResponse is the non-stream chat-completion shape DeepSeek /
// OpenAI / Groq all return.
type OpenAIResponse struct {
	ID      string         `json:"id"`
	Object  string         `json:"object"`
	Created int64          `json:"created"`
	Model   string         `json:"model"`
	Choices []OpenAIChoice `json:"choices"`
	Usage   OpenAIUsage    `json:"usage"`
}

type OpenAIChoice struct {
	Index        int           `json:"index"`
	Message      OpenAIMessage `json:"message"`
	FinishReason string        `json:"finish_reason"` // stop|length|tool_calls|content_filter
}

type OpenAIUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// OpenAIStreamChunk is what we read from upstream's SSE stream. Each
// `data: {…}` SSE line decodes into this.
type OpenAIStreamChunk struct {
	ID      string                   `json:"id"`
	Object  string                   `json:"object"` // chat.completion.chunk
	Created int64                    `json:"created"`
	Model   string                   `json:"model"`
	Choices []OpenAIStreamChoice     `json:"choices"`
	Usage   *OpenAIUsage             `json:"usage,omitempty"`
}

type OpenAIStreamChoice struct {
	Index        int             `json:"index"`
	Delta        OpenAIDelta     `json:"delta"`
	FinishReason *string         `json:"finish_reason,omitempty"`
}

type OpenAIDelta struct {
	Role      string                  `json:"role,omitempty"`
	Content   string                  `json:"content,omitempty"`
	ToolCalls []OpenAIDeltaToolCall   `json:"tool_calls,omitempty"`
}

type OpenAIDeltaToolCall struct {
	Index    int                  `json:"index"`
	ID       string               `json:"id,omitempty"`
	Type     string               `json:"type,omitempty"`
	Function *OpenAIDeltaFunction `json:"function,omitempty"`
}

type OpenAIDeltaFunction struct {
	Name      string `json:"name,omitempty"`
	Arguments string `json:"arguments,omitempty"`
}
