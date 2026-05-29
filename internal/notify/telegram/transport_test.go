package telegram

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
		{"token only", Config{BotToken: "123:ABC"}, true},
		{"chat only", Config{ChatID: "42"}, true},
		{"valid", Config{BotToken: "123:ABC", ChatID: "42"}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := tc.cfg.Validate(); (err != nil) != tc.wantErr {
				t.Errorf("Validate() err=%v want error=%v", err, tc.wantErr)
			}
		})
	}
}

// Happy path: stand up a fake Telegram API, assert the request hits the
// /bot<token>/sendMessage path with a JSON body carrying chat_id + text,
// and that an {"ok":true} response yields a nil error.
func TestTransport_Deliver_HappyPath(t *testing.T) {
	var capturedPath string
	var capturedBody []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.Path
		capturedBody, _ = io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true,"result":{"message_id":7}}`))
	}))
	defer srv.Close()

	tp := New(Config{
		BotToken:    "123456:ABCDEF",
		ChatID:      "-1001234567890",
		APIBaseURL:  srv.URL,
		HTTPTimeout: 2 * time.Second,
	})
	err := tp.Deliver(context.Background(), notify.Event{
		ID: "e1", Kind: notify.KindToolStuck, Severity: notify.SeverityWarning,
		Title: "psi · Bash 卡 47s", Body: "command: go test ./...",
		Project: "psi", Tool: "claude", Time: time.Unix(1700000000, 0),
	})
	if err != nil {
		t.Fatalf("Deliver error: %v", err)
	}

	if capturedPath != "/bot123456:ABCDEF/sendMessage" {
		t.Errorf("path = %q, want /bot<token>/sendMessage", capturedPath)
	}
	var sent map[string]any
	if err := json.Unmarshal(capturedBody, &sent); err != nil {
		t.Fatalf("body not JSON: %v\n%s", err, capturedBody)
	}
	if sent["chat_id"] != "-1001234567890" {
		t.Errorf("chat_id = %v, want the configured group id", sent["chat_id"])
	}
	text, _ := sent["text"].(string)
	if !strings.Contains(text, "psi · Bash 卡 47s") {
		t.Errorf("text missing title: %q", text)
	}
	if !strings.Contains(text, "tool_stuck") {
		t.Errorf("text missing kind footer: %q", text)
	}
}

// Telegram returns HTTP 200 with {"ok":false,...} for soft errors like a
// bad chat_id. That MUST surface as a delivery error, else a typo'd chat
// looks like success and the user never sees the bug.
func TestTransport_Deliver_OkFalseIsError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":false,"description":"Bad Request: chat not found"}`))
	}))
	defer srv.Close()

	tp := New(Config{BotToken: "t", ChatID: "c", APIBaseURL: srv.URL, HTTPTimeout: 2 * time.Second})
	err := tp.Deliver(context.Background(), notify.Event{ID: "e1", Title: "x"})
	if err == nil || !strings.Contains(err.Error(), "chat not found") {
		t.Errorf("expected error containing the description; got %v", err)
	}
}

func TestTransport_Deliver_Non200IsError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	tp := New(Config{BotToken: "t", ChatID: "c", APIBaseURL: srv.URL, HTTPTimeout: 2 * time.Second})
	if err := tp.Deliver(context.Background(), notify.Event{ID: "e1", Title: "x"}); err == nil {
		t.Error("expected error on HTTP 401")
	}
}

func TestBuildText_OrdersFields(t *testing.T) {
	got := buildText(notify.Event{
		Kind: notify.KindSessionDone, Severity: notify.SeveritySuccess,
		Title: "Done", Body: "all green", Project: "memorus", Tool: "codex",
		Time: time.Unix(1700000000, 0).UTC(),
	})
	for _, want := range []string{"Done", "all green", "📂 memorus", "🔧 codex", "session_done"} {
		if !strings.Contains(got, want) {
			t.Errorf("buildText missing %q in:\n%s", want, got)
		}
	}
	// Title must lead the message.
	if !strings.HasPrefix(got, "Done") {
		t.Errorf("title should be first line, got:\n%s", got)
	}
}

func TestTransport_DoesNotSupportApproval(t *testing.T) {
	tp := New(Config{BotToken: "t", ChatID: "c"})
	if tp.SupportsApproval() {
		t.Error("push-only telegram transport must return false for SupportsApproval")
	}
}
