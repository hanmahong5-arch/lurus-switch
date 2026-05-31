package gateway

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"strings"
	"time"
)

type contextKey string

const metaKey contextKey = "request_meta"

// withAuth is middleware that validates per-app tokens and injects RequestMeta.
func (s *Server) withAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := extractBearerToken(r)
		if token == "" {
			writeOpenAIError(w, http.StatusUnauthorized, "missing_api_key",
				"Authorization header with Bearer token is required. Get your token from Lurus Switch.")
			return
		}

		// Look up app by token.
		appID := s.registry.LookupByToken(token)
		if appID == "" {
			writeOpenAIError(w, http.StatusUnauthorized, "invalid_api_key",
				"Invalid API key. Check your token in Lurus Switch → Connected Apps.")
			return
		}

		// Update last-seen (non-blocking).
		go s.registry.TouchLastSeen(appID)

		// Pull the chargeback dimensions off the registry record so
		// downstream metering can attribute the cost to the right
		// employee + cost-center. Empty when the admin hasn't bound
		// the app yet — the chargeback page surfaces those in the
		// "unattributed" bucket.
		empID, cc := s.registry.LookupOwnership(appID)

		// Stable correlation key. Prefer the client's idempotency header so an
		// SDK retry of the SAME request dedups against one booking; fall back
		// to a vendor request id, then to a generated id. Priority matters:
		// Idempotency-Key is the explicit "this is a retry of that" contract,
		// while X-Request-Id is best-effort. (A client that reuses one fixed
		// key across DIFFERENT requests would under-count — that is a client
		// contract violation, documented here so the behavior is intentional.)
		reqID := firstNonEmpty(
			strings.TrimSpace(r.Header.Get("Idempotency-Key")),
			strings.TrimSpace(r.Header.Get("X-Request-Id")),
		)
		if reqID == "" {
			reqID = generateRequestID()
		}

		// Inject metadata for downstream handlers.
		meta := &RequestMeta{
			AppID:           appID,
			StartTime:       time.Now(),
			RequestID:       reqID,
			OwnerEmployeeID: empID,
			CostCenter:      cc,
		}
		ctx := context.WithValue(r.Context(), metaKey, meta)
		next(w, r.WithContext(ctx))
	}
}

// firstNonEmpty returns the first argument that is not the empty string, or ""
// when every argument is empty.
func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}

// generateRequestID mints a random correlation id for requests that arrive
// without a client idempotency / request header. Same 8-byte hex shape as
// metering.generateRecordID so ids are visually consistent across the codebase.
func generateRequestID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// getMeta extracts RequestMeta from context. Returns nil if not present.
func getMeta(r *http.Request) *RequestMeta {
	v := r.Context().Value(metaKey)
	if v == nil {
		return nil
	}
	return v.(*RequestMeta)
}

// extractBearerToken gets the token from the Authorization header.
func extractBearerToken(r *http.Request) string {
	auth := r.Header.Get("Authorization")
	if auth == "" {
		return ""
	}
	const prefix = "Bearer "
	if !strings.HasPrefix(auth, prefix) {
		return ""
	}
	return strings.TrimSpace(auth[len(prefix):])
}

// writeOpenAIError writes an error response in OpenAI API format.
func writeOpenAIError(w http.ResponseWriter, status int, errType, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	// OpenAI error format: {"error": {"message": "...", "type": "...", "code": "..."}}
	w.Write([]byte(`{"error":{"message":"` + escapeJSON(message) + `","type":"` + errType + `","code":"` + errType + `"}}`))
}

// escapeJSON does minimal JSON string escaping for error messages.
func escapeJSON(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	s = strings.ReplaceAll(s, "\n", `\n`)
	s = strings.ReplaceAll(s, "\r", `\r`)
	s = strings.ReplaceAll(s, "\t", `\t`)
	return s
}
