package notify

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

// stubTransport is a minimal Transport for asserting bus behaviour. It
// records every event it received plus an error count so tests can
// confirm fan-out, filtering, and tracer callbacks.
type stubTransport struct {
	name           string
	supportApprove bool
	failNext       error
	mu             sync.Mutex
	got            []Event
}

func (s *stubTransport) Name() string             { return s.name }
func (s *stubTransport) SupportsApproval() bool   { return s.supportApprove }
func (s *stubTransport) Deliver(_ context.Context, ev Event) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.failNext != nil {
		err := s.failNext
		s.failNext = nil
		return err
	}
	s.got = append(s.got, ev)
	return nil
}
func (s *stubTransport) received() []Event {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]Event, len(s.got))
	copy(out, s.got)
	return out
}

func TestBus_FansOutToAllTransports(t *testing.T) {
	b := NewBus()
	a := &stubTransport{name: "a"}
	c := &stubTransport{name: "c"}
	b.Register(a)
	b.Register(c)

	n := b.Publish(context.Background(), Event{ID: "1", Kind: KindToolStuck, Title: "hi"})
	if n != 2 {
		t.Errorf("expected 2 successful deliveries, got %d", n)
	}
	if len(a.received()) != 1 || len(c.received()) != 1 {
		t.Errorf("a=%d c=%d", len(a.received()), len(c.received()))
	}
}

// Approval-class events must skip transports that can't carry an
// approval round-trip. Otherwise a card appears on a channel the user
// can never reply to, which is worse than not appearing at all.
func TestBus_ApprovalEventSkipsNonApprovalTransports(t *testing.T) {
	b := NewBus()
	noApprove := &stubTransport{name: "webhook-only"}
	withApprove := &stubTransport{name: "interactive", supportApprove: true}
	b.Register(noApprove)
	b.Register(withApprove)

	ev := Event{
		ID: "approval-1", Kind: KindBashGuardApproval, Title: "rm -rf?",
		Approval: &ApprovalRequest{Reply: make(chan Decision, 1)},
	}
	b.Publish(context.Background(), ev)

	if len(noApprove.received()) != 0 {
		t.Errorf("non-approval transport must not see approval events; got %d", len(noApprove.received()))
	}
	if len(withApprove.received()) != 1 {
		t.Errorf("approval transport should receive 1; got %d", len(withApprove.received()))
	}
}

// Register on an existing name must REPLACE rather than duplicate, so the
// Settings UI can rewire credentials without leaking transport instances.
func TestBus_RegisterIsIdempotentByName(t *testing.T) {
	b := NewBus()
	v1 := &stubTransport{name: "feishu"}
	v2 := &stubTransport{name: "feishu"}
	b.Register(v1)
	b.Register(v2)

	b.Publish(context.Background(), Event{ID: "x", Title: "hi"})
	if len(v1.received()) != 0 {
		t.Errorf("v1 should have been replaced; got %d events", len(v1.received()))
	}
	if len(v2.received()) != 1 {
		t.Errorf("v2 should have received 1; got %d", len(v2.received()))
	}
}

// A transport returning an error must NOT crash the bus or stop other
// transports from receiving. The tracer callback should also see the
// error so the UI can surface "Feishu delivery failed" without polling
// the transport itself.
func TestBus_TransportErrorDoesNotPoisonOthers(t *testing.T) {
	b := NewBus()
	bad := &stubTransport{name: "bad", failNext: errors.New("502 bad gateway")}
	good := &stubTransport{name: "good"}
	b.Register(bad)
	b.Register(good)

	var seen []struct {
		name string
		err  error
	}
	b.SetTracer(func(name string, _ Event, err error) {
		seen = append(seen, struct {
			name string
			err  error
		}{name, err})
	})

	n := b.Publish(context.Background(), Event{ID: "y", Title: "hi"})
	if n != 1 {
		t.Errorf("only the good transport should succeed; got %d", n)
	}
	if len(seen) != 2 {
		t.Errorf("tracer should have seen both attempts; got %d", len(seen))
	}
}

func TestBus_RecentRingBufferCaps(t *testing.T) {
	b := NewBus()
	b.maxRecent = 4 // keep test cheap
	for i := 0; i < 10; i++ {
		b.Publish(context.Background(), Event{ID: "e", Time: time.Now()})
	}
	if got := len(b.Recent()); got != 4 {
		t.Errorf("recent ring should cap at 4; got %d", got)
	}
}
