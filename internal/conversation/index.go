package conversation

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

const indexFileName = "conversation-index.json"

// ConversationMeta is the row the index stores per session. The frontend
// uses this for facet filtering — no need to crack the JSONL until the
// user actually opens a session.
type ConversationMeta struct {
	Tool            string    `json:"tool"`
	SessionID       string    `json:"sessionID"`
	Cwd             string    `json:"cwd,omitempty"`
	Path            string    `json:"path"`
	Model           string    `json:"model,omitempty"`
	StartedAt       time.Time `json:"startedAt,omitempty"`
	EndedAt         time.Time `json:"endedAt,omitempty"`
	MessageCount    int       `json:"messageCount"`
	UserMessages    int       `json:"userMessages"`
	AssistantMessages int     `json:"assistantMessages"`
	TotalTokens     int64     `json:"totalTokens"`
	ToolList        []string  `json:"toolList,omitempty"`
	HasErrors       bool      `json:"hasErrors"`
	HasDLPHits      bool      `json:"hasDLPHits,omitempty"` // set by audit-join at read time

	// Fork metadata (P3). Populated from the sibling .lurus.json file.
	ParentSessionID string `json:"parentSessionID,omitempty"`
	ForkPointUUID   string `json:"forkPointUUID,omitempty"`

	// mtime of the underlying JSONL — used to decide incremental rebuild.
	FileModTime int64 `json:"fileModTime"`
	FileSize    int64 `json:"fileSize"`
}

// ConversationFilter narrows ListConversations results.
type ConversationFilter struct {
	Tool         string `json:"tool"`         // exact match; "" = any
	CwdSubstring string `json:"cwdSubstring"` // case-sensitive substring; "" = any
	Model        string `json:"model"`        // exact; "" = any
	StartAfter   string `json:"startAfter"`   // RFC3339; "" = any
	EndBefore    string `json:"endBefore"`    // RFC3339; "" = any
	OnlyDLPHits  bool   `json:"onlyDLPHits"`
	Search       string `json:"search"`       // SessionID or cwd substring
}

// Index is the on-disk catalogue of every session JSONL Switch has seen.
// Rebuild is mtime-driven so a 5,000-session history doesn't re-parse
// every poll. Reads use a copy-on-write snapshot so List doesn't block
// concurrent Rebuilds.
type Index struct {
	mu       sync.RWMutex
	path     string
	rows     []ConversationMeta
}

// NewIndex opens the on-disk index at appDataDir/conversation-index.json,
// creating it if missing. Returns immediately — first Rebuild is the
// caller's responsibility (and runs in the background).
func NewIndex(appDataDir string) (*Index, error) {
	if err := os.MkdirAll(appDataDir, 0o755); err != nil {
		return nil, fmt.Errorf("conversation index: mkdir: %w", err)
	}
	idx := &Index{path: filepath.Join(appDataDir, indexFileName)}
	if err := idx.load(); err != nil && !os.IsNotExist(err) {
		// Corrupt index is non-fatal — we'll rebuild fresh.
		idx.rows = nil
	}
	return idx, nil
}

func (i *Index) load() error {
	data, err := os.ReadFile(i.path)
	if err != nil {
		return err
	}
	var rows []ConversationMeta
	if err := json.Unmarshal(data, &rows); err != nil {
		return err
	}
	i.rows = rows
	return nil
}

