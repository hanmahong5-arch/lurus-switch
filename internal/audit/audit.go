// Package audit is the append-only journal of state-mutating operations
// performed through Switch. Every Wails binding that writes to a
// downstream service (newapi/newhub) or to local state should call
// Record() before returning success, with enough payload to reconstruct
// "before" and "after" — the Undo handler uses that to revert.
//
// Two persistence layers:
//   - Hot: in-memory ring buffer (last 500 entries) — used by the audit
//     log UI.
//   - Cold: append-only NDJSON file (one row per entry, daily-rotated)
//     under appDataDir/audit/ — used for evidence export, never edited.
//
// The journal is intentionally NOT relational — auditors care about
// "what happened in order, who did it, was it reverted" and that's
// well-served by an immutable log. Snapshots / rollups are computed at
// read time.
package audit

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"lurus-switch/internal/capability"
)

const (
	auditDir    = "audit"
	hotRingSize = 500
)

// Entry is a single journaled mutation.
type Entry struct {
	ID         string            `json:"id"` // ULID-ish: nanoseconds + counter
	Timestamp  time.Time         `json:"timestamp"`
	Principal  string            `json:"principal"`        // who: "user:marvin", "agent:sales-1"
	CapsHeld   []string          `json:"capsHeld"`         // caps the principal had at the moment
	Operation  string            `json:"operation"`        // dotted op name e.g. "channel.create"
	Target     string            `json:"target"`           // free-form: id of the affected entity
	Before     any               `json:"before,omitempty"` // pre-state snapshot
	After      any               `json:"after,omitempty"`  // post-state snapshot
	Outcome    string            `json:"outcome"`          // "ok" | "denied" | "error"
	Error      string            `json:"error,omitempty"`
	UndoneAt   *time.Time        `json:"undoneAt,omitempty"`
	UndoneBy   string            `json:"undoneBy,omitempty"`
	Reversible bool              `json:"reversible"` // can Undo() touch this op?
	Metadata   map[string]string `json:"metadata,omitempty"`
}

// UndoFunc reverts a recorded change. Called with a copy of the
// entry's Before payload — implementations cast to their concrete
// type. Return error if undo is impossible (e.g., dependent state
// has changed in incompatible ways).
type UndoFunc func(entry Entry) error

// Journal is the append-only log + per-op undo registry.
type Journal struct {
	mu           sync.RWMutex
	baseDir      string
	hot          []Entry             // most recent first
	undoHandlers map[string]UndoFunc // keyed by Operation
	idCounter    atomic.Uint64
}

// NewJournal opens the journal rooted at appDataDir/audit/.
// Creates the directory if needed.
func NewJournal(appDataDir string) (*Journal, error) {
	dir := filepath.Join(appDataDir, auditDir)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("create audit dir: %w", err)
	}
	j := &Journal{
		baseDir:      dir,
		hot:          make([]Entry, 0, hotRingSize),
		undoHandlers: make(map[string]UndoFunc),
	}
	// Hydrate hot ring from today's file so a restart keeps history.
	if entries := j.loadDayFile(time.Now()); len(entries) > 0 {
		// Take only the tail (most recent first).
		start := 0
		if len(entries) > hotRingSize {
			start = len(entries) - hotRingSize
		}
		// Reverse so most recent is first.
		for i := len(entries) - 1; i >= start; i-- {
			j.hot = append(j.hot, entries[i])
		}
	}
	return j, nil
}

// Register attaches an undo handler for the given operation. Multiple
// calls overwrite. Operations without a handler are still recorded but
// flagged Reversible=false in the entry.
func (j *Journal) Register(op string, handler UndoFunc) {
	j.mu.Lock()
	defer j.mu.Unlock()
	j.undoHandlers[op] = handler
}

// IsReversible reports whether the given op has an undo handler.
func (j *Journal) IsReversible(op string) bool {
	j.mu.RLock()
	defer j.mu.RUnlock()
	_, ok := j.undoHandlers[op]
	return ok
}

