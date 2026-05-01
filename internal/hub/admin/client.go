// Package admin is a typed Go client for the Lurus Hub (lurus-newhub) admin
// API. It is consumed by Switch's Reseller-mode pages — channel/token/
// redemption/log/governance — and shields the frontend from the raw HTTP
// envelope (`{success, message, data}`) and Hub-specific status semantics.
//
// Endpoint coverage corresponds to roadmap S-Xa.5 and is intentionally
// minimal: the most-used CRUD plus pagination. Less-used endpoints (multi-
// key management, OpenRouter sync, audit query) get added as their owning
// pages reach the queue in Sprint 4c-4d (Phase B).
//
// Hub auth: Hub middleware reads the raw access token from the
// `Authorization` header (no `Bearer` prefix required for native tokens; it
// is accepted for Lurus Platform session tokens). The client passes whatever
// the caller configured.
package admin

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	defaultTimeout = 20 * time.Second
	envelopeData   = "data"
)

// Config configures the Hub admin client.
type Config struct {
	// BaseURL is the Hub root (no trailing slash), e.g. "https://hub.acme.example".
	BaseURL string
	// Token is the admin access token (Hub-issued or Identity session token).
	// Sent verbatim in the Authorization header.
	Token string
	// Timeout is the per-request deadline. Defaults to defaultTimeout when zero.
	Timeout time.Duration
	// HTTPClient overrides the underlying HTTP client. Optional — a sane
	// default is constructed when nil.
	HTTPClient *http.Client
}

// Client talks to a Hub admin API.
type Client struct {
	baseURL string
	token   string
	http    *http.Client
}

// New constructs a Client from cfg. Returns an error when BaseURL is missing
// or malformed; missing Token is permitted (callers can defer auth, e.g.
// the ResellerSetupWizard configures Token after Hub provisioning).
func New(cfg Config) (*Client, error) {
	base := strings.TrimRight(strings.TrimSpace(cfg.BaseURL), "/")
	if base == "" {
		return nil, errors.New("hub admin: BaseURL is required")
	}
	if _, err := url.Parse(base); err != nil {
		return nil, fmt.Errorf("hub admin: invalid BaseURL %q: %w", base, err)
	}
	httpClient := cfg.HTTPClient
	if httpClient == nil {
		timeout := cfg.Timeout
		if timeout == 0 {
			timeout = defaultTimeout
		}
		httpClient = &http.Client{Timeout: timeout}
	}
	return &Client{
		baseURL: base,
		token:   cfg.Token,
		http:    httpClient,
	}, nil
}

// HubError represents a non-2xx HTTP response or a `success:false` envelope.
type HubError struct {
	HTTPStatus int    // HTTP status code returned by Hub (0 for transport errors)
	Code       string // machine-readable code if Hub provided one (currently optional)
	Message    string // human-readable message ("无权进行此操作", etc.)
}

// Error implements the error interface.
func (e *HubError) Error() string {
	if e == nil {
		return ""
	}
	if e.Message == "" {
		return fmt.Sprintf("hub admin: HTTP %d", e.HTTPStatus)
	}
	return fmt.Sprintf("hub admin: %s (HTTP %d)", e.Message, e.HTTPStatus)
}

// IsUnauthorized reports whether err is a HubError with HTTP 401, signalling
// the admin token is missing or invalid. Callers (e.g. ResellerSetupWizard)
// can use this to bounce the user back to the token configuration step.
func IsUnauthorized(err error) bool {
	var hubErr *HubError
	if errors.As(err, &hubErr) && hubErr.HTTPStatus == http.StatusUnauthorized {
		return true
	}
	return false
}

// envelope is the standard Hub response shape produced by common.ApiSuccess
// / common.ApiError in newhub.
type envelope struct {
	Success bool            `json:"success"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

// do performs an HTTP request, unwraps the envelope, and decodes data into
// out (when out is non-nil). Pass body=nil for GET/DELETE; pass query=nil
// when no query parameters are needed.
func (c *Client) do(ctx context.Context, method, path string, query url.Values, body any, out any) error {
	if c == nil {
		return errors.New("hub admin: nil client")
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	endpoint := c.baseURL + path
	if len(query) > 0 {
		endpoint += "?" + query.Encode()
	}

	var reqBody io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("hub admin: marshal request body: %w", err)
		}
		reqBody = bytes.NewReader(raw)
	}

	req, err := http.NewRequestWithContext(ctx, method, endpoint, reqBody)
	if err != nil {
		return fmt.Errorf("hub admin: build request: %w", err)
	}
	if reqBody != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/json")
	if c.token != "" {
		req.Header.Set("Authorization", c.token)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return &HubError{Message: err.Error()}
	}
	defer resp.Body.Close()

	rawBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return &HubError{HTTPStatus: resp.StatusCode, Message: "read response body: " + err.Error()}
	}

	// 401 short-circuits before envelope parsing (Hub may emit non-envelope
	// payload from middleware before reaching the success/message wrapper).
	if resp.StatusCode == http.StatusUnauthorized {
		msg := tryExtractMessage(rawBody)
		if msg == "" {
			msg = "unauthorized"
		}
		return &HubError{HTTPStatus: resp.StatusCode, Message: msg}
	}

	// Other non-2xx with a JSON envelope still gets unwrapped below, but if
	// it's pure HTML / plain text we surface a generic error.
	var env envelope
	if err := json.Unmarshal(rawBody, &env); err != nil {
		return &HubError{
			HTTPStatus: resp.StatusCode,
			Message:    fmt.Sprintf("non-JSON response (%d bytes)", len(rawBody)),
		}
	}
	if !env.Success {
		msg := env.Message
		if msg == "" {
			msg = fmt.Sprintf("HTTP %d", resp.StatusCode)
		}
		return &HubError{HTTPStatus: resp.StatusCode, Message: msg}
	}

	if out != nil && len(env.Data) > 0 && string(env.Data) != "null" {
		if err := json.Unmarshal(env.Data, out); err != nil {
			return fmt.Errorf("hub admin: decode data: %w", err)
		}
	}
	return nil
}

// tryExtractMessage scans an arbitrary 401 body for a `message` field — best-
// effort only, not all 401 bodies are envelopes.
func tryExtractMessage(raw []byte) string {
	var probe struct {
		Message string `json:"message"`
	}
	if err := json.Unmarshal(raw, &probe); err != nil {
		return ""
	}
	return probe.Message
}

// pageQuery converts a ListOpts into URL query params understood by Hub's
// common.GetPageQuery (`p`, `page_size`).
func (o *ListOpts) pageQuery() url.Values {
	v := url.Values{}
	if o == nil {
		return v
	}
	if o.Page > 0 {
		v.Set("p", strconv.Itoa(o.Page))
	}
	if o.PageSize > 0 {
		v.Set("page_size", strconv.Itoa(o.PageSize))
	}
	if o.Keyword != "" {
		v.Set("keyword", o.Keyword)
	}
	for k, raw := range o.Extra {
		v.Set(k, raw)
	}
	return v
}
