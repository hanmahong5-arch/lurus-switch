// Package dlp is a Data Loss Prevention scanner for content flowing
// through the gateway. It runs a configurable pattern library against
// inbound prompts and outbound completions, producing a Result that
// callers can use to redact, block, or pass through the content.
//
// Design goals:
//   - Pure Go, no external services. Default patterns ship with the
//     binary so a fresh install has reasonable defaults.
//   - Per-pattern policy (allow / redact / block / warn) so an org can
//     start permissive and tighten over time.
//   - Idempotent — Scan() does not mutate the input string. Apply()
//     returns a new string with hits redacted in place, leaving the
//     original for audit.
//   - Audit-friendly — every hit produces a stable identifier (pattern
//     name + offset) so the same flagged content shows up consistently
//     in the audit log.
package dlp

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
)

// Severity ranks how seriously the operator should treat a match.
type Severity string

const (
	SeverityInfo     Severity = "info"     // notable but not concerning
	SeverityWarning  Severity = "warning"  // worth reviewing
	SeverityCritical Severity = "critical" // hard policy violation
)

// Policy decides what happens when a Pattern fires.
type Policy string

const (
	// PolicyAllow lets the content through unchanged. Useful for
	// patterns that exist only to populate the audit log.
	PolicyAllow Policy = "allow"
	// PolicyRedact replaces the matched substring with a stable token
	// (e.g. "[REDACTED:CC]") and lets the rest pass.
	PolicyRedact Policy = "redact"
	// PolicyBlock rejects the request entirely — the gateway should
	// surface a 451 / "your prompt contains restricted data" error.
	PolicyBlock Policy = "block"
	// PolicyWarn allows the content but emits a warning event so the
	// operator's dashboard can flag the user.
	PolicyWarn Policy = "warn"
)

// Pattern is a named regex with metadata. The library ships with a
// curated default set; operators can add custom patterns at runtime.
type Pattern struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Regex       string   `json:"regex"`
	Severity    Severity `json:"severity"`
	Policy      Policy   `json:"policy"`
	// Tags help group patterns in the UI (e.g. "pii", "secrets", "internal").
	Tags []string `json:"tags,omitempty"`

	compiled *regexp.Regexp
}

// Hit is one match of one pattern against one input.
type Hit struct {
	PatternName string `json:"patternName"`
	Severity    Severity `json:"severity"`
	Policy      Policy   `json:"policy"`
	Start       int      `json:"start"` // byte offset
	End         int      `json:"end"`
	// Snippet is a short anonymized excerpt of the match — useful for
	// the audit log without leaking the full sensitive value.
	Snippet string `json:"snippet"`
}

// Result aggregates everything found by a single Scan() call.
type Result struct {
	Hits         []Hit   `json:"hits"`
	HighestPolicy Policy `json:"highestPolicy"` // most severe policy among hits
	Blocked       bool   `json:"blocked"`
	// Redacted is the input with every PolicyRedact hit replaced. For
	// PolicyBlock the field equals the input — caller is expected to
	// reject the request rather than continuing.
	Redacted string `json:"redacted"`
}

// HitRecord is a recorded hit + the context where it was caught. The
// scanner keeps a small ring of recent hits so the admin UI can show
// "what got blocked / redacted in the last hour" without a separate
// audit pipeline. For long-term forensic audit, the journal package is
// the right tool — this ring is for at-a-glance observability.
type HitRecord struct {
	Timestamp time.Time `json:"timestamp"`
	Source    string    `json:"source"` // "gateway.request" / "test" / etc.
	Path      string    `json:"path"`   // e.g. "/v1/chat/completions"
	Hit       Hit       `json:"hit"`
}

// hitRingSize bounds memory for the recent-hits ring. 200 is enough to
// cover the last few minutes of a busy gateway without hogging RAM.
const hitRingSize = 200

// Scanner holds the active set of patterns. Use NewScanner() with the
// curated defaults, then Add() / Remove() to customize per deployment.
type Scanner struct {
	mu       sync.RWMutex
	patterns []*Pattern
	hits     []HitRecord // most-recent first; capped at hitRingSize
}

// NewScanner returns a Scanner pre-populated with default patterns.
func NewScanner() *Scanner {
	s := &Scanner{}
	for _, p := range DefaultPatterns() {
		// Best-effort compile; bad defaults would be a programmer bug
		// caught in TestDefaultPatternsCompile.
		_ = s.Add(p)
	}
	return s
}

// Add validates the pattern's regex and appends it. Returns an error
// if the regex is malformed or the name conflicts.
func (s *Scanner) Add(p Pattern) error {
	if strings.TrimSpace(p.Name) == "" {
		return fmt.Errorf("pattern name cannot be empty")
	}
	if strings.TrimSpace(p.Regex) == "" {
		return fmt.Errorf("pattern %q has empty regex", p.Name)
	}
	rx, err := regexp.Compile(p.Regex)
	if err != nil {
		return fmt.Errorf("pattern %q: regex compile: %w", p.Name, err)
	}
	if p.Severity == "" {
		p.Severity = SeverityWarning
	}
	if p.Policy == "" {
		p.Policy = PolicyWarn
	}
	p.compiled = rx

	s.mu.Lock()
	defer s.mu.Unlock()
	for _, existing := range s.patterns {
		if existing.Name == p.Name {
			return fmt.Errorf("pattern %q already registered", p.Name)
		}
	}
	s.patterns = append(s.patterns, &p)
	return nil
}

