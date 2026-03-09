package analytics

import (
	"fmt"
	"os"
	"sync"
	"testing"
	"time"
)

// setupTrackerScenario redirects analytics path to a temp dir and returns a
// fresh Tracker ready for scenario testing.
func setupTrackerScenario(t *testing.T) *Tracker {
	t.Helper()
	tmp := t.TempDir()
	t.Setenv("APPDATA", tmp)
	t.Setenv("HOME", tmp)
	t.Setenv("XDG_CONFIG_HOME", tmp)

	tr, err := NewTracker()
	if err != nil {
		t.Fatalf("NewTracker: %v", err)
	}
	return tr
}

// recordMust records an event and fails the test on error.
func recordMust(t *testing.T, tr *Tracker, e Event) {
	t.Helper()
	if err := tr.Record(e); err != nil {
		t.Fatalf("Record(%+v): %v", e, err)
	}
}

// ============================================================
// Scenario: User installs, uninstalls, reinstalls
// ============================================================

// TestScenario_UserInstallsUninstallsReinstalls_AllEventsRecorded simulates
// a user who: installs a tool → decides to remove it → installs again.
// All three events must appear in the report with correct counts.
func TestScenario_UserInstallsUninstallsReinstalls_AllEventsRecorded(t *testing.T) {
	tr := setupTrackerScenario(t)

	recordMust(t, tr, Event{Tool: "claude", Action: "install", Success: true})
	recordMust(t, tr, Event{Tool: "claude", Action: "uninstall", Success: true})
	recordMust(t, tr, Event{Tool: "claude", Action: "install", Success: true})

	report, err := tr.GetReport(7)
	if err != nil {
		t.Fatalf("GetReport: %v", err)
	}

	if report.ToolActions["claude"]["install"] != 2 {
		t.Errorf("install count = %d, want 2", report.ToolActions["claude"]["install"])
	}
	if report.ToolActions["claude"]["uninstall"] != 1 {
		t.Errorf("uninstall count = %d, want 1", report.ToolActions["claude"]["uninstall"])
	}
}

// ============================================================
// Scenario: User installs multiple tools in one session
// ============================================================

// TestScenario_UserInstallsAllTools_EachToolCounted simulates a user installing
// all five tools one after another in a single setup session.
func TestScenario_UserInstallsAllTools_EachToolCounted(t *testing.T) {
	tr := setupTrackerScenario(t)

	tools := []string{"claude", "codex", "gemini", "picoclaw", "nullclaw"}
	for _, tool := range tools {
		recordMust(t, tr, Event{Tool: tool, Action: "install", Success: true})
	}

	report, err := tr.GetReport(7)
	if err != nil {
		t.Fatalf("GetReport: %v", err)
	}

	for _, tool := range tools {
		if report.ToolActions[tool]["install"] != 1 {
			t.Errorf("tool %q install count = %d, want 1", tool, report.ToolActions[tool]["install"])
		}
	}
}

// ============================================================
// Scenario: User uses same tool for many actions
// ============================================================

// TestScenario_UserAppliesPromptAndUpdatesConfig_BothCounted verifies that
// different action types for the same tool are counted independently.
func TestScenario_UserAppliesPromptAndUpdatesConfig_BothCounted(t *testing.T) {
	tr := setupTrackerScenario(t)

	recordMust(t, tr, Event{Tool: "claude", Action: "config", Success: true, Detail: "added MCP server"})
	recordMust(t, tr, Event{Tool: "claude", Action: "prompt_apply", Success: true, Detail: "BMAD template"})
	recordMust(t, tr, Event{Tool: "claude", Action: "config", Success: true, Detail: "changed theme"})
	recordMust(t, tr, Event{Tool: "claude", Action: "update", Success: false, Detail: "network timeout"})

	report, err := tr.GetReport(7)
	if err != nil {
		t.Fatalf("GetReport: %v", err)
	}

	if report.ToolActions["claude"]["config"] != 2 {
		t.Errorf("config count = %d, want 2", report.ToolActions["claude"]["config"])
	}
	if report.ToolActions["claude"]["prompt_apply"] != 1 {
		t.Errorf("prompt_apply count = %d, want 1", report.ToolActions["claude"]["prompt_apply"])
	}
	if report.ToolActions["claude"]["update"] != 1 {
		t.Errorf("update count = %d, want 1", report.ToolActions["claude"]["update"])
	}
}

// ============================================================
// Scenario: Old events are excluded from report window
// ============================================================