func (i *Index) save() error {
	data, err := json.MarshalIndent(i.rows, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(i.path, data, 0o600)
}

// ReindexResult summarises what Rebuild did. Surfaced through the Wails
// binding so the UI can render a "scanned N, added M, updated K" toast.
type ReindexResult struct {
	Scanned int `json:"scanned"`
	Added   int `json:"added"`
	Updated int `json:"updated"`
	Removed int `json:"removed"`
	Errors  int `json:"errors"`
}

// Rebuild walks every supported tool's session directory and refreshes
// the index incrementally: existing rows with unchanged mtime are kept
// as-is; new or changed JSONLs are reparsed; deleted files are dropped.
func (i *Index) Rebuild() ReindexResult {
	files := DiscoverAll()
	res := ReindexResult{Scanned: len(files)}

	i.mu.Lock()
	defer i.mu.Unlock()

	prev := make(map[string]ConversationMeta, len(i.rows))
	for _, r := range i.rows {
		prev[indexKey(r.Tool, r.SessionID)] = r
	}

	seen := make(map[string]struct{}, len(files))
	var next []ConversationMeta
	for _, f := range files {
		k := indexKey(f.Tool, f.SessionID)
		seen[k] = struct{}{}
		if existing, ok := prev[k]; ok && existing.FileModTime == f.ModTime && existing.FileSize == f.Size {
			next = append(next, existing)
			continue
		}
		meta, err := summarize(f)
		if err != nil {
			res.Errors++
			continue
		}
		if _, ok := prev[k]; ok {
			res.Updated++
		} else {
			res.Added++
		}
		next = append(next, meta)
	}
	for k := range prev {
		if _, ok := seen[k]; !ok {
			res.Removed++
		}
	}

	// Sort newest-first by EndedAt (falling back to mtime) so the UI
	// shows recent sessions without re-sorting client-side.
	sort.SliceStable(next, func(a, b int) bool {
		ta := next[a].EndedAt
		if ta.IsZero() {
			ta = time.Unix(0, next[a].FileModTime)
		}
		tb := next[b].EndedAt
		if tb.IsZero() {
			tb = time.Unix(0, next[b].FileModTime)
		}
		return ta.After(tb)
	})

	i.rows = next
	_ = i.save()
	return res
}

func indexKey(tool, sessionID string) string {
	return tool + "::" + sessionID
}

// summarize parses a session file just deeply enough to fill in the
// per-session metadata. It does NOT keep the events in memory after
// the function returns — large sessions are too big to hoard.
func summarize(f SessionFile) (ConversationMeta, error) {
	events, err := ParseFile(f.Path)
	if err != nil {
		return ConversationMeta{}, err
	}
	meta := ConversationMeta{
		Tool:        f.Tool,
		SessionID:   f.SessionID,
		Cwd:         f.Cwd,
		Path:        f.Path,
		FileModTime: f.ModTime,
		FileSize:    f.Size,
	}
	toolSet := map[string]struct{}{}
	for _, e := range events {
		meta.MessageCount++
		if e.Timestamp.IsZero() {
			// skip timestamp tracking but still count
		} else if meta.StartedAt.IsZero() || e.Timestamp.Before(meta.StartedAt) {
			meta.StartedAt = e.Timestamp
		}
		if e.Timestamp.After(meta.EndedAt) {
			meta.EndedAt = e.Timestamp
		}
		switch e.Type {
		case EventUser:
			meta.UserMessages++
		case EventAssistant:
			meta.AssistantMessages++
		case EventToolUse:
			if e.ToolName != "" {
				toolSet[e.ToolName] = struct{}{}
			}
		}
		if e.Model != "" && meta.Model == "" {
			meta.Model = e.Model
		}
		meta.TotalTokens += e.InputTokens + e.OutputTokens
	}
	for name := range toolSet {
		meta.ToolList = append(meta.ToolList, name)
	}
	sort.Strings(meta.ToolList)

	// Sidecar metadata (P3 fork bookkeeping). Cheap stat — skipped on miss.
	if sc, err := readForkSidecar(f.Path); err == nil {
		meta.ParentSessionID = sc.ParentSessionID
		meta.ForkPointUUID = sc.ForkPointUUID
	}

	return meta, nil
}

// List returns a filtered, newest-first slice of conversations. The
// underlying slice is never mutated — callers may hold the result safely.
func (i *Index) List(filter ConversationFilter) []ConversationMeta {
	i.mu.RLock()
	defer i.mu.RUnlock()

	var startAfter, endBefore time.Time
	if filter.StartAfter != "" {
		startAfter, _ = time.Parse(time.RFC3339, filter.StartAfter)
	}
	if filter.EndBefore != "" {
		endBefore, _ = time.Parse(time.RFC3339, filter.EndBefore)
	}

	out := make([]ConversationMeta, 0, len(i.rows))
	for _, r := range i.rows {
		if filter.Tool != "" && r.Tool != filter.Tool {
			continue
		}
		if filter.CwdSubstring != "" && !containsFold(r.Cwd, filter.CwdSubstring) {
			continue
		}
		if filter.Model != "" && r.Model != filter.Model {
			continue
		}
		if !startAfter.IsZero() && r.StartedAt.Before(startAfter) {
			continue
		}
		if !endBefore.IsZero() && !r.EndedAt.IsZero() && r.EndedAt.After(endBefore) {
			continue
		}
		if filter.OnlyDLPHits && !r.HasDLPHits {
			continue
		}
		if filter.Search != "" && !containsFold(r.SessionID, filter.Search) && !containsFold(r.Cwd, filter.Search) {
			continue
		}
		out = append(out, r)
	}
	return out
}

// Get returns the indexed metadata for one session, or false if missing.
func (i *Index) Get(tool, sessionID string) (ConversationMeta, bool) {
	i.mu.RLock()
	defer i.mu.RUnlock()
	for _, r := range i.rows {
		if r.Tool == tool && r.SessionID == sessionID {
			return r, true
		}
	}
	return ConversationMeta{}, false
}

// MarkDLPHits stamps the HasDLPHits flag for the listed (tool,sessionID)
// pairs. Called by the binding layer after joining against the audit
// journal — saves the join cost on every List call.
func (i *Index) MarkDLPHits(hits map[string]bool) {
	i.mu.Lock()
	defer i.mu.Unlock()
	for k, v := range hits {
		for idx := range i.rows {
			if indexKey(i.rows[idx].Tool, i.rows[idx].SessionID) == k {
				i.rows[idx].HasDLPHits = v
			}
		}
	}
}

func containsFold(s, sub string) bool {
	if sub == "" {
		return true
	}
	ls := toLower(s)
	lsub := toLower(sub)
	for i := 0; i+len(lsub) <= len(ls); i++ {
		if ls[i:i+len(lsub)] == lsub {
			return true
		}
	}
	return false
}

func toLower(s string) string {
	out := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			c += 'a' - 'A'
		}
		out[i] = c
	}
	return string(out)
}