// Remove drops a pattern by name. Returns false if not found.
func (s *Scanner) Remove(name string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, p := range s.patterns {
		if p.Name == name {
			s.patterns = append(s.patterns[:i], s.patterns[i+1:]...)
			return true
		}
	}
	return false
}

// SetPolicy mutates the policy for a registered pattern. Used by the
// admin UI ("turn this from warn to block").
func (s *Scanner) SetPolicy(name string, policy Policy) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, p := range s.patterns {
		if p.Name == name {
			p.Policy = policy
			return true
		}
	}
	return false
}

// Patterns returns a copy of the active pattern list, sorted by name.
// Used by the admin UI to render the policy table.
func (s *Scanner) Patterns() []Pattern {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Pattern, 0, len(s.patterns))
	for _, p := range s.patterns {
		// Drop the compiled field on the wire; consumers don't need it.
		copy := *p
		copy.compiled = nil
		out = append(out, copy)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// Scan runs every active pattern against the input. Hits that don't
// overlap are all reported; overlapping hits are kept for audit but
// only the first is used for redaction (so we don't double-mask).
func (s *Scanner) Scan(input string) Result {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var hits []Hit
	highest := PolicyAllow

	for _, p := range s.patterns {
		matches := p.compiled.FindAllStringIndex(input, -1)
		for _, m := range matches {
			hits = append(hits, Hit{
				PatternName: p.Name,
				Severity:    p.Severity,
				Policy:      p.Policy,
				Start:       m[0],
				End:         m[1],
				Snippet:     anonymize(input[m[0]:m[1]]),
			})
			if policyRank(p.Policy) > policyRank(highest) {
				highest = p.Policy
			}
		}
	}

	// Sort hits by Start so redaction is deterministic.
	sort.SliceStable(hits, func(i, j int) bool { return hits[i].Start < hits[j].Start })

	res := Result{
		Hits:          hits,
		HighestPolicy: highest,
		Blocked:       highest == PolicyBlock,
	}

	// Apply redaction (block leaves text intact — caller rejects request).
	if highest == PolicyBlock {
		res.Redacted = input
	} else {
		res.Redacted = applyRedaction(input, hits)
	}

	return res
}

// applyRedaction replaces matched ranges with a stable token. We walk
// hits sorted by Start, skipping overlapping spans.
func applyRedaction(input string, hits []Hit) string {
	if len(hits) == 0 {
		return input
	}
	var b strings.Builder
	b.Grow(len(input))
	cursor := 0
	for _, h := range hits {
		if h.Policy != PolicyRedact {
			continue
		}
		if h.Start < cursor {
			continue // overlaps a previously redacted span
		}
		b.WriteString(input[cursor:h.Start])
		b.WriteString(fmt.Sprintf("[REDACTED:%s]", h.PatternName))
		cursor = h.End
	}
	b.WriteString(input[cursor:])
	return b.String()
}

// anonymize keeps the first char + length info for the audit log so
// auditors can recognize the kind of value without seeing the secret.
func anonymize(s string) string {
	switch {
	case len(s) <= 4:
		return strings.Repeat("*", len(s))
	case len(s) <= 12:
		return string(s[0]) + strings.Repeat("*", len(s)-2) + string(s[len(s)-1])
	default:
		return s[:2] + strings.Repeat("*", len(s)-4) + s[len(s)-2:]
	}
}

// policyRank orders policies by severity for "highest wins" computation.
func policyRank(p Policy) int {
	switch p {
	case PolicyBlock:
		return 4
	case PolicyRedact:
		return 3
	case PolicyWarn:
		return 2
	case PolicyAllow:
		return 1
	default:
		return 0
	}
}

// RecordHits stamps a list of hits with the given source/path metadata
// and pushes them into the ring buffer for later observability calls.
// Safe to invoke with zero hits — no-op in that case.
func (s *Scanner) RecordHits(source, path string, hits []Hit) {
	if len(hits) == 0 {
		return
	}
	now := time.Now()
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, h := range hits {
		s.hits = append([]HitRecord{{
			Timestamp: now,
			Source:    source,
			Path:      path,
			Hit:       h,
		}}, s.hits...)
	}
	if len(s.hits) > hitRingSize {
		s.hits = s.hits[:hitRingSize]
	}
}

// RecentHits returns up to limit most-recent hit records (newest first).
// limit ≤ 0 returns all available.
func (s *Scanner) RecentHits(limit int) []HitRecord {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if limit <= 0 || limit > len(s.hits) {
		limit = len(s.hits)
	}
	out := make([]HitRecord, limit)
	copy(out, s.hits[:limit])
	return out
}

// HitStats summarizes the recent ring for the admin dashboard tile.
type HitStats struct {
	Total       int            `json:"total"`
	BySeverity  map[string]int `json:"bySeverity"`
	ByPolicy    map[string]int `json:"byPolicy"`
	ByPattern   map[string]int `json:"byPattern"`
	BySource    map[string]int `json:"bySource"`
}

// Stats returns counters across the entire recent-hit ring.
func (s *Scanner) Stats() HitStats {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := HitStats{
		BySeverity: make(map[string]int),
		ByPolicy:   make(map[string]int),
		ByPattern:  make(map[string]int),
		BySource:   make(map[string]int),
	}
	out.Total = len(s.hits)
	for _, r := range s.hits {
		out.BySeverity[string(r.Hit.Severity)]++
		out.ByPolicy[string(r.Hit.Policy)]++
		out.ByPattern[r.Hit.PatternName]++
		out.BySource[r.Source]++
	}
	return out
}
