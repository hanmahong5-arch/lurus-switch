package main

import (
	"lurus-switch/internal/audit"
	"lurus-switch/internal/capability"
)

// requireAndAudit checks the capability gate and, regardless of grant
// outcome, records the attempt to the audit journal. Returns a non-nil
// error when the gate denies — callers must early-return on err so the
// underlying operation never runs.
//
// The pattern at every wrapped binding is:
//
//	func (a *App) DangerousOp(input X) (err error) {
//	    if err = a.requireAndAudit(capability.CapXxx, "domain.action", target, input); err != nil {
//	        return err
//	    }
//	    defer func() { a.recordOutcome("domain.action", target, input, err) }()
//	    ...do the work...
//	}
//
// Two records are emitted: "denied" (or pre-write attempt) and the
// final outcome via recordOutcome (with error or after-state).
func (a *App) requireAndAudit(cap capability.Cap, op, target string, input any) error {
	if err := capability.RequireCurrent(cap); err != nil {
		if a.auditJournal != nil {
			a.auditJournal.Record(op, target, nil, input, err)
		}
		return err
	}
	return nil
}

// recordOutcome journals the post-call state (or error). Safe to call
// from `defer` — no-ops if the journal is nil.
func (a *App) recordOutcome(op, target string, after any, err error) {
	if a.auditJournal == nil {
		return
	}
	a.auditJournal.Record(op, target, nil, after, err)
}

// auditOp is the single sentinel both the binding wrapper and the
// frontend agree on for op identifiers. Centralizing here so they
// don't drift.
const (
	auditOpChannelCreate      = "channel.create"
	auditOpChannelUpdate      = "channel.update"
	auditOpChannelDelete      = "channel.delete"
	auditOpChannelDeleteBatch = "channel.delete_batch"
	auditOpTokenCreate        = "token.create"
	auditOpTokenUpdate        = "token.update"
	auditOpTokenDelete        = "token.delete"
	auditOpTokenDeleteBatch   = "token.delete_batch"
	auditOpRedemptionDelete   = "redemption.delete"
	auditOpModelSwitch        = "model.switch"
)

// Tiny aliases so binding files don't need to import the capability
// package directly.
func capChannelWrite() capability.Cap { return capability.CapChannelWrite }
func capTokenCreate() capability.Cap  { return capability.CapTokenCreate }
func capTokenRevoke() capability.Cap  { return capability.CapTokenRevoke }
func capRedemptionDelete() capability.Cap { return capability.CapRedemptionDelete }
func capPricingWrite() capability.Cap     { return capability.CapPricingWrite }

// stringField extracts a string-coerced field from a free-form map.
// Used to derive audit "target" identifiers from request payloads
// without panicking on missing or wrong-typed keys.
func stringField(m map[string]any, key string) string {
	if m == nil {
		return ""
	}
	v, ok := m[key]
	if !ok || v == nil {
		return ""
	}
	switch t := v.(type) {
	case string:
		return t
	case int:
		return fmtIntID(t)
	case int64:
		return fmtIntID(int(t))
	case float64:
		return fmtIntID(int(t))
	default:
		return ""
	}
}

func fmtIntID(id int) string {
	return _itoa(id)
}

// _itoa is a stripped-down Itoa to avoid importing strconv just for
// audit target formatting. Negative IDs are unexpected (DB IDs are
// positive integers) so we don't handle them specially.
func _itoa(n int) string {
	if n == 0 {
		return "0"
	}
	negative := n < 0
	if negative {
		n = -n
	}
	buf := make([]byte, 0, 12)
	for n > 0 {
		buf = append(buf, byte('0'+n%10))
		n /= 10
	}
	// reverse
	for i, j := 0, len(buf)-1; i < j; i, j = i+1, j-1 {
		buf[i], buf[j] = buf[j], buf[i]
	}
	if negative {
		return "-" + string(buf)
	}
	return string(buf)
}

// Suppress unused-import linting.
var _ = audit.Stats{}
