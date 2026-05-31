package gateway

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

// FallbackChain holds an ordered list of upstream endpoints to try.
// If the primary upstream fails (5xx, timeout, connection refused),
// the gateway walks the chain until one succeeds or all are exhausted.
//
// Design lessons from NadirClaw (2026-04-11):
//   - Primary model may be rate-limited on free tiers (Groq: 30 RPM)
//   - Fallback should be automatic and transparent to the CLI tool
//   - Record which endpoint actually served the request for debugging
type FallbackChain struct {
	mu       sync.RWMutex
	entries  []FallbackEntry
	maxRetry int // max entries to try (0 = try all)

	// observer is invoked once per upstream attempt with the endpoint
	// name (or "primary"), a success/failure verdict, and the measured
	// round-trip latency in milliseconds. Used by the relay router's
	// circuit breaker + latency feedback loop; nil-safe.
	observer func(endpointName string, ok bool, errMsg string, latencyMs int64)
}

// SetObserver wires a per-attempt callback into the chain. The observer
// fires after every primary AND fallback attempt — success returns
// ok=true, anything that trips shouldFallback returns ok=false. The
// latencyMs argument is the wall-clock duration of the upstream HTTP
// round trip; 0 when the attempt was short-circuited before dialing.
func (fc *FallbackChain) SetObserver(fn func(endpointName string, ok bool, errMsg string, latencyMs int64)) {
	fc.mu.Lock()
	defer fc.mu.Unlock()
	fc.observer = fn
}

// FallbackEntry is one upstream endpoint in the chain.
type FallbackEntry struct {
	Name     string `json:"name"`     // display name (e.g. "Groq-Free", "DeepSeek")
	URL      string `json:"url"`      // base URL (without /v1)
	Token    string `json:"token"`    // API key / bearer token
	Priority int    `json:"priority"` // lower = tried first
}

// NewFallbackChain creates a chain. Entries are tried in order of priority (ascending).
func NewFallbackChain(entries []FallbackEntry) *FallbackChain {
	return &FallbackChain{
		entries: entries,
	}
}

// Entries returns a copy of the current chain.
func (fc *FallbackChain) Entries() []FallbackEntry {
	fc.mu.RLock()
	defer fc.mu.RUnlock()
	out := make([]FallbackEntry, len(fc.entries))
	copy(out, fc.entries)
	return out
}

// shouldFallback returns true if the error or HTTP status warrants trying
// the next entry in the chain.
//
// Two failure classes roll over to the backup upstream:
//   - server-side faults: 5xx, 429 rate-limit, and transport errors
//     (connection refused, timeout, DNS failure);
//   - upstream auth / quota rejections: 401 (key revoked or banned), 403
//     (key forbidden or region block), 402 (upstream out of credit).
//
// The auth/quota class is the one resellers care about: a banned or drained
// key on the primary endpoint must fail over to a backup instead of being
// handed straight back to the caller. Because these attempts return ok=false
// to the observer, the circuit breaker also trips the dead endpoint open
// after the failure threshold instead of hammering it with every request.
//
// Genuine *client* 4xx (400 malformed, 404 unknown route, 422 …) are NOT
// retried — the next endpoint would reject them identically, so cascading
// only wastes a round trip.
func shouldFallback(resp *http.Response, err error) bool {
	if err != nil {
		return true // connection refused, timeout, DNS failure
	}
	if resp == nil {
		return true
	}
	// 5xx = server error → try next
	if resp.StatusCode >= 500 {
		return true
	}
	switch resp.StatusCode {
	case http.StatusTooManyRequests, // 429 rate limited
		http.StatusUnauthorized,    // 401 upstream key revoked / banned
		http.StatusForbidden,       // 403 upstream key forbidden / region block
		http.StatusPaymentRequired: // 402 upstream out of credit
		return true
	}
	return false
}

