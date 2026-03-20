package gateway

import (
	"context"
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

		// Inject metadata for downstream handlers.
		meta := &RequestMeta{
			AppID:     appID,
			StartTime: time.Now(),
		}
		ctx := context.WithValue(r.Context(), metaKey, meta)
		next(w, r.WithContext(ctx))
	}
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
