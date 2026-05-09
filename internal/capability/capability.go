// Package capability is the access-control layer for Switch's system
// services. Every Wails binding that mutates state should go through a
// capability check at the top of the handler — this prevents an agent
// (or a prompt-injected LLM) from calling, say, DeleteUser when its
// scope was only meant to cover read-only diagnostics.
//
// The package is deliberately tiny: a small set of named caps, a Token
// type that carries them, and a Require helper that returns an error
// when a cap is missing. Audit is layered on top by recording every
// Require call (success and failure alike) — see internal/audit.
package capability

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
)

// Cap is a string-typed capability name. Use the constants below; new
// caps should be declared here with a one-line description so we have a
// single source of truth.
type Cap string

const (
	// Pricing: model/group ratios, model price, group ratio.
	CapPricingRead  Cap = "pricing.read"
	CapPricingWrite Cap = "pricing.write"

	// Channels: upstream provider config (most dangerous: holds API keys).
	CapChannelRead  Cap = "channel.read"
	CapChannelWrite Cap = "channel.write"
	CapChannelTest  Cap = "channel.test"

	// Models: registering / pricing / catalog management.
	CapModelRead     Cap = "model.read"
	CapModelRegister Cap = "model.register"
	CapModelPricing  Cap = "model.pricing"

	// Users: tenants on the gateway side.
	CapUserRead   Cap = "user.read"   // read all users
	CapUserSelf   Cap = "user.self"   // read only own user
	CapUserCreate Cap = "user.create"
	CapUserModify Cap = "user.modify" // edit non-destructive
	CapUserDelete Cap = "user.delete"
	CapUserFreeze Cap = "user.freeze"

	// Tokens: API keys for users.
	CapTokenRead   Cap = "token.read"
	CapTokenCreate Cap = "token.create"
	CapTokenRevoke Cap = "token.revoke"

	// Logs: usage history (potentially sensitive — prompts, costs).
	CapLogReadOwn Cap = "log.read.own" // own user's logs
	CapLogReadAll Cap = "log.read.all" // any user

	// Redemption codes (top-up / activation).
	CapRedemptionRead   Cap = "redemption.read"
	CapRedemptionCreate Cap = "redemption.create"
	CapRedemptionDelete Cap = "redemption.delete"

	// System options (the /api/option/ surface — most powerful).
	CapOptionRead  Cap = "option.read"
	CapOptionWrite Cap = "option.write"

	// Notifications: outbound to user / admin.
	CapNotifyUser  Cap = "notify.user"
	CapNotifyAdmin Cap = "notify.admin"

	// Audit log access.
	CapAuditRead Cap = "audit.read"
	CapAuditUndo Cap = "audit.undo"

	// Wildcard — only granted to the human-in-the-loop principal
	// (i.e. the desktop user running the app, not agents). Avoid
	// granting * to anything that runs unattended.
	CapAll Cap = "*"
)

// Description returns a one-line human description of a cap. Used by
// the UI to render cap chips.
var Description = map[Cap]string{
	CapPricingRead:      "Read model / group / channel pricing",
	CapPricingWrite:     "Modify pricing ratios (writes affect billing!)",
	CapChannelRead:      "List and inspect upstream channels",
	CapChannelWrite:     "Add / edit / delete channels (holds API keys)",
	CapChannelTest:      "Send a test request to a channel",
	CapModelRead:        "Read the model catalog",
	CapModelRegister:    "Register a new model in the catalog",
	CapModelPricing:     "Set per-model pricing",
	CapUserRead:         "Read all gateway users",
	CapUserSelf:         "Read only the principal's own user",
	CapUserCreate:       "Provision a new user",
	CapUserModify:       "Edit a user (non-destructive fields)",
	CapUserDelete:       "Delete a user (irreversible)",
	CapUserFreeze:       "Freeze / unfreeze a user (reversible)",
	CapTokenRead:        "Read API tokens (always masked)",
	CapTokenCreate:      "Issue a new API token",
	CapTokenRevoke:      "Revoke an API token",
	CapLogReadOwn:       "Read the principal's own request logs",
	CapLogReadAll:       "Read any user's request logs (sensitive)",
	CapRedemptionRead:   "List redemption codes",
	CapRedemptionCreate: "Mint redemption codes",
	CapRedemptionDelete: "Invalidate redemption codes",
	CapOptionRead:       "Read system options",
	CapOptionWrite:      "Write system options (most powerful — affects all users)",
	CapNotifyUser:       "Send email / IM to a user",
	CapNotifyAdmin:      "Send notifications to the operator",
	CapAuditRead:        "Read the audit journal",
	CapAuditUndo:        "Undo a journaled change",
	CapAll:              "Wildcard — full access (human-in-the-loop only)",
}

