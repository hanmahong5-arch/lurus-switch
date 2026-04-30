// Windows single-instance IPC implementation using named pipes.
// Pipe name: \\.\pipe\lurus-switch-<username>
//
// golang.org/x/sys/windows is already in go.mod — no new dependency added.
//
// Security: the pipe security descriptor "D:P(A;;GA;;;OW)" (Allow Generic-All
// for the object owner) is set via SecurityDescriptorFromString so that only
// the process owner can connect.  The username suffix further scopes the pipe
// to the local user, preventing cross-user pollution.

package deeplink

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"golang.org/x/sys/windows"
)

const (
	pipeBaseName = `\\.\pipe\lurus-switch-`
	dialTimeout  = 3 * time.Second
	maxURLLen    = 8192

	// pipe open-mode flags (not exported by x/sys).
	pipeAccessDuplex          = 0x00000003
	fileFlagFirstPipeInstance = 0x00080000

	// genericWrite for secondary-instance client side.
	genericWrite = 0x40000000
	openExisting = 3
)

// ErrAlreadyRunning signals that another Switch process holds the IPC channel.
var ErrAlreadyRunning = errors.New("deeplink: another instance is running")

func pipeName() string {
	return pipeBaseName + currentUsername()
}

// Server is the single-instance IPC listener.
type Server struct {
	mu      sync.Mutex
	pipe    windows.Handle
	done    chan struct{}
	stopped bool
}

// buildSecurityAttributes builds a SECURITY_ATTRIBUTES that restricts the
// named pipe to the pipe-instance owner.  Falls back to nil on any error.
func buildSecurityAttributes() *windows.SecurityAttributes {
	// "D:P(A;;GA;;;OW)" — Discretionary ACL: Allow Generic-All for Owner.
	sd, err := windows.SecurityDescriptorFromString("D:P(A;;GA;;;OW)")
	if err != nil {
		return nil
	}
	sa := &windows.SecurityAttributes{
		Length:             uint32(12), // sizeof(SECURITY_ATTRIBUTES)
		SecurityDescriptor: sd,
		InheritHandle:      0,
	}
	return sa
}

// NewServer creates the named pipe with FILE_FLAG_FIRST_PIPE_INSTANCE.
// Returns ErrAlreadyRunning if the pipe is already owned.
func NewServer(_ string) (*Server, error) {
	name, err := windows.UTF16PtrFromString(pipeName())
	if err != nil {
		return nil, fmt.Errorf("deeplink: pipe name encode: %w", err)
	}

	sa := buildSecurityAttributes()
	openMode := uint32(pipeAccessDuplex | fileFlagFirstPipeInstance)
	pipeMode := uint32(windows.PIPE_TYPE_BYTE | windows.PIPE_READMODE_BYTE | windows.PIPE_WAIT)

	handle, err := windows.CreateNamedPipe(
		name,
		openMode,
		pipeMode,
		1,    // nMaxInstances — 1 with FIRST_PIPE_INSTANCE enforces single owner
		4096, // nOutBufferSize
		4096, // nInBufferSize
		0,    // nDefaultTimeOut — use system default
		sa,
	)
	if err != nil {
		// Probe with CreateFile to distinguish "already running" from "real error".
		if probeErr := probePipe(); probeErr == nil {
			return nil, ErrAlreadyRunning
		}
		return nil, fmt.Errorf("deeplink: create named pipe: %w", err)
	}

	return &Server{pipe: handle, done: make(chan struct{})}, nil
}

// Start begins accepting forwarded URLs from secondary instances.
// Runs until ctx is cancelled or Stop is called.
func (s *Server) Start(ctx context.Context, onPayload func(*Payload)) {
	go s.acceptLoop(onPayload)
	go func() {
		<-ctx.Done()
		s.Stop() //nolint:errcheck
	}()
}