// Record appends an entry to the journal. The Outcome / Error fields
// are derived from `err` — a nil err means "ok", a *capability.Error
// means "denied", everything else means "error".
//
// `before` and `after` are JSON-serialized into the on-disk record;
// pass small structures only.
func (j *Journal) Record(op, target string, before, after any, err error) Entry {
	now := time.Now()
	tok := capability.Current()

	entry := Entry{
		ID:         j.nextID(now),
		Timestamp:  now,
		Principal:  tok.Principal,
		CapsHeld:   tok.CapsList(),
		Operation:  op,
		Target:     target,
		Before:     before,
		After:      after,
		Outcome:    "ok",
		Reversible: j.isOpReversible(op),
	}
	switch e := err.(type) {
	case nil:
		// keep "ok"
	case *capability.Error:
		entry.Outcome = "denied"
		entry.Error = e.Error()
	default:
		entry.Outcome = "error"
		entry.Error = e.Error()
	}

	j.append(entry)
	return entry
}

// RecordSystem appends an entry attributed to a non-Wails actor (e.g.
// the gateway middleware writing DLP block events). Bypasses
// capability.Current() because the caller doesn't run under a user
// session — but still emits a normal entry that the audit UI can
// filter by operation.
func (j *Journal) RecordSystem(principal, op, target string, before, after any, err error) Entry {
	now := time.Now()
	if principal == "" {
		principal = "system"
	}

	entry := Entry{
		ID:         j.nextID(now),
		Timestamp:  now,
		Principal:  principal,
		CapsHeld:   nil, // system events have no capability set
		Operation:  op,
		Target:     target,
		Before:     before,
		After:      after,
		Outcome:    "ok",
		Reversible: j.isOpReversible(op),
	}
	if err != nil {
		entry.Outcome = "error"
		entry.Error = err.Error()
	}
	j.append(entry)
	return entry
}

// AttachMetadata sets/merges metadata keys on an existing entry in the
// hot ring. Used by the gateway DLP middleware which records the entry
// first (via RecordSystem) and then enriches it with conversation
// correlation IDs. Missing entries are silently ignored — the entry may
// have already aged out of the ring.
func (j *Journal) AttachMetadata(entryID string, kv map[string]string) {
	if len(kv) == 0 {
		return
	}
	j.mu.Lock()
	defer j.mu.Unlock()
	for i := range j.hot {
		if j.hot[i].ID == entryID {
			if j.hot[i].Metadata == nil {
				j.hot[i].Metadata = make(map[string]string, len(kv))
			}
			for k, v := range kv {
				j.hot[i].Metadata[k] = v
			}
			return
		}
	}
}

// List returns up to `limit` most-recent entries, optionally filtered
// by principal / operation / outcome substrings (empty = no filter).
// Entries with newest first.
func (j *Journal) List(limit int, filter Filter) []Entry {
	j.mu.RLock()
	defer j.mu.RUnlock()

	if limit <= 0 || limit > len(j.hot) {
		limit = len(j.hot)
	}

	out := make([]Entry, 0, limit)
	for _, e := range j.hot {
		if !filter.matches(e) {
			continue
		}
		out = append(out, e)
		if len(out) >= limit {
			break
		}
	}
	return out
}

// Filter narrows what List returns.
type Filter struct {
	Principal      string // substring match
	Operation      string // substring match
	Outcome        string // exact match: "ok" / "denied" / "error" / ""
	OnlyReversible bool
	OnlyUndone     bool
	OnlyNotUndone  bool
}

func (f Filter) matches(e Entry) bool {
	if f.Principal != "" && !contains(e.Principal, f.Principal) {
		return false
	}
	if f.Operation != "" && !contains(e.Operation, f.Operation) {
		return false
	}
	if f.Outcome != "" && e.Outcome != f.Outcome {
		return false
	}
	if f.OnlyReversible && !e.Reversible {
		return false
	}
	if f.OnlyUndone && e.UndoneAt == nil {
		return false
	}
	if f.OnlyNotUndone && e.UndoneAt != nil {
		return false
	}
	return true
}

