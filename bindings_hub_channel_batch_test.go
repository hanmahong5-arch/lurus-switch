package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestHubBatchSetChannelTag verifies the binding forwards ids + tag to
// Hub's POST /api/channel/batch/tag endpoint.
func TestHubBatchSetChannelTag(t *testing.T) {
	var got struct {
		Ids []int  `json:"ids"`
		Tag string `json:"tag"`
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/channel/batch/tag" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		_ = json.NewDecoder(r.Body).Decode(&got)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"success": true})
	}))
	defer srv.Close()

	withFakeHub(t, srv.URL)
	app := &App{}

	if err := app.HubBatchSetChannelTag([]int{7, 8, 9}, "vip"); err != nil {
		t.Fatalf("HubBatchSetChannelTag: %v", err)
	}
	if got.Tag != "vip" || len(got.Ids) != 3 {
		t.Errorf("unexpected payload: %+v", got)
	}
}

// TestHubFetchChannelModels verifies the binding round-trips the model list.
func TestHubFetchChannelModels(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/channel/fetch_models/5" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"success": true,
			"data":    []string{"claude-opus-4-7", "claude-sonnet-4-6"},
		})
	}))
	defer srv.Close()

	withFakeHub(t, srv.URL)
	app := &App{}

	models, err := app.HubFetchChannelModels(5)
	if err != nil {
		t.Fatalf("HubFetchChannelModels: %v", err)
	}
	if len(models) != 2 || models[0] != "claude-opus-4-7" {
		t.Errorf("unexpected models: %v", models)
	}
}

// TestHubFixChannelAbilities verifies the binding sends a no-body POST.
func TestHubFixChannelAbilities(t *testing.T) {
	hit := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/channel/fix" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		hit = true
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"success": true})
	}))
	defer srv.Close()

	withFakeHub(t, srv.URL)
	app := &App{}

	if err := app.HubFixChannelAbilities(); err != nil {
		t.Fatalf("HubFixChannelAbilities: %v", err)
	}
	if !hit {
		t.Error("server handler never invoked")
	}
}