// TryUpstream attempts the request against the primary upstream first.
// If it fails and a fallback chain is configured, tries each fallback in order.
// Returns the response, the name of the endpoint that succeeded, and any final error.
//
// TryUpstream is the cfg-driven legacy entry point: caller provides one
// primary URL+token and the chain pulls fallbacks from its persisted
// entries. Use TryUpstreamChain when the caller (e.g. relay router)
// already knows the full ordered chain.
func (fc *FallbackChain) TryUpstream(
	ctx context.Context,
	method, path, query string,
	body []byte,
	headers http.Header,
	primaryURL, primaryToken string,
) (resp *http.Response, servedBy string, err error) {
	fc.mu.RLock()
	entries := make([]FallbackEntry, 0, 1+len(fc.entries))
	entries = append(entries, FallbackEntry{Name: "primary", URL: primaryURL, Token: primaryToken})
	for _, e := range fc.entries {
		if e.URL == "" {
			continue
		}
		entries = append(entries, e)
	}
	fc.mu.RUnlock()
	return fc.TryUpstreamChain(ctx, method, path, query, body, headers, entries)
}

// TryUpstreamChain attempts the request against the provided ordered
// chain in sequence. The first entry is treated as the primary; the
// rest are tried in order on 5xx / 429 / connection failure. Observer
// fires once per attempt. Returns the response, name of the entry that
// succeeded, and any final error.
//
// The chain is provided by the caller (router-driven), so this method
// does NOT consult fc.entries — that path is exclusively TryUpstream's.
func (fc *FallbackChain) TryUpstreamChain(
	ctx context.Context,
	method, path, query string,
	body []byte,
	headers http.Header,
	chain []FallbackEntry,
) (resp *http.Response, servedBy string, err error) {
	if len(chain) == 0 {
		return nil, "", fmt.Errorf("upstream chain is empty")
	}
	if ctx == nil {
		ctx = context.Background()
	}

	client := &http.Client{Timeout: upstreamTimeout}
	fc.mu.RLock()
	observer := fc.observer
	fc.mu.RUnlock()

	for _, entry := range chain {
		if entry.URL == "" {
			continue
		}
		var latencyMs int64
		resp, latencyMs, err = fc.doRequest(ctx, client, method, entry.URL, path, query, body, headers, entry.Token)
		if !shouldFallback(resp, err) {
			if observer != nil {
				observer(entry.Name, true, "", latencyMs)
			}
			return resp, entry.Name, nil
		}
		if observer != nil {
			msg := ""
			if err != nil {
				msg = err.Error()
			} else if resp != nil {
				msg = fmt.Sprintf("status %d", resp.StatusCode)
			}
			observer(entry.Name, false, msg, latencyMs)
		}
		if resp != nil {
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
		}
	}

	if err != nil {
		return nil, "", fmt.Errorf("all upstream endpoints failed, last error: %w", err)
	}
	if resp != nil {
		return nil, "", fmt.Errorf("all upstream endpoints failed (last status: %d)", resp.StatusCode)
	}
	return nil, "", fmt.Errorf("all upstream endpoints had empty URLs")
}

// doRequest performs one upstream HTTP call and returns the response
// plus the measured wall-clock latency in milliseconds.
func (fc *FallbackChain) doRequest(
	ctx context.Context,
	client *http.Client,
	method, baseURL, path, query string,
	body []byte,
	headers http.Header,
	token string,
) (*http.Response, int64, error) {
	targetURL := strings.TrimRight(baseURL, "/") + path
	if query != "" {
		targetURL += "?" + query
	}

	// Carry the caller's context so an upstream call is cancelled when the
	// client disconnects or the request deadline fires — a bare
	// http.NewRequest leaks the goroutine + upstream connection until the
	// 5-minute client timeout, which on a streaming hot path means the
	// gateway keeps paying for tokens nobody is reading.
	req, err := http.NewRequestWithContext(ctx, method, targetURL, bytes.NewReader(body))
	if err != nil {
		return nil, 0, err
	}

	// Copy original headers.
	for k, vv := range headers {
		for _, v := range vv {
			req.Header.Add(k, v)
		}
	}
	// Override auth with this entry's token.
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	start := time.Now()
	resp, err := client.Do(req)
	latency := time.Since(start).Milliseconds()
	return resp, latency, err
}
