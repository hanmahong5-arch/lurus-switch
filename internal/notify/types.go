// Package notify is Switch's outbound notification layer — the bridge
// from in-process events (livesession watcher, bashguard hits, budget
// crossings, etc.) to messaging platforms the user already lives in
// (Feishu, Telegram, Slack, …).
//
// Design borrowed from cc-connect's bridge pattern, clean-room rewritten:
//
//	┌──────────────────────┐   Publish(Event)   ┌──────────────────┐
//	│ rules engine         │ ─────────────────▶ │ Bus              │
//	│ livesession.Watcher  │                    │ (fanout)         │
//	│ bashguard hook       │                    └────────┬─────────┘
//	└──────────────────────┘                             │
//	                                          Subscribe ▼
//	                                         ┌──────────────────┐
//	                                         │ Transport(s)     │
//	                                         │  ├─ feishu       │
//	                                         │  ├─ telegram (TBD)│
//	                                         │  └─ slack    (TBD)│
//	                                         └──────────────────┘
//
// Approval events are the interactive subclass: the publisher provides an
// ID and a reply channel; whichever transport surfaces a button-tap (or
// the timeout fires) sends the decision back through the channel.
package notify

import "time"

// Severity classifies an event's urgency. Transports translate this to
// colour/icon conventions native to each platform (Feishu colour cards,
// Slack attachment colours, Telegram emoji prefixes).
type Severity string

const (
	SeverityInfo    Severity = "info"
	SeveritySuccess Severity = "success"
	SeverityWarning Severity = "warning"
	SeverityError   Severity = "error"
)

// Kind tags the source of the event so transports can ignore types they
// don't care about (the user can also filter by kind in settings).
type Kind string

const (
	// KindToolStuck — a pending tool_use has been running longer than the
	// configured threshold (default 60s).
	KindToolStuck Kind = "tool_stuck"
	// KindSessionDone — a session went from active to idle for >5min, i.e.
	// "Claude finished and is waiting for you".
	KindSessionDone Kind = "session_done"
	// KindBudgetAlert — cumulative cost crossed a configured ceiling.
	KindBudgetAlert Kind = "budget_alert"
	// KindBashGuardApproval — a Bash-Guard rule matched a dangerous
	// command and is requesting human approval to allow it.
	KindBashGuardApproval Kind = "bashguard_approval"
	// KindTest — synthetic event the Settings page sends to verify the
	// transport credentials are wired correctly.
	KindTest Kind = "test"
)

// Event is the unit the bus moves around. Concrete fields stay flat
// because every transport's payload codec ends up referencing them by
// name; nesting would mean every transport learning two shapes.
type Event struct {
	ID       string    `json:"id"`       // stable identifier, used for dedup + approval round-trip
	Time     time.Time `json:"time"`
	Kind     Kind      `json:"kind"`
	Severity Severity  `json:"severity"`
	Title    string    `json:"title"`    // one-line headline (will be the card title)
	Body     string    `json:"body"`     // optional detail (≤ 500 chars; transports may truncate further)

	// Project + Tool are surfaced as a small label row on each platform so
	// the user can tell at a glance which session triggered the event.
	Project string `json:"project,omitempty"`
	Tool    string `json:"tool,omitempty"`

	// Approval is non-nil only for KindBashGuardApproval events. The
	// publisher fills this with a fresh channel; whichever side observes
	// the user's tap (or the timeout) sends a Decision through it.
	Approval *ApprovalRequest `json:"-"`
}

// ApprovalRequest is the interactive subclass of Event. The button-tap
// from the messaging platform comes back through Reply; if no one taps
// before the publisher's deadline, the publisher closes Reply itself.
type ApprovalRequest struct {
	// Command + Reason describe what's being approved — for rendering only.
	Command string
	Reason  string
	RuleID  string
	// Reply is a buffered channel (size 1). Transports send the user's
	// choice here. The publisher reads with a context-bounded select.
	Reply chan Decision
}

// Decision is what comes back from a tap. Allow / Block are obvious;
// Timeout means no one acted before the publisher's deadline (the safer
// default is to treat that as Block, but the publisher decides).
type Decision string

const (
	DecisionAllow   Decision = "allow"
	DecisionBlock   Decision = "block"
	DecisionTimeout Decision = "timeout"
)
