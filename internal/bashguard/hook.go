package bashguard

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

// HookInput is the JSON payload Claude Code's PreToolUse hook delivers
// on stdin. Other CLIs (Codex/Gemini) use different schemas — handled
// in their own ParseXxx helpers when they're added.
type HookInput struct {
	HookEventName string `json:"hook_event_name"`
	ToolName      string `json:"tool_name"`
	ToolInput     struct {
		Command string `json:"command"`
	} `json:"tool_input"`
	Cwd string `json:"cwd,omitempty"`
}

// BlockEntry is one row in the audit log persisted to disk so the UI
// can show "what we blocked recently".
type BlockEntry struct {
	Time     time.Time `json:"time"`
	Tool     string    `json:"tool"`
	Command  string    `json:"command"`
	RuleID   string    `json:"ruleId"`
	Reason   string    `json:"reason"`
	Severity string    `json:"severity"`
	Cwd      string    `json:"cwd,omitempty"`
}

// HandleStdin runs as the PreToolUse hook in CLI mode. Reads JSON from
// stdin, evaluates against rules, and exits:
//   - exit 0 → allow (Claude Code proceeds)
//   - exit 2 → block (Claude Code aborts; stderr shown to user/agent)
//
// The 0/2 contract is what Claude Code's hooks expect (per official docs).
// Any other exit code is treated as "non-blocking error" and Claude
// proceeds, so we deliberately stick to 0/2.
func HandleStdin(stdin io.Reader, stderr io.Writer, logPath string, e *Engine) int {
	body, err := io.ReadAll(stdin)
	if err != nil {
		fmt.Fprintln(stderr, "[lurus-bashguard] read stdin:", err)
		return 0 // fail-open on infrastructure errors
	}
	var in HookInput
	if jerr := json.Unmarshal(body, &in); jerr != nil {
		// Not the format we know — let it through rather than break the
		// CLI. The user can re-enable strict mode if they want.
		return 0
	}
	cmd := in.ToolInput.Command
	if cmd == "" {
		return 0
	}
	res := e.Evaluate(cmd)
	if res.Allowed {
		return 0
	}
	// Log the block before signalling Claude.
	_ = appendBlockLog(logPath, BlockEntry{
		Time: time.Now(), Tool: in.ToolName, Command: cmd,
		RuleID: res.Rule.ID, Reason: res.Rule.ReasonEn, Severity: string(res.Rule.Severity),
		Cwd: in.Cwd,
	})
	fmt.Fprintf(stderr, "🛡  Lurus Bash-Guard blocked: %s\n", res.Rule.ReasonEn)
	fmt.Fprintf(stderr, "   Rule: %s (%s)\n", res.Rule.ID, res.Rule.Severity)
	if res.Rule.Reference != "" {
		fmt.Fprintf(stderr, "   Reference: %s\n", res.Rule.Reference)
	}
	return 2
}

func appendBlockLog(path string, entry BlockEntry) error {
	if path == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	line, err := json.Marshal(entry)
	if err != nil {
		return err
	}
	if _, err := f.Write(append(line, '\n')); err != nil {
		return err
	}
	return nil
}

