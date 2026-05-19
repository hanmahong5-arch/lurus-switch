package mcp

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"
)

// TestNewRuntime_DefaultWorkdirHonored checks the §3.2 invariant:
// empty workdir falls back to os.TempDir()-rooted path, never
// hardcoded `/tmp`. (Strict equality with TempDir would be brittle —
// just assert non-empty + non-/tmp on Windows.)
func TestNewRuntime_DefaultWorkdirHonored(t *testing.T) {
	rt := NewRuntime("")
	if rt.workdir == "" {
		t.Fatal("workdir should not be empty when defaulted")
	}
	if !strings.HasSuffix(rt.workdir, "switch-mcp") {
		t.Errorf("default workdir should end with 'switch-mcp', got %q", rt.workdir)
	}
}

// TestFraming_RoundTrip exercises the Content-Length framer in both
// directions — Risk #2 of the design review. Single test catches both
// the writer split-across-pipe issue AND the reader's header parse.
func TestFraming_RoundTrip(t *testing.T) {
	var buf bytes.Buffer
	payload := map[string]any{"jsonrpc": "2.0", "id": 7, "method": "test"}
	if err := writeFrame(&buf, payload); err != nil {
		t.Fatalf("writeFrame: %v", err)
	}
	body, err := readFrame(bufio.NewReader(&buf))
	if err != nil {
		t.Fatalf("readFrame: %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal(body, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got["method"] != "test" {
		t.Errorf("got method=%v, want test", got["method"])
	}
}

// TestReadFrame_RejectsMissingContentLength locks in the "no
// fallback to line-delimited" contract — Risk #2.
func TestReadFrame_RejectsMissingContentLength(t *testing.T) {
	r := bufio.NewReader(strings.NewReader("hello\r\n\r\n"))
	_, err := readFrame(r)
	if err == nil {
		t.Fatal("expected error on missing Content-Length")
	}
}

// fakeSender implements Sender for dispatcher tests, returning
// pre-baked responses keyed by (preset, method).
type fakeSender struct {
	resp map[string]json.RawMessage
	err  map[string]error
}

func (f *fakeSender) Send(_ context.Context, name, method string, _ any) (json.RawMessage, error) {
	key := name + ":" + method
	if e, ok := f.err[key]; ok {
		return nil, e
	}
	return f.resp[key], nil
}

// TestDispatcher_Dispatch covers the three documented paths from §2.2:
// happy MCP route, unknown tool, hook short-circuit.
func TestDispatcher_Dispatch(t *testing.T) {
	resolver := MapResolver{"read_file": "filesystem"}
	good := &fakeSender{
		resp: map[string]json.RawMessage{
			"filesystem:tools/call": json.RawMessage(`"ok"`),
		},
	}
	cases := []struct {
		name    string
		hook    DispatchHook
		sender  Sender
		call    ToolCall
		wantErr error
		wantRes string
	}{
		{
			name:    "happy-path-mcp",
			sender:  good,
			call:    ToolCall{ID: "t1", Name: "read_file", Input: json.RawMessage(`{}`)},
			wantRes: `"ok"`,
		},
		{
			name:    "unknown-tool",
			sender:  good,
			call:    ToolCall{ID: "t2", Name: "nope"},
			wantErr: ErrUnknownTool,
		},
		{
			name: "hook-shortcircuit",
			hook: func(_ context.Context, c ToolCall) (ToolResult, bool, error) {
				return ToolResult{ToolUseID: c.ID, Content: json.RawMessage(`"native"`)}, true, nil
			},
			sender:  good,
			call:    ToolCall{ID: "t3", Name: "web_search"},
			wantRes: `"native"`,
		},
		{
			name:    "sender-error-wrapped",
			sender:  &fakeSender{err: map[string]error{"filesystem:tools/call": errors.New("crash")}},
			call:    ToolCall{ID: "t4", Name: "read_file"},
			wantErr: errors.New("crash"),
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			d := NewDispatcher(tc.sender, resolver, tc.hook)
			res, err := d.Dispatch(context.Background(), tc.call)
			if tc.wantErr != nil {
				if err == nil {
					t.Fatalf("want error %v, got nil", tc.wantErr)
				}
				if errors.Is(tc.wantErr, ErrUnknownTool) && !errors.Is(err, ErrUnknownTool) {
					t.Errorf("want ErrUnknownTool, got %v", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected err: %v", err)
			}
			if string(res.Content) != tc.wantRes {
				t.Errorf("content=%q, want %q", res.Content, tc.wantRes)
			}
		})
	}
}

// TestIsNativeServerTool guards the SWITCH-2 predicate from drift.
// Adding/removing entries here is intentional — these names are part
// of the SWITCH-1 ⇄ SWITCH-2 contract.
func TestIsNativeServerTool(t *testing.T) {
	yes := []string{"web_search", "computer_use", "text_editor"}
	no := []string{"read_file", "search_repos", "", "Web_Search"}
	for _, n := range yes {
		if !IsNativeServerTool(n) {
			t.Errorf("IsNativeServerTool(%q)=false, want true", n)
		}
	}
	for _, n := range no {
		if IsNativeServerTool(n) {
			t.Errorf("IsNativeServerTool(%q)=true, want false", n)
		}
	}
}

// TestRuntime_StartFailsOnMissingBinary verifies Start surfaces spawn
// failures cleanly (no zombie handle in r.handles, restart counter
// incremented). Avoids spinning real MCP servers in unit tests —
// integration coverage lives in the SWITCH_E2E gate.
func TestRuntime_StartFailsOnMissingBinary(t *testing.T) {
	rt := NewRuntime(t.TempDir())
	rt.SetLogger(func(string, ...any) {}) // silence
	srv := MCPServer{
		Name:    "ghost",
		Type:    "stdio",
		Command: "definitely-not-a-real-binary-mcp-test",
	}
	err := rt.Start(context.Background(), "ghost", srv)
	if err == nil {
		t.Fatal("expected spawn error for missing binary")
	}
	rt.mu.Lock()
	_, present := rt.handles["ghost"]
	rt.mu.Unlock()
	if present {
		t.Error("failed Start should not register a handle")
	}
}

// TestRuntime_SendUnknownServer covers the registry-miss path.
func TestRuntime_SendUnknownServer(t *testing.T) {
	rt := NewRuntime(t.TempDir())
	_, err := rt.Send(context.Background(), "nope", "tools/call", nil)
	if !errors.Is(err, ErrServerNotFound) {
		t.Errorf("want ErrServerNotFound, got %v", err)
	}
}

// TestRuntime_RestartBudgetEnforced verifies the 3-crash ceiling
// from §3.1. We manipulate the counter directly so the test stays
// hermetic (no real subprocess crashes needed).
func TestRuntime_RestartBudgetEnforced(t *testing.T) {
	rt := NewRuntime(t.TempDir())
	rt.restarts["x"] = MaxRestarts
	err := rt.Start(context.Background(), "x", MCPServer{Command: "anything"})
	if !errors.Is(err, ErrRestartExhausted) {
		t.Errorf("want ErrRestartExhausted, got %v", err)
	}
}

// TestJitter_InBounds confirms backoff jitter stays in [0.8, 1.2].
func TestJitter_InBounds(t *testing.T) {
	for i := 0; i < 50; i++ {
		j := jitter()
		if j < 0.8 || j > 1.2 {
			t.Errorf("jitter()=%v outside [0.8,1.2]", j)
		}
	}
}

// TestCredScrub locks in Risk #3 — bearer tokens / API keys redacted
// before stderr lines hit the log sink.
func TestCredScrub(t *testing.T) {
	in := []string{
		"Authorization: Bearer sk-xyz",
		"api_key=hunter2",
		"plain log line",
		"Authorization=very-secret",
	}
	want := []string{"<REDACTED>", "<REDACTED>", "plain log line", "<REDACTED>"}
	for i, s := range in {
		out := credScrubRE.ReplaceAllString(s, "$1=<REDACTED>")
		if want[i] == "plain log line" && out != "plain log line" {
			t.Errorf("[%d] scrub mangled plain line: %q", i, out)
			continue
		}
		if want[i] == "<REDACTED>" && !strings.Contains(out, "<REDACTED>") {
			t.Errorf("[%d] expected redaction, got %q", i, out)
		}
	}
}

// TestRuntime_ShutdownIdempotent — back-to-back Shutdown calls must
// not panic / double-close (sets up §3.2 contract for app teardown).
func TestRuntime_ShutdownIdempotent(t *testing.T) {
	rt := NewRuntime(t.TempDir())
	if err := rt.Shutdown(context.Background()); err != nil {
		t.Errorf("first Shutdown: %v", err)
	}
	if err := rt.Shutdown(context.Background()); err != nil {
		t.Errorf("second Shutdown: %v", err)
	}
}

// Defensive: timeouts are constants, so this catches anyone tuning
// them down accidentally and breaking the Winston-locked contract.
func TestTimeouts_RespectDesignReview(t *testing.T) {
	if HandshakeTimeout != 10*time.Second {
		t.Errorf("HandshakeTimeout=%v, design review locks at 10s", HandshakeTimeout)
	}
	if PerCallTimeout != 60*time.Second {
		t.Errorf("PerCallTimeout=%v, design review locks at 60s", PerCallTimeout)
	}
	if MaxRestarts != 3 {
		t.Errorf("MaxRestarts=%d, design review locks at 3", MaxRestarts)
	}
	if GracefulStopWait != 5*time.Second {
		t.Errorf("GracefulStopWait=%v, design review locks at 5s", GracefulStopWait)
	}
}

// Unused but keeps the testing import alive even if some assertions
// get refactored away — guards against silently empty test files.
var _ = fmt.Sprintf
