package gateway

import (
	"encoding/json"
	"strings"

	"lurus-switch/internal/dlp"
)

// DLP middleware integration. The gateway holds an optional *dlp.Scanner
// (set by services.go at boot via SetDLPScanner). Each inbound proxy
// request body is scanned BEFORE forwarding upstream:
//
//   - PolicyBlock hit         → return 451 + dlp_blocked error
//   - PolicyRedact hit        → swap body for the redacted version
//   - PolicyWarn / PolicyAllow → forward original body, but the hit is
//                                still recorded for observability
//
// Hits are recorded into the scanner's ring buffer regardless of policy,
// so the admin DLP page can display "what got caught lately" without a
// separate audit pipeline.
//
// Why not response scanning too? The streaming case (SSE chunks) makes
// post-hoc redaction tricky — you'd have to re-frame the chunks, and
// any redaction lands in the user's terminal mid-stream. We start with
// request-only and revisit response scanning when there's a concrete ask.

// SetDLPScanner injects (or clears, with nil) the DLP scanner. Safe to
// call after Start; the next request picks up the change.
func (s *Server) SetDLPScanner(scanner *dlp.Scanner) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.dlpScanner = scanner
}

// SetDLPAuditFn injects (or clears, with nil) the audit callback. The
// fn is invoked once per blocked or redacted request; payload carries
// the path, app, and matched pattern names so an admin can later
// reconstruct the event. The optional metadata map carries the
// conversation correlation keys (tool / sessionID / messageUUID) when
// the request originated from a CLI session we recognise.
func (s *Server) SetDLPAuditFn(fn func(op, target string, payload any, metadata map[string]string)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.dlpAuditFn = fn
}

// applyDLPRequest scans the inbound request body and either returns the
// (possibly redacted) body to forward, or signals that the request must
// be blocked. When no scanner is configured this is a fast no-op.
//
// Side-effects: records hits into the scanner ring; if an audit fn is
// wired, emits a "dlp.block" or "dlp.redact" entry per blocking or
// redacting event for forensic durability.
//
// Returns:
//   - newBody:    body to forward (== input on no-redact)
//   - blocked:    true means caller must reject the request
//   - blockReason: human-readable string for the error response
func (s *Server) applyDLPRequest(body []byte, path string) (newBody []byte, blocked bool, blockReason string) {
	s.mu.Lock()
	scanner := s.dlpScanner
	auditFn := s.dlpAuditFn
	s.mu.Unlock()
	if scanner == nil || len(body) == 0 {
		return body, false, ""
	}
	res := scanner.Scan(string(body))
	scanner.RecordHits("gateway.request", path, res.Hits)

	if res.Blocked {
		if auditFn != nil {
			auditFn("dlp.block", path, dlpAuditPayload(path, res.Hits), extractSessionMetadata(body))
		}
		return body, true, dlpBlockReason(res.Hits)
	}
	if res.HighestPolicy == dlp.PolicyRedact && res.Redacted != string(body) {
		if auditFn != nil {
			auditFn("dlp.redact", path, dlpAuditPayload(path, res.Hits), extractSessionMetadata(body))
		}
		return []byte(res.Redacted), false, ""
	}
	return body, false, ""
}

// extractSessionMetadata pulls conversation-correlation fields out of an
// Anthropic Messages API request body. Claude Code sends a `metadata`
// object with `user_id` and, in newer CLIs, the encoded session/message
// UUIDs. We tolerate either flat or nested shape because the CLI's
// schema has drifted across versions.
//
// Returns nil when the body isn't valid JSON or carries no recognisable
// session ID — the audit entry then has no metadata map, which is fine.
func extractSessionMetadata(body []byte) map[string]string {
	if len(body) == 0 {
		return nil
	}
	var raw struct {
		Metadata map[string]any `json:"metadata"`
		// Claude Code also surfaces the IDs at the top level in some versions.
		SessionID   string `json:"session_id"`
		MessageID   string `json:"message_id"`
		MessageUUID string `json:"message_uuid"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil
	}
	out := map[string]string{}
	pick := func(key string, m map[string]any) string {
		if v, ok := m[key]; ok {
			if s, ok := v.(string); ok {
				return s
			}
		}
		return ""
	}
	sid := raw.SessionID
	if sid == "" && raw.Metadata != nil {
		sid = pick("session_id", raw.Metadata)
		if sid == "" {
			sid = pick("sessionId", raw.Metadata)
		}
	}
	muid := raw.MessageUUID
	if muid == "" {
		muid = raw.MessageID
	}
	if muid == "" && raw.Metadata != nil {
		muid = pick("message_uuid", raw.Metadata)
		if muid == "" {
			muid = pick("messageUuid", raw.Metadata)
		}
	}
	if sid == "" && muid == "" {
		return nil
	}
	// Tool fingerprinting is best-effort. Claude Code stamps "claude-cli"
	// in its user_id; we just default to "claude" when we see a sessionID
	// look that's UUID-ish.
	tool := ""
	if raw.Metadata != nil {
		if v := pick("user_id", raw.Metadata); strings.Contains(v, "claude") {
			tool = "claude"
		}
	}
	if tool == "" && looksLikeUUID(sid) {
		tool = "claude"
	}
	if tool != "" {
		out["conv_tool"] = tool
	}
	if sid != "" {
		out["conv_session_id"] = sid
	}
	if muid != "" {
		out["conv_message_uuid"] = muid
	}
	return out
}

func looksLikeUUID(s string) bool {
	if len(s) != 36 {
		return false
	}
	for i, c := range s {
		switch i {
		case 8, 13, 18, 23:
			if c != '-' {
				return false
			}
		default:
			if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
				return false
			}
		}
	}
	return true
}

// dlpAuditPayload trims a Result down to the metadata an auditor cares
// about. We deliberately omit the redacted body — the journal isn't a
// place to store potentially-sensitive prompt content; the request was
// already filtered.
func dlpAuditPayload(path string, hits []dlp.Hit) map[string]any {
	patterns := make([]string, 0, len(hits))
	seen := map[string]struct{}{}
	for _, h := range hits {
		if _, ok := seen[h.PatternName]; ok {
			continue
		}
		seen[h.PatternName] = struct{}{}
		patterns = append(patterns, h.PatternName)
	}
	return map[string]any{
		"path":     path,
		"hitCount": len(hits),
		"patterns": patterns,
	}
}

// dlpBlockReason composes a short, user-facing message naming the
// pattern(s) that triggered the block. Repeated patterns are deduped so
// the message stays terse even when the prompt had three of the same
// secret in it.
func dlpBlockReason(hits []dlp.Hit) string {
	if len(hits) == 0 {
		return "DLP policy violation"
	}
	seen := map[string]struct{}{}
	var names []string
	for _, h := range hits {
		if h.Policy != dlp.PolicyBlock {
			continue
		}
		if _, ok := seen[h.PatternName]; ok {
			continue
		}
		seen[h.PatternName] = struct{}{}
		names = append(names, h.PatternName)
	}
	if len(names) == 0 {
		return "DLP policy violation"
	}
	return "Lurus Switch blocked this request: matched DLP rule(s) " + strings.Join(names, ", ") + ". See DLP admin page to review or adjust the policy."
}
