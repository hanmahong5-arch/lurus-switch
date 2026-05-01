package redemption

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// redeemEndpoint is the Hub path for anonymous redemption-code exchange.
//
// We deliberately do not use the legacy newapi `/api/user/topup` here — that
// endpoint requires an authenticated user token, which is exactly what the
// EndUser doesn't yet have. The Hub team adds a paired anonymous endpoint
// at `/api/v2/switch/redeem` that:
//   - accepts {code, fingerprint, app_version} unauthenticated
//   - server-side calls newapi RedeemCodeV2 + provisions a UserToken
//   - returns {user_token, user_id, quota, expires_at, tenant_slug}
//
// Until that endpoint ships on the Hub, RedeemHubURL pointed at it will
// 404 — Switch surfaces that as a friendly "Hub 暂未启用激活码兑换" message
// instead of a raw HTTP error.
const redeemEndpoint = "/api/v2/switch/redeem"

// RedeemRequest is the JSON body sent to the Hub redeem endpoint.
type RedeemRequest struct {
	Code        string `json:"code"`
	Fingerprint string `json:"fingerprint"`
	AppVersion  string `json:"app_version,omitempty"`
}

// RedeemResponse is the JSON body the Hub returns on a successful exchange.
// Field names mirror the existing newapi `/user/topup` response plus the
// additions for anonymous flow.
type RedeemResponse struct {
	UserToken  string `json:"user_token"`
	UserID     int    `json:"user_id"`
	Quota      int64  `json:"quota"`
	ExpiresAt  int64  `json:"expires_at,omitempty"` // unix seconds
	TenantSlug string `json:"tenant_slug,omitempty"`
}

// RedeemError is a typed error so the UI can map specific failures (used
// code, expired, network, server) to localized messages.
type RedeemError struct {
	Kind    RedeemErrorKind
	Message string
	Cause   error
}

// RedeemErrorKind enumerates the cases the EndUser activation page reacts to.
type RedeemErrorKind string

const (
	ErrInvalidInput   RedeemErrorKind = "invalid_input"   // empty / malformed inputs
	ErrNetwork        RedeemErrorKind = "network"         // DNS, TCP, TLS failures
	ErrCodeNotFound   RedeemErrorKind = "code_not_found"  // Hub says code doesn't exist
	ErrCodeUsed       RedeemErrorKind = "code_used"       // already redeemed
	ErrCodeExpired    RedeemErrorKind = "code_expired"    // past ExpiredTime
	ErrCodeDisabled   RedeemErrorKind = "code_disabled"   // operator disabled
	ErrEndpointAbsent RedeemErrorKind = "endpoint_absent" // Hub doesn't expose redeem yet (404)
	ErrServer         RedeemErrorKind = "server"          // 5xx, malformed envelope
)

// Error implements the error interface.
func (e *RedeemError) Error() string {
	if e == nil {
		return ""
	}
	if e.Cause != nil {
		return fmt.Sprintf("%s: %s (%v)", e.Kind, e.Message, e.Cause)
	}
	return fmt.Sprintf("%s: %s", e.Kind, e.Message)
}

// Unwrap exposes the underlying cause to errors.Is / errors.As.
func (e *RedeemError) Unwrap() error { return e.Cause }

// IsRedeemError narrows an error to *RedeemError; convenience wrapper for
// the binding layer.
func IsRedeemError(err error) (*RedeemError, bool) {
	var re *RedeemError
	if errors.As(err, &re) {
		return re, true
	}
	return nil, false
}

// Redeemer talks to the Hub redeem endpoint. Reuses one HTTP client so
// keepalive + timeouts can be tuned in one place.
type Redeemer struct {
	httpClient *http.Client
	appVersion string
}

// NewRedeemer constructs a Redeemer with the given app version (used in
// the User-Agent and request body for diagnostics on the Hub side).
func NewRedeemer(appVersion string) *Redeemer {
	return &Redeemer{
		httpClient: &http.Client{Timeout: 15 * time.Second},
		appVersion: appVersion,
	}
}

