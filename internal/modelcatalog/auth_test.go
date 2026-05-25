package modelcatalog

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func newAuthUpstream(t *testing.T, respModel string, status int) (*httptest.Server, *string) {
	t.Helper()
	var captured string
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/chat/completions") {
			t.Errorf("expected /chat/completions, got %s", r.URL.Path)
		}
		var body struct {
			Model string `json:"model"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		captured = body.Model
		if status != http.StatusOK {
			w.WriteHeader(status)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"model":   respModel,
			"choices": []map[string]any{{"index": 0, "message": map[string]string{"role": "assistant", "content": "ok"}}},
		})
	}))
	return s, &captured
}

func TestProbeAuthenticity_Match(t *testing.T) {
	srv, captured := newAuthUpstream(t, "claude-sonnet-4-6", http.StatusOK)
	defer srv.Close()

	results := ProbeAuthenticity(context.Background(), ProviderEndpoint{
		ID: "p1", Name: "p1", BaseURL: srv.URL, APIKey: "k",
	}, []string{"claude-sonnet-4-6"})

	if len(results) != 1 {
		t.Fatalf("got %d results, want 1", len(results))
	}
	if results[0].Verdict != VerdictMatch {
		t.Errorf("verdict = %v, want match (note=%s)", results[0].Verdict, results[0].Note)
	}
	if *captured != "claude-sonnet-4-6" {
		t.Errorf("upstream received model = %q, want claude-sonnet-4-6", *captured)
	}
}

func TestProbeAuthenticity_Mismatch(t *testing.T) {
	// Requested claude-opus, server says it served claude-haiku — silent
	// downgrade exactly the kind of fraud this check exists to catch.
	srv, _ := newAuthUpstream(t, "claude-haiku-4-5", http.StatusOK)
	defer srv.Close()

	results := ProbeAuthenticity(context.Background(), ProviderEndpoint{
		ID: "p", Name: "p", BaseURL: srv.URL,
	}, []string{"claude-opus-4-7"})

	if results[0].Verdict != VerdictMismatch {
		t.Errorf("verdict = %v, want mismatch", results[0].Verdict)
	}
	if results[0].ReportedModel != "claude-haiku-4-5" {
		t.Errorf("ReportedModel = %q", results[0].ReportedModel)
	}
}

func TestProbeAuthenticity_Inconclusive_NoModelField(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"choices": []map[string]any{{"index": 0}}})
	}))
	defer srv.Close()

	results := ProbeAuthenticity(context.Background(), ProviderEndpoint{
		ID: "p", Name: "p", BaseURL: srv.URL,
	}, []string{"claude-opus-4-7"})

	if results[0].Verdict != VerdictInconclusive {
		t.Errorf("verdict = %v, want inconclusive", results[0].Verdict)
	}
}

func TestProbeAuthenticity_AuthFailure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	results := ProbeAuthenticity(context.Background(), ProviderEndpoint{
		ID: "p", Name: "p", BaseURL: srv.URL,
	}, []string{"x"})

	if results[0].Verdict != VerdictAuth {
		t.Errorf("verdict = %v, want auth", results[0].Verdict)
	}
}

func TestProbeAuthenticity_Timeout(t *testing.T) {
	// Server that holds the request open longer than the probe budget.
	// Constructed by setting an unreachably-short context.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// hang for longer than probe budget — but we'll short-circuit
		// via a parent context with 1ms timeout below
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()
	results := ProbeAuthenticity(ctx, ProviderEndpoint{
		ID: "p", Name: "p", BaseURL: srv.URL,
	}, []string{"x"})

	v := results[0].Verdict
	if v != VerdictTimeout && v != VerdictUnreachable {
		t.Errorf("verdict = %v, want timeout or unreachable", v)
	}
}

func TestProbeAuthenticity_PrefixMatch(t *testing.T) {
	// Upstream canonicalises "claude-sonnet-4-6" → "claude-sonnet-4-6-20250601".
	// Our matcher must treat that as a match, not a mismatch.
	srv, _ := newAuthUpstream(t, "claude-sonnet-4-6-20250601", http.StatusOK)
	defer srv.Close()

	results := ProbeAuthenticity(context.Background(), ProviderEndpoint{
		ID: "p", Name: "p", BaseURL: srv.URL,
	}, []string{"claude-sonnet-4-6"})

	if results[0].Verdict != VerdictMatch {
		t.Errorf("dated-variant should match base, got %v", results[0].Verdict)
	}
}

func TestProbeAuthenticity_EmptyModelsSkipped(t *testing.T) {
	srv, _ := newAuthUpstream(t, "x", http.StatusOK)
	defer srv.Close()
	res := ProbeAuthenticity(context.Background(), ProviderEndpoint{
		ID: "p", Name: "p", BaseURL: srv.URL,
	}, []string{"", "  ", "x"})
	if len(res) != 1 {
		t.Errorf("empty / whitespace models should be skipped, got %d results", len(res))
	}
}
