package analytics

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func setupTracker(t *testing.T) *Tracker {
	t.Helper()
	tmp := t.TempDir()
	t.Setenv("APPDATA", tmp)
	t.Setenv("HOME", tmp)
	t.Setenv("XDG_CONFIG_HOME", tmp)

	tracker, err := NewTracker()
	if err != nil {
		t.Fatalf("NewTracker error: %v", err)
	}
	return tracker
}

func TestNewTracker(t *testing.T) {
	tracker := setupTracker(t)
	if tracker == nil {
		t.Fatal("expected non-nil tracker")
	}
	if tracker.path == "" {
		t.Error("tracker path should not be empty")
	}
}

func TestTracker_Record_Basic(t *testing.T) {
	tracker := setupTracker(t)

	e := Event{
		Tool:    "claude",
		Action:  "install",
		Success: true,
	}
	if err := tracker.Record(e); err != nil {
		t.Fatalf("Record error: %v", err)
	}

	// Verify file was created
	if _, err := os.Stat(tracker.path); os.IsNotExist(err) {
		t.Error("analytics file should have been created after Record")
	}
}

func TestTracker_Record_AutoTimestamp(t *testing.T) {
	tracker := setupTracker(t)

	e := Event{
		Tool:   "codex",
		Action: "update",
	}
	before := time.Now().Add(-time.Second)
	if err := tracker.Record(e); err != nil {
		t.Fatalf("Record error: %v", err)
	}
	after := time.Now().Add(time.Second)

	// Read the report to verify a valid timestamp was written
	report, err := tracker.GetReport(1)
	if err != nil {
		t.Fatalf("GetReport error: %v", err)
	}

	// Should have recorded today's activity
	today := before.Format("2006-01-02")
	_ = after
	count := report.DailyActive[today]
	if count != 1 {
		t.Errorf("DailyActive[%s] = %d, want 1", today, count)
	}
}

func TestTracker_Record_WithTimestamp(t *testing.T) {
	tracker := setupTracker(t)

	ts := time.Now().Format(time.RFC3339)
	e := Event{
		Timestamp: ts,
		Tool:      "gemini",
		Action:    "config",
		Success:   true,
	}
	if err := tracker.Record(e); err != nil {
		t.Fatalf("Record error: %v", err)
	}

	report, err := tracker.GetReport(7)
	if err != nil {
		t.Fatalf("GetReport error: %v", err)
	}
	if report.ToolActions["gemini"]["config"] != 1 {
		t.Errorf("expected 1 config action for gemini, got %d", report.ToolActions["gemini"]["config"])
	}
}

func TestTracker_GetReport_EmptyFile(t *testing.T) {
	tracker := setupTracker(t)

	report, err := tracker.GetReport(7)
	if err != nil {
		t.Fatalf("GetReport on missing file error: %v", err)
	}
	if len(report.ToolActions) != 0 {
		t.Errorf("expected empty ToolActions, got %v", report.ToolActions)
	}
	if len(report.DailyActive) != 0 {
		t.Errorf("expected empty DailyActive, got %v", report.DailyActive)
	}
}

func TestTracker_GetReport_MultiplEvents(t *testing.T) {
	tracker := setupTracker(t)

	events := []Event{
		{Tool: "claude", Action: "install", Success: true},
		{Tool: "claude", Action: "config", Success: true},
		{Tool: "codex", Action: "install", Success: false},
		{Tool: "claude", Action: "install", Success: true},
	}
	for _, e := range events {
		if err := tracker.Record(e); err != nil {
			t.Fatalf("Record error: %v", err)
		}
	}

	report, err := tracker.GetReport(7)
	if err != nil {
		t.Fatalf("GetReport error: %v", err)
	}

	if report.ToolActions["claude"]["install"] != 2 {
		t.Errorf("claude install count = %d, want 2", report.ToolActions["claude"]["install"])
	}
	if report.ToolActions["claude"]["config"] != 1 {
		t.Errorf("claude config count = %d, want 1", report.ToolActions["claude"]["config"])
	}
	if report.ToolActions["codex"]["install"] != 1 {
		t.Errorf("codex install count = %d, want 1", report.ToolActions["codex"]["install"])
	}
}

func TestTracker_GetReport_FilterOldEvents(t *testing.T) {
	tracker := setupTracker(t)

	// Write an old event manually (10 days ago)
	oldTS := time.Now().AddDate(0, 0, -10).Format(time.RFC3339)
	oldLine := `{"ts":"` + oldTS + `","tool":"claude","action":"install","success":true}` + "\n"
	if err := os.WriteFile(tracker.path, []byte(oldLine), 0644); err != nil {
		t.Fatalf("WriteFile error: %v", err)
	}

	// GetReport for only 7 days — should exclude the old event
	report, err := tracker.GetReport(7)
	if err != nil {
		t.Fatalf("GetReport error: %v", err)
	}
	if len(report.ToolActions) != 0 {
		t.Errorf("old events should be filtered out, got %v", report.ToolActions)
	}
}

func TestTracker_GetReport_DefaultDays(t *testing.T) {
	tracker := setupTracker(t)

	// days=0 should default to 7
	report, err := tracker.GetReport(0)
	if err != nil {
		t.Fatalf("GetReport error: %v", err)
	}
	if report == nil {
		t.Error("expected non-nil report")
	}
}

func TestTracker_GetReport_SkipsInvalidLines(t *testing.T) {
	tracker := setupTracker(t)

	// Write mixed valid/invalid lines
	validTS := time.Now().Format(time.RFC3339)
	content := "not valid json\n" +
		`{"ts":"` + validTS + `","tool":"gemini","action":"update","success":true}` + "\n" +
		"\n" +
		`{"ts":"badinvalidts","tool":"codex","action":"install","success":true}` + "\n"

	if err := os.WriteFile(tracker.path, []byte(content), 0644); err != nil {
		t.Fatalf("WriteFile error: %v", err)
	}

	report, err := tracker.GetReport(7)
	if err != nil {
		t.Fatalf("GetReport error: %v", err)
	}
	// Only the valid event with parseable timestamp should be counted
	if report.ToolActions["gemini"]["update"] != 1 {
		t.Errorf("gemini update count = %d, want 1", report.ToolActions["gemini"]["update"])
	}
}

func TestTracker_Record_Detail(t *testing.T) {
	tracker := setupTracker(t)

	e := Event{
		Tool:    "picoclaw",
		Action:  "config",
		Success: true,
		Detail:  "updated api_base",
	}
	if err := tracker.Record(e); err != nil {
		t.Fatalf("Record error: %v", err)
	}

	// Verify file is non-empty
	data, err := os.ReadFile(tracker.path)
	if err != nil {
		t.Fatalf("ReadFile error: %v", err)
	}
	if len(data) == 0 {
		t.Error("analytics file should not be empty after recording")
	}
}

func TestTrackerPath_APPDATA(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("APPDATA", tmp)

	p, err := analyticsPath()
	if err != nil {
		t.Fatalf("analyticsPath error: %v", err)
	}
	expected := filepath.Join(tmp, "lurus-switch", "analytics.jsonl")
	if p != expected {
		t.Errorf("path = %q, want %q", p, expected)
	}
}
