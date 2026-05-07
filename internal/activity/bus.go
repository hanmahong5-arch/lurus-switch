// Package activity is a tiny in-process pub/sub for operation events
// (install / configure / probe / etc). The frontend subscribes to a
// single Wails event channel and renders a live "what is Switch doing
// right now" panel — solving the 2026 user complaint that long-running
// flows feel like black boxes ("the spinner just spins for 30s").
//
// Events are fire-and-forget; the bus never blocks the caller, even if
// the frontend isn't listening yet (e.g. during early startup).
package activity

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// Phase is the lifecycle status of an Event.
type Phase string

const (
	PhaseStart    Phase = "start"
	PhaseProgress Phase = "progress"
	PhaseDone     Phase = "done"
	PhaseError    Phase = "error"
)

// Event is what gets pushed to the frontend.
type Event struct {
	ID         string    `json:"id"`         // unique per operation
	Phase      Phase     `json:"phase"`
	TitleZh    string    `json:"titleZh"`    // short bilingual headline
	TitleEn    string    `json:"titleEn"`
	DetailZh   string    `json:"detailZh,omitempty"`  // optional sub-line
	DetailEn   string    `json:"detailEn,omitempty"`
	Progress   int       `json:"progress,omitempty"`   // 0..100; 0 = indeterminate
	Total      int       `json:"total,omitempty"`      // for "step X of Total"
	Step       int       `json:"step,omitempty"`
	Error      string    `json:"error,omitempty"`      // populated on PhaseError
	StartedAt  time.Time `json:"startedAt"`
	UpdatedAt  time.Time `json:"updatedAt"`
}

// EventName is the Wails event channel.
const EventName = "activity:event"

// Bus is the singleton event router. Pure best-effort emission to the
// Wails runtime; failures (including a missing context) are swallowed.
type Bus struct {
	mu      sync.RWMutex
	ctx     context.Context
	idGen   atomic.Uint64
}

func New() *Bus { return &Bus{} }

// Bind ties the bus to the Wails runtime context. Safe to call once
// the Wails app has started (i.e. inside OnStartup).
func (b *Bus) Bind(ctx context.Context) {
	b.mu.Lock()
	b.ctx = ctx
	b.mu.Unlock()
}

// Op is a fluent helper that scopes a single operation. Use:
//
//	op := bus.Op("install-claude", "安装 Claude Code", "Installing Claude Code")
//	op.Progress("下载中", "Downloading", 25, 4, 1)
//	... eventually ...
//	op.Done("已安装", "Installed")
//
// On failure, op.Error(...) emits a PhaseError event with the message.
type Op struct {
	bus       *Bus
	id        string
	titleZh   string
	titleEn   string
	startedAt time.Time
}

func (b *Bus) Op(id, titleZh, titleEn string) *Op {
	if id == "" {
		id = b.newID()
	}
	op := &Op{
		bus: b, id: id, titleZh: titleZh, titleEn: titleEn,
		startedAt: time.Now(),
	}
	b.emit(Event{
		ID: id, Phase: PhaseStart,
		TitleZh: titleZh, TitleEn: titleEn,
		StartedAt: op.startedAt, UpdatedAt: op.startedAt,
	})
	return op
}

// Progress fires PhaseProgress. progress is 0..100; pass 0 for an
// indeterminate spinner. step / total describe a multi-step flow.
func (o *Op) Progress(detailZh, detailEn string, progress, total, step int) {
	o.bus.emit(Event{
		ID: o.id, Phase: PhaseProgress,
		TitleZh: o.titleZh, TitleEn: o.titleEn,
		DetailZh: detailZh, DetailEn: detailEn,
		Progress: progress, Total: total, Step: step,
		StartedAt: o.startedAt, UpdatedAt: time.Now(),
	})
}

func (o *Op) Done(detailZh, detailEn string) {
	o.bus.emit(Event{
		ID: o.id, Phase: PhaseDone,
		TitleZh: o.titleZh, TitleEn: o.titleEn,
		DetailZh: detailZh, DetailEn: detailEn,
		Progress: 100,
		StartedAt: o.startedAt, UpdatedAt: time.Now(),
	})
}

func (o *Op) Error(err string) {
	o.bus.emit(Event{
		ID: o.id, Phase: PhaseError,
		TitleZh: o.titleZh, TitleEn: o.titleEn,
		Error: err,
		StartedAt: o.startedAt, UpdatedAt: time.Now(),
	})
}

func (b *Bus) emit(ev Event) {
	b.mu.RLock()
	ctx := b.ctx
	b.mu.RUnlock()
	if ctx == nil {
		return // no UI yet; fine
	}
	// EventsEmit is safe to call concurrently and from any goroutine.
	wailsRuntime.EventsEmit(ctx, EventName, ev)
}

func (b *Bus) newID() string {
	n := b.idGen.Add(1)
	return "act-" + time.Now().Format("150405") + "-" + itoa(n)
}

func itoa(n uint64) string {
	if n == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}

// ─── Convenience for one-shot ops that don't need progress streaming ─

// Run wraps a function in start/done/error events. The returned error
// is the original — the bus only mirrors it to the frontend.
func (b *Bus) Run(id, titleZh, titleEn string, fn func() error) error {
	op := b.Op(id, titleZh, titleEn)
	err := fn()
	if err != nil {
		op.Error(err.Error())
		return err
	}
	op.Done("", "")
	return nil
}
