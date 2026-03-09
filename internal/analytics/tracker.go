package analytics

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

// Event represents a single user action event
type Event struct {
	Timestamp string `json:"ts"`
	Tool      string `json:"tool"`
	Action    string `json:"action"` // "install" | "update" | "uninstall" | "config" | "prompt_apply"
	Success   bool   `json:"success"`
	Detail    string `json:"detail,omitempty"`
}

// UsageReport summarizes events from the analytics log
type UsageReport struct {
	ToolActions  map[string]map[string]int `json:"toolActions"`  // tool → action → count
	DailyActive  map[string]int            `json:"dailyActive"`  // date → event count
	ConfigCounts map[string]int            `json:"configCounts"` // tool → config count
	PromptCount  int                       `json:"promptCount"`
}

// Tracker appends events to a local JSONL file
type Tracker struct {
	path string
	mu   sync.Mutex
}

// NewTracker creates a tracker writing to ~/.lurus-switch/analytics.jsonl
func NewTracker() (*Tracker, error) {
	p, err := analyticsPath()
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
		return nil, fmt.Errorf("failed to create analytics directory: %w", err)
	}
	return &Tracker{path: p}, nil
}

// analyticsPath returns the JSONL file path
func analyticsPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}

	var base string
	switch runtime.GOOS {
	case "windows":
		appData := os.Getenv("APPDATA")
		if appData == "" {
			appData = filepath.Join(home, "AppData", "Roaming")
		}
		base = filepath.Join(appData, "lurus-switch")
	case "darwin":
		base = filepath.Join(home, "Library", "Application Support", "lurus-switch")
	default:
		base = filepath.Join(home, ".lurus-switch")
	}

	return filepath.Join(base, "analytics.jsonl"), nil
}

// Record appends one event to the JSONL log
func (t *Tracker) Record(e Event) error {
	if e.Timestamp == "" {
		e.Timestamp = time.Now().Format(time.RFC3339)
	}

	data, err := json.Marshal(e)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	f, err := os.OpenFile(t.path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open analytics file: %w", err)
	}
	defer f.Close()

	_, err = fmt.Fprintln(f, string(data))
	return err
}

// GetReport reads the log and generates a summary for the past `days` days
func (t *Tracker) GetReport(days int) (*UsageReport, error) {
	if days <= 0 {
		days = 7
	}

	cutoff := time.Now().AddDate(0, 0, -days)
	report := &UsageReport{
		ToolActions:  make(map[string]map[string]int),
		DailyActive:  make(map[string]int),
		ConfigCounts: make(map[string]int),
	}

	t.mu.Lock()
	f, err := os.Open(t.path)
	t.mu.Unlock()

	if err != nil {
		if os.IsNotExist(err) {
			return report, nil
		}
		return nil, fmt.Errorf("failed to open analytics file: %w", err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var e Event
		if err := json.Unmarshal([]byte(line), &e); err != nil {
			continue
		}

		ts, err := time.Parse(time.RFC3339, e.Timestamp)
		if err != nil || ts.Before(cutoff) {
			continue
		}

		// Tool action counts
		if _, ok := report.ToolActions[e.Tool]; !ok {
			report.ToolActions[e.Tool] = make(map[string]int)
		}
		report.ToolActions[e.Tool][e.Action]++

		// Daily active counts
		day := ts.Format("2006-01-02")
		report.DailyActive[day]++
	}

	return report, scanner.Err()
}
