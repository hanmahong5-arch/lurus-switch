package mcp

import (
	"bufio"
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sync"
	"sync/atomic"
	"time"
)

// Timeout / backoff constants — see Winston design review §3.3.
// Not env-overridable on purpose: the dispatcher contract relies on
// these holding for the whole session.
const (
	HandshakeTimeout  = 10 * time.Second
	GracefulStopWait  = 5 * time.Second
	MaxRestarts       = 3
	BackoffBase       = 200 * time.Millisecond
	BackoffJitterFrac = 20 // ±20%
)

// ErrHandshakeTimeout is returned by Start when the child fails to
// respond to initialize within HandshakeTimeout.
var ErrHandshakeTimeout = errors.New("mcp: handshake timeout")

// ErrServerNotFound is returned by Send when no handle for that preset
// is registered.
var ErrServerNotFound = errors.New("mcp: server not found")

// ErrRestartExhausted is returned by Start when MaxRestarts crashes
// have accumulated for one preset.
var ErrRestartExhausted = errors.New("mcp: restart budget exhausted")

// credScrubRE redacts auth-bearing tokens from stderr before logging.
// Risk #3 from Winston review (defense-in-depth, server may log secrets).
var credScrubRE = regexp.MustCompile(`(?i)(authorization|api[-_]?key|token|secret)\s*[:=]\s*\S+`)

// handle owns one MCP subprocess and its IO plumbing.
type handle struct {
	name    string
	cmd     *exec.Cmd
	stdin   io.WriteCloser
	stdout  io.ReadCloser
	stderr  io.ReadCloser
	done    chan struct{} // closed on Stop; readers select on this
	pending sync.Map      // id -> chan json.RawMessage
	nextID  atomic.Int64
}

// Runtime is the in-process supervisor described in design review §2.1.
// Owns a map of preset-name → handle, each backed by one os.Process plus
// stdin writer + stdout reader + stderr reader goroutines.
type Runtime struct {
	mu       sync.Mutex
	handles  map[string]*handle
	restarts map[string]int // preset name → crashes-this-session
	workdir  string
	logf     func(format string, args ...any) // pluggable (defaults to stderr)
}

// NewRuntime constructs a Runtime. workdir, if empty, defaults to
// `<os.TempDir()>/switch-mcp` — Windows-safe per §3.2 (resolves to
// `%LOCALAPPDATA%\Temp\switch-mcp`, never hardcoded `/tmp`).
func NewRuntime(workdir string) *Runtime {
	if workdir == "" {
		workdir = filepath.Join(os.TempDir(), "switch-mcp")
	}
	return &Runtime{
		handles:  make(map[string]*handle),
		restarts: make(map[string]int),
		workdir:  workdir,
		logf:     func(f string, a ...any) { fmt.Fprintf(os.Stderr, "mcp: "+f+"\n", a...) },
	}
}

// SetLogger swaps the diagnostic sink (used by tests + production wire).
func (r *Runtime) SetLogger(fn func(format string, args ...any)) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if fn != nil {
		r.logf = fn
	}
}

// Start spawns the child described by srv under preset name `name`,
// performs the initialize handshake (10s budget), and registers the
// handle. Counts crashes against the MaxRestarts budget.
func (r *Runtime) Start(ctx context.Context, name string, srv MCPServer) error {
	r.mu.Lock()
	if r.restarts[name] >= MaxRestarts {
		r.mu.Unlock()
		return fmt.Errorf("%w: %s (%d crashes)", ErrRestartExhausted, name, r.restarts[name])
	}
	r.mu.Unlock()

	dir := filepath.Join(r.workdir, name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("mcp: workdir: %w", err)
	}

	cmd := exec.CommandContext(ctx, srv.Command, srv.Args...)
	cmd.Dir = dir
	// env_passthrough allow-list — never inherit os.Environ() wholesale
	// (credential hygiene, §2.3).
	if len(srv.Env) > 0 {
		envSlice := make([]string, 0, len(srv.Env))
		for k, v := range srv.Env {
			envSlice = append(envSlice, k+"="+v)
		}
		cmd.Env = envSlice
	}

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("mcp: stdin pipe: %w", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("mcp: stdout pipe: %w", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("mcp: stderr pipe: %w", err)
	}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("mcp: spawn %s: %w", srv.Command, err)
	}

	h := &handle{
		name:   name,
		cmd:    cmd,
		stdin:  stdin,
		stdout: stdout,
		stderr: stderr,
		done:   make(chan struct{}),
	}
	go r.readStdout(h)
	go r.readStderr(h)

	if err := r.handshake(ctx, h); err != nil {
		_ = r.stopHandle(h)
		r.mu.Lock()
		r.restarts[name]++
		r.mu.Unlock()
		return err
	}

	r.mu.Lock()
	r.handles[name] = h
	r.mu.Unlock()
	return nil
}

// handshake sends the initialize JSON-RPC frame and waits up to
// HandshakeTimeout for a response. Used by Start; not exported.
func (r *Runtime) handshake(parent context.Context, h *handle) error {
	ctx, cancel := context.WithTimeout(parent, HandshakeTimeout)
	defer cancel()
	id := h.nextID.Add(1)
	ch := make(chan json.RawMessage, 1)
	h.pending.Store(id, ch)
	defer h.pending.Delete(id)

	frame := map[string]any{"jsonrpc": "2.0", "id": id, "method": "initialize"}
	if err := writeFrame(h.stdin, frame); err != nil {
		return fmt.Errorf("mcp: handshake write: %w", err)
	}
	select {
	case <-ch:
		return nil
	case <-ctx.Done():
		return fmt.Errorf("%w: %s", ErrHandshakeTimeout, h.name)
	}
}

