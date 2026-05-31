package translator

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

// StreamTranslator converts an OpenAI SSE stream into an Anthropic SSE
// stream incrementally. Anthropic's protocol is event-driven (named
// SSE events for message_start / content_block_start / _delta / _stop
// / message_delta / message_stop), while OpenAI sends a single
// undifferentiated stream of choice deltas — so we own quite a bit of
// state to bridge them.
//
// Usage:
//
//	tr := NewStreamTranslator("msg_id", "claude-...", inputTokens)
//	tr.Run(upstreamReader, anthropicWriter, flushFn)
//
// flushFn is called after every event so the client sees real-time
// streaming; pass http.Flusher.Flush.
type StreamTranslator struct {
	msgID            string
	model            string
	inputTokens      int

	// Per-stream state.
	textBlockOpen    bool
	textBlockIdx     int
	toolBlocks       map[int]*toolBlockState // OpenAI delta tool_call.index → our content block info
	nextBlockIdx     int
	finalStopReason  string
	finalOutputTokens int

	// Token-stream breakdowns captured from the upstream's final usage chunk.
	// Both are subsets already inside the prompt / completion totals; the
	// gateway normalizes (subtract cached out of billed input) when metering.
	finalCachedTokens    int
	finalReasoningTokens int
}

type toolBlockState struct {
	contentIdx int // our Anthropic content_block index
	id         string
	name       string
}

func NewStreamTranslator(msgID, model string, inputTokens int) *StreamTranslator {
	return &StreamTranslator{
		msgID:        msgID,
		model:        model,
		inputTokens:  inputTokens,
		toolBlocks:   make(map[int]*toolBlockState),
		textBlockIdx: -1,
	}
}

// Usage returns the token counts captured during Run: the input (prompt)
// count from the upstream's final usage chunk — or the seed value passed to
// NewStreamTranslator when upstream omitted one — and the output (completion)
// count. Both are 0 if the upstream sent no usage chunk. Call after Run
// returns; the caller meters and feeds the budget guard from these values.
func (s *StreamTranslator) Usage() (inputTokens, outputTokens int) {
	return s.inputTokens, s.finalOutputTokens
}

// CacheUsage returns the token-stream breakdowns captured during Run: the
// prompt-cache read count (a subset of the input total) and the reasoning /
// "thinking" count (a subset of the output total). Both are 0 when the
// upstream's usage chunk omitted the details. Call after Run; the gateway
// passes these through so cache-read bills at the discounted rate and
// reasoning shows up on the dashboard without being billed twice.
func (s *StreamTranslator) CacheUsage() (cachedTokens, reasoningTokens int) {
	return s.finalCachedTokens, s.finalReasoningTokens
}

// Run consumes the upstream stream and writes Anthropic SSE events to
// out. flush is invoked after each event so an http.Flusher can push
// bytes downstream immediately. Returns when upstream EOFs or errors.
//
// The "final usage" event is emitted last with completion-token count,
// regardless of whether upstream included a usage chunk (we fall back
// to 0 if it didn't).
func (s *StreamTranslator) Run(upstream io.Reader, out io.Writer, flush func()) error {
	if err := s.writeMessageStart(out, flush); err != nil {
		return err
	}

	scanner := bufio.NewScanner(upstream)
	// Server-sent events can have lines longer than the default 64 KB
	// scanner buffer when tool-call argument deltas are big.
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		payload := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if payload == "" || payload == "[DONE]" {
			continue
		}
		var chunk OpenAIStreamChunk
		if err := json.Unmarshal([]byte(payload), &chunk); err != nil {
			// Malformed — skip rather than abort the stream.
			continue
		}
		if err := s.processChunk(&chunk, out, flush); err != nil {
			return err
		}
	}
	if err := scanner.Err(); err != nil && err != io.EOF {
		return fmt.Errorf("scan upstream: %w", err)
	}

	// Wrap up: close any open content blocks, emit message_delta + stop.
	if err := s.closeOpenBlocks(out, flush); err != nil {
		return err
	}
	if err := s.writeMessageDelta(out, flush); err != nil {
		return err
	}
	return s.writeMessageStop(out, flush)
}

