package slack

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"lurus-switch/internal/notify"
)

func TestConfig_Validate(t *testing.T) {
	cases := []struct {
		name    string
		cfg     Config
		wantErr bool
	}{
		{"empty", Config{}, true},
		{"http not https", Config{WebhookURL: "http://hooks.slack.com/services/x"}, true},
		{"valid", Config{WebhookURL: "https://hooks.slack.com/services/T/B/X"}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := tc.cfg.Validate(); (err != nil) != tc.wantErr {
				t.Errorf("Validate() err=%v want error=%v", err, tc.wantErr)
			}
		})
	}
}

// Happy path: assert the POST body carries one attachment whose colour
// tracks severity and whose title is the event title.
func TestTransport_Deliver_HappyPath(t *testing.T) {
	var capturedBody []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedBody, _ = io.ReadAll(r.Body)
		_, _ = w.Write([]byte("ok"))
	}))
	defer srv.Close()

	tp := New(Config{WebhookURL: srv.URL, HTTPTimeout: 2 * time.Second})
	err := tp.Deliver(context.Background(), notify.Event{
		ID: "e1", Kind: notify.KindToolStuck, Severity: notify.SeverityError,
		Title: "psi · Bash 卡死", Body: "command: rm -rf /",
		Project: "psi", Tool: "claude", Time: time.Unix(1700000000, 0),
	})
	if err != nil {
		t.Fatalf("Deliver error: %v", err)
	}

	var sent struct {
		Attachments []map[string]any `json:"attachments"`
	}
	if err := json.Unmarshal(capturedBody, &sent); err != nil {
		t.Fatalf("body not JSON: %v\n%s", err, capturedBody)
	}
	if len(sent.Attachments) != 1 {
		t.Fatalf("attachments = %d, want 1", len(sent.Attachments))
	}
	att := sent.Attachments[0]
	if att["color"] != "danger" {
		t.Errorf("error severity should map to danger; got %v", att["color"])
	}
	if title, _ := att["title"].(string); !strings.Contains(title, "psi") {
		t.Errorf("title should carry project name; got %v", att["title"])
	}
	if footer, _ := att["footer"].(string); !strings.Contains(footer, "tool_stuck") {
		t.Errorf("footer should carry kind; got %v", att["footer"])
	}
}

func TestTransport_Deliver_Non200IsError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte("no_service"))
	}))
	defer srv.Close()

	tp := New(Config{WebhookURL: srv.URL, HTTPTimeout: 2 * time.Second})
	err := tp.Deliver(context.Background(), notify.Event{ID: "e1", Title: "x"})
	if err == nil || !strings.Contains(err.Error(), "404") {
		t.Errorf("expected error mentioning HTTP 404; got %v", err)
	}
}

func TestAttachmentColor_BySeverity(t *testing.T) {
	cases := map[notify.Severity]string{
		notify.SeveritySuccess: "good",
		notify.SeverityWarning: "warning",
		notify.SeverityError:   "danger",
		notify.SeverityInfo:    defaultColor,
	}
	for sev, want := range cases {
		if got := attachmentColor(sev); got != want {
			t.Errorf("attachmentColor(%q) = %q, want %q", sev, got, want)
		}
	}
}

func TestTransport_DoesNotSupportApproval(t *testing.T) {
	tp := New(Config{WebhookURL: "https://hooks.slack.com/services/x"})
	if tp.SupportsApproval() {
		t.Error("webhook slack transport must return false for SupportsApproval")
	}
}