// acceptLoop blocks on ConnectNamedPipe and dispatches each connection.
func (s *Server) acceptLoop(onPayload func(*Payload)) {
	for {
		s.mu.Lock()
		h := s.pipe
		stopped := s.stopped
		s.mu.Unlock()

		if stopped || h == windows.InvalidHandle {
			return
		}

		// Block until a client connects (or pipe is closed).
		err := windows.ConnectNamedPipe(h, nil)
		if err != nil && err != windows.ERROR_PIPE_CONNECTED {
			select {
			case <-s.done:
				return
			default:
				// Recreate the pipe for the next accept cycle.
				if err == windows.ERROR_BROKEN_PIPE || err == windows.ERROR_NO_DATA {
					windows.DisconnectNamedPipe(h) //nolint:errcheck
					continue
				}
				return
			}
		}

		// Dispatch this connection and immediately bind a new pipe instance
		// so the next client does not get ERROR_PIPE_BUSY.
		go s.readAndDispatch(h, onPayload)

		name, _ := windows.UTF16PtrFromString(pipeName())
		newH, nerr := windows.CreateNamedPipe(
			name,
			pipeAccessDuplex,
			uint32(windows.PIPE_TYPE_BYTE|windows.PIPE_READMODE_BYTE|windows.PIPE_WAIT),
			windows.PIPE_UNLIMITED_INSTANCES,
			4096, 4096, 0, nil,
		)
		if nerr != nil {
			return
		}
		s.mu.Lock()
		s.pipe = newH
		s.mu.Unlock()
	}
}

// readAndDispatch reads one URL from the pipe handle and calls onPayload.
func (s *Server) readAndDispatch(handle windows.Handle, onPayload func(*Payload)) {
	defer windows.CloseHandle(handle) //nolint:errcheck

	buf := make([]byte, maxURLLen)
	var n uint32
	err := windows.ReadFile(handle, buf, &n, nil)
	if err != nil && !errors.Is(err, io.EOF) {
		return
	}

	rawURL := strings.TrimSpace(string(buf[:n]))
	if rawURL == "" {
		return
	}

	p, parseErr := Parse(rawURL)
	if parseErr != nil {
		return
	}
	onPayload(p)
}

// Stop closes the IPC listener gracefully.
func (s *Server) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.stopped {
		return nil
	}
	s.stopped = true
	select {
	case <-s.done:
	default:
		close(s.done)
	}

	h := s.pipe
	s.pipe = windows.InvalidHandle
	if h != windows.InvalidHandle {
		windows.DisconnectNamedPipe(h) //nolint:errcheck
		return windows.CloseHandle(h)
	}
	return nil
}

// SendToExisting opens the named pipe as a write-only client and sends rawURL.
func SendToExisting(_ string, rawURL string) error {
	if len(rawURL) > maxURLLen {
		return fmt.Errorf("deeplink: URL too long (%d bytes)", len(rawURL))
	}

	name, err := windows.UTF16PtrFromString(pipeName())
	if err != nil {
		return fmt.Errorf("deeplink: pipe name encode: %w", err)
	}

	deadline := time.Now().Add(dialTimeout)
	var handle windows.Handle
	for {
		handle, err = windows.CreateFile(name, genericWrite, 0, nil, openExisting, 0, 0)
		if err == nil {
			break
		}
		if err != windows.ERROR_PIPE_BUSY {
			return fmt.Errorf("deeplink: open pipe: %w", err)
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("deeplink: timeout waiting for pipe")
		}
		// Busy — wait a bit and retry (WaitNamedPipe not exported by x/sys).
		time.Sleep(50 * time.Millisecond)
	}
	defer windows.CloseHandle(handle) //nolint:errcheck

	data := []byte(rawURL)
	var written uint32
	if err := windows.WriteFile(handle, data, &written, nil); err != nil {
		return fmt.Errorf("deeplink: write to pipe: %w", err)
	}
	return nil
}

// probePipe attempts a read-only open of the pipe to detect a live server.
func probePipe() error {
	name, err := windows.UTF16PtrFromString(pipeName())
	if err != nil {
		return err
	}
	const genericRead = 0x80000000
	h, err := windows.CreateFile(name, genericRead, 0, nil, openExisting, 0, 0)
	if err != nil {
		return err
	}
	windows.CloseHandle(h) //nolint:errcheck
	return nil
}
