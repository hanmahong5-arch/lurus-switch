package modelcatalog

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestDefaultCatalog_HasModels(t *testing.T) {
	cat := DefaultCatalog()
	if len(cat.Models) == 0 {
		t.Fatal("default catalog should have models")
	}
	// Verify essential models exist
	ids := map[string]bool{}
	for _, m := range cat.Models {
		ids[m.ID] = true
	}
	for _, want := range []string{"deepseek-chat", "deepseek-reasoner", "claude-sonnet-4-20250514"} {
		if !ids[want] {
			t.Errorf("default catalog missing %q", want)
		}
	}
}

func TestDefaultCatalog_RecommendedModels(t *testing.T) {
	cat := DefaultCatalog()
	recCount := 0
	for _, m := range cat.Models {
		if m.Recommended {
			recCount++
		}
	}
	if recCount == 0 {
		t.Error("should have at least one recommended model")
	}
}

func TestFetch_FromAPI(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/pricing" {
			http.NotFound(w, r)
			return
		}
		models := []map[string]interface{}{
			{"model_name": "deepseek-chat", "model_ratio": 0.07, "completion_ratio": 0.14},
			{"model_name": "gpt-4o", "model_ratio": 1.25, "completion_ratio": 5.0},
		}
		json.NewEncoder(w).Encode(models)
	}))
	defer srv.Close()

	tmpDir := t.TempDir()
	mgr := NewManager(tmpDir)
	cat, err := mgr.Fetch(context.Background(), srv.URL, "")
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	if len(cat.Models) != 2 {
		t.Fatalf("expected 2 models, got %d", len(cat.Models))
	}
	if cat.Models[0].ID != "deepseek-chat" {
		t.Errorf("first model = %q, want deepseek-chat", cat.Models[0].ID)
	}
}

func TestFetch_FallbackToCache(t *testing.T) {
	tmpDir := t.TempDir()
	// Write a cached catalog
	cached := &Catalog{
		FetchedAt: time.Now().Add(-30 * time.Minute), // 30 min old, within TTL
		Models: []Model{
			{ID: "cached-model", DisplayName: "Cached", Provider: "Test"},
		},
	}
	data, _ := json.Marshal(cached)
	os.WriteFile(filepath.Join(tmpDir, cacheFile), data, 0600)

	mgr := NewManager(tmpDir)
	cat, err := mgr.Fetch(context.Background(), "", "")
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	if len(cat.Models) != 1 || cat.Models[0].ID != "cached-model" {
		t.Errorf("expected cached model, got %v", cat.Models)
	}
}

func TestFetch_FallbackToDefault(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := NewManager(tmpDir)
	// No API, no cache → should return defaults
	cat, err := mgr.Fetch(context.Background(), "", "")
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	if len(cat.Models) == 0 {
		t.Error("should fall back to default catalog")
	}
}

func TestInferProvider(t *testing.T) {
	tests := []struct {
		modelID string
		want    string
	}{
		{"deepseek-chat", "DeepSeek"},
		{"deepseek-reasoner", "DeepSeek"},
		{"qwen-plus", "Alibaba"},
		{"qwen-max", "Alibaba"},
		{"glm-4-plus", "Zhipu"},
		{"claude-sonnet-4-20250514", "Anthropic"},
		{"gpt-4o", "OpenAI"},
		{"gemini-pro", "Google"},
		{"unknown-model", "Other"},
	}
	for _, tt := range tests {
		got := inferProvider(tt.modelID)
		if got != tt.want {
			t.Errorf("inferProvider(%q) = %q, want %q", tt.modelID, got, tt.want)
		}
	}
}