func (s *StreamTranslator) processChunk(chunk *OpenAIStreamChunk, out io.Writer, flush func()) error {
	if chunk.Usage != nil {
		// Final usage chunk — record but don't emit yet (Anthropic puts
		// usage on message_delta which comes after content blocks close).
		s.finalOutputTokens = chunk.Usage.CompletionTokens
		// If upstream gave us a more accurate prompt count, prefer it.
		if chunk.Usage.PromptTokens > 0 {
			s.inputTokens = chunk.Usage.PromptTokens
		}
		// Capture the cache-read / reasoning subsets so the gateway can bill
		// the cache stream at its discounted rate (see CacheUsage).
		s.finalCachedTokens = chunk.Usage.PromptTokensDetails.CachedTokens
		s.finalReasoningTokens = chunk.Usage.CompletionTokensDetails.ReasoningTokens
	}
	if len(chunk.Choices) == 0 {
		return nil
	}
	choice := chunk.Choices[0]
	if choice.FinishReason != nil && *choice.FinishReason != "" {
		s.finalStopReason = mapFinishReason(*choice.FinishReason,
			len(s.toolBlocks) > 0)
	}

	// Text delta.
	if choice.Delta.Content != "" {
		if !s.textBlockOpen {
			idx := s.nextBlockIdx
			s.nextBlockIdx++
			s.textBlockIdx = idx
			s.textBlockOpen = true
			if err := s.writeContentBlockStart(out, flush, idx, ContentBlock{Type: "text"}); err != nil {
				return err
			}
		}
		if err := s.writeTextDelta(out, flush, s.textBlockIdx, choice.Delta.Content); err != nil {
			return err
		}
	}

	// Tool-call deltas. OpenAI streams tool calls incrementally — first
	// chunk has id/type/function.name, later chunks add to
	// function.arguments.
	for _, tc := range choice.Delta.ToolCalls {
		state, ok := s.toolBlocks[tc.Index]
		if !ok {
			// First chunk for this tool — close any open text block,
			// open a new tool_use content block.
			if s.textBlockOpen {
				if err := s.writeContentBlockStop(out, flush, s.textBlockIdx); err != nil {
					return err
				}
				s.textBlockOpen = false
			}
			contentIdx := s.nextBlockIdx
			s.nextBlockIdx++
			state = &toolBlockState{contentIdx: contentIdx, id: tc.ID}
			if tc.Function != nil {
				state.name = tc.Function.Name
			}
			s.toolBlocks[tc.Index] = state
			if err := s.writeContentBlockStart(out, flush, contentIdx, ContentBlock{
				Type:  "tool_use",
				ID:    state.id,
				Name:  state.name,
				Input: json.RawMessage(`{}`),
			}); err != nil {
				return err
			}
		}
		// Subsequent chunks may carry an argument fragment.
		if tc.Function != nil && tc.Function.Arguments != "" {
			if err := s.writeJSONDelta(out, flush, state.contentIdx, tc.Function.Arguments); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *StreamTranslator) closeOpenBlocks(out io.Writer, flush func()) error {
	if s.textBlockOpen {
		if err := s.writeContentBlockStop(out, flush, s.textBlockIdx); err != nil {
			return err
		}
		s.textBlockOpen = false
	}
	for _, state := range s.toolBlocks {
		if err := s.writeContentBlockStop(out, flush, state.contentIdx); err != nil {
			return err
		}
	}
	return nil
}

// ─── SSE event writers ────────────────────────────────────────────

func (s *StreamTranslator) writeMessageStart(out io.Writer, flush func()) error {
	body := map[string]any{
		"type": "message_start",
		"message": map[string]any{
			"id":            s.msgID,
			"type":          "message",
			"role":          "assistant",
			"model":         s.model,
			"content":       []any{},
			"stop_reason":   nil,
			"stop_sequence": nil,
			"usage": map[string]any{
				"input_tokens":  s.inputTokens,
				"output_tokens": 0,
			},
		},
	}
	return writeSSE(out, flush, "message_start", body)
}

func (s *StreamTranslator) writeContentBlockStart(out io.Writer, flush func(), index int, block ContentBlock) error {
	return writeSSE(out, flush, "content_block_start", map[string]any{
		"type":          "content_block_start",
		"index":         index,
		"content_block": block,
	})
}

func (s *StreamTranslator) writeTextDelta(out io.Writer, flush func(), index int, text string) error {
	return writeSSE(out, flush, "content_block_delta", map[string]any{
		"type":  "content_block_delta",
		"index": index,
		"delta": map[string]any{"type": "text_delta", "text": text},
	})
}

func (s *StreamTranslator) writeJSONDelta(out io.Writer, flush func(), index int, partialJSON string) error {
	return writeSSE(out, flush, "content_block_delta", map[string]any{
		"type":  "content_block_delta",
		"index": index,
		"delta": map[string]any{"type": "input_json_delta", "partial_json": partialJSON},
	})
}

func (s *StreamTranslator) writeContentBlockStop(out io.Writer, flush func(), index int) error {
	return writeSSE(out, flush, "content_block_stop", map[string]any{
		"type":  "content_block_stop",
		"index": index,
	})
}

func (s *StreamTranslator) writeMessageDelta(out io.Writer, flush func()) error {
	stop := s.finalStopReason
	if stop == "" {
		stop = "end_turn"
	}
	return writeSSE(out, flush, "message_delta", map[string]any{
		"type": "message_delta",
		"delta": map[string]any{
			"stop_reason":   stop,
			"stop_sequence": nil,
		},
		"usage": map[string]any{"output_tokens": s.finalOutputTokens},
	})
}

func (s *StreamTranslator) writeMessageStop(out io.Writer, flush func()) error {
	return writeSSE(out, flush, "message_stop", map[string]any{"type": "message_stop"})
}

// writeSSE emits a named SSE event with a JSON payload, then flushes.
//
// SSE spec:  "event: <name>\ndata: <json>\n\n"
func writeSSE(out io.Writer, flush func(), eventName string, body any) error {
	data, err := json.Marshal(body)
	if err != nil {
		return err
	}
	if _, err := fmt.Fprintf(out, "event: %s\ndata: %s\n\n", eventName, data); err != nil {
		return err
	}
	if flush != nil {
		flush()
	}
	return nil
}
