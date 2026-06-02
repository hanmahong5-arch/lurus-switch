package deeplink

import (
	"encoding/json"
	"strings"
	"testing"
)

// TestGenerate_RoundTrip verifies that Parse(Generate(p)) produces an
// equivalent Payload (matching Type and Data) for every valid type.
func TestGenerate_RoundTrip(t *testing.T) {
	types := []string{"provider", "mcp", "prompt", "skill"}

	for _, typ := range types {
		t.Run(typ, func(t *testing.T) {
			data, err := json.Marshal(map[string]string{"key": "value", "type": typ})
			if err != nil {
				t.Fatalf("json.Marshal: %v", err)
			}

			original := &Payload{
				Type: typ,
				Data: json.RawMessage(data),
			}

			rawURL, err := Generate(original)
			if err != nil {
				t.Fatalf("Generate: %v", err)
			}

			parsed, err := Parse(rawURL)
			if err != nil {
				t.Fatalf("Parse: %v", err)
			}

			if parsed.Type != original.Type {
				t.Errorf("Type = %q, want %q", parsed.Type, original.Type)
			}

			// Compare Data as unmarshalled maps so key order does not matter.
			var gotObj, wantObj map[string]string
			if err := json.Unmarshal(parsed.Data, &gotObj); err != nil {
				t.Fatalf("unmarshal parsed.Data: %v", err)
			}
			if err := json.Unmarshal(original.Data, &wantObj); err != nil {
				t.Fatalf("unmarshal original.Data: %v", err)
			}
			for k, v := range wantObj {
				if gotObj[k] != v {
					t.Errorf("Data[%q] = %q, want %q", k, gotObj[k], v)
				}
			}
			if len(gotObj) != len(wantObj) {
				t.Errorf("Data length mismatch: got %d keys, want %d", len(gotObj), len(wantObj))
			}
		})
	}
}

// TestGenerate_URLQueryForm verifies the generated URL uses the query form.
func TestGenerate_URLQueryForm(t *testing.T) {
	p := &Payload{
		Type: "provider",
		Data: json.RawMessage(`{"model":"gpt-4"}`),
	}
	rawURL, err := Generate(p)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if !strings.HasPrefix(rawURL, "switch://import?") {
		t.Errorf("expected query form URL, got %q", rawURL)
	}
	if !strings.Contains(rawURL, "type=provider") {
		t.Errorf("URL missing type param: %q", rawURL)
	}
	if !strings.Contains(rawURL, "data=") {
		t.Errorf("URL missing data param: %q", rawURL)
	}
}

// TestGenerate_ErrorUnknownType verifies that an unknown type is rejected.
func TestGenerate_ErrorUnknownType(t *testing.T) {
	p := &Payload{
		Type: "malicious",
		Data: json.RawMessage(`{"x":1}`),
	}
	_, err := Generate(p)
	if err == nil {
		t.Fatal("expected error for unknown type")
	}
	if !strings.Contains(err.Error(), "unknown type") {
		t.Errorf("error message mismatch: %v", err)
	}
}

// TestGenerate_ErrorNonObjectData_Array verifies that a JSON array is rejected.
func TestGenerate_ErrorNonObjectData_Array(t *testing.T) {
	p := &Payload{
		Type: "provider",
		Data: json.RawMessage(`["a","b"]`),
	}
	_, err := Generate(p)
	if err == nil {
		t.Fatal("expected error for JSON array data")
	}
	if !strings.Contains(err.Error(), "JSON object") {
		t.Errorf("error message mismatch: %v", err)
	}
}

// TestGenerate_ErrorNonObjectData_String verifies that a JSON string is rejected.
func TestGenerate_ErrorNonObjectData_String(t *testing.T) {
	p := &Payload{
		Type: "mcp",
		Data: json.RawMessage(`"hello"`),
	}
	_, err := Generate(p)
	if err == nil {
		t.Fatal("expected error for JSON string data")
	}
}

// TestGenerate_ErrorOversizeData verifies that data exceeding 64 KiB is rejected.
func TestGenerate_ErrorOversizeData(t *testing.T) {
	// Build a JSON object whose raw JSON exceeds maxDataBytes.
	big := map[string]string{"x": strings.Repeat("A", maxDataBytes)}
	bigJSON, _ := json.Marshal(big)
	p := &Payload{
		Type: "skill",
		Data: json.RawMessage(bigJSON),
	}
	_, err := Generate(p)
	if err == nil {
		t.Fatal("expected error for oversized data")
	}
	if !strings.Contains(err.Error(), "too large") {
		t.Errorf("error message mismatch: %v", err)
	}
}

// TestGenerate_ErrorNilPayload verifies that a nil Payload is rejected.
func TestGenerate_ErrorNilPayload(t *testing.T) {
	_, err := Generate(nil)
	if err == nil {
		t.Fatal("expected error for nil payload")
	}
}

// TestGenerate_ParseRetainsRaw verifies that the round-tripped Payload has Raw
// set to the generated URL (not the original empty Raw field).
func TestGenerate_ParseRetainsRaw(t *testing.T) {
	p := &Payload{
		Type: "prompt",
		Data: json.RawMessage(`{"title":"hello"}`),
	}
	rawURL, err := Generate(p)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	parsed, err := Parse(rawURL)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if parsed.Raw != rawURL {
		t.Errorf("parsed.Raw = %q, want %q", parsed.Raw, rawURL)
	}
}
