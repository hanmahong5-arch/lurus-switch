package configapply

import (
	"strings"
	"testing"
)

func TestDiffSummary(t *testing.T) {
	cases := []struct {
		name     string
		before   string
		after    string
		expected string
	}{
		{"identical", "a\nb\nc\n", "a\nb\nc\n", "no change"},
		{"add 2", "a\nb\n", "a\nb\nc\nd\n", "+2 -0"},
		{"del 2", "a\nb\nc\nd\n", "a\nb\n", "+0 -2"},
		{"replace 1", "a\nb\nc\n", "a\nx\nc\n", "+1 -1"},
		{"empty to content", "", "hello\n", "+1 -0"},
		{"content to empty", "hello\n", "", "+0 -1"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := DiffSummary(c.before, c.after)
			if got != c.expected {
				t.Errorf("DiffSummary(%q, %q) = %q, want %q", c.before, c.after, got, c.expected)
			}
		})
	}
}

func TestUnifiedDiff_NoChange(t *testing.T) {
	if got := UnifiedDiff("a\nb\n", "a\nb\n", 3); got != "" {
		t.Errorf("expected empty diff, got %q", got)
	}
}

func TestUnifiedDiff_Replace(t *testing.T) {
	got := UnifiedDiff("a\nb\nc\n", "a\nx\nc\n", 3)
	if !strings.Contains(got, "-b") {
		t.Errorf("expected '-b' in diff, got: %s", got)
	}
	if !strings.Contains(got, "+x") {
		t.Errorf("expected '+x' in diff, got: %s", got)
	}
}

func TestUnifiedDiff_Append(t *testing.T) {
	got := UnifiedDiff("a\nb\n", "a\nb\nc\n", 3)
	if !strings.Contains(got, "+c") {
		t.Errorf("expected '+c' in diff, got: %s", got)
	}
	if strings.Contains(got, "-a") || strings.Contains(got, "-b") {
		t.Errorf("expected no deletions, got: %s", got)
	}
}

func TestCountLines(t *testing.T) {
	cases := []struct {
		input    string
		expected int
	}{
		{"", 0},
		{"a", 1},
		{"a\n", 1},
		{"a\nb", 2},
		{"a\nb\n", 2},
		{"a\nb\nc\n", 3},
	}
	for _, c := range cases {
		got := countLines(c.input)
		if got != c.expected {
			t.Errorf("countLines(%q) = %d, want %d", c.input, got, c.expected)
		}
	}
}
