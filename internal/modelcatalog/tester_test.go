package modelcatalog

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func collect(ch <-chan TestResult) map[string]TestResult {
	out := make(map[string]TestResult)
	for r := range ch {
		out[r.ProviderID] = r
	}
	return out
}

func TestProbe_OK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"data":[{"id":"gpt-4o"},{"id":"gpt-4o-mini"}]}`))
	}))
	defer srv.Close()

	tr := NewTester()
	res := collect(tr.RunHealthCheck(context.Background(), []ProviderEndpoint{
		{ID: "p1", Name: "P1", BaseURL: srv.URL},
	}))
	got := res["p1"]
	if got.Status != StatusOK {
		t.Fatalf("status = %s, want ok", got.Status)
	}
	if len(got.Models) != 2 {
		t.Errorf("models = %v, want 2", got.Models)
	}
}

func TestProbe_Auth(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	tr := NewTester()
	res := collect(tr.RunHealthCheck(context.Background(), []ProviderEndpoint{{ID: "p1", BaseURL: srv.URL}}))
	if res["p1"].Status != StatusAuth {
		t.Errorf("status = %s, want auth", res["p1"].Status)
	}
}

func TestProbe_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	tr := NewTester()
	res := collect(tr.RunHealthCheck(context.Background(), []ProviderEndpoint{{ID: "p1", BaseURL: srv.URL}}))
	if res["p1"].Status != StatusError {
		t.Errorf("status = %s, want error", res["p1"].Status)
	}
}

func TestProbe_Timeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		w.Write([]byte(`{"data":[]}`))
	}))
	defer srv.Close()

	tr := &Tester{Workers: 2, Timeout: 20 * time.Millisecond}
	res := collect(tr.RunHealthCheck(context.Background(), []ProviderEndpoint{{ID: "p1", BaseURL: srv.URL}}))
	if res["p1"].Status != StatusTimeout {
		t.Errorf("status = %s, want timeout", res["p1"].Status)
	}
}

func TestProbe_Unreachable(t *testing.T) {
	tr := &Tester{Workers: 2, Timeout: time.Second}
	// Reserved TEST-NET-1 address that won't connect.
	res := collect(tr.RunHealthCheck(context.Background(), []ProviderEndpoint{
		{ID: "p1", BaseURL: "http://192.0.2.1:9/v1"},
	}))
	st := res["p1"].Status
	if st != StatusUnreachable && st != StatusTimeout {
		t.Errorf("status = %s, want unreachable/timeout", st)
	}
}

func TestProbe_EmptyBaseURL(t *testing.T) {
	tr := NewTester()
	res := collect(tr.RunHealthCheck(context.Background(), []ProviderEndpoint{{ID: "p1"}}))
	if res["p1"].Status != StatusError {
		t.Errorf("status = %s, want error for empty base URL", res["p1"].Status)
	}
}

func TestRunHealthCheck_AllResultsDelivered(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"data":[{"id":"m"}]}`))
	}))
	defer srv.Close()

	eps := make([]ProviderEndpoint, 25) // exceed worker pool to exercise the semaphore
	for i := range eps {
		eps[i] = ProviderEndpoint{ID: string(rune('a' + i)), BaseURL: srv.URL}
	}
	tr := NewTester()
	res := collect(tr.RunHealthCheck(context.Background(), eps))
	if len(res) != 25 {
		t.Errorf("got %d results, want 25 (concurrency lost results?)", len(res))
	}
}

func TestModelsEndpoint(t *testing.T) {
	cases := map[string]string{
		"https://api.deepseek.com":   "https://api.deepseek.com/v1/models",
		"https://api.openai.com/v1":  "https://api.openai.com/v1/models",
		"https://x.test/v1/models":   "https://x.test/v1/models",
		"http://localhost:11434/v1/": "http://localhost:11434/v1/models",
	}
	for in, want := range cases {
		if got := modelsEndpoint(in); got != want {
			t.Errorf("modelsEndpoint(%q) = %q, want %q", in, got, want)
		}
	}
}
