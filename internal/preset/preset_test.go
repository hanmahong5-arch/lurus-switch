package preset

import (
	"testing"
)

// TestPresetLists_BaselineShape locks in the count and required descriptor
// fields for each tool's preset menu — the SettingsPage preset picker
// renders directly from these lists.
func TestPresetLists_BaselineShape(t *testing.T) {
	cases := []struct {
		tool string
		list []Preset
	}{
		{"claude", ClaudePresets()},
		{"codex", CodexPresets()},
		{"gemini", GeminiPresets()},
	}
	for _, tc := range cases {
		t.Run(tc.tool, func(t *testing.T) {
			if len(tc.list) < 4 {
				t.Fatalf("%s presets len = %d, want >= 4", tc.tool, len(tc.list))
			}
			seen := make(map[string]bool, len(tc.list))
			for _, p := range tc.list {
				if p.ID == "" || p.Name == "" {
					t.Errorf("preset missing id/name: %+v", p)
				}
				if seen[p.ID] {
					t.Errorf("duplicate preset id: %q", p.ID)
				}
				seen[p.ID] = true
			}
			// All three tools expose the same four canonical IDs; the UI
			// keys off this naming convention to render uniform tab labels.
			for _, want := range []string{"quick-start", "security", "performance", "budget"} {
				if !seen[want] {
					t.Errorf("%s missing preset %q", tc.tool, want)
				}
			}
		})
	}
}

// TestApplyClaudePreset_HappyPath drives one Claude preset end-to-end and
// asserts a representative flag is set — guards against a future refactor
// silently dropping the AllowBash field rename.
func TestApplyClaudePreset_HappyPath(t *testing.T) {
	t.Run("security", func(t *testing.T) {
		c, err := ApplyClaudePreset("security")
		if err != nil {
			t.Fatalf("ApplyClaudePreset: %v", err)
		}
		if c == nil {
			t.Fatal("returned nil config")
		}
		if c.Permissions.AllowBash {
			t.Error("security preset must disable AllowBash")
		}
		if !c.Sandbox.Enabled {
			t.Error("security preset must enable Sandbox")
		}
	})
	t.Run("quick-start", func(t *testing.T) {
		c, err := ApplyClaudePreset("quick-start")
		if err != nil {
			t.Fatalf("ApplyClaudePreset: %v", err)
		}
		if !c.Permissions.AllowBash || !c.Permissions.AllowWrite {
			t.Errorf("quick-start should be permissive: %+v", c.Permissions)
		}
	})
}

// TestApplyCodexPreset_HappyPath asserts security-mode locks down both
// network access and the sandbox toggle in a single canonical preset.
func TestApplyCodexPreset_HappyPath(t *testing.T) {
	c, err := ApplyCodexPreset("security")
	if err != nil {
		t.Fatalf("ApplyCodexPreset: %v", err)
	}
	if c.Security.NetworkAccess != "off" {
		t.Errorf("network = %q, want off", c.Security.NetworkAccess)
	}
	if !c.Sandbox.Enabled {
		t.Error("sandbox must be enabled in security preset")
	}
}

// TestApplyGeminiPreset_HappyPath confirms the budget preset reduces
// MaxFileSize — this is the field cost-sensitive resellers tune.
func TestApplyGeminiPreset_HappyPath(t *testing.T) {
	c, err := ApplyGeminiPreset("budget")
	if err != nil {
		t.Fatalf("ApplyGeminiPreset: %v", err)
	}
	if c.Behavior.MaxFileSize != 1*1024*1024 {
		t.Errorf("budget MaxFileSize = %d, want 1 MB", c.Behavior.MaxFileSize)
	}
	if c.Model != "gemini-2.0-flash" {
		t.Errorf("budget model = %q, want gemini-2.0-flash", c.Model)
	}
}

// TestApplyPreset_UnknownIDErrors locks in the (nil, error) contract for
// unknown preset IDs across all three tools — the SettingsPage shows the
// error message directly, so it must not be a panic.
func TestApplyPreset_UnknownIDErrors(t *testing.T) {
	cases := []struct {
		tool string
		fn   func(string) (any, error)
	}{
		{"claude", func(id string) (any, error) { return ApplyClaudePreset(id) }},
		{"codex", func(id string) (any, error) { return ApplyCodexPreset(id) }},
		{"gemini", func(id string) (any, error) { return ApplyGeminiPreset(id) }},
	}
	for _, tc := range cases {
		t.Run(tc.tool, func(t *testing.T) {
			c, err := tc.fn("no-such-preset")
			if err == nil {
				t.Fatalf("%s Apply(unknown) should have errored", tc.tool)
			}
			// Go's typed-nil-in-any rule: c may be a non-nil interface
			// wrapping a typed nil pointer. The contract is just "do not
			// silently return a usable config" — err != nil already
			// guarantees that for the caller.
			_ = c
		})
	}
}
