package provider

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestModelsEndpoint(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"https://api.openai.com/v1", "https://api.openai.com/v1/models"},
		{"https://api.openai.com/v1/", "https://api.openai.com/v1/models"},
		{"https://api.deepseek.com", "https://api.deepseek.com/v1/models"},
		{"https://api.cohere.com/v2", "https://api.cohere.com/v2/models"},
		{"http://localhost:11434/v1", "http://localhost:11434/v1/models"},
		{"https://api.example.com/v1/models", "https://api.example.com/v1/models"},
		{"https://api.example.com/models", "https://api.example.com/models"},
	}
	for _, c := range cases {
		if got := modelsEndpoint(c.in); got != c.want {
			t.Errorf("modelsEndpoint(%q) = %q; want %q", c.in, got, c.want)
		}
	}
}

func TestFetchModels_Success(t *testing.T) {
	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		if r.URL.Path != "/v1/models" {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":[
			{"id":"gpt-4"},
			{"id":"gpt-3.5-turbo"},
			{"id":"gpt-4"}
		]}`))
	}))
	defer srv.Close()

	models, err := FetchModels(context.Background(), srv.URL, "sk-test")
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if gotAuth != "Bearer sk-test" {
		t.Errorf("auth header = %q; want Bearer sk-test", gotAuth)
	}
	// Sorted + deduplicated.
	want := []string{"gpt-3.5-turbo", "gpt-4"}
	if len(models) != len(want) {
		t.Fatalf("got %d models, want %d (%v)", len(models), len(want), models)
	}
	for i := range want {
		if models[i] != want[i] {
			t.Errorf("models[%d] = %q; want %q", i, models[i], want[i])
		}
	}
}

func TestFetchModels_NoAuthHeaderWhenKeyEmpty(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "" {
			t.Errorf("auth header should be empty, got %q", got)
		}
		_, _ = w.Write([]byte(`{"data":[{"id":"llama3"}]}`))
	}))
	defer srv.Close()

	if _, err := FetchModels(context.Background(), srv.URL, ""); err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
}

func TestFetchModels_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":"invalid key"}`))
	}))
	defer srv.Close()

	_, err := FetchModels(context.Background(), srv.URL, "bad")
	if err == nil {
		t.Fatal("expected error for 401")
	}
	if !strings.Contains(err.Error(), "401") {
		t.Errorf("error should mention status: %v", err)
	}
}

func TestFetchModels_EmptyList(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"data":[]}`))
	}))
	defer srv.Close()

	_, err := FetchModels(context.Background(), srv.URL, "")
	if err == nil {
		t.Fatal("expected error for empty list")
	}
	if !strings.Contains(err.Error(), "no models") {
		t.Errorf("error should mention empty list: %v", err)
	}
}

func TestFetchModels_MalformedJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`<html>not json</html>`))
	}))
	defer srv.Close()

	if _, err := FetchModels(context.Background(), srv.URL, ""); err == nil {
		t.Fatal("expected parse error")
	}
}

func TestFetchModels_EmptyBaseURL(t *testing.T) {
	if _, err := FetchModels(context.Background(), "  ", "key"); err == nil {
		t.Fatal("expected error for empty base URL")
	}
}
