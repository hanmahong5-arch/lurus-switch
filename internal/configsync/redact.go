package configsync

import (
	"bytes"
	"encoding/json"
	"strings"
)

// redactedKeyNames is the case-insensitive denylist of JSON key names whose
// values are blanked when includeKeys is false. This is a BLACKLIST — adding
// a component that stores a secret under a new key name requires extending
// this list, which is why the PR checklist forces a secrets review.
var redactedKeyNames = map[string]struct{}{
	"apikey":          {},
	"api_key":         {},
	"anthropicapikey": {},
	"openaikey":       {},
	"openaiapikey":    {},
	"authclientid":    {},
	"admintoken":      {},
	"usertoken":       {},
	"secret":          {},
	"secretkey":       {},
	"password":        {},
	"apikeyb64":       {}, // custom-providers on-disk obfuscated key
}

// redactedKeySuffixes catches per-provider variants like "deepseekApiKey".
var redactedKeySuffixes = []string{"apikey", "api_key", "_token", "secret"}

const redactedPlaceholder = "__REDACTED__"

// shouldRedactKey reports whether a JSON key name matches the denylist.
func shouldRedactKey(key string) bool {
	k := strings.ToLower(key)
	if _, ok := redactedKeyNames[k]; ok {
		return true
	}
	for _, suf := range redactedKeySuffixes {
		if strings.HasSuffix(k, suf) && len(k) > len(suf) {
			return true
		}
	}
	return false
}

// redactJSON walks an arbitrary JSON document and blanks the value of any key
// matching the denylist (recursively, including inside arrays). Non-JSON or
// empty input is returned unchanged so the caller can pass any file blindly.
func redactJSON(raw []byte) []byte {
	var v any
	if err := json.Unmarshal(raw, &v); err != nil {
		return raw // not JSON — leave to TOML/line redactor or pass through
	}
	redactValue(v)
	out, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return raw
	}
	return out
}

func redactValue(v any) {
	switch t := v.(type) {
	case map[string]any:
		for k, child := range t {
			if shouldRedactKey(k) {
				if s, ok := child.(string); ok && s != "" {
					t[k] = redactedPlaceholder
					continue
				}
				if child == nil {
					continue
				}
			}
			redactValue(child)
		}
	case []any:
		for _, child := range t {
			redactValue(child)
		}
	}
}

// redactTOMLLines blanks the right-hand side of any TOML assignment whose key
// matches the denylist. Comments and structure are preserved. This is a
// line-based pass — sufficient for the flat key=value shape of codex's
// config.toml; it does not attempt to parse nested tables semantically.
func redactTOMLLines(raw []byte) []byte {
	lines := bytes.Split(raw, []byte("\n"))
	for i, line := range lines {
		trimmed := strings.TrimSpace(string(line))
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		eq := strings.Index(trimmed, "=")
		if eq < 0 {
			continue
		}
		key := strings.TrimSpace(trimmed[:eq])
		if shouldRedactKey(key) {
			indent := line[:len(line)-len(bytes.TrimLeft(line, " \t"))]
			lines[i] = append(append([]byte{}, indent...), []byte(key+" = \""+redactedPlaceholder+"\"")...)
		}
	}
	return bytes.Join(lines, []byte("\n"))
}
