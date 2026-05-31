package translator

import (
	"bytes"
	"strings"
	"testing"
)

// TestStreamTranslator_CaptureCacheUsage verifies that the streaming
// translator pulls the cache-read / reasoning breakdowns out of the upstream's
// final usage chunk, so the gateway can bill the cache stream at its discounted
// rate (and show reasoning volume) on the Claude Code streaming path.
func TestStreamTranslator_CaptureCacheUsage(t *testing.T) {
	upstream := strings.NewReader(
		`data: {"choices":[{"delta":{"content":"hi"}}]}` + "\n\n" +
			`data: {"choices":[{"delta":{},"finish_reason":"stop"}]}` + "\n\n" +
			`data: {"choices":[],"usage":{"prompt_tokens":1000,"completion_tokens":120,"total_tokens":1120,"prompt_tokens_details":{"cached_tokens":700},"completion_tokens_details":{"reasoning_tokens":50}}}` + "\n\n" +
			"data: [DONE]\n\n",
	)

	tr := NewStreamTranslator("msg_x", "claude-sonnet-4-6", 0)
	var out bytes.Buffer
	if err := tr.Run(upstream, &out, nil); err != nil {
		t.Fatalf("Run: %v", err)
	}

	inTok, outTok := tr.Usage()
	if inTok != 1000 || outTok != 120 {
		t.Fatalf("Usage() = (%d, %d), want (1000, 120)", inTok, outTok)
	}
	cached, reasoning := tr.CacheUsage()
	if cached != 700 {
		t.Errorf("cached = %d, want 700", cached)
	}
	if reasoning != 50 {
		t.Errorf("reasoning = %d, want 50", reasoning)
	}
}

// TestStreamTranslator_NoUsageDetailsDefaultsZero guards the common case where
// the upstream omits the *_details blocks: CacheUsage must report 0, not panic
// or carry stale state.
func TestStreamTranslator_NoUsageDetailsDefaultsZero(t *testing.T) {
	upstream := strings.NewReader(
		`data: {"choices":[{"delta":{"content":"hi"}}]}` + "\n\n" +
			`data: {"choices":[],"usage":{"prompt_tokens":12,"completion_tokens":8,"total_tokens":20}}` + "\n\n" +
			"data: [DONE]\n\n",
	)

	tr := NewStreamTranslator("msg_y", "claude-sonnet-4-6", 0)
	var out bytes.Buffer
	if err := tr.Run(upstream, &out, nil); err != nil {
		t.Fatalf("Run: %v", err)
	}
	cached, reasoning := tr.CacheUsage()
	if cached != 0 || reasoning != 0 {
		t.Fatalf("CacheUsage() = (%d, %d), want (0, 0) when details omitted", cached, reasoning)
	}
}