// Send dispatches a JSON-RPC call to the named server with a 60s
// deadline owned by the caller's ctx (dispatcher adds the timeout).
// Returns the raw result payload — caller unmarshals.
func (r *Runtime) Send(ctx context.Context, name string, method string, params any) (json.RawMessage, error) {
	r.mu.Lock()
	h, ok := r.handles[name]
	r.mu.Unlock()
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrServerNotFound, name)
	}
	id := h.nextID.Add(1)
	ch := make(chan json.RawMessage, 1)
	h.pending.Store(id, ch)
	defer h.pending.Delete(id)

	frame := map[string]any{"jsonrpc": "2.0", "id": id, "method": method, "params": params}
	if err := writeFrame(h.stdin, frame); err != nil {
		return nil, fmt.Errorf("mcp: send write: %w", err)
	}
	select {
	case raw := <-ch:
		return raw, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// Stop gracefully shuts down a single server: closes stdin, waits up
// to GracefulStopWait, then proc.Kill().
func (r *Runtime) Stop(name string) error {
	r.mu.Lock()
	h, ok := r.handles[name]
	if ok {
		delete(r.handles, name)
	}
	r.mu.Unlock()
	if !ok {
		return ErrServerNotFound
	}
	return r.stopHandle(h)
}

func (r *Runtime) stopHandle(h *handle) error {
	close(h.done) // unblock readers (Risk #1)
	_ = h.stdin.Close()
	exited := make(chan error, 1)
	go func() { exited <- h.cmd.Wait() }()
	select {
	case <-exited:
		return nil
	case <-time.After(GracefulStopWait):
		_ = h.cmd.Process.Kill()
		<-exited
		return nil
	}
}

// Shutdown stops every registered server. Safe to call multiple times.
func (r *Runtime) Shutdown(ctx context.Context) error {
	r.mu.Lock()
	names := make([]string, 0, len(r.handles))
	for n := range r.handles {
		names = append(names, n)
	}
	r.mu.Unlock()
	var firstErr error
	for _, n := range names {
		if err := r.Stop(n); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

// readStdout demuxes JSON-RPC responses to per-id channels. Exits on
// EOF, read error, or done-channel close (Risk #1).
func (r *Runtime) readStdout(h *handle) {
	reader := bufio.NewReader(h.stdout)
	for {
		select {
		case <-h.done:
			return
		default:
		}
		payload, err := readFrame(reader)
		if err != nil {
			if err != io.EOF {
				r.logf("server=%s stdout read: %v", h.name, err)
			}
			return
		}
		var env struct {
			ID     int64           `json:"id"`
			Result json.RawMessage `json:"result"`
			Error  json.RawMessage `json:"error"`
		}
		if err := json.Unmarshal(payload, &env); err != nil {
			r.logf("server=%s stdout unmarshal: %v", h.name, err)
			continue
		}
		v, ok := h.pending.Load(env.ID)
		if !ok {
			continue
		}
		ch := v.(chan json.RawMessage)
		// Use Result on success, Error otherwise — caller inspects.
		raw := env.Result
		if len(raw) == 0 {
			raw = env.Error
		}
		select {
		case ch <- raw:
		default:
		}
	}
}

// readStderr forwards child diagnostics to logf, scrubbing credentials
// per Risk #3.
func (r *Runtime) readStderr(h *handle) {
	scanner := bufio.NewScanner(h.stderr)
	for scanner.Scan() {
		select {
		case <-h.done:
			return
		default:
		}
		line := credScrubRE.ReplaceAllString(scanner.Text(), "$1=<REDACTED>")
		r.logf("server=%s stderr: %s", h.name, line)
	}
}

// writeFrame encodes payload + Content-Length header per MCP spec
// (Risk #2 — line-delimited JSON splits across Windows 4KB pipe
// boundaries on large tool results).
func writeFrame(w io.Writer, payload any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	header := fmt.Sprintf("Content-Length: %d\r\n\r\n", len(body))
	if _, err := w.Write([]byte(header)); err != nil {
		return err
	}
	_, err = w.Write(body)
	return err
}

// readFrame parses one Content-Length-prefixed JSON-RPC message.
func readFrame(r *bufio.Reader) ([]byte, error) {
	var contentLen int
	for {
		line, err := r.ReadString('\n')
		if err != nil {
			return nil, err
		}
		line = trimCRLF(line)
		if line == "" {
			break // end of headers
		}
		var n int
		if _, err := fmt.Sscanf(line, "Content-Length: %d", &n); err == nil {
			contentLen = n
		}
	}
	if contentLen <= 0 {
		return nil, fmt.Errorf("mcp: missing Content-Length header")
	}
	buf := make([]byte, contentLen)
	if _, err := io.ReadFull(r, buf); err != nil {
		return nil, err
	}
	return buf, nil
}

func trimCRLF(s string) string {
	for len(s) > 0 && (s[len(s)-1] == '\r' || s[len(s)-1] == '\n') {
		s = s[:len(s)-1]
	}
	return s
}

// jitter returns a random ±BackoffJitterFrac% multiplier in [0.8, 1.2].
// Used by callers (dispatcher) implementing 3-attempt backoff (§3.1).
// crypto/rand defense against thundering-herd on shared crash.
func jitter() float64 {
	max := big.NewInt(2 * BackoffJitterFrac)
	n, err := rand.Int(rand.Reader, max)
	if err != nil {
		return 1.0
	}
	return 1.0 + (float64(n.Int64())-BackoffJitterFrac)/100.0
}