// ReadRecentBlocks reads the tail of the JSONL audit log. Used by the
// UI to populate the "Recent blocks" panel.
func ReadRecentBlocks(path string, max int) ([]BlockEntry, error) {
	if max <= 0 {
		max = 50
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	out := make([]BlockEntry, 0, max)
	// Walk from end to start so we get newest first.
	end := len(data)
	for end > 0 && len(out) < max {
		// Find previous newline
		start := end - 1
		for start > 0 && data[start-1] != '\n' {
			start--
		}
		line := data[start:end]
		if len(line) > 0 && line[len(line)-1] == '\n' {
			line = line[:len(line)-1]
		}
		if len(line) > 0 {
			var entry BlockEntry
			if json.Unmarshal(line, &entry) == nil {
				out = append(out, entry)
			}
		}
		end = start
		if end > 0 {
			end-- // skip the newline
		}
	}
	return out, nil
}

// ─── Claude Code hook installer ─────────────────────────────────────

// HookInstallStatus is the user-facing state of the Bash-Guard
// integration with each CLI we know how to wire.
type HookInstallStatus struct {
	Tool       string `json:"tool"`        // "claude" for now
	Installed  bool   `json:"installed"`
	HookCmd    string `json:"hookCmd"`     // current command in settings.json
	ConfigPath string `json:"configPath"`
	Issue      string `json:"issue,omitempty"`
}

const (
	HookMatcher = "Bash"
	// The marker is embedded as a comment-like field in the hook entry so
	// we can find/remove our own hook without touching user-installed ones.
	HookMarker = "lurus-bashguard"
)

// InstallClaudeHook adds (or refreshes) a PreToolUse Bash hook in
// ~/.claude/settings.json that executes `cmd` with the bashguard
// arguments. Idempotent — re-running just updates the path.
func InstallClaudeHook(settingsPath, hookCommand string) error {
	if settingsPath == "" {
		return fmt.Errorf("settings path required")
	}
	if hookCommand == "" {
		return fmt.Errorf("hook command required")
	}
	raw := readSettings(settingsPath)

	hooks, _ := raw["hooks"].(map[string]interface{})
	if hooks == nil {
		hooks = map[string]interface{}{}
		raw["hooks"] = hooks
	}
	preToolUse, _ := hooks["PreToolUse"].([]interface{})

	// Filter out any existing lurus-bashguard entry so re-install just
	// upserts. Keep user's other hooks untouched.
	cleaned := preToolUse[:0]
	for _, h := range preToolUse {
		if !isOurHook(h) {
			cleaned = append(cleaned, h)
		}
	}

	entry := map[string]interface{}{
		"matcher": HookMatcher,
		"hooks": []interface{}{
			map[string]interface{}{
				"type":    "command",
				"command": hookCommand,
				"_lurus":  HookMarker, // sentinel so we recognize ourselves on uninstall
			},
		},
	}
	hooks["PreToolUse"] = append(cleaned, entry)
	return writeSettings(settingsPath, raw)
}

// UninstallClaudeHook removes only our own hook entry, leaving any
// user-added PreToolUse hooks intact.
func UninstallClaudeHook(settingsPath string) error {
	raw := readSettings(settingsPath)
	hooks, _ := raw["hooks"].(map[string]interface{})
	if hooks == nil {
		return nil
	}
	preToolUse, _ := hooks["PreToolUse"].([]interface{})
	cleaned := preToolUse[:0]
	for _, h := range preToolUse {
		if !isOurHook(h) {
			cleaned = append(cleaned, h)
		}
	}
	if len(cleaned) == 0 {
		delete(hooks, "PreToolUse")
	} else {
		hooks["PreToolUse"] = cleaned
	}
	return writeSettings(settingsPath, raw)
}

// CheckClaudeHook reports whether our hook is currently registered in
// the settings file. Used by the UI to render the toggle state.
func CheckClaudeHook(settingsPath string) HookInstallStatus {
	st := HookInstallStatus{Tool: "claude", ConfigPath: settingsPath}
	raw := readSettings(settingsPath)
	hooks, _ := raw["hooks"].(map[string]interface{})
	if hooks == nil {
		return st
	}
	preToolUse, _ := hooks["PreToolUse"].([]interface{})
	for _, h := range preToolUse {
		if isOurHook(h) {
			st.Installed = true
			if entry, _ := h.(map[string]interface{}); entry != nil {
				if hs, _ := entry["hooks"].([]interface{}); len(hs) > 0 {
					if hh, _ := hs[0].(map[string]interface{}); hh != nil {
						st.HookCmd, _ = hh["command"].(string)
					}
				}
			}
			return st
		}
	}
	return st
}

func isOurHook(h interface{}) bool {
	entry, _ := h.(map[string]interface{})
	if entry == nil {
		return false
	}
	hs, _ := entry["hooks"].([]interface{})
	for _, hh := range hs {
		hm, _ := hh.(map[string]interface{})
		if hm == nil {
			continue
		}
		if marker, _ := hm["_lurus"].(string); marker == HookMarker {
			return true
		}
	}
	return false
}

func readSettings(path string) map[string]interface{} {
	data, err := os.ReadFile(path)
	if err != nil {
		return map[string]interface{}{}
	}
	var raw map[string]interface{}
	if json.Unmarshal(data, &raw) != nil {
		return map[string]interface{}{}
	}
	return raw
}

func writeSettings(path string, raw map[string]interface{}) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	body, err := json.MarshalIndent(raw, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, body, 0o644)
}