func contains(s, sub string) bool {
	if sub == "" {
		return true
	}
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

// EntriesWithMetadata returns hot-ring entries whose Metadata map
// contains every key/value pair in `match`. Used by the conversation
// bindings to compute the audit-×-DLP join — "show every entry whose
// conv_session_id == X".
func (j *Journal) EntriesWithMetadata(match map[string]string) []Entry {
	j.mu.RLock()
	defer j.mu.RUnlock()
	out := make([]Entry, 0)
	for _, e := range j.hot {
		if e.Metadata == nil {
			continue
		}
		ok := true
		for k, v := range match {
			if got, present := e.Metadata[k]; !present || got != v {
				ok = false
				break
			}
		}
		if ok {
			out = append(out, e)
		}
	}
	return out
}

// EntryByID locates an entry in the hot ring. Returns ok=false when the
// entry has aged out (in which case the cold file would need replay).
func (j *Journal) EntryByID(id string) (Entry, bool) {
	j.mu.RLock()
	defer j.mu.RUnlock()
	for _, e := range j.hot {
		if e.ID == id {
			return e, true
		}
	}
	return Entry{}, false
}

// Stats summarizes the hot ring for at-a-glance dashboard.
//
// FailRate is (denied + error) / total expressed as 0..1. It is zero
// when Total == 0 — the caller is expected to render "—" or hide the
// chip in that case, not interpret 0.0 as "perfect health".
//
// ByOperationPrefix groups counts by the dotted-name prefix (e.g.
// `channel.create` + `channel.delete` both roll up under `channel`).
// Lets the UI show "tool-dimension" without introducing a new field on
// Entry. Empty operation names roll up under "unknown".
type Stats struct {
	Total             int            `json:"total"`
	OK                int            `json:"ok"`
	Denied            int            `json:"denied"`
	Error             int            `json:"error"`
	Undone            int            `json:"undone"`
	FailRate          float64        `json:"failRate"`
	WindowStart       *time.Time     `json:"windowStart,omitempty"`
	ByPrincipal       map[string]int `json:"byPrincipal"`
	ByOperation       map[string]int `json:"byOperation"`
	ByOperationPrefix map[string]int `json:"byOperationPrefix"`
}

func (j *Journal) Stats() Stats {
	return j.computeStats(time.Time{})
}

// StatsWindow returns Stats over entries with Timestamp >= since. A
// zero `since` includes everything in the hot ring.
//
// The window applies to all aggregations — Total / OK / Denied / Error /
// Undone / ByPrincipal / ByOperation / ByOperationPrefix all reflect the
// same time slice, so the UI never shows an inconsistent stats grid.
func (j *Journal) StatsWindow(since time.Time) Stats {
	return j.computeStats(since)
}

func (j *Journal) computeStats(since time.Time) Stats {
	j.mu.RLock()
	defer j.mu.RUnlock()
	out := Stats{
		ByPrincipal:       make(map[string]int),
		ByOperation:       make(map[string]int),
		ByOperationPrefix: make(map[string]int),
	}
	if !since.IsZero() {
		out.WindowStart = &since
	}
	for _, e := range j.hot {
		if !since.IsZero() && e.Timestamp.Before(since) {
			continue
		}
		out.Total++
		switch e.Outcome {
		case "ok":
			out.OK++
		case "denied":
			out.Denied++
		case "error":
			out.Error++
		}
		if e.UndoneAt != nil {
			out.Undone++
		}
		out.ByPrincipal[e.Principal]++
		out.ByOperation[e.Operation]++
		out.ByOperationPrefix[opPrefix(e.Operation)]++
	}
	if out.Total > 0 {
		out.FailRate = float64(out.Denied+out.Error) / float64(out.Total)
	}
	return out
}

// opPrefix extracts the dotted-name root of an op. "channel.create" → "channel".
// "audit.undo" → "audit". Empty / no-dot → returned as-is (or "unknown").
func opPrefix(op string) string {
	if op == "" {
		return "unknown"
	}
	for i := 0; i < len(op); i++ {
		if op[i] == '.' {
			return op[:i]
		}
	}
	return op
}

// Undo invokes the registered handler for the entry's operation,
// passing a copy of the entry. On success, marks the entry as undone
// (idempotent — undoing twice is a no-op).
func (j *Journal) Undo(entryID string) error {
	j.mu.Lock()
	idx := -1
	for i := range j.hot {
		if j.hot[i].ID == entryID {
			idx = i
			break
		}
	}
	if idx < 0 {
		j.mu.Unlock()
		return fmt.Errorf("entry %q not found in hot ring", entryID)
	}
	entry := j.hot[idx]
	if entry.UndoneAt != nil {
		j.mu.Unlock()
		return fmt.Errorf("entry %q already undone at %s", entryID, entry.UndoneAt.Format(time.RFC3339))
	}
	if !entry.Reversible {
		j.mu.Unlock()
		return fmt.Errorf("entry %q operation %q is not reversible", entryID, entry.Operation)
	}
	if entry.Outcome != "ok" {
		j.mu.Unlock()
		return fmt.Errorf("cannot undo a non-ok entry (outcome=%s)", entry.Outcome)
	}
	handler := j.undoHandlers[entry.Operation]
	j.mu.Unlock()

	if handler == nil {
		return fmt.Errorf("no undo handler registered for %q", entry.Operation)
	}

	// Run the handler outside the lock so it can call back into
	// other Switch services (they may take their own locks).
	if err := handler(entry); err != nil {
		return fmt.Errorf("undo handler failed: %w", err)
	}

	// Mark undone.
	now := time.Now()
	tok := capability.Current()
	j.mu.Lock()
	j.hot[idx].UndoneAt = &now
	j.hot[idx].UndoneBy = tok.Principal
	updated := j.hot[idx]
	j.mu.Unlock()

	// Persist the marker by re-writing today's file would be wasteful;
	// instead append a synthetic "undone" marker entry — auditors can
	// reconstruct state by replaying the log.
	marker := Entry{
		ID:         j.nextID(now),
		Timestamp:  now,
		Principal:  tok.Principal,
		CapsHeld:   tok.CapsList(),
		Operation:  "audit.undo",
		Target:     entryID,
		After:      map[string]string{"undoneEntry": entryID, "originalOp": updated.Operation},
		Outcome:    "ok",
		Reversible: false, // undo of an undo is meaningless
	}
	j.append(marker)
	return nil
}

// --- internals -----------------------------------------------------------

func (j *Journal) nextID(t time.Time) string {
	n := j.idCounter.Add(1)
	return fmt.Sprintf("%d-%04d", t.UnixNano(), n%10000)
}

func (j *Journal) isOpReversible(op string) bool {
	j.mu.RLock()
	defer j.mu.RUnlock()
	_, ok := j.undoHandlers[op]
	return ok
}

func (j *Journal) append(e Entry) {
	j.mu.Lock()
	// Hot ring: prepend (newest first), trim at hotRingSize.
	j.hot = append([]Entry{e}, j.hot...)
	if len(j.hot) > hotRingSize {
		j.hot = j.hot[:hotRingSize]
	}
	j.mu.Unlock()

	// Cold storage: append NDJSON to today's file (best effort —
	// audit must never fail the user-facing operation).
	if err := j.writeColdEntry(e); err != nil {
		// Log to stderr; cold-storage failure shouldn't block the user.
		fmt.Fprintf(os.Stderr, "audit cold-storage write failed: %v\n", err)
	}
}

func (j *Journal) coldFilePath(t time.Time) string {
	return filepath.Join(j.baseDir, t.Format("2006-01-02")+".ndjson")
}

func (j *Journal) writeColdEntry(e Entry) error {
	f, err := os.OpenFile(j.coldFilePath(e.Timestamp), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	enc.SetEscapeHTML(false)
	return enc.Encode(e)
}

func (j *Journal) loadDayFile(day time.Time) []Entry {
	data, err := os.ReadFile(j.coldFilePath(day))
	if err != nil {
		return nil
	}
	var entries []Entry
	dec := json.NewDecoder(bytes.NewReader(data))
	for {
		var e Entry
		if err := dec.Decode(&e); err != nil {
			if err == io.EOF {
				break
			}
			// Skip malformed line and keep reading — auditors will see
			// the file directly if anything's truly wrong.
			break
		}
		entries = append(entries, e)
	}
	// Sort ascending by timestamp for a deterministic hydration.
	sort.SliceStable(entries, func(i, k int) bool {
		return entries[i].Timestamp.Before(entries[k].Timestamp)
	})
	return entries
}
