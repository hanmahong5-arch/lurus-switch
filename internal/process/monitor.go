package process

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

// ProcessInfo contains information about a running CLI tool process
type ProcessInfo struct {
	PID     int    `json:"pid"`
	Tool    string `json:"tool"`
	Command string `json:"command"`
	Status  string `json:"status"`
	Memory  uint64 `json:"memory"`
	Since   string `json:"since"`
}

// ringBuffer is a fixed-capacity FIFO for output lines
type ringBuffer struct {
	lines []string
	cap   int
	mu    sync.Mutex
}

func newRingBuffer(cap int) *ringBuffer {
	return &ringBuffer{lines: make([]string, 0, cap), cap: cap}
}

func (r *ringBuffer) append(line string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if len(r.lines) >= r.cap {
		r.lines = r.lines[1:]
	}
	r.lines = append(r.lines, line)
}

func (r *ringBuffer) get(max int) []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	if max <= 0 || max >= len(r.lines) {
		cp := make([]string, len(r.lines))
		copy(cp, r.lines)
		return cp
	}
	start := len(r.lines) - max
	cp := make([]string, max)
	copy(cp, r.lines[start:])
	return cp
}

// session represents a managed child process
type session struct {
	cmd    *exec.Cmd
	output *ringBuffer
	cancel context.CancelFunc
}

// Monitor manages CLI tool process discovery and lifecycle
type Monitor struct {
	sessions map[string]*session
	mu       sync.Mutex
}

// NewMonitor creates a new process monitor
func NewMonitor() *Monitor {
	return &Monitor{
		sessions: make(map[string]*session),
	}
}

// knownTools maps binary names to tool IDs
var knownTools = map[string]string{
	"claude": "claude",
	"codex":  "codex",
	"gemini": "gemini",
	"pclaw":  "picoclaw",
	"nclaw":  "nullclaw",
}

// ListCLIProcesses returns currently running CLI tool processes
func (m *Monitor) ListCLIProcesses(ctx context.Context) ([]ProcessInfo, error) {
	switch runtime.GOOS {
	case "windows":
		return m.listWindowsProcesses(ctx)
	default:
		return m.listUnixProcesses(ctx)
	}
}

func (m *Monitor) listWindowsProcesses(ctx context.Context) ([]ProcessInfo, error) {
	var results []ProcessInfo

	for binary, tool := range knownTools {
		cmd := exec.CommandContext(ctx, "tasklist", "/FI", fmt.Sprintf("IMAGENAME eq %s.exe", binary), "/FO", "CSV", "/NH")
		hideWindowCmd(cmd)
		out, err := cmd.Output()
		if err != nil {
			continue
		}

		scanner := bufio.NewScanner(strings.NewReader(string(out)))
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "" || strings.HasPrefix(line, "INFO:") {
				continue
			}
			// CSV format: "imageName","PID","sessionName","sessionNum","memUsage"
			parts := strings.Split(line, ",")
			if len(parts) < 5 {
				continue
			}
			pidStr := strings.Trim(parts[1], `"`)
			pid, err := strconv.Atoi(pidStr)
			if err != nil {
				continue
			}
			memStr := strings.Trim(parts[4], `" K`)
			memStr = strings.ReplaceAll(memStr, ",", "")
			memKB, _ := strconv.ParseUint(memStr, 10, 64)

			results = append(results, ProcessInfo{
				PID:    pid,
				Tool:   tool,
				Status: "running",
				Memory: memKB * 1024,
				Since:  time.Now().Format(time.RFC3339),
			})
		}
	}

	return results, nil
}

func (m *Monitor) listUnixProcesses(ctx context.Context) ([]ProcessInfo, error) {
	// ps -eo pid,comm,args with newline-separated output
	cmd := exec.CommandContext(ctx, "ps", "-eo", "pid,comm,args")
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("ps failed: %w", err)
	}

	var results []ProcessInfo
	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	scanner.Scan() // skip header
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		pid, err := strconv.Atoi(fields[0])
		if err != nil {
			continue
		}
		comm := fields[1]
		tool, ok := knownTools[comm]
		if !ok {
			continue
		}
		cmdline := strings.Join(fields[2:], " ")
		results = append(results, ProcessInfo{
			PID:     pid,
			Tool:    tool,
			Command: cmdline,
			Status:  "running",
			Since:   time.Now().Format(time.RFC3339),
		})
	}

	return results, nil
}