// AllCaps returns every registered Cap, sorted, useful for UI pickers
// and for granting "everything" to a privileged principal.
func AllCaps() []Cap {
	out := make([]Cap, 0, len(Description))
	for c := range Description {
		out = append(out, c)
	}
	sort.Slice(out, func(i, j int) bool { return string(out[i]) < string(out[j]) })
	return out
}

// Token carries the set of caps a principal may exercise. Tokens are
// lightweight — pass by value through context.
type Token struct {
	Principal string         // human-readable: "user:marvin", "agent:sales-1", "system"
	Caps      map[Cap]bool   // explicit grant set
	notes     string         // optional reason / parent grant chain
}

// NewToken creates a Token for principal granted the given caps.
func NewToken(principal string, caps ...Cap) Token {
	m := make(map[Cap]bool, len(caps))
	for _, c := range caps {
		m[c] = true
	}
	return Token{Principal: principal, Caps: m}
}

// AllToken is the privileged wildcard token used for the desktop user
// — i.e., the human running the GUI. Agents must never get this.
func AllToken(principal string) Token {
	return Token{Principal: principal, Caps: map[Cap]bool{CapAll: true}}
}

// Has returns true iff the token grants the given cap (directly or via
// CapAll).
func (t Token) Has(c Cap) bool {
	if t.Caps == nil {
		return false
	}
	return t.Caps[CapAll] || t.Caps[c]
}

// CapsList returns granted caps, sorted, for UI display.
func (t Token) CapsList() []string {
	out := make([]string, 0, len(t.Caps))
	for c := range t.Caps {
		out = append(out, string(c))
	}
	sort.Strings(out)
	return out
}

// String makes Token Stringer-friendly for logs.
func (t Token) String() string {
	return fmt.Sprintf("Token{principal=%s, caps=[%s]}",
		t.Principal, strings.Join(t.CapsList(), " "))
}

// --- Context plumbing ---------------------------------------------------

type ctxKey struct{}

// WithToken attaches a token to ctx. Bindings should call this once at
// the top (e.g. via a Wails request middleware).
func WithToken(ctx context.Context, t Token) context.Context {
	return context.WithValue(ctx, ctxKey{}, t)
}

// FromContext extracts the token, or returns the zero Token (no caps)
// if none is set. Zero token will fail every Require call, which is
// the safe default.
func FromContext(ctx context.Context) Token {
	if v, ok := ctx.Value(ctxKey{}).(Token); ok {
		return v
	}
	return Token{}
}

// Require returns nil iff the token in ctx grants the cap, otherwise
// an *Error suitable for surfacing through Wails.
func Require(ctx context.Context, c Cap) error {
	t := FromContext(ctx)
	if t.Has(c) {
		return nil
	}
	return &Error{Required: c, Principal: t.Principal}
}

// Error is returned when Require denies. The frontend renders a
// "Permission denied" banner when it sees this code.
type Error struct {
	Required  Cap
	Principal string
}

func (e *Error) Error() string {
	if e.Principal == "" {
		return fmt.Sprintf("permission denied: requires %s", e.Required)
	}
	return fmt.Sprintf("permission denied: %s lacks %s", e.Principal, e.Required)
}

// --- Process-wide default ------------------------------------------------

// The desktop binary runs under the human user — for now we grant CapAll
// at boot. Agents will be given scoped tokens once the agent supervisor
// learns to attach them. Storing the "current" token globally so
// non-context-aware call sites (most existing bindings) can still gate.
var (
	mu      sync.RWMutex
	current = AllToken("desktop-user")
)

// Current returns the process-wide token. Treat as a temporary fallback
// while individual bindings get migrated to per-call ctx tokens.
func Current() Token {
	mu.RLock()
	defer mu.RUnlock()
	return current
}

// SetCurrent overrides the process-wide token (e.g. when the user
// downgrades to an agent-scoped role for testing).
func SetCurrent(t Token) {
	mu.Lock()
	defer mu.Unlock()
	current = t
}

// RequireCurrent is a convenience for bindings that don't have a ctx
// (most existing ones don't). It checks the process-wide current token.
func RequireCurrent(c Cap) error {
	if Current().Has(c) {
		return nil
	}
	return &Error{Required: c, Principal: Current().Principal}
}
