//go:build !windows

// Non-Windows single-instance IPC using a Unix domain socket + lock file.
//
// Socket path: <dataDir>/deeplink.sock
// Lock file:   <dataDir>/deeplink.lock
//
// The socket file has mode 0600, restricting access to the owning user.
// If the socket exists but no process owns the lock file, the stale socket
// is removed before binding.

package deeplink

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

const (
	dialTimeout = 3 * time.Second
	maxURLLen   = 8192
)

// ErrAlreadyRunning signals that another Switch process holds the IPC channel.
var ErrAlreadyRunning = errors.New("deeplink: another instance is running")

func socketPath(dataDir string) string {
	return filepath.Join(dataDir, "deeplink.sock")
}

func lockPath(dataDir string) string {
	return filepath.Join(dataDir, "deeplink.lock")
}

// Server is the single-instance IPC listener.
type Server struct {
	listener net.Listener
	lockFile *os.File
	dataDir  string
}

// NewServer creates the Unix socket listener.
// Returns ErrAlreadyRunning if a live instance already holds the socket.
func NewServer(dataDir string) (*Server, error) {
	if err := os.MkdirAll(dataDir, 0700); err != nil {
		return nil, fmt.Errorf("deeplink: mkdir dataDir: %w", err)
	}

	lp := lockPath(dataDir)

	// Try to obtain an exclusive advisory lock on the lock file.
	lf, err := os.OpenFile(lp, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return nil, fmt.Errorf("deeplink: open lock file: %w", err)
	}

	if err := syscall.Flock(int(lf.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		lf.Close()
		return nil, ErrAlreadyRunning
	}

	// We own the lock; remove a stale socket if present.
	sp := socketPath(dataDir)
	os.Remove(sp) //nolint:errcheck

	ln, err := net.Listen("unix", sp)
	if err != nil {
		syscall.Flock(int(lf.Fd()), syscall.LOCK_UN) //nolint:errcheck
		lf.Close()
		return nil, fmt.Errorf("deeplink: listen unix socket: %w", err)
	}

	// Restrict socket to owner only.
	os.Chmod(sp, 0600) //nolint:errcheck

	return &Server{listener: ln, lockFile: lf, dataDir: dataDir}, nil
}

// Start begins accepting forwarded URLs from secondary instances.
func (s *Server) Start(ctx context.Context, onPayload func(*Payload)) {
	go func() {
		for {
			conn, err := s.listener.Accept()
			if err != nil {
				return
			}
			go handleConn(conn, onPayload)
		}
	}()

	go func() {
		<-ctx.Done()
		s.Stop() //nolint:errcheck
	}()
}

// Stop closes the listener and releases the lock.
func (s *Server) Stop() error {
	errs := []error{s.listener.Close()}
	if s.lockFile != nil {
		syscall.Flock(int(s.lockFile.Fd()), syscall.LOCK_UN) //nolint:errcheck
		errs = append(errs, s.lockFile.Close())
		os.Remove(lockPath(s.dataDir))  //nolint:errcheck
		os.Remove(socketPath(s.dataDir)) //nolint:errcheck
	}
	for _, e := range errs {
		if e != nil {
			return e
		}
	}
	return nil
}

// SendToExisting dials the running instance socket and writes rawURL.
func SendToExisting(dataDir, rawURL string) error {
	if len(rawURL) > maxURLLen {
		return fmt.Errorf("deeplink: URL too long (%d bytes)", len(rawURL))
	}

	conn, err := net.DialTimeout("unix", socketPath(dataDir), dialTimeout)
	if err != nil {
		return fmt.Errorf("deeplink: dial existing instance: %w", err)
	}
	defer conn.Close()

	_, err = io.WriteString(conn, rawURL)
	return err
}

// handleConn reads a URL from one connection and dispatches it.
func handleConn(conn net.Conn, onPayload func(*Payload)) {
	defer conn.Close()
	conn.SetDeadline(time.Now().Add(5 * time.Second)) //nolint:errcheck

	buf := make([]byte, maxURLLen)
	n, err := conn.Read(buf)
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
