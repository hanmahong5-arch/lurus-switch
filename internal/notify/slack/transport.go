// Package slack implements notify.Transport for Slack Incoming Webhooks.
// Outbound push only — interactive Block Kit actions need a separate
// request receiver (a later transport mode); this posts a coloured
// attachment whose colour tracks event severity.
//
// API reference:
//
//	https://api.slack.com/messaging/webhooks
//	https://api.slack.com/reference/messaging/attachments
//
// We use the legacy `attachments` shape (not Block Kit) deliberately: a
// single attachment with a `color` bar is the cheapest way to carry the
// severity signal, and every Incoming Webhook supports it.
package slack

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"lurus-switch/internal/notify"
)

// Name is the stable identifier the bus uses for this transport.
const Name = "slack"

// defaultColor is the attachment bar colour for non-severity events
// (Info / Test). Lurus brand blue.
const defaultColor = "#2563eb"

// Config is the per-user webhook setup persisted to disk. The URL comes
// from Slack's "Incoming Webhooks → Add New Webhook to Workspace" flow.
type Config struct {
	// WebhookURL is the full Incoming Webhook URL, e.g.
	// https://hooks.slack.com/services/T000/B000/XXXX
	WebhookURL string `json:"webhookUrl"`
	// HTTPTimeout overrides the default 10s outbound HTTP timeout. Zero
	// uses the default; primarily a knob for tests.
	HTTPTimeout time.Duration `json:"-"`
}

// Validate reports whether the config is usable. Used by the Settings UI
// to surface a save error before the transport is registered.
func (c Config) Validate() error {
	if strings.TrimSpace(c.WebhookURL) == "" {
		return fmt.Errorf("Slack Webhook URL 必填")
	}
	if !strings.HasPrefix(c.WebhookURL, "https://") {
		return fmt.Errorf("Slack Webhook URL 必须是 https://")
	}
	return nil
}

// Transport implements notify.Transport against a Slack Incoming Webhook.
type Transport struct {
	cfg    Config
	client *http.Client
}

// New returns a Transport ready for Bus.Register. Call Validate on cfg
// before construction — New itself doesn't refuse a bad config so callers
// can register a placeholder and re-register once the form is complete.
func New(cfg Config) *Transport {
	timeout := cfg.HTTPTimeout
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	return &Transport{cfg: cfg, client: &http.Client{Timeout: timeout}}
}

// Name returns the bus-wide identifier.
func (t *Transport) Name() string { return Name }

// SupportsApproval reports false — the webhook variant can't carry a
// button-tap round-trip (that needs an interactivity request URL).
func (*Transport) SupportsApproval() bool { return false }

// Deliver renders ev to a Slack attachment and POSTs it. Like feishu,
// delivery only requires a non-empty URL (allowing http:// so httptest
// works); the https check lives in Validate for the user-facing form.
func (t *Transport) Deliver(ctx context.Context, ev notify.Event) error {
	if strings.TrimSpace(t.cfg.WebhookURL) == "" {
		return fmt.Errorf("slack transport not configured")
	}
	body, err := json.Marshal(buildPayload(ev))
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
		return fmt.Errorf("slack http: %w", err)
	}
	defer resp.Body.Close()

	// Slack Incoming Webhooks return HTTP 200 with body "ok" on success and
	// a non-200 with a plain-text reason ("invalid_payload", "no_service",
	// …) on failure — unlike feishu/telegram there's no 200-but-failed JSON
	// envelope, so the status code is the whole signal.
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("slack webhook returned HTTP %d", resp.StatusCode)
	}
	return nil
}

// buildPayload assembles the JSON body: a single coloured attachment
// carrying title / body / footer, with the colour bar set by severity.
func buildPayload(ev notify.Event) map[string]any {
	tm := ev.Time
	if tm.IsZero() {
		tm = time.Now()
	}

	footer := make([]string, 0, 3)
	if ev.Project != "" {
		footer = append(footer, "📂 "+ev.Project)
	}
	if ev.Tool != "" {
		footer = append(footer, "🔧 "+ev.Tool)
	}
	footer = append(footer, string(ev.Kind))

	attachment := map[string]any{
		"color":  attachmentColor(ev.Severity),
		"title":  ev.Title,
		"text":   ev.Body,
		"footer": strings.Join(footer, "  ·  "),
		"ts":     tm.Unix(),
	}
	return map[string]any{
		"attachments": []any{attachment},
	}
}

// attachmentColor maps Severity onto Slack's documented attachment colour
// presets ("good"/"warning"/"danger"), falling back to brand blue.
func attachmentColor(s notify.Severity) string {
	switch s {
	case notify.SeveritySuccess:
		return "good"
	case notify.SeverityWarning:
		return "warning"
	case notify.SeverityError:
		return "danger"
	default:
		return defaultColor
	}
}
