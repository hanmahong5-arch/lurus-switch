package admin

import (
	"context"
	"encoding/json"
	"net/http"
	"reflect"
	"testing"
)

func TestBatchSetChannelTag_PostsIDsAndTag(t *testing.T) {
	var got struct {
		Ids []int  `json:"ids"`
		Tag string `json:"tag"`
	}
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/channel/batch/tag" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		_ = json.NewDecoder(r.Body).Decode(&got)
		envRespond(w, nil)
	})
	if err := c.BatchSetChannelTag(context.Background(), []int{1, 2, 3}, "premium"); err != nil {
		t.Fatalf("BatchSetChannelTag: %v", err)
	}
	if !reflect.DeepEqual(got.Ids, []int{1, 2, 3}) || got.Tag != "premium" {
		t.Errorf("unexpected payload: %+v", got)
	}
}

func TestEnableChannelsByTag(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/channel/tag/enabled" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		envRespond(w, nil)
	})
	if err := c.EnableChannelsByTag(context.Background(), "premium"); err != nil {
		t.Fatalf("EnableChannelsByTag: %v", err)
	}
}

func TestDisableChannelsByTag(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/channel/tag/disabled" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		envRespond(w, nil)
	})
	if err := c.DisableChannelsByTag(context.Background(), "legacy"); err != nil {
		t.Fatalf("DisableChannelsByTag: %v", err)
	}
}

func TestEditChannelTag_UsesPutWithNewTag(t *testing.T) {
	var got struct {
		Tag    string `json:"tag"`
		NewTag string `json:"new_tag"`
	}
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut || r.URL.Path != "/api/channel/tag" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		_ = json.NewDecoder(r.Body).Decode(&got)
		envRespond(w, nil)
	})
	if err := c.EditChannelTag(context.Background(), "old", "new"); err != nil {
		t.Fatalf("EditChannelTag: %v", err)
	}
	if got.Tag != "old" || got.NewTag != "new" {
		t.Errorf("unexpected payload: %+v", got)
	}
}

func TestFetchChannelModels(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/channel/fetch_models/42" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		envRespond(w, []string{"gpt-4", "gpt-4o", "o1"})
	})
	models, err := c.FetchChannelModels(context.Background(), 42)
	if err != nil {
		t.Fatalf("FetchChannelModels: %v", err)
	}
	if !reflect.DeepEqual(models, []string{"gpt-4", "gpt-4o", "o1"}) {
		t.Errorf("unexpected models: %v", models)
	}
}

func TestFixChannelAbilities(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/channel/fix" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		envRespond(w, nil)
	})
	if err := c.FixChannelAbilities(context.Background()); err != nil {
		t.Fatalf("FixChannelAbilities: %v", err)
	}
}
