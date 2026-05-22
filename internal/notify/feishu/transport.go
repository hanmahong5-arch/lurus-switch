// Package feishu implements notify.Transport for Lark / Feishu custom-bot
// webhooks. This is the "phase 1" half of cc-connect-style integration:
// outbound push only — interactive cards (approval round-trip) require
// the open-platform app + WebSocket long-conn, which lands as a separate
// transport mode in phase 2.
//
// The webhook endpoint shape is documented at:
//
//	https://open.feishu.cn/document/client-docs/bot-v3/add-custom-bot
//
// We deliberately produce *interactive* card messages (not plain text) —
// they render with a coloured header that matches event severity and
// scale better when we later add buttons. The "card" payload is identical
// across webhook + app modes, so phase 2 just swaps the dispatch shell.
package feishu

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"lurus-switch/internal/notify"
)

// Name is the stable identifier the bus uses for this transport.
const Name = "feishu"

// Config is the per-user webhook setup persisted to disk. Both fields
// originate in the Feishu group's "Bots → Custom Bot" configuration UI.
type Config struct {
	// WebhookURL is the full URL from the custom bot's "Webhook 地址" field.
	// Example: https://open.feishu.cn/open-apis/bot/v2/hook/abc123...
	WebhookURL string `json:"webhookUrl"`
	// Secret is the optional signing secret. Empty = no signing.
	Secret string `json:"secret,omitempty"`
	// HTTPTimeout overrides the default 10s outbound HTTP timeout. Zero
	// uses the default; primarily a knob for tests.
	HTTPTimeout time.Duration `json:"-"`
}

// Validate reports whether the config is usable. Used by the Settings UI
// to disable the "Save" button until the user fills in the webhook URL.
func (c Config) Validate() error {
	if strings.TrimSpace(c.WebhookURL) == "" {
		return fmt.Errorf("Feishu webhook URL 必填")
	}
	if !strings.HasPrefix(c.WebhookURL, "https://") {
		return fmt.Errorf("Feishu webhook URL 必须是 https://")
	}
	return nil
}

// Transport implements notify.Transport against a Feishu custom-bot webhook.
type Transport struct {
	cfg    Config
	client *http.Client
}

// New returns a Transport ready for Bus.Register. Call Validate on cfg
// before construction — New itself doesn't refuse a bad config so callers
// can register a placeholder and re-register once the user finishes
// filling in the form.
func New(cfg Config) *Transport {
	timeout := cfg.HTTPTimeout
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	return &Transport{cfg: cfg, client: &http.Client{Timeout: timeout}}
}

// Name returns the bus-wide identifier.
func (t *Transport) Name() string { return Name }

// SupportsApproval reports false for the webhook variant — Feishu custom
// bots can't receive button-tap callbacks. The open-platform app variant
// (phase 2) will return true and override this method by registering a
// different Transport type.
func (*Transport) SupportsApproval() bool { return false }

// Deliver renders ev to a Feishu interactive card and POSTs it. The
// request body's shape varies depending on whether signing is enabled
// (cfg.Secret non-empty).
func (t *Transport) Deliver(ctx context.Context, ev notify.Event) error {
	// Settings UI invokes Validate() at save-time; here we only need a
	// non-empty URL. Allowing http:// at delivery time keeps integration
	// tests (httptest) working without weakening the user-facing form.
	if strings.TrimSpace(t.cfg.WebhookURL) == "" {
		return fmt.Errorf("feishu transport not configured")
	}
	body, err := t.buildPayload(ev)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, t.cfg.WebhookURL, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json; charset=utf-8")

	resp, err := t.client.Do(req)
	if err != nil {
		return fmt.Errorf("feishu http: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("feishu webhook returned HTTP %d", resp.StatusCode)
	}
	// Feishu always returns HTTP 200 then encodes the real result inside
	// the JSON body's `code` field (0 = ok, non-zero = error). Read it so
	// "webhook URL invalid" doesn't silently look like success.
	var envelope struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
		// Body wasn't JSON — uncommon but treat as success since HTTP 200.
		return nil
	}
	if envelope.Code != 0 {
		return fmt.Errorf("feishu rejected message (code=%d): %s", envelope.Code, envelope.Msg)
	}
	return nil
}

