// Package telegram implements notify.Transport for the Telegram Bot API
// sendMessage method. Like feishu, this is outbound push only: Telegram
// bots can receive callbacks, but that needs a long-poll / webhook
// receiver (a separate transport mode), so this just posts a message.
//
// API reference:
//
//	https://core.telegram.org/bots/api#sendmessage
//
// We send plain text (no parse_mode). Telegram's MarkdownV2 requires
// escaping a long list of characters and silently 400s the whole message
// on a single un-escaped one — not worth it for an event card whose text
// comes from arbitrary project / command strings.
package telegram

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
const Name = "telegram"

// defaultAPIBaseURL is Telegram's public Bot API host. Config.APIBaseURL
// overrides it so integration tests can point at an httptest server.
const defaultAPIBaseURL = "https://api.telegram.org"

// Config is the per-user bot setup persisted to disk. Both BotToken and
// ChatID come from the Telegram side: the token from @BotFather, the
// chat ID from the target conversation.
type Config struct {
	// BotToken is the @BotFather token, e.g. "123456:ABC-DEF...".
	BotToken string `json:"botToken"`
	// ChatID is the target conversation: a numeric user ID for DMs or a
	// "-100..." group/channel ID. Kept as a string so leading "-" and
	// large 64-bit channel IDs survive JSON round-trips intact.
	ChatID string `json:"chatId"`
	// APIBaseURL overrides the Telegram host. Empty = defaultAPIBaseURL;
	// primarily a knob for tests.
	APIBaseURL string `json:"apiBaseUrl,omitempty"`
	// HTTPTimeout overrides the default 10s outbound HTTP timeout. Zero
	// uses the default; primarily a knob for tests.
	HTTPTimeout time.Duration `json:"-"`
}

// Validate reports whether the config is usable. Used by the Settings UI
// to decide whether to register the transport / surface a save error.
func (c Config) Validate() error {
	if strings.TrimSpace(c.BotToken) == "" {
		return fmt.Errorf("Telegram Bot Token 必填")
	}
	if strings.TrimSpace(c.ChatID) == "" {
		return fmt.Errorf("Telegram Chat ID 必填")
	}
	return nil
}

// Transport implements notify.Transport against the Telegram Bot API.
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

// SupportsApproval reports false — the push-only variant can't carry a
// button-tap round-trip (that needs an update receiver, a later mode).
func (*Transport) SupportsApproval() bool { return false }

// Deliver renders ev to plain text and POSTs it to sendMessage. Like
// feishu, delivery only needs a populated config; the strict (https)
// validation lives in Validate so httptest can drive an http:// server.
func (t *Transport) Deliver(ctx context.Context, ev notify.Event) error {
	if strings.TrimSpace(t.cfg.BotToken) == "" || strings.TrimSpace(t.cfg.ChatID) == "" {
		return fmt.Errorf("telegram transport not configured")
	}
	body, err := json.Marshal(map[string]any{
		"chat_id": t.cfg.ChatID,
		"text":    buildText(ev),
	})
	if err != nil {
		return err
	}
	url := t.baseURL() + "/bot" + t.cfg.BotToken + "/sendMessage"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json; charset=utf-8")

	resp, err := t.client.Do(req)
	if err != nil {
		return fmt.Errorf("telegram http: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("telegram sendMessage returned HTTP %d", resp.StatusCode)
	}
	// Telegram returns HTTP 200 with {"ok":false,"description":...} for
	// soft errors like a bad chat_id, so the status code alone isn't
	// enough — same "200-but-failed" trap feishu has. Read the envelope so
	// a typo'd token / chat doesn't silently look like success.
	var envelope struct {
		OK          bool   `json:"ok"`
		Description string `json:"description"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
		// Body wasn't JSON — uncommon but treat as success since HTTP 200.
		return nil
	}
	if !envelope.OK {
		return fmt.Errorf("telegram rejected message: %s", envelope.Description)
	}
	return nil
}

// baseURL returns the configured API host without a trailing slash, or
// the public default when unset.
func (t *Transport) baseURL() string {
	if u := strings.TrimSpace(t.cfg.APIBaseURL); u != "" {
		return strings.TrimRight(u, "/")
	}
	return defaultAPIBaseURL
}

// buildText assembles the plain-text message body. Layout mirrors the
// feishu card's information order: title, body, project/tool chips, then
// a "kind · timestamp" footer line.
func buildText(ev notify.Event) string {
	var b strings.Builder
	b.WriteString(ev.Title)
	if ev.Body != "" {
		b.WriteString("\n\n")
		b.WriteString(ev.Body)
	}

	var chips []string
	if ev.Project != "" {
		chips = append(chips, "📂 "+ev.Project)
	}
	if ev.Tool != "" {
		chips = append(chips, "🔧 "+ev.Tool)
	}
	if len(chips) > 0 {
		b.WriteString("\n\n")
		b.WriteString(strings.Join(chips, "  ·  "))
	}

	tm := ev.Time
	if tm.IsZero() {
		tm = time.Now()
	}
	b.WriteString("\n\n")
	b.WriteString(fmt.Sprintf("%s · %s", ev.Kind, tm.Format("2006-01-02 15:04:05")))
	return b.String()
}
