package main

import (
	"fmt"
	"os"
	"path/filepath"

	"lurus-switch/internal/bashguard"
)

// ============================
// Bash-Guard Bindings
// ============================
//
// The PreToolUse deny-list hook for Claude Code's shell tool. These
// bindings let the UI install/uninstall the hook, preview rules without
// touching live config, and tail the audit log.

// BashGuardListRules returns the deny-list rules. UI uses this to show
// what Switch is willing to block.
func (a *App) BashGuardListRules() []*bashguard.Rule {
	return bashguard.DefaultRules()
}

// BashGuardTestCommand evaluates a command without installing/running
// any hook — pure preview so users can test their workflows without
// touching the live CLI integration.
func (a *App) BashGuardTestCommand(cmd string) (*bashguard.MatchResult, error) {
	eng, err := bashguard.NewEngine(bashguard.DefaultRules())
	if err != nil {
		return nil, err
	}
	r := eng.Evaluate(cmd)
	return &r, nil
}

// BashGuardClaudeStatus reports whether the PreToolUse hook is wired
// into the user's claude settings and (if so) what the hook command is.
func (a *App) BashGuardClaudeStatus() bashguard.HookInstallStatus {
	return bashguard.CheckClaudeHook(claudeSettingsPath())
}

// BashGuardInstallClaude wires the PreToolUse hook to call this very
// executable with --bashguard, so subsequent Claude Code shell tool
// invocations route through Switch's deny-list.
func (a *App) BashGuardInstallClaude() error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	hookCmd := fmt.Sprintf("%q --bashguard", exe)
	return bashguard.InstallClaudeHook(claudeSettingsPath(), hookCmd)
}

// BashGuardUninstallClaude removes only our hook entry, preserving any
// user-managed PreToolUse hooks.
func (a *App) BashGuardUninstallClaude() error {
	return bashguard.UninstallClaudeHook(claudeSettingsPath())
}

// BashGuardRecentBlocks returns the tail of the audit log so the UI
// can show what got blocked recently.
func (a *App) BashGuardRecentBlocks(max int) ([]bashguard.BlockEntry, error) {
	logPath := filepath.Join(appDataBaseDir(), "bashguard-blocks.jsonl")
	return bashguard.ReadRecentBlocks(logPath, max)
}

func claudeSettingsPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".claude", "settings.json")
}