// Redeem exchanges a code for an Activation against the Hub at hubURL.
// Returns *Activation on success, *RedeemError otherwise.
//
// The fingerprint is captured here (not from the caller) so we can be
// certain the activation is bound to the live machine, not a stale value
// passed by a buggy caller.
func (r *Redeemer) Redeem(ctx context.Context, hubURL, code string) (*Activation, error) {
	hubURL = strings.TrimRight(strings.TrimSpace(hubURL), "/")
	code = strings.TrimSpace(code)
	if hubURL == "" {
		return nil, &RedeemError{Kind: ErrInvalidInput, Message: "Hub URL 必填"}
	}
	if _, err := url.Parse(hubURL); err != nil {
		return nil, &RedeemError{Kind: ErrInvalidInput, Message: "Hub URL 格式不正确", Cause: err}
	}
	if code == "" {
		return nil, &RedeemError{Kind: ErrInvalidInput, Message: "激活码必填"}
	}
	fp := DeviceFingerprint()
	body := RedeemRequest{
		Code:        code,
		Fingerprint: fp,
		AppVersion:  r.appVersion,
	}
	raw, err := json.Marshal(body)
	if err != nil {
		return nil, &RedeemError{Kind: ErrInvalidInput, Message: "无法序列化请求体", Cause: err}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, hubURL+redeemEndpoint, bytes.NewReader(raw))
	if err != nil {
		return nil, &RedeemError{Kind: ErrInvalidInput, Message: "构造请求失败", Cause: err}
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-Device-Fingerprint", fp)
	if r.appVersion != "" {
		req.Header.Set("User-Agent", "lurus-switch/"+r.appVersion)
	}

	resp, err := r.httpClient.Do(req)
	if err != nil {
		return nil, &RedeemError{Kind: ErrNetwork, Message: "无法连接到 Hub", Cause: err}
	}
	defer resp.Body.Close()
	rawBody, _ := io.ReadAll(resp.Body)

	// 404 from the Hub almost always means "this Hub build doesn't expose
	// /api/v2/switch/redeem yet" — distinct from a wrong code.
	if resp.StatusCode == http.StatusNotFound {
		return nil, &RedeemError{
			Kind:    ErrEndpointAbsent,
			Message: "该 Hub 暂未启用激活码兑换 endpoint，请联系经销商升级 newhub。",
		}
	}

	// Try to parse the standard Hub envelope first; fall back to raw text
	// when the body isn't JSON (HTML 502 from a reverse proxy, etc.).
	var env struct {
		Success bool            `json:"success"`
		Message string          `json:"message"`
		Data    json.RawMessage `json:"data"`
	}
	if jsonErr := json.Unmarshal(rawBody, &env); jsonErr != nil {
		return nil, &RedeemError{
			Kind:    ErrServer,
			Message: fmt.Sprintf("Hub 返回非 JSON 响应（HTTP %d）", resp.StatusCode),
		}
	}

	if !env.Success {
		return nil, classifyRedeemFailure(resp.StatusCode, env.Message)
	}

	var data RedeemResponse
	if len(env.Data) > 0 && string(env.Data) != "null" {
		if err := json.Unmarshal(env.Data, &data); err != nil {
			return nil, &RedeemError{Kind: ErrServer, Message: "Hub 响应数据解析失败", Cause: err}
		}
	}
	if data.UserToken == "" {
		return nil, &RedeemError{Kind: ErrServer, Message: "Hub 未返回用户 token"}
	}

	act := &Activation{
		HubURL:      hubURL,
		TenantSlug:  data.TenantSlug,
		UserToken:   data.UserToken,
		UserID:      data.UserID,
		Quota:       data.Quota,
		Fingerprint: fp,
		ActivatedAt: time.Now().UTC(),
	}
	if data.ExpiresAt > 0 {
		act.ExpiresAt = time.Unix(data.ExpiresAt, 0).UTC()
	}
	return act, nil
}

// classifyRedeemFailure maps the Hub's `success:false` message to a typed
// error kind. The Hub message text is the best signal we have — we match
// substrings rather than exact equality so minor copy edits on the Hub
// side don't break Switch.
func classifyRedeemFailure(httpStatus int, message string) error {
	low := strings.ToLower(message)
	switch {
	case strings.Contains(low, "已使用") || strings.Contains(low, "used") || strings.Contains(low, "redeemed"):
		return &RedeemError{Kind: ErrCodeUsed, Message: defaultMessage(message, "激活码已被使用")}
	case strings.Contains(low, "过期") || strings.Contains(low, "expire"):
		return &RedeemError{Kind: ErrCodeExpired, Message: defaultMessage(message, "激活码已过期")}
	case strings.Contains(low, "禁用") || strings.Contains(low, "disabled") || strings.Contains(low, "revoked"):
		return &RedeemError{Kind: ErrCodeDisabled, Message: defaultMessage(message, "激活码已被禁用")}
	case strings.Contains(low, "不存在") || strings.Contains(low, "not found") || strings.Contains(low, "invalid"):
		return &RedeemError{Kind: ErrCodeNotFound, Message: defaultMessage(message, "激活码不存在或无效")}
	case httpStatus >= 500:
		return &RedeemError{Kind: ErrServer, Message: defaultMessage(message, "Hub 内部错误，请稍后重试")}
	default:
		return &RedeemError{Kind: ErrCodeNotFound, Message: defaultMessage(message, "激活失败")}
	}
}

// defaultMessage returns msg if non-empty, fallback otherwise. Hub messages
// are usually Chinese already; this just guards against empty strings.
func defaultMessage(msg, fallback string) string {
	if strings.TrimSpace(msg) == "" {
		return fallback
	}
	return msg
}
