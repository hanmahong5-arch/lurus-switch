package main

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	goruntime "runtime"
	"strings"
	"time"
)

// ============================
// Terminal Launch Binding
// ============================
//
// LaunchToolInTerminal spawns an external terminal window and runs the CLI
// tool's launch command inside it. This is the "double-click to start
// Claude Code" affordance for non-technical users — the desktop GUI does
// what they'd otherwise type into a terminal themselves.
//
// Behaviour by platform:
//
//   - Windows: prefer Windows Terminal (`wt.exe`), fall back to cmd.exe
//     via `start cmd /k <cmd>`. Both keep the window open after the
//     process exits so users can read errors.
//   - macOS: `open -a Terminal <script>` invoking a tiny shell wrapper.
//   - Linux: try x-terminal-emulator → gnome-terminal → xterm.
//
// The launched process is detached from Switch (does not inherit Switch's
// stdin/stdout). If the binary isn't on PATH we surface a typed error so
// the UI can route to the install action instead of showing a raw shell
// error like "command not found".

// toolLaunchSpec is the per-tool launch recipe. Each entry has the binary
// users would normally type plus any "just-work" flags that bypass the
// CLI's interactive permission gates — Switch's whole point is to be a
// 360-grade desktop tool that runs the CLI for the user, so an interactive
// "are you sure?" prompt on every shell command would defeat the purpose.
// Power-user flags chosen per upstream docs:
//   - claude: --dangerously-skip-permissions (canonical, see Claude Code
//     issue #10077 / BashGuardModal copy).
//   - codex: --dangerously-bypass-approvals-and-sandbox.
//   - gemini: --yolo.
//   - claw family: no documented bypass flag → empty.
type toolLaunchSpec struct {
	Bin  string
	Args []string
}

var toolLaunchCmd = map[string]toolLaunchSpec{
	"claude":   {"claude", []string{"--dangerously-skip-permissions"}},
	"codex":    {"codex", []string{"--dangerously-bypass-approvals-and-sandbox"}},
	"gemini":   {"gemini", []string{"--yolo"}},
	"picoclaw": {"picoclaw", nil},
	"nullclaw": {"nullclaw", nil},
	"zeroclaw": {"zeroclaw", nil},
	"openclaw": {"openclaw", nil},
}

// LaunchToolInTerminal opens a fresh terminal window and runs the tool's
// launch command. Returns an error with a stable prefix ("not-found",
// "no-terminal", "spawn") so the frontend can pick the right toast copy.
func (a *App) LaunchToolInTerminal(tool string) error {
	spec, ok := toolLaunchCmd[strings.ToLower(strings.TrimSpace(tool))]
	if !ok {
		return fmt.Errorf("not-found: unknown tool %q", tool)
	}

	// Resolve the binary before we spawn — if it isn't on PATH we want a
	// clean, typed error rather than a flash-of-terminal-window then a
	// confusing "command not found" inside the spawned shell.
	if _, err := exec.LookPath(spec.Bin); err != nil {
		return fmt.Errorf("not-found: %s 不在 PATH 中，请先安装或修复 PATH", spec.Bin)
	}

	// bun's global-install shims occasionally end up pointing at a moved or
	// deleted node_modules entry (image #18 in this session: claude shim
	// printing "Bun failed to remap this bin"). The terminal flashes the
	// failure and closes; user can't tell why. Probe once with a short
	// timeout — if the binary itself is broken, surface a typed error with
	// the exact repair command so the UI can show "重新安装" instead of
	// launching a doomed terminal.
	if hint := probeBrokenShim(spec.Bin); hint != "" {
		return fmt.Errorf("broken-bin: %s", hint)
	}

	full := strings.TrimSpace(spec.Bin + " " + strings.Join(spec.Args, " "))

	switch goruntime.GOOS {
	case "windows":
		return launchWindowsTerminal(full)
	case "darwin":
		return launchMacTerminal(full)
	case "linux":
		return launchLinuxTerminal(full)
	}
	return errors.New("no-terminal: unsupported OS")
}

