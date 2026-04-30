package deeplink

import (
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"
)

// encodeData is a helper that base64url-encodes a JSON value.
func encodeData(t *testing.T, v any) string {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
	return base64.RawURLEncoding.EncodeToString(b)
}

func TestParse_QueryForm_AllTypes(t *testing.T) {
	types := []string{"provider", "mcp", "prompt", "skill"}
	data := encodeData(t, map[string]string{"key": "value"})

	for _, typ := range types {
		t.Run(typ, func(t *testing.T) {
			raw := "switch://import?type=" + typ + "&data=" + data
			p, err := Parse(raw)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if p.Type != typ {
				t.Errorf("Type = %q, want %q", p.Type, typ)
			}
			if p.Raw != raw {
				t.Errorf("Raw mismatch")
			}
			var obj map[string]string
			if err := json.Unmarshal(p.Data, &obj); err != nil {
				t.Errorf("Data not valid JSON: %v", err)
			}
		})
	}
}

func TestParse_PathForm(t *testing.T) {
	types := []string{"provider", "mcp", "prompt", "skill"}
	data := encodeData(t, map[string]string{"model": "gpt-4"})

	for _, typ := range types {
		t.Run("path/"+typ, func(t *testing.T) {
			raw := "switch://import/" + typ + "?data=" + data
			p, err := Parse(raw)
			if err != nil {
				t.Fatalf("unexpected error for %s: %v", typ, err)
			}
			if p.Type != typ {
				t.Errorf("Type = %q, want %q", p.Type, typ)
			}
		})
	}
}

func TestParse_CaseInsensitiveType(t *testing.T) {
	data := encodeData(t, map[string]any{"x": 1})
	raw := "switch://import?type=PROVIDER&data=" + data
	p, err := Parse(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.Type != "provider" {
		t.Errorf("Type = %q, want %q", p.Type, "provider")
	}
}

func TestParse_ErrorUnknownScheme(t *testing.T) {
	data := encodeData(t, map[string]any{})
	raw := "https://import?type=provider&data=" + data
	_, err := Parse(raw)
	if err == nil {
		t.Fatal("expected error for wrong scheme")
	}
	if !strings.Contains(err.Error(), "unsupported scheme") {
		t.Errorf("error message mismatch: %v", err)
	}
}

func TestParse_ErrorUnknownType(t *testing.T) {
	data := encodeData(t, map[string]any{})
	raw := "switch://import?type=malicious&data=" + data
	_, err := Parse(raw)
	if err == nil {
		t.Fatal("expected error for unknown type")
	}
	if !strings.Contains(err.Error(), "unknown type") {
		t.Errorf("error message mismatch: %v", err)
	}
}

func TestParse_ErrorMissingType(t *testing.T) {
	data := encodeData(t, map[string]any{})
	raw := "switch://import?data=" + data
	_, err := Parse(raw)
	if err == nil {
		t.Fatal("expected error for missing type")
	}
	if !strings.Contains(err.Error(), "missing type") {
		t.Errorf("error message mismatch: %v", err)
	}
}

func TestParse_ErrorMissingData(t *testing.T) {
	raw := "switch://import?type=provider"
	_, err := Parse(raw)
	if err == nil {
		t.Fatal("expected error for missing data")
	}
	if !strings.Contains(err.Error(), "missing data") {
		t.Errorf("error message mismatch: %v", err)
	}
}

func TestParse_ErrorInvalidBase64(t *testing.T) {
	raw := "switch://import?type=provider&data=!!!notbase64!!!"
	_, err := Parse(raw)
	if err == nil {
		t.Fatal("expected error for invalid base64")
	}
	if !strings.Contains(err.Error(), "base64url") {
		t.Errorf("error message mismatch: %v", err)
	}
}

func TestParse_ErrorDataTooLarge(t *testing.T) {
	// Create a JSON object whose base64url encoding exceeds 64 KiB.
	big := map[string]string{"x": strings.Repeat("A", maxDataBytes)}
	bigJSON, _ := json.Marshal(big)
	enc := base64.RawURLEncoding.EncodeToString(bigJSON)
	raw := "switch://import?type=provider&data=" + enc
	_, err := Parse(raw)
	if err == nil {
		t.Fatal("expected error for oversized data")
	}
	if !strings.Contains(err.Error(), "too large") {
		t.Errorf("error message mismatch: %v", err)
	}
}

func TestParse_ErrorDataNotObject_Array(t *testing.T) {
	enc := base64.RawURLEncoding.EncodeToString([]byte(`["a","b"]`))
	raw := "switch://import?type=provider&data=" + enc
	_, err := Parse(raw)
	if err == nil {
		t.Fatal("expected error for JSON array")
	}
	if !strings.Contains(err.Error(), "JSON object") {
		t.Errorf("error message mismatch: %v", err)
	}
}

func TestParse_ErrorDataNotObject_String(t *testing.T) {
	enc := base64.RawURLEncoding.EncodeToString([]byte(`"hello"`))
	raw := "switch://import?type=provider&data=" + enc
	_, err := Parse(raw)
	if err == nil {
		t.Fatal("expected error for JSON string")
	}
}

func TestParse_ErrorEmptyURL(t *testing.T) {
	_, err := Parse("")
	if err == nil {
		t.Fatal("expected error for empty URL")
	}
}

func TestParse_ErrorMalformedURL(t *testing.T) {
	_, err := Parse("://broken url")
	if err == nil {
		t.Fatal("expected error for malformed URL")
	}
}

func TestParse_PaddedBase64URL(t *testing.T) {
	// Ensure padded base64url also works.
	obj := map[string]string{"a": "b"}
	b, _ := json.Marshal(obj)
	enc := base64.URLEncoding.EncodeToString(b) // with padding
	raw := "switch://import?type=mcp&data=" + enc
	p, err := Parse(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.Type != "mcp" {
		t.Errorf("Type = %q, want %q", p.Type, "mcp")
	}
}

func TestParse_RawURLPreserved(t *testing.T) {
	data := encodeData(t, map[string]any{"id": "test-123"})
	raw := "switch://import?type=skill&data=" + data
	p, err := Parse(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.Raw != raw {
		t.Errorf("Raw = %q, want %q", p.Raw, raw)
	}
}
