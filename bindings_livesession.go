package main

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	"lurus-switch/internal/conversation"
	"lurus-switch/internal/livesession"
)

// ============================
// Live-Session Inspector Bindings
// ============================
//
// Surfaces the in-process Watcher to the frontend. The watcher itself runs
// continuously from app startup (see app.go); the frontend pulls via
// GetLiveSessions on initial load and again on every "livesession:update"
// event the backend emits.

// GetLiveSessions returns every active session within the watcher's
// default activity window. Use GetAllLiveSessions to include idle history.
func (a *App) GetLiveSessions() []livesession.LiveSession {
	if a.liveWatcher == nil {
		return []livesession.LiveSession{}
	}
	return a.liveWatcher.SnapshotActive()
}

// GetAllLiveSessions returns every known session — including idle ones
// kept in memory for resume. The Live Inspector page uses this when the
// "show idle" toggle is on.
func (a *App) GetAllLiveSessions() []livesession.LiveSession {
	if a.liveWatcher == nil {
		return []livesession.LiveSession{}
	}
	return a.liveWatcher.Snapshot()
}

// transcriptMaxEvents caps the slice returned to the frontend. Real
// sessions easily run into thousands of JSONL lines; ferrying that across
// the WebView IPC bridge and rendering it in React both choke. 500 is
// roughly "one productive afternoon of pair-programming with a CLI" and
// keeps the drawer responsive. We always keep the tail (newest events) —
// it's the most interesting end for a live session.
const transcriptMaxEvents = 500

// allowedTranscriptRoots lists the directory roots a Wails caller is
// permitted to read transcript JSONLs from. Every other location is
// refused. This is defence-in-depth: an attacker who landed XSS inside
// the Wails WebView could otherwise pass arbitrary paths and exfiltrate
// any file on the user's disk via this binding.
//
// Returned paths are absolute, already symlink-resolved.
func allowedTranscriptRoots() []string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return nil
	}
	candidates := []string{
		filepath.Join(home, ".claude", "projects"),
		filepath.Join(home, ".codex", "sessions"),
		filepath.Join(home, ".gemini", "sessions"),
	}
	var roots []string
	for _, c := range candidates {
		// EvalSymlinks fails on non-existent paths, which is fine — we
		// just skip those rather than refuse all access.
		resolved, err := filepath.EvalSymlinks(c)
		if err != nil {
			roots = append(roots, filepath.Clean(c))
			continue
		}
		roots = append(roots, resolved)
	}
	return roots
}

// isUnderRoot reports whether `path` is contained within `root`, comparing
// the cleaned absolute forms. filepath.Rel returns a relative path that
// starts with ".." when path is outside root, which is what we screen on.
func isUnderRoot(path, root string) bool {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return false
	}
	if rel == "." {
		return true
	}
	if strings.HasPrefix(rel, "..") {
		return false
	}
	if strings.Contains(rel, ".."+string(os.PathSeparator)) {
		return false
	}
	return true
}

// GetSessionTranscript returns the parsed JSONL transcript for a single
// live session. Returns at most the last `transcriptMaxEvents` entries
// (oldest of those first) so the WebView stays responsive on long
// sessions. The path is refused unless it resolves to a location under
// the user's `~/.claude/projects`, `~/.codex/sessions`, or
// `~/.gemini/sessions` tree.
func (a *App) GetSessionTranscript(path string) ([]conversation.Event, error) {
	if path == "" {
		return nil, errors.New("transcript path is empty")
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	// EvalSymlinks first so a symlink planted inside the allow-listed
	// tree can't be used to escape it. If the file doesn't exist yet,
	// Abs+Clean is the best we can do — the ParseFile call below will
	// surface the real "no such file" error to the caller.
	resolved, err := filepath.EvalSymlinks(abs)
	if err != nil {
		resolved = filepath.Clean(abs)
	}
	roots := allowedTranscriptRoots()
	if len(roots) == 0 {
		return nil, errors.New("no allowed transcript roots resolvable on this host")
	}
	allowed := false
	for _, root := range roots {
		if isUnderRoot(resolved, root) {
			allowed = true
			break
		}
	}
	if !allowed {
		return nil, errors.New("transcript path is outside the allowed CLI session directories")
	}

	events, err := conversation.ParseFile(resolved)
	if err != nil {
		return nil, err
	}
	if len(events) > transcriptMaxEvents {
		events = events[len(events)-transcriptMaxEvents:]
	}
	return events, nil
}
