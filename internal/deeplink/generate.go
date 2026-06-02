package deeplink

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
)

// Generate encodes a Payload into a switch:// deep-link URL that Parse can
// decode back to an equivalent Payload.
//
// The returned URL always uses the query form:
//
//	switch://import?type=<type>&data=<base64url>
//
// Generate enforces the same invariants as Parse:
//   - Type must be one of the known values (provider, mcp, prompt, skill)
//   - Data must be a valid JSON object (not array, string, number, etc.)
//   - Encoded Data must not exceed 64 KiB
//
// Payload.Raw is ignored; the caller receives a freshly-generated URL.
func Generate(p *Payload) (string, error) {
	if p == nil {
		return "", fmt.Errorf("deeplink: nil payload")
	}

	// Validate type against the same allowlist used by Parse.
	if !validTypes[p.Type] {
		return "", fmt.Errorf("deeplink: unknown type %q (allowed: provider, mcp, prompt, skill)", p.Type)
	}

	// Validate that Data is a JSON object.
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(p.Data, &obj); err != nil {
		return "", fmt.Errorf("deeplink: data must be a JSON object: %w", err)
	}

	// Enforce size cap before encoding.
	if len(p.Data) > maxDataBytes {
		return "", fmt.Errorf("deeplink: data payload too large (%d bytes, max %d)", len(p.Data), maxDataBytes)
	}

	enc := base64.RawURLEncoding.EncodeToString(p.Data)

	q := url.Values{}
	q.Set("type", p.Type)
	q.Set("data", enc)

	return Scheme + "://import?" + q.Encode(), nil
}