// buildPayload assembles the JSON body. When signing is enabled, the
// HMAC and timestamp ride alongside msg_type / card at the top level —
// Feishu's docs are clear that signed fields go OUTSIDE the card.
func (t *Transport) buildPayload(ev notify.Event) ([]byte, error) {
	payload := map[string]any{
		"msg_type": "interactive",
		"card":     buildCard(ev),
	}
	if t.cfg.Secret != "" {
		ts := time.Now().Unix()
		payload["timestamp"] = strconv.FormatInt(ts, 10)
		payload["sign"] = signTimestamp(ts, t.cfg.Secret)
	}
	return json.Marshal(payload)
}

// signTimestamp returns the base64(HMAC-SHA256(empty, key=ts+"\n"+secret))
// digest required by Feishu's "签名校验" feature. The empty payload is
// intentional — that's what the Feishu reference implementation hashes.
func signTimestamp(ts int64, secret string) string {
	key := strconv.FormatInt(ts, 10) + "\n" + secret
	h := hmac.New(sha256.New, []byte(key))
	// Per Feishu spec we hash an empty body.
	_, _ = h.Write(nil)
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}

// headerTemplate maps Severity onto Feishu's coloured-header presets.
// The values come from Feishu's documented `template` enum.
func headerTemplate(s notify.Severity) string {
	switch s {
	case notify.SeverityError:
		return "red"
	case notify.SeverityWarning:
		return "orange"
	case notify.SeveritySuccess:
		return "green"
	default:
		return "blue"
	}
}

// buildCard turns an Event into the "card" field. Layout:
//
//   ┌────────────────────────────────────┐
//   │ Title                          (header)
//   ├────────────────────────────────────┤
//   │ Project / Tool chip row            │
//   │ Body text (lark_md)                │
//   │ Footer: kind + timestamp           │
//   └────────────────────────────────────┘
func buildCard(ev notify.Event) map[string]any {
	elements := []any{}

	// Project + tool chip row, only when at least one is present.
	if ev.Project != "" || ev.Tool != "" {
		parts := []string{}
		if ev.Project != "" {
			parts = append(parts, "📂 "+escapeMd(ev.Project))
		}
		if ev.Tool != "" {
			parts = append(parts, "🔧 "+escapeMd(ev.Tool))
		}
		elements = append(elements, map[string]any{
			"tag": "div",
			"text": map[string]any{
				"tag":     "lark_md",
				"content": strings.Join(parts, "  ·  "),
			},
		})
	}

	if ev.Body != "" {
		elements = append(elements, map[string]any{
			"tag": "div",
			"text": map[string]any{
				"tag":     "lark_md",
				"content": ev.Body,
			},
		})
	}

	// Always include a footer with kind + timestamp so the user can spot
	// the source class at a glance. Feishu's "note" element renders this
	// in a subtler grey.
	t := ev.Time
	if t.IsZero() {
		t = time.Now()
	}
	elements = append(elements, map[string]any{
		"tag": "note",
		"elements": []any{
			map[string]any{
				"tag":     "plain_text",
				"content": fmt.Sprintf("%s · %s", ev.Kind, t.Format("2006-01-02 15:04:05")),
			},
		},
	})

	return map[string]any{
		"header": map[string]any{
			"template": headerTemplate(ev.Severity),
			"title": map[string]any{
				"tag":     "plain_text",
				"content": ev.Title,
			},
		},
		"elements": elements,
	}
}

// escapeMd protects user-supplied strings from being interpreted as
// lark_md syntax (e.g. an asterisk in a project name would otherwise
// become bold). Conservative — escape the few characters Feishu's
// lark_md flavour treats as control characters.
func escapeMd(s string) string {
	r := strings.NewReplacer(
		"*", "\\*",
		"_", "\\_",
		"`", "\\`",
		"[", "\\[",
	)
	return r.Replace(s)
}
