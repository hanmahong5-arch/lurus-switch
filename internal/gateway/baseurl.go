package gateway

import "strings"

// NormalizeChannelBaseURL fixes a common misconfiguration where a channel's
// base URL includes a trailing "/v1" or "/v1/". Since the gateway appends
// the request path (which already starts with /v1/chat/completions), having
// /v1 in the base URL produces a doubled path: /v1/v1/chat/completions → 404.
//
// Learned the hard way from Groq integration (2026-04-11):
//   - Correct:   https://api.groq.com/openai        → /openai/v1/chat/completions ✓
//   - WRONG:     https://api.groq.com/openai/v1      → /openai/v1/v1/chat/completions ✗
//
// This function strips trailing /v1 so the proxy can safely append the full path.
// It also handles /v1/ (with trailing slash) variants.
func NormalizeChannelBaseURL(baseURL string) string {
	u := strings.TrimRight(baseURL, "/")
	if strings.HasSuffix(u, "/v1") {
		u = strings.TrimSuffix(u, "/v1")
	}
	return u
}

// ValidateChannelBaseURL returns a warning if the base URL looks misconfigured.
// Returns empty string if the URL is fine.
func ValidateChannelBaseURL(baseURL string) string {
	u := strings.TrimRight(baseURL, "/")
	if strings.HasSuffix(u, "/v1") {
		return "Base URL ends with /v1 — this will cause path duplication (/v1/v1/...). The /v1 suffix will be auto-stripped."
	}
	if strings.HasSuffix(u, "/v1beta") {
		return "Base URL ends with /v1beta — verify this is correct for the provider (Gemini uses /v1beta natively)."
	}
	if !strings.HasPrefix(u, "http://") && !strings.HasPrefix(u, "https://") {
		return "Base URL must start with http:// or https://."
	}
	return ""
}
