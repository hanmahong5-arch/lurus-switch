package metering

import (
	"encoding/csv"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestMarshalRecordsCSV_RoundTripWithID(t *testing.T) {
	ts := time.Date(2026, 5, 31, 8, 30, 0, 0, time.UTC)
	records := []Record{
		{ID: "abc123", AppID: "claude-code", Model: "claude-sonnet-4-6", TokensIn: 1000, TokensOut: 500, CacheReadTokens: 200, StatusCode: 200, CachedHit: true, Timestamp: ts},
		{ID: "def456", AppID: "codex", Model: "gpt-4o", TokensIn: 800, TokensOut: 300, StatusCode: 200, Timestamp: ts.Add(time.Minute)},
	}

	out, err := MarshalRecordsCSV(records)
	if err != nil {
		t.Fatalf("MarshalRecordsCSV: %v", err)
	}

	r := csv.NewReader(strings.NewReader(out))
	rows, err := r.ReadAll()
	if err != nil {
		t.Fatalf("parse csv: %v", err)
	}
	if len(rows) != 3 { // header + 2 records
		t.Fatalf("row count = %d, want 3", len(rows))
	}
	if rows[0][0] != "id" {
		t.Errorf("first column header = %q, want id", rows[0][0])
	}
	// Record.ID is the join key — it must survive the round-trip in column 0.
	if rows[1][0] != "abc123" || rows[2][0] != "def456" {
		t.Errorf("record IDs not preserved: %q, %q", rows[1][0], rows[2][0])
	}
	// Spot-check a couple of value columns (tokensIn is column index 4).
	if rows[1][4] != "1000" {
		t.Errorf("row1 tokensIn = %q, want 1000", rows[1][4])
	}
}

func TestMarshalRecordsCSV_EmptyIsHeaderOnly(t *testing.T) {
	out, err := MarshalRecordsCSV(nil)
	if err != nil {
		t.Fatalf("MarshalRecordsCSV(nil): %v", err)
	}
	r := csv.NewReader(strings.NewReader(out))
	rows, err := r.ReadAll()
	if err != nil {
		t.Fatalf("parse csv: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("empty export should be header-only, got %d rows", len(rows))
	}
}

func TestMarshalRecordsJSON_RoundTrip(t *testing.T) {
	records := []Record{
		{ID: "j1", Model: "claude-sonnet-4-6", TokensIn: 10, TokensOut: 5, Timestamp: time.Unix(1_700_000_000, 0).UTC()},
	}
	data, err := MarshalRecordsJSON(records)
	if err != nil {
		t.Fatalf("MarshalRecordsJSON: %v", err)
	}
	var back []Record
	if err := json.Unmarshal(data, &back); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(back) != 1 || back[0].ID != "j1" || back[0].TokensIn != 10 {
		t.Errorf("JSON round-trip lost data: %+v", back)
	}
}

func TestMarshalRecordsJSON_NilSerializesToEmptyArray(t *testing.T) {
	data, err := MarshalRecordsJSON(nil)
	if err != nil {
		t.Fatalf("MarshalRecordsJSON(nil): %v", err)
	}
	if strings.TrimSpace(string(data)) != "[]" {
		t.Errorf("nil should serialize to [], got %q", string(data))
	}
}

func TestExportRange_FiltersByWindow(t *testing.T) {
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	base := time.Date(2026, 5, 31, 12, 0, 0, 0, time.UTC)
	store.Record(Record{ID: "in1", Model: "x", Timestamp: base})
	store.Record(Record{ID: "out1", Model: "x", Timestamp: base.AddDate(0, 0, 3)})

	got := store.ExportRange(base.Add(-time.Hour), base.Add(time.Hour))
	if len(got) != 1 || got[0].ID != "in1" {
		t.Errorf("ExportRange window filter wrong: %+v", got)
	}
}
