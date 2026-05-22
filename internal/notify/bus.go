package notify

import (
	"context"
	"log"
	"sync"
)

// Transport is anything that can render an Event onto a remote surface.
// Implementations: feishu, telegram, slack, … Each ships in its own
// sub-package so platform SDKs don't pollute the core dependency graph.
type Transport interface {
	// Name is a stable identifier ("feishu", "telegram", …) used for
	// logging and to let the UI show per-transport delivery status.
	Name() string
	// Deliver sends the event. ctx cancellation must abort the send.
	// Returning an error logs a delivery failure but does NOT remove the
	// transport from the bus — transient HTTP errors should not silently
	// disable notifications until the next app restart.
	Deliver(ctx context.Context, ev Event) error
	// SupportsApproval reports whether this transport can carry
	// interactive approval round-trips. Webhook-only transports return
	// false and the bus will refuse to dispatch ApprovalRequest events
	// to them, so a Bash-Guard prompt never goes out somewhere it can't
	// come back from.
	SupportsApproval() bool
}

// Bus is the in-process pub/sub. The publisher calls Publish; every
// registered Transport's Deliver runs concurrently. Slow transports
// can't block fast ones. Synchronous semantics on Publish (waits for
// all transports) keep the test surface small; in practice every send
// is bounded by the transport's own context timeout.
type Bus struct {
	mu         sync.RWMutex
	transports []Transport

	// recent is a small ring buffer of delivered events, surfaced to the
	// Notifications page in the UI so the user can see "what got sent
	// without checking my phone". Bounded by maxRecent.
	recent     []Event
	maxRecent  int

	// onDeliver is called after each Deliver attempt for tracing. Tests
	// substitute their own to assert dispatch behaviour.
	onDeliver func(transportName string, ev Event, err error)
}

const defaultMaxRecent = 30

// NewBus returns an empty bus. Register transports with Register.
func NewBus() *Bus {
	return &Bus{maxRecent: defaultMaxRecent}
}

// Register installs a transport. Idempotent on name — re-registering the
// same name replaces the previous instance, so a UI settings flush can
// rewire credentials without restarting the app.
func (b *Bus) Register(tp Transport) {
	if tp == nil {
		return
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	for i, existing := range b.transports {
		if existing.Name() == tp.Name() {
			b.transports[i] = tp
			return
		}
	}
	b.transports = append(b.transports, tp)
}

// Unregister removes a transport by name. Used when the user disables a
// platform in settings.
func (b *Bus) Unregister(name string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	out := b.transports[:0]
	for _, t := range b.transports {
		if t.Name() != name {
			out = append(out, t)
		}
	}
	b.transports = out
}

// SetTracer overrides the delivery tracing hook. Tests use this to assert
// what got dispatched; production passes nil (logs to stdlib log only).
func (b *Bus) SetTracer(fn func(transportName string, ev Event, err error)) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.onDeliver = fn
}

// Publish fans the event out to every registered transport. Returns the
// number of successful deliveries (handy for tests; production callers
// usually ignore the count).
//
// Approval events go ONLY to transports whose SupportsApproval is true.
// Other transports never see them — a "we couldn't ask" state is far
// less confusing than rendering an approval card on a channel the user
// can never reply to.
func (b *Bus) Publish(ctx context.Context, ev Event) int {
	b.mu.RLock()
	transports := append([]Transport(nil), b.transports...)
	tracer := b.onDeliver
	b.mu.RUnlock()

	var wg sync.WaitGroup
	var successCount int
	var counterMu sync.Mutex

	for _, tp := range transports {
		if ev.Approval != nil && !tp.SupportsApproval() {
			continue
		}
		wg.Add(1)
		go func(tp Transport) {
			defer wg.Done()
			err := tp.Deliver(ctx, ev)
			if tracer != nil {
				tracer(tp.Name(), ev, err)
			}
			if err != nil {
				log.Printf("notify: transport %s deliver failed: %v", tp.Name(), err)
				return
			}
			counterMu.Lock()
			successCount++
			counterMu.Unlock()
		}(tp)
	}
	wg.Wait()

	b.recordRecent(ev)
	return successCount
}

func (b *Bus) recordRecent(ev Event) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.recent = append(b.recent, ev)
	if len(b.recent) > b.maxRecent {
		b.recent = b.recent[len(b.recent)-b.maxRecent:]
	}
}

// Recent returns a copy of the recent-events ring buffer, newest last.
// The UI uses this for the "what got pushed lately" panel.
func (b *Bus) Recent() []Event {
	b.mu.RLock()
	defer b.mu.RUnlock()
	out := make([]Event, len(b.recent))
	copy(out, b.recent)
	return out
}

// TransportNames returns the registered transports' names. Stable order
// matches registration order — the UI shows them top-to-bottom that way.
func (b *Bus) TransportNames() []string {
	b.mu.RLock()
	defer b.mu.RUnlock()
	out := make([]string, len(b.transports))
	for i, t := range b.transports {
		out[i] = t.Name()
	}
	return out
}