// echoThenRun returns a single shell command string that prints the
// to-be-executed command (so the user sees *what* Switch dispatched) and
// then runs it. Uses `cmd /k` semantics — the window stays open after the
// tool exits so error output remains visible.
func echoThenRunWindows(full string) string {
	// `echo Launching: <full>` then `<full>`. Using && so a failure in
	// echo (vanishingly unlikely) still doesn't suppress the actual run.
	return "echo Launching: " + full + " && " + full
}

func echoThenRunPosix(full string) string {
	// printf is more portable than echo on some shells (busybox etc.).
	return "printf '%s\\n' 'Launching: " + full + "' ; " + full
}

// bunShimPackages maps each tool whose launcher binary is a bun-installed
// shim to the npm package the user should reinstall. Tools whose binary is
// a real native executable (claws) are absent — for those a broken bin is
// best surfaced by re-running the installer, not a `bun install` retry.
var bunShimPackages = map[string]string{
	"claude":   "@anthropic-ai/claude-code",
	"codex":    "@openai/codex",
	"gemini":   "@google/gemini-cli",
	"openclaw": "openclaw",
}

// probeBrokenShim runs `<bin> --version` with a 2s timeout and watches for
// the bun-shim error signature. Returns an empty string when the binary
// looks healthy (or when we don't have a known reinstall recipe for it),
// otherwise a human-readable hint that includes the suggested fix command.
func probeBrokenShim(bin string) string {
	pkg, ok := bunShimPackages[bin]
	if !ok {
		return ""
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, bin, "--version").CombinedOutput()
	if err == nil {
		return ""
	}
	combined := strings.ToLower(string(out) + " " + err.Error())
	switch {
	case strings.Contains(combined, "bun failed to remap"):
		return fmt.Sprintf("%s 的 bun shim 已失效（node_modules 损坏）— 终端运行：bun install -g %s --force", bin, pkg)
	case strings.Contains(combined, "could not create process"),
		strings.Contains(combined, "cannot find module"):
		return fmt.Sprintf("%s 启动器损坏 — 终端运行：bun install -g %s --force", bin, pkg)
	}
	return ""
}

func launchWindowsTerminal(full string) error {
	payload := echoThenRunWindows(full)
	// Prefer Windows Terminal if available — it gives users a tabbed
	// modern shell. `wt -- cmd /k <payload>` keeps the window open after
	// the tool exits so any final output (including stack traces) stays
	// visible.
	if _, err := exec.LookPath("wt.exe"); err == nil {
		c := exec.Command("wt.exe", "-w", "0", "new-tab", "cmd.exe", "/k", payload)
		if err := c.Start(); err == nil {
			return nil
		}
		// wt installed but blocked (AppLocker, group policy) → fall through.
	}
	// cmd.exe /c start "" cmd /k <payload> — `start ""` spawns a detached
	// console window; /k keeps it open after the inner command finishes.
	c := exec.Command("cmd.exe", "/c", "start", "", "cmd.exe", "/k", payload)
	if err := c.Start(); err != nil {
		return fmt.Errorf("spawn: %w", err)
	}
	return nil
}

func launchMacTerminal(full string) error {
	// `do script` types the payload into a fresh Terminal.app window. We
	// pre-echo with printf so the user sees the launched command, mirroring
	// the Windows behaviour.
	payload := echoThenRunPosix(full)
	script := fmt.Sprintf(`tell application "Terminal" to do script %q`, payload)
	c := exec.Command("osascript", "-e", script)
	if err := c.Start(); err != nil {
		return fmt.Errorf("spawn: %w", err)
	}
	return nil
}

func launchLinuxTerminal(full string) error {
	payload := echoThenRunPosix(full) + " ; exec bash"
	candidates := [][]string{
		{"x-terminal-emulator", "-e", "bash", "-c", payload},
		{"gnome-terminal", "--", "bash", "-c", payload},
		{"konsole", "-e", "bash", "-c", payload},
		{"xterm", "-e", "bash", "-c", payload},
	}
	for _, args := range candidates {
		if _, err := exec.LookPath(args[0]); err != nil {
			continue
		}
		c := exec.Command(args[0], args[1:]...)
		if err := c.Start(); err == nil {
			return nil
		}
	}
	return errors.New("no-terminal: no usable terminal emulator found")
}
