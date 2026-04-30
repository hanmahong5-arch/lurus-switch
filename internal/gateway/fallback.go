package gateway

import (
	"bytes"
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

// SetEntries replaces the fallback chain entries atomically.
func (fc *FallbackChain) SetEntries(entries []FallbackEntry) {
	fc.mu.Lock()
	defer fc.mu.Unlock()
	fc.entries = entries
}

// Entries returns a copy of the current chain.
func (fc *FallbackChain) Entries() []FallbackEntry {
	fc.mu.RLock()
	defer fc.mu.RUnlock()
	out := make([]FallbackEntry, len(fc.entries))
	copy(out, fc.entries)
	return out
}

// shouldFallback returns true if the error or HTTP status warrants trying the next entry.
// Only server-side failures trigger fallback — client errors (4xx) do not.
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
	// 429 = rate limited → try next
	if resp.StatusCode == http.StatusTooManyRequests {
		return true
	}
	return false
}

// TryUpstream attempts the request against the primary upstream first.
// If it fails and a fallback chain is configured, tries each fallback in order.
// Returns the response, the name of the endpoint that succeeded, and any final error.
func (fc *FallbackChain) TryUpstream(
	method, path, query string,
	body []byte,
	headers http.Header,
	primaryURL, primaryToken string,
) (resp *http.Response, servedBy string, err error) {
	client := &http.Client{Timeout: upstreamTimeout}

	// Try primary first.
	resp, err = fc.doRequest(client, method, primaryURL, path, query, body, headers, primaryToken)
	if !shouldFallback(resp, err) {
		return resp, "primary", nil
	}
	// Close failed response body if any.
	if resp != nil {
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}

	// Walk fallback chain.
	fc.mu.RLock()
	entries := make([]FallbackEntry, len(fc.entries))
	copy(entries, fc.entries)
	fc.mu.RUnlock()

	for _, entry := range entries {
		if entry.URL == "" {
			continue
		}
		resp, err = fc.doRequest(client, method, entry.URL, path, query, body, headers, entry.Token)
		if !shouldFallback(resp, err) {
			return resp, entry.Name, nil
		}
		if resp != nil {
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
		}
	}

	// All entries exhausted.
	if err != nil {
		return nil, "", fmt.Errorf("all upstream endpoints failed, last error: %w", err)
	}
	return nil, "", fmt.Errorf("all upstream endpoints failed (last status: %d)", resp.StatusCode)
}

func (fc *FallbackChain) doRequest(
	client *http.Client,
	method, baseURL, path, query string,
	body []byte,
	headers http.Header,
	token string,
) (*http.Response, error) {
	targetURL := strings.TrimRight(baseURL, "/") + path
	if query != "" {
		targetURL += "?" + query
	}

	req, err := http.NewRequest(method, targetURL, bytes.NewReader(body))
	if err != nil {
		return nil, err
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
	_ = time.Since(start) // available for future latency tracking
	return resp, err
}
