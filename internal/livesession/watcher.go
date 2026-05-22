package livesession

import (
	"sync"
	"time"

	"lurus-switch/internal/conversation"
)

// pollInterval is how often Watcher re-walks the projects directory and
// tails files for new lines. 2s is the sweet spot — fast enough that the
// UI feels live, slow enough to keep CPU under 1% with dozens of stale
// sessions on disk.
const pollInterval = 2 * time.Second

// staleAfter is the age at which a session stops getting polled. We keep
// the in-memory state around so a resumed session reattaches cheaply, but
// active polling is reserved for files that look alive.
const staleAfter = 30 * time.Minute

// activeWindow is the cut-off for "show this in the UI by default".
// Sessions older than this slide to the inactive shelf.
const activeWindow = 5 * time.Minute

// Watcher polls Claude/Codex/Gemini transcript directories and folds new
// events into per-session live state. Construct via New; call Start once;
// call Stop to release the polling goroutine.
type Watcher struct {
	mu       sync.RWMutex
	sessions map[string]*sessionState // key: file path
	offsets  map[string]int64         // bytes read per path

	stop chan struct{}
	tick *time.Ticker

	// emit is called after every poll cycle if any session's state changed.
	// Frontends use this to wake up their fetcher (event push over Wails).
	emit func()

	// now is overridable for tests; production uses time.Now.
	now func() time.Time
}

// New returns a Watcher that fires `onChange` whenever any session was
// updated by the most recent poll. onChange may be nil.
func New(onChange func()) *Watcher {
	return &Watcher{
		sessions: map[string]*sessionState{},
		offsets:  map[string]int64{},
		emit:     onChange,
		now:      time.Now,
	}
}

// Start launches the background poll loop. Calling Start twice is a no-op.
func (w *Watcher) Start() {
	w.mu.Lock()
	if w.tick != nil {
		w.mu.Unlock()
		return
	}
	w.stop = make(chan struct{})
	w.tick = time.NewTicker(pollInterval)
	w.mu.Unlock()

	go w.loop()
}

// Stop halts the background poll loop. Safe to call repeatedly.
func (w *Watcher) Stop() {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.tick == nil {
		return
	}
	w.tick.Stop()
	close(w.stop)
	w.tick = nil
}

// Snapshot returns a copy of every session, sorted by recency descending.
// Callers safely hold the slice — internal state is deep-copied via the
// per-session snapshot() projection.
func (w *Watcher) Snapshot() []LiveSession {
	w.mu.RLock()
	defer w.mu.RUnlock()
	now := w.now()
	out := make([]LiveSession, 0, len(w.sessions))
	for _, s := range w.sessions {
		out = append(out, s.snapshot(now))
	}
	sortByLastActivityDesc(out)
	return out
}

// SnapshotActive is Snapshot filtered to sessions within activeWindow.
func (w *Watcher) SnapshotActive() []LiveSession {
	all := w.Snapshot()
	now := w.now()
	live := make([]LiveSession, 0, len(all))
	for _, s := range all {
		if now.Sub(s.LastActivity) <= activeWindow {
			live = append(live, s)
		}
	}
	return live
}

func (w *Watcher) loop() {
	// Do one immediate pass so the first Snapshot() after Start returns
	// something useful instead of an empty slice.
	w.pollOnce()
	for {
		select {
		case <-w.stop:
			return
		case <-w.tick.C:
			w.pollOnce()
		}
	}
}

func (w *Watcher) pollOnce() {
	files := conversation.DiscoverAll()

	w.mu.Lock()
	changed := false
	seenPaths := make(map[string]bool, len(files))
	now := w.now()

	for _, sf := range files {
		seenPaths[sf.Path] = true
		state, exists := w.sessions[sf.Path]
		off := w.offsets[sf.Path]

		// Skip files we've already drained completely AND that haven't
		// grown — most discovered files are stale historical transcripts.
		if exists && off >= sf.Size && now.Sub(state.lastActivity) > staleAfter {
			continue
		}

		// Tail-read any new bytes.
		if sf.Size > off {
			events, newOff, err := readNew(sf.Path, off)
			if err == nil {
				if !exists {
					state = newState(sf.SessionID, sf.Tool, sf.Cwd, sf.Path)
					w.sessions[sf.Path] = state
				}
				if len(events) > 0 {
					state.applyEvents(events)
					changed = true
				}
				w.offsets[sf.Path] = newOff
			}
		}
	}

	// Forget paths that have disappeared (user manually pruned ~/.claude
	// or rotated transcripts). Keep recent state in memory; only drop
	// when the file is gone AND the session is stale.
	for path, state := range w.sessions {
		if seenPaths[path] {
			continue
		}
		if now.Sub(state.lastActivity) > staleAfter {
			delete(w.sessions, path)
			delete(w.offsets, path)
			changed = true
		}
	}
	w.mu.Unlock()

	if changed && w.emit != nil {
		w.emit()
	}
}

func sortByLastActivityDesc(out []LiveSession) {
	for i := 1; i < len(out); i++ {
		for j := i; j > 0 && out[j].LastActivity.After(out[j-1].LastActivity); j-- {
			out[j], out[j-1] = out[j-1], out[j]
		}
	}
}
