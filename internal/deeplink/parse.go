package deeplink

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"
)

// Scheme is the URL scheme owned by this application.
const Scheme = "switch"

// maxDataBytes is the maximum allowed size for the decoded data payload (64 KiB).
const maxDataBytes = 64 * 1024

// validTypes is the allowlist of accepted import type values.
var validTypes = map[string]bool{
	"provider": true,
	"mcp":      true,
	"prompt":   true,
	"skill":    true,
}

// Payload is the parsed import request decoded from a switch:// URL.
type Payload struct {
	// Type is the import category: "provider" | "mcp" | "prompt" | "skill".
	Type string `json:"type"`
	// Data is the opaque decoded JSON object; the frontend validates its shape.
	Data json.RawMessage `json:"data"`
	// Raw is the original URL preserved for audit logging.
	Raw string `json:"raw"`
}

// Parse decodes and validates a "switch://..." deep-link URL into a Payload.
//
// Accepted forms:
//
//	switch://import?type=<type>&data=<base64url>
//	switch://import/<type>?data=<base64url>
//
// Parse rejects:
//   - wrong URL scheme (not "switch")
//   - unknown type values
//   - non-base64url data
//   - decoded data larger than 64 KiB
//   - data that does not decode to a JSON object
func Parse(rawURL string) (*Payload, error) {
	if rawURL == "" {
		return nil, errors.New("deeplink: empty URL")
	}

	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("deeplink: malformed URL: %w", err)
	}

	// Validate scheme.
	if !strings.EqualFold(u.Scheme, Scheme) {
		return nil, fmt.Errorf("deeplink: unsupported scheme %q (want %q)", u.Scheme, Scheme)
	}

	q := u.Query()

	// Resolve type from query param or URL path segment.
	// Path form: switch://import/<type>?data=...
	// Query form: switch://import?type=<type>&data=...
	typVal := q.Get("type")
	if typVal == "" {
		// Try path: u.Host == "import", u.Path might be "/<type>" or empty.
		// url.Parse with opaque scheme: for "switch://import/provider?data=..."
		// u.Host = "import", u.Path = "/provider"
		path := strings.TrimPrefix(u.Path, "/")
		path = strings.TrimSuffix(path, "/")
		if path != "" {
			typVal = path
		}
	}

	if typVal == "" {
		return nil, errors.New("deeplink: missing type parameter")
	}

	typVal = strings.ToLower(typVal)
	if !validTypes[typVal] {
		return nil, fmt.Errorf("deeplink: unknown type %q (allowed: provider, mcp, prompt, skill)", typVal)
	}

	// Decode base64url data.
	enc := q.Get("data")
	if enc == "" {
		return nil, errors.New("deeplink: missing data parameter")
	}

	// Accept both standard and URL-safe base64 (with or without padding).
	decoded, err := base64.RawURLEncoding.DecodeString(enc)
	if err != nil {
		// Fallback: try padded base64url.
		decoded, err = base64.URLEncoding.DecodeString(enc)
		if err != nil {
			return nil, fmt.Errorf("deeplink: data is not valid base64url: %w", err)
		}
	}

	// Enforce size cap before JSON parsing.
	if len(decoded) > maxDataBytes {
		return nil, fmt.Errorf("deeplink: data payload too large (%d bytes, max %d)", len(decoded), maxDataBytes)
	}

	// Validate that data is a JSON object (not array, string, number, etc.).
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(decoded, &obj); err != nil {
		return nil, fmt.Errorf("deeplink: data must be a JSON object: %w", err)
	}

	return &Payload{
		Type: typVal,
		Data: json.RawMessage(decoded),
		Raw:  rawURL,
	}, nil
}
