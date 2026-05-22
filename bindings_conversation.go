package main

import (
	"fmt"
	"os"
	"path/filepath"

	"lurus-switch/internal/audit"
	"lurus-switch/internal/conversation"
)

// ConversationEvents bundles a session's parsed events with the index
// metadata. Returned by GetConversation so the frontend can render the
// Timeline + header in one round-trip.
type ConversationEvents struct {
	Meta   conversation.ConversationMeta `json:"meta"`
	Events []conversation.Event          `json:"events"`
}

// ListConversations returns indexed session metadata, filtered. A fresh
// reindex runs in the background if the index is empty so first-launch
// users see something within seconds.
func (a *App) ListConversations(filter conversation.ConversationFilter) ([]conversation.ConversationMeta, error) {
	if a.conversationIndex == nil {
		return nil, fmt.Errorf("conversation index not initialised")
	}
	rows := a.conversationIndex.List(filter)
	if len(rows) == 0 {
		// Cold start — kick off the first reindex and return whatever
		// the immediate list call has.
		go safeGo("conversation-reindex-cold", func() { a.conversationIndex.Rebuild() })
	}
	a.stampDLPHitsOnto(rows)
	return rows, nil
}

// GetConversation parses one session's JSONL on demand. The result is
// not cached — sessions are append-only, so re-reading is cheap and
// always current.
func (a *App) GetConversation(tool, sessionID string) (*ConversationEvents, error) {
	if a.conversationIndex == nil {
		return nil, fmt.Errorf("conversation index not initialised")
	}
	meta, ok := a.conversationIndex.Get(tool, sessionID)
	if !ok {
		return nil, fmt.Errorf("conversation %s/%s not in index — try Reindex", tool, sessionID)
	}
	events, err := conversation.ParseFile(meta.Path)
	if err != nil {
		return nil, fmt.Errorf("parse session: %w", err)
	}
	return &ConversationEvents{Meta: meta, Events: events}, nil
}

// ReindexConversations forces a full discovery + reparse pass. Returns a
// summary the UI surfaces as a toast.
func (a *App) ReindexConversations() (*conversation.ReindexResult, error) {
	if a.conversationIndex == nil {
		return nil, fmt.Errorf("conversation index not initialised")
	}
	r := a.conversationIndex.Rebuild()
	return &r, nil
}

// ExportConversation writes a Markdown or JSON dump of the session to
// appDataDir/conversation-exports/. Returns the absolute path.
func (a *App) ExportConversation(tool, sessionID, format string, redactToolResults bool) (string, error) {
	conv, err := a.GetConversation(tool, sessionID)
	if err != nil {
		return "", err
	}
	outDir := filepath.Join(appDataBaseDir(), "conversation-exports")
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return "", err
	}
	f := conversation.ExportMarkdown
	if format == "json" {
		f = conversation.ExportJSON
	}
	return conversation.Export(conv.Meta, conv.Events, conversation.ExportOptions{
		Format:            f,
		OutputDir:         outDir,
		RedactToolResults: redactToolResults,
	})
}

// GetDLPHitsForSession returns the audit entries whose metadata maps the
// (tool, sessionID) we were given. The hot ring caps at 500 entries so
// this is bounded; older hits live in the daily NDJSON cold files which
// we don't replay yet (the audit page already gives full access).
func (a *App) GetDLPHitsForSession(tool, sessionID string) ([]audit.Entry, error) {
	if a.auditJournal == nil {
		return []audit.Entry{}, nil
	}
	match := map[string]string{
		conversation.MetaTool:      tool,
		conversation.MetaSessionID: sessionID,
	}
	return a.auditJournal.EntriesWithMetadata(match), nil
}

// GetSessionsForDLPHit takes an audit entry ID and resolves the session
// it points to via its metadata map. Returns the indexed meta when known,
// or an empty slice when the entry has no session correlation stamped.
func (a *App) GetSessionsForDLPHit(entryID string) ([]conversation.ConversationMeta, error) {
	if a.auditJournal == nil || a.conversationIndex == nil {
		return []conversation.ConversationMeta{}, nil
	}
	entry, ok := a.auditJournal.EntryByID(entryID)
	if !ok {
		return []conversation.ConversationMeta{}, nil
	}
	if entry.Metadata == nil {
		return []conversation.ConversationMeta{}, nil
	}
	tool := entry.Metadata[conversation.MetaTool]
	sid := entry.Metadata[conversation.MetaSessionID]
	if tool == "" || sid == "" {
		return []conversation.ConversationMeta{}, nil
	}
	if meta, ok := a.conversationIndex.Get(tool, sid); ok {
		return []conversation.ConversationMeta{meta}, nil
	}
	return []conversation.ConversationMeta{}, nil
}

// GetProjectContextFiles enumerates CLAUDE.md / AGENTS.md / .cursorrules
// under the given working directory. Used by the session detail drawer.
func (a *App) GetProjectContextFiles(cwd string) []conversation.ContextFile {
	return conversation.FindContextFiles(cwd)
}

// ForkConversation duplicates the JSONL up to and including messageUUID
// into a new session file in the same project directory. The new session
// is picked up by the Claude CLI via `claude --resume <newSessionID>`.
// Records parentage in a sibling .lurus.json sidecar.
func (a *App) ForkConversation(tool, sessionID, messageUUID string) (*conversation.ForkResult, error) {
	if a.conversationIndex == nil {
		return nil, fmt.Errorf("conversation index not initialised")
	}
	meta, ok := a.conversationIndex.Get(tool, sessionID)
	if !ok {
		return nil, fmt.Errorf("conversation %s/%s not in index", tool, sessionID)
	}
	parent := conversation.SessionFile{
		Tool:      tool,
		SessionID: sessionID,
		Path:      meta.Path,
		Cwd:       meta.Cwd,
	}
	res, err := conversation.Fork(parent, messageUUID)
	if err != nil {
		return nil, err
	}
	// Reindex in the background so the new child shows up in the next
	// ListConversations call. Cheap because it's mtime-driven.
	go safeGo("conversation-reindex-after-fork", func() { a.conversationIndex.Rebuild() })
	return &res, nil
}

// stampDLPHitsOnto enriches a freshly-listed row set with HasDLPHits.
// O(rows × entries) but both are bounded (rows ≤ index size, entries ≤
// 500) so this stays cheap.
func (a *App) stampDLPHitsOnto(rows []conversation.ConversationMeta) {
	if a.auditJournal == nil {
		return
	}
	for i := range rows {
		hits := a.auditJournal.EntriesWithMetadata(map[string]string{
			conversation.MetaTool:      rows[i].Tool,
			conversation.MetaSessionID: rows[i].SessionID,
		})
		rows[i].HasDLPHits = len(hits) > 0
	}
}