// KillProcess sends SIGINT (or TerminateProcess on Windows) and waits up to 3 seconds
func (m *Monitor) KillProcess(ctx context.Context, pid int) error {
	proc, err := os.FindProcess(pid)
	if err != nil {
		return fmt.Errorf("process %d not found: %w", pid, err)
	}

	if runtime.GOOS == "windows" {
		// Windows: kill immediately via taskkill
		cmd := exec.CommandContext(ctx, "taskkill", "/PID", strconv.Itoa(pid), "/F")
		hideWindowCmd(cmd)
		return cmd.Run()
	}

	// Unix: SIGINT then wait
	if err := proc.Signal(os.Interrupt); err != nil {
		return proc.Kill()
	}

	done := make(chan struct{})
	go func() {
		proc.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-time.After(3 * time.Second):
		return proc.Kill()
	}
}

// LaunchTool starts a CLI tool in a managed session and returns the session ID
func (m *Monitor) LaunchTool(ctx context.Context, tool string, args []string) (string, error) {
	binary, err := resolveBinary(tool)
	if err != nil {
		return "", err
	}

	sessionID := fmt.Sprintf("%s-%d", tool, time.Now().UnixMilli())

	sessionCtx, cancel := context.WithCancel(ctx)
	cmd := exec.CommandContext(sessionCtx, binary, args...)

	buf := newRingBuffer(500)

	// Pipe combined stdout+stderr into the ring buffer
	pr, pw := io.Pipe()
	cmd.Stdout = pw
	cmd.Stderr = pw

	if err := cmd.Start(); err != nil {
		cancel()
		pr.Close()
		pw.Close()
		return "", fmt.Errorf("failed to start %s: %w", tool, err)
	}

	m.mu.Lock()
	m.sessions[sessionID] = &session{cmd: cmd, output: buf, cancel: cancel}
	m.mu.Unlock()

	// Read output in a goroutine
	go func() {
		scanner := bufio.NewScanner(pr)
		for scanner.Scan() {
			buf.append(scanner.Text())
		}
		pr.Close()
	}()

	// Cleanup on process exit
	go func() {
		cmd.Wait()
		pw.Close()
		m.mu.Lock()
		delete(m.sessions, sessionID)
		m.mu.Unlock()
	}()

	return sessionID, nil
}

// GetOutput returns the most recent lines from a session's output buffer
func (m *Monitor) GetOutput(sessionID string, maxLines int) ([]string, error) {
	m.mu.Lock()
	s, ok := m.sessions[sessionID]
	m.mu.Unlock()
	if !ok {
		return nil, fmt.Errorf("session not found: %s", sessionID)
	}
	return s.output.get(maxLines), nil
}

// StopSession terminates a managed session
func (m *Monitor) StopSession(sessionID string) error {
	m.mu.Lock()
	s, ok := m.sessions[sessionID]
	m.mu.Unlock()
	if !ok {
		return fmt.Errorf("session not found: %s", sessionID)
	}
	s.cancel()
	return nil
}

// resolveBinary finds the executable name for a tool ID
func resolveBinary(tool string) (string, error) {
	switch tool {
	case "claude":
		return "claude", nil
	case "codex":
		return "codex", nil
	case "gemini":
		return "gemini", nil
	case "picoclaw":
		return "pclaw", nil
	case "nullclaw":
		return "nclaw", nil
	default:
		return "", fmt.Errorf("unknown tool: %s", tool)
	}
}

// hideWindowCmd suppresses the console window on Windows (stub; real impl in exec_windows.go)
func hideWindowCmd(cmd *exec.Cmd) {
	hideWindowProcess(cmd)
}
