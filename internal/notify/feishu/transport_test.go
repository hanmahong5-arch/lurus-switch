package feishu

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
		{"http not https", Config{WebhookURL: "http://insecure.example/x"}, true},
		{"valid", Config{WebhookURL: "https://open.feishu.cn/open-apis/bot/v2/hook/abc"}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.cfg.Validate()
			if (err != nil) != tc.wantErr {
				t.Errorf("Validate() err=%v want error=%v", err, tc.wantErr)
			}
		})
	}
}

// End-to-end happy path: spin up a test HTTP server that pretends to be
// Feishu's webhook endpoint, assert the body looks like a Feishu card,
// and confirm Deliver returns nil on a "code:0" response.
func TestTransport_Deliver_HappyPath(t *testing.T) {
	var capturedBody []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedBody, _ = io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"code":0,"msg":"ok"}`))
	}))
	defer srv.Close()

	tp := New(Config{WebhookURL: srv.URL, HTTPTimeout: 2 * time.Second})
	err := tp.Deliver(context.Background(), notify.Event{
		ID: "e1", Kind: notify.KindToolStuck, Severity: notify.SeverityWarning,
		Title: "psi · Bash 卡 47s", Body: "command: go test ./...",
		Project: "psi", Tool: "claude", Time: time.Now(),
	})
	if err != nil {
		t.Fatalf("Deliver error: %v", err)
	}

	var sent map[string]any
	if err := json.Unmarshal(capturedBody, &sent); err != nil {
		t.Fatalf("body not JSON: %v\n%s", err, capturedBody)
	}
	if sent["msg_type"] != "interactive" {
		t.Errorf("msg_type = %v, want interactive", sent["msg_type"])
	}
	card, _ := sent["card"].(map[string]any)
	header, _ := card["header"].(map[string]any)
	if tpl, _ := header["template"].(string); tpl != "orange" {
		t.Errorf("warning severity should map to orange; got %q", tpl)
	}
	title, _ := header["title"].(map[string]any)
	if got, _ := title["content"].(string); !strings.Contains(got, "psi") {
		t.Errorf("title should contain project name; got %q", got)
	}
}

// Feishu always returns HTTP 200, then encodes the real result in `code`.
// A non-zero code MUST surface as a delivery error — otherwise typo'd
// webhook URLs look like success and the user never sees the bug.
func TestTransport_Deliver_FeishuLevelError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"code":19021,"msg":"sign match fail or timestamp expired"}`))
	}))
	defer srv.Close()

	tp := New(Config{WebhookURL: srv.URL, HTTPTimeout: 2 * time.Second})
	err := tp.Deliver(context.Background(), notify.Event{ID: "e1", Title: "x"})
	if err == nil || !strings.Contains(err.Error(), "19021") {
		t.Errorf("expected error containing 19021; got %v", err)
	}
}

// When signing is configured, the request must include `timestamp` and
// `sign` fields at the top level alongside msg_type — that's the wire
// shape Feishu's server validates against.
func TestTransport_Deliver_SignsWhenSecretSet(t *testing.T) {
	var sent map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&sent)
		_, _ = w.Write([]byte(`{"code":0}`))
	}))
	defer srv.Close()

	tp := New(Config{
		WebhookURL:  srv.URL,
		Secret:      "supersecret",
		HTTPTimeout: 2 * time.Second,
	})
	if err := tp.Deliver(context.Background(), notify.Event{ID: "x", Title: "t"}); err != nil {
		t.Fatalf("Deliver error: %v", err)
	}
	if _, ok := sent["timestamp"]; !ok {
		t.Error("signed message missing timestamp")
	}
	if _, ok := sent["sign"]; !ok {
		t.Error("signed message missing sign")
	}
	// Without a secret, those fields must NOT appear.
	tp2 := New(Config{WebhookURL: srv.URL, HTTPTimeout: 2 * time.Second})
	sent = nil
	_ = tp2.Deliver(context.Background(), notify.Event{ID: "x", Title: "t"})
	if _, ok := sent["sign"]; ok {
		t.Error("unsigned message should not include sign")
	}
}

// Webhook-mode transport must declare it can NOT carry approval round-
// trips, so the bus filters approval events away from it (a card with
// buttons that don't work would be worse than no card).
func TestTransport_DoesNotSupportApproval(t *testing.T) {
	tp := New(Config{WebhookURL: "https://example.com/x"})
	if tp.SupportsApproval() {
		t.Error("webhook transport must return false for SupportsApproval")
	}
}
