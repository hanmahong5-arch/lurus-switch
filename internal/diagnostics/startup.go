// Package diagnostics records how long Switch takes to come up, broken
// down by phase, so the user (and we) can see whether a new feature is
// dragging out cold start.
//
// The model is deliberately tiny: a process-wide Recorder that callers
// poke with Mark(name) at meaningful boundaries during startup. Each
// Mark closes the previous phase and opens a new one. Snapshot() freezes
// the current trace; Persist() keeps the last few traces on disk so the
// UI can show a "vs. last launch" delta.
//
// There is exactly one Recorder per process (the package-level singleton)
// because "startup" is a process-global event — wiring an instance through
// every constructor would be noise. It is mutex-guarded so the background
// safeGo goroutines that finish late can Mark() safely.
package diagnostics

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const (
	diagDir       = "diagnostics"
	startupFile   = "startup-traces.json"
	maxKeptTraces = 5
	dirPerm       = 0o755
	filePerm      = 0o600
)

// Phase is one named segment of startup. Duration is filled in when the
// next Mark arrives (or when Snapshot is taken for the final phase).
type Phase struct {
	Name       string    `json:"name"`
	StartedAt  time.Time `json:"startedAt"`
	DurationMs int64     `json:"durationMs"`
}

// Trace is a complete startup timeline.
//
// ColdStartMS is the wall-clock from MarkStart() to the moment Snapshot()
// was taken. GUIReadyMS is captured separately at the point the Wails
// startup() callback returns — it is the "window is interactive" milestone,
// distinct from "all background services settled" (ColdStartMS), so the UI
// can show both without one misleading the other.
type Trace struct {
	ColdStartMS int64     `json:"coldStartMs"`
	GUIReadyMS  int64     `json:"guiReadyMs"`
	Phases      []Phase   `json:"phases"`
	CapturedAt  time.Time `json:"capturedAt"`
}

// Recorder accumulates phases for a single process lifetime.
type Recorder struct {
	mu       sync.Mutex
	start    time.Time
	guiReady time.Time
	phases   []Phase
	lastMark time.Time
	started  bool
}

// Default is the process-wide recorder. main() calls Default.MarkStart()
// as its first statement; app.startup() sprinkles Default.Mark(...).
var Default = &Recorder{}

// MarkStart stamps t0. Idempotent — a second call is ignored so a CLI
// fast-path that returns early before the GUI can't reset the clock.
func (r *Recorder) MarkStart() {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.started {
		return
	}
	now := time.Now()
	r.start = now
	r.lastMark = now
	r.started = true
}

// Mark closes the in-flight phase (attributing elapsed time to the
// previously-named boundary) and opens a new one called name. A Mark
// before MarkStart is a no-op — we never want a negative duration.
func (r *Recorder) Mark(name string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if !r.started {
		return
	}
	now := time.Now()
	r.phases = append(r.phases, Phase{
		Name:       name,
		StartedAt:  r.lastMark,
		DurationMs: now.Sub(r.lastMark).Milliseconds(),
	})
	r.lastMark = now
}

// MarkGUIReady records the instant the GUI became interactive. Idempotent.
func (r *Recorder) MarkGUIReady() {
	r.mu.Lock()
	defer r.mu.Unlock()
	if !r.started || !r.guiReady.IsZero() {
		return
	}
	r.guiReady = time.Now()
}

// Snapshot returns a copy of the current trace. Safe to call repeatedly;
// each call recomputes ColdStartMS against now.
func (r *Recorder) Snapshot() Trace {
	r.mu.Lock()
	defer r.mu.Unlock()
	now := time.Now()
	phases := make([]Phase, len(r.phases))
	copy(phases, r.phases)
	t := Trace{
		Phases:     phases,
		CapturedAt: now,
	}
	if r.started {
		t.ColdStartMS = now.Sub(r.start).Milliseconds()
	}
	if !r.guiReady.IsZero() {
		t.GUIReadyMS = r.guiReady.Sub(r.start).Milliseconds()
	}
	return t
}

// Persist appends the current snapshot to the on-disk history, keeping
// only the most recent maxKeptTraces. Returns the history it wrote (newest
// first) so the caller can avoid a re-read.
func (r *Recorder) Persist(appDataDir string) ([]Trace, error) {
	current := r.Snapshot()
	dir := filepath.Join(appDataDir, diagDir)
	if err := os.MkdirAll(dir, dirPerm); err != nil {
		return nil, err
	}
	path := filepath.Join(dir, startupFile)

	history := loadTraces(path) // newest first; tolerant of missing/corrupt
	history = append([]Trace{current}, history...)
	if len(history) > maxKeptTraces {
		history = history[:maxKeptTraces]
	}

	data, err := json.MarshalIndent(history, "", "  ")
	if err != nil {
		return nil, err
	}
	if err := os.WriteFile(path, data, filePerm); err != nil {
		return nil, err
	}
	return history, nil
}

// History reads the persisted traces (newest first). Missing file yields
// an empty slice, not an error — a first-ever launch has no history.
func History(appDataDir string) []Trace {
	return loadTraces(filepath.Join(appDataDir, diagDir, startupFile))
}

func loadTraces(path string) []Trace {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var traces []Trace
	if err := json.Unmarshal(data, &traces); err != nil {
		return nil
	}
	return traces
}
