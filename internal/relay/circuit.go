package relay

import (
	"sync"
	"time"
)

// CircuitStatus reports the breaker state for one endpoint.
type CircuitStatus string

const (
	StatusClosed   CircuitStatus = "closed"    // healthy, traffic flows
	StatusOpen     CircuitStatus = "open"      // failing, all traffic short-circuited
	StatusHalfOpen CircuitStatus = "half_open" // probing — next request decides
)

const (
	defaultFailureThreshold = 3
	defaultCooldown         = 30 * time.Second
)

// CircuitState is the breaker's per-endpoint observable state. Exposed
// verbatim through the Wails binding so the UI can render badges.
type CircuitState struct {
	EndpointID         string        `json:"endpointID"`
	Status             CircuitStatus `json:"status"`
	ConsecutiveFailures int          `json:"consecutiveFailures"`
	LastFailureMs      int64         `json:"lastFailureMs,omitempty"` // unix-millis
	NextProbeMs        int64         `json:"nextProbeMs,omitempty"`   // unix-millis, half-open at-or-after
	LastError          string        `json:"lastError,omitempty"`
}

// CircuitBreaker keeps a state machine per endpoint ID. Safe for
// concurrent use; reads use an RWMutex so the tray's 10s tooltip refresh
// doesn't contend with live request flow.
type CircuitBreaker struct {
	mu               sync.RWMutex
	states           map[string]*CircuitState
	failureThreshold int
	cooldown         time.Duration
	now              func() time.Time // injectable for tests
}

// NewCircuitBreaker returns a breaker with defaults that match what
// production traffic has shown is safe (3 fails before opening, 30s
// cooldown). Tests can override via NewCircuitBreakerForTest.
func NewCircuitBreaker() *CircuitBreaker {
	return &CircuitBreaker{
		states:           map[string]*CircuitState{},
		failureThreshold: defaultFailureThreshold,
		cooldown:         defaultCooldown,
		now:              time.Now,
	}
}

// NewCircuitBreakerForTest exposes the failure threshold / cooldown / clock
// knobs that the production constructor hides.
func NewCircuitBreakerForTest(threshold int, cooldown time.Duration, now func() time.Time) *CircuitBreaker {
	return &CircuitBreaker{
		states:           map[string]*CircuitState{},
		failureThreshold: threshold,
		cooldown:         cooldown,
		now:              now,
	}
}

// Allow reports whether a request to endpointID is currently permitted.
// In closed and half-open states it returns true (half-open lets exactly
// one probe through, after which Record success/failure transitions it).
func (b *CircuitBreaker) Allow(endpointID string) bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	st, ok := b.states[endpointID]
	if !ok {
		return true
	}
	if st.Status == StatusOpen {
		if b.now().UnixMilli() >= st.NextProbeMs {
			st.Status = StatusHalfOpen
			return true
		}
		return false
	}
	return true
}

// RecordSuccess closes the breaker for endpointID.
func (b *CircuitBreaker) RecordSuccess(endpointID string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	st, ok := b.states[endpointID]
	if !ok {
		return
	}
	st.Status = StatusClosed
	st.ConsecutiveFailures = 0
	st.LastError = ""
}

// RecordFailure increments the consecutive-failures counter and trips
// the breaker once it crosses the threshold.
func (b *CircuitBreaker) RecordFailure(endpointID, errMsg string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	st, ok := b.states[endpointID]
	if !ok {
		st = &CircuitState{EndpointID: endpointID, Status: StatusClosed}
		b.states[endpointID] = st
	}
	st.ConsecutiveFailures++
	st.LastFailureMs = b.now().UnixMilli()
	st.LastError = errMsg
	if st.ConsecutiveFailures >= b.failureThreshold {
		st.Status = StatusOpen
		st.NextProbeMs = b.now().Add(b.cooldown).UnixMilli()
	}
}

// Reset clears state for one endpoint, returning it to closed.
func (b *CircuitBreaker) Reset(endpointID string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	delete(b.states, endpointID)
}

// Snapshot returns the current per-endpoint state map. The returned map
// is a deep copy — callers may mutate freely.
func (b *CircuitBreaker) Snapshot() map[string]CircuitState {
	b.mu.RLock()
	defer b.mu.RUnlock()
	out := make(map[string]CircuitState, len(b.states))
	for k, v := range b.states {
		out[k] = *v
	}
	return out
}
