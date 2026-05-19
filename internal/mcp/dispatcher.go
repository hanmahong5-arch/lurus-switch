package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

// PerCallTimeout is the dispatcher-owned deadline for a single MCP
// tools/call frame. Caller may wrap in their own context for session
// budgets (§2.2 — session bookkeeping is the Switch domain layer).
const PerCallTimeout = 60 * time.Second

// ErrUnknownTool means the dispatcher can't resolve the Anthropic
// tool_use name to any registered MCP server. SWITCH-2 native handler
// catches this and decides whether to pass through.
var ErrUnknownTool = errors.New("mcp: unknown tool")

// ToolCall mirrors Anthropic Messages tool_use block shape.
type ToolCall struct {
	ID    string          `json:"id"`
	Name  string          `json:"name"`
	Input json.RawMessage `json:"input"`
}

// ToolResult mirrors the tool_result block shape returned to Anthropic.
type ToolResult struct {
	ToolUseID string          `json:"tool_use_id"`
	Content   json.RawMessage `json:"content"`
	IsError   bool            `json:"is_error,omitempty"`
}

// Resolver maps a tool name (as the model produced it) onto the MCP
// preset name that owns the binary. Provided by Switch's preset
// registry — kept as an interface so SWITCH-1 doesn't have to commit
// to a registry impl.
type Resolver interface {
	Resolve(tool string) (preset string, ok bool)
}

// ToolCallDispatcher is the public seam SWITCH-2 plugs into. The
// SWITCH-2 translator owns the decision tree (native pass-through vs
// MCP route); SWITCH-1 provides the MCP leg.
type ToolCallDispatcher interface {
	Dispatch(ctx context.Context, call ToolCall) (ToolResult, error)
}

// Sender is the subset of Runtime that Dispatcher needs — defined
// here so tests can mock without spinning a real process.
type Sender interface {
	Send(ctx context.Context, name string, method string, params any) (json.RawMessage, error)
}

// Dispatcher routes Anthropic tool_use blocks to MCP tools/call frames.
// Owns per-call timeout (60s) but NOT session budget.
type Dispatcher struct {
	sender   Sender
	resolver Resolver
	hook     DispatchHook
}

// DispatchHook lets the caller intercept a call before it hits MCP —
// the SWITCH-2 transparency seam. Return handled=true to short-circuit
// with the supplied result (native server-tool already produced output);
// handled=false to fall through to normal MCP dispatch.
type DispatchHook func(ctx context.Context, call ToolCall) (result ToolResult, handled bool, err error)

// NewDispatcher wires a runtime and a tool→preset resolver. hook may be
// nil for pure-MCP-only mode (handy for early bring-up + tests).
func NewDispatcher(s Sender, r Resolver, hook DispatchHook) *Dispatcher {
	return &Dispatcher{sender: s, resolver: r, hook: hook}
}

// Dispatch resolves the tool, applies the hook (for native passthrough),
// and on miss invokes MCP with PerCallTimeout. Returns ErrUnknownTool
// when no resolver hit + no hook handled.
func (d *Dispatcher) Dispatch(ctx context.Context, call ToolCall) (ToolResult, error) {
	if d.hook != nil {
		res, handled, err := d.hook(ctx, call)
		if handled {
			return res, err
		}
	}
	if d.resolver == nil {
		return ToolResult{}, fmt.Errorf("%w: %s (no resolver)", ErrUnknownTool, call.Name)
	}
	preset, ok := d.resolver.Resolve(call.Name)
	if !ok {
		return ToolResult{}, fmt.Errorf("%w: %s", ErrUnknownTool, call.Name)
	}
	cctx, cancel := context.WithTimeout(ctx, PerCallTimeout)
	defer cancel()
	params := map[string]any{"name": call.Name, "arguments": call.Input}
	raw, err := d.sender.Send(cctx, preset, "tools/call", params)
	if err != nil {
		return ToolResult{ToolUseID: call.ID, IsError: true}, err
	}
	return ToolResult{ToolUseID: call.ID, Content: raw}, nil
}

// IsNativeServerTool is the SWITCH-2 predicate per §4. Anthropic's
// messages.create natively serves these tool names — translator should
// pass them through without invoking MCP. Note: the "web_search"
// preset name in this package collides with the native server-tool
// name; SWITCH-2 owns the per-request choice (see preset comments).
func IsNativeServerTool(name string) bool {
	switch name {
	case "web_search", "computer_use", "text_editor":
		return true
	}
	return false
}

// MapResolver is a trivial Resolver backed by a name→preset map.
// Suitable for early bring-up and tests; the production registry will
// be richer (per-session enable/disable, etc).
type MapResolver map[string]string

// Resolve implements Resolver.
func (m MapResolver) Resolve(tool string) (string, bool) {
	p, ok := m[tool]
	return p, ok
}