// TestScenario_UserHasOldInstalls_OnlyRecentEventsCounted verifies that events
// older than the requested window are filtered out. This matters when a user
// asks "what did I do this week?" — last month's data should not appear.
func TestScenario_UserHasOldInstalls_OnlyRecentEventsCounted(t *testing.T) {
	tr := setupTrackerScenario(t)

	// Old event: 10 days ago (outside a 7-day window)
	oldTS := time.Now().AddDate(0, 0, -10).Format(time.RFC3339)
	oldLine := fmt.Sprintf(`{"ts":%q,"tool":"claude","action":"install","success":true}`, oldTS) + "\n"

	// Recent event: today
	recentTS := time.Now().Format(time.RFC3339)
	recentLine := fmt.Sprintf(`{"ts":%q,"tool":"codex","action":"install","success":true}`, recentTS) + "\n"

	os.WriteFile(tr.path, []byte(oldLine+recentLine), 0644)

	report, err := tr.GetReport(7)
	if err != nil {
		t.Fatalf("GetReport: %v", err)
	}

	if report.ToolActions["claude"]["install"] != 0 {
		t.Errorf("old claude event should be excluded, got count %d", report.ToolActions["claude"]["install"])
	}
	if report.ToolActions["codex"]["install"] != 1 {
		t.Errorf("recent codex event should be included, got count %d", report.ToolActions["codex"]["install"])
	}
}

// TestScenario_UserAsksFor1DayReport_YesterdayExcluded verifies the edge case
// where the window is exactly 1 day. An event from yesterday should be out.
func TestScenario_UserAsksFor1DayReport_YesterdayExcluded(t *testing.T) {
	tr := setupTrackerScenario(t)

	// Yesterday: exactly 25 hours ago (safely outside 1-day window)
	yesterdayTS := time.Now().Add(-25 * time.Hour).Format(time.RFC3339)
	line := fmt.Sprintf(`{"ts":%q,"tool":"gemini","action":"config","success":true}`, yesterdayTS) + "\n"
	os.WriteFile(tr.path, []byte(line), 0644)

	report, err := tr.GetReport(1)
	if err != nil {
		t.Fatalf("GetReport(1): %v", err)
	}
	if report.ToolActions["gemini"]["config"] != 0 {
		t.Errorf("event from 25h ago should be excluded from 1-day report, got %d", report.ToolActions["gemini"]["config"])
	}
}

// ============================================================
// Scenario: Corrupt log file — app must not crash
// ============================================================

// TestScenario_CorruptedLogFile_ValidEventsStillCounted simulates a log file
// that has been partially corrupted (e.g., disk I/O error mid-write). The
// GetReport call must: not crash, and count all lines that are still valid.
func TestScenario_CorruptedLogFile_ValidEventsStillCounted(t *testing.T) {
	tr := setupTrackerScenario(t)

	goodTS := time.Now().Format(time.RFC3339)
	content := "this is garbage\n" +
		fmt.Sprintf(`{"ts":%q,"tool":"claude","action":"install","success":true}`, goodTS) + "\n" +
		"{truncated json....\n" +
		fmt.Sprintf(`{"ts":%q,"tool":"codex","action":"update","success":true}`, goodTS) + "\n" +
		"\n" +
		`{"ts":"not-a-date","tool":"gemini","action":"config","success":true}` + "\n"

	os.WriteFile(tr.path, []byte(content), 0644)

	report, err := tr.GetReport(7)
	if err != nil {
		t.Fatalf("GetReport on corrupt file: %v", err)
	}

	// Only the two valid-timestamp lines should be counted
	if report.ToolActions["claude"]["install"] != 1 {
		t.Errorf("claude install count = %d, want 1", report.ToolActions["claude"]["install"])
	}
	if report.ToolActions["codex"]["update"] != 1 {
		t.Errorf("codex update count = %d, want 1", report.ToolActions["codex"]["update"])
	}
	// Line with invalid timestamp must be excluded
	if report.ToolActions["gemini"]["config"] != 0 {
		t.Errorf("invalid-timestamp event should be excluded, got %d", report.ToolActions["gemini"]["config"])
	}
}

// ============================================================
// Scenario: Concurrent installs (user rapidly spawns installs)
// ============================================================

// TestScenario_ConcurrentInstalls_NoEventDropped simulates the user rapidly
// triggering installs for multiple tools simultaneously. Every event must be
// recorded; no events may be silently dropped or cause a data race.
func TestScenario_ConcurrentInstalls_NoEventDropped(t *testing.T) {
	tr := setupTrackerScenario(t)

	const workers = 5
	const eventsPerWorker = 10
	tools := []string{"claude", "codex", "gemini", "picoclaw", "nullclaw"}

	var wg sync.WaitGroup
	for i, tool := range tools {
		wg.Add(1)
		go func(idx int, toolName string) {
			defer wg.Done()
			for j := 0; j < eventsPerWorker; j++ {
				recordMust(t, tr, Event{Tool: toolName, Action: "install", Success: true})
			}
		}(i, tool)
	}
	wg.Wait()

	report, err := tr.GetReport(7)
	if err != nil {
		t.Fatalf("GetReport after concurrent writes: %v", err)
	}

	total := 0
	for _, actions := range report.ToolActions {
		total += actions["install"]
	}

	if total != workers*eventsPerWorker {
		t.Errorf("total install events = %d, want %d (concurrent writes lost events)", total, workers*eventsPerWorker)
	}
}

// ============================================================
// Scenario: App update — failed events don't pollute counts
// ============================================================

// TestScenario_FailedActions_ReportedButNotHidden verifies that failed events
// (Success=false) are counted in ToolActions just like successful ones. The
// dashboard may want to show failure statistics.
func TestScenario_FailedActions_ReportedAndCounted(t *testing.T) {
	tr := setupTrackerScenario(t)

	recordMust(t, tr, Event{Tool: "codex", Action: "install", Success: true})
	recordMust(t, tr, Event{Tool: "codex", Action: "install", Success: false, Detail: "network error"})
	recordMust(t, tr, Event{Tool: "codex", Action: "install", Success: false, Detail: "permission denied"})

	report, err := tr.GetReport(7)
	if err != nil {
		t.Fatalf("GetReport: %v", err)
	}

	// All three install events (1 success + 2 fail) should be counted
	if report.ToolActions["codex"]["install"] != 3 {
		t.Errorf("codex install count (all outcomes) = %d, want 3", report.ToolActions["codex"]["install"])
	}
}

// ============================================================
// Scenario: Day boundary — DailyActive counts
// ============================================================

// TestScenario_UserActiveTwoDays_DailyCountsCorrect verifies that events on
// two different calendar days produce distinct DailyActive entries.
func TestScenario_UserActiveTwoDays_DailyCountsCorrect(t *testing.T) {
	tr := setupTrackerScenario(t)

	today := time.Now()
	yesterday := today.AddDate(0, 0, -1)

	// 3 events today, 2 events yesterday
	for i := 0; i < 3; i++ {
		recordMust(t, tr, Event{
			Timestamp: today.Format(time.RFC3339),
			Tool:      "claude",
			Action:    "config",
			Success:   true,
		})
	}
	for i := 0; i < 2; i++ {
		recordMust(t, tr, Event{
			Timestamp: yesterday.Format(time.RFC3339),
			Tool:      "claude",
			Action:    "install",
			Success:   true,
		})
	}

	report, err := tr.GetReport(7)
	if err != nil {
		t.Fatalf("GetReport: %v", err)
	}

	todayKey := today.Format("2006-01-02")
	yesterdayKey := yesterday.Format("2006-01-02")

	if report.DailyActive[todayKey] != 3 {
		t.Errorf("today DailyActive = %d, want 3", report.DailyActive[todayKey])
	}
	if report.DailyActive[yesterdayKey] != 2 {
		t.Errorf("yesterday DailyActive = %d, want 2", report.DailyActive[yesterdayKey])
	}
}

// ============================================================
// Scenario: GetReport with zero/negative days defaults to 7
// ============================================================

// TestScenario_NegativeDaysParam_TreatedAs7Days verifies that a programming
// error (passing 0 or negative days) doesn't crash or return an empty report
// when data exists within the last 7 days.
func TestScenario_NegativeDaysParam_TreatedAs7Days(t *testing.T) {
	tr := setupTrackerScenario(t)

	recordMust(t, tr, Event{Tool: "claude", Action: "install", Success: true})

	for _, days := range []int{0, -1, -99} {
		report, err := tr.GetReport(days)
		if err != nil {
			t.Fatalf("GetReport(%d): %v", days, err)
		}
		if report.ToolActions["claude"]["install"] != 1 {
			t.Errorf("GetReport(%d) should use 7-day window and find event, got %d",
				days, report.ToolActions["claude"]["install"])
		}
	}
}
