package conversation

import (
	"bufio"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// ForkSidecar lives next to a forked session JSONL as <sid>.lurus.json.
// The Claude CLI ignores extra files in the project directory, so we use
// the sidecar to record parentage without touching the JSONL schema.
type ForkSidecar struct {
	ParentSessionID string    `json:"parentSessionID"`
	ForkPointUUID   string    `json:"forkPointUUID"`
	ForkedAt        time.Time `json:"forkedAt"`
}

// ForkResult is the return value of the public ForkConversation binding.
type ForkResult struct {
	NewSessionID  string `json:"newSessionID"`
	NewPath       string `json:"newPath"`
	ParentPath    string `json:"parentPath"`
	ForkPointUUID string `json:"forkPointUUID"`
	MessagesKept  int    `json:"messagesKept"`
}

// Fork copies `parent` up to (and including) the JSONL line whose `uuid`
// equals forkPointUUID, into a sibling file with a new session ID. The
// new session keeps the same encoded-cwd directory so the Claude CLI's
// `--resume <new-sid>` picks it up out of the box.
func Fork(parent SessionFile, forkPointUUID string) (ForkResult, error) {
	if forkPointUUID == "" {
		return ForkResult{}, fmt.Errorf("fork: forkPointUUID required")
	}
	src, err := os.Open(parent.Path)
	if err != nil {
		return ForkResult{}, fmt.Errorf("fork: open parent: %w", err)
	}
	defer src.Close()

	newSID, err := newSessionID()
	if err != nil {
		return ForkResult{}, err
	}
	newPath := filepath.Join(filepath.Dir(parent.Path), newSID+".jsonl")
	dst, err := os.OpenFile(newPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return ForkResult{}, fmt.Errorf("fork: create child: %w", err)
	}
	dstClosed := false
	closeDst := func() {
		if !dstClosed {
			dst.Close()
			dstClosed = true
		}
	}
	defer closeDst()

	scanner := bufio.NewScanner(src)
	buf := make([]byte, 0, 1024*1024)
	scanner.Buffer(buf, 16*1024*1024)
	w := bufio.NewWriter(dst)

	kept := 0
	matched := false
	for scanner.Scan() {
		line := scanner.Bytes()
		if _, err := w.Write(line); err != nil {
			return ForkResult{}, err
		}
		if err := w.WriteByte('\n'); err != nil {
			return ForkResult{}, err
		}
		kept++
		// Cheap UUID match: parse only the uuid field.
		var probe struct{ UUID string `json:"uuid"` }
		_ = json.Unmarshal(line, &probe)
		if probe.UUID == forkPointUUID {
			matched = true
			break
		}
	}
	if err := w.Flush(); err != nil {
		return ForkResult{}, err
	}
	if err := scanner.Err(); err != nil {
		return ForkResult{}, err
	}
	// Close before any rename / unlink — Windows refuses to remove open files.
	closeDst()
	if !matched {
		// Don't leave a half-formed child behind.
		_ = os.Remove(newPath)
		return ForkResult{}, fmt.Errorf("fork: messageUUID %q not found in parent", forkPointUUID)
	}

	// Sidecar.
	sc := ForkSidecar{
		ParentSessionID: parent.SessionID,
		ForkPointUUID:   forkPointUUID,
		ForkedAt:        time.Now(),
	}
	if err := writeForkSidecar(newPath, sc); err != nil {
		// Best-effort — the fork itself is already on disk and usable
		// without the sidecar. Surface the error so the UI can decide.
		return ForkResult{
			NewSessionID:  newSID,
			NewPath:       newPath,
			ParentPath:    parent.Path,
			ForkPointUUID: forkPointUUID,
			MessagesKept:  kept,
		}, fmt.Errorf("fork: write sidecar: %w", err)
	}

	return ForkResult{
		NewSessionID:  newSID,
		NewPath:       newPath,
		ParentPath:    parent.Path,
		ForkPointUUID: forkPointUUID,
		MessagesKept:  kept,
	}, nil
}

func sidecarPath(jsonlPath string) string {
	dir := filepath.Dir(jsonlPath)
	base := filepath.Base(jsonlPath)
	stem := base
	if ext := filepath.Ext(base); ext != "" {
		stem = base[:len(base)-len(ext)]
	}
	return filepath.Join(dir, stem+".lurus.json")
}

func writeForkSidecar(jsonlPath string, sc ForkSidecar) error {
	data, err := json.MarshalIndent(sc, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(sidecarPath(jsonlPath), data, 0o600)
}

func readForkSidecar(jsonlPath string) (ForkSidecar, error) {
	data, err := os.ReadFile(sidecarPath(jsonlPath))
	if err != nil {
		return ForkSidecar{}, err
	}
	var sc ForkSidecar
	if err := json.Unmarshal(data, &sc); err != nil {
		return ForkSidecar{}, err
	}
	return sc, nil
}

// newSessionID produces a UUIDv4-shape string without pulling in the
// uuid dependency just for this one call site.
func newSessionID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	// Version 4 / variant 1 bits.
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%s-%s-%s-%s-%s",
		hex.EncodeToString(b[0:4]),
		hex.EncodeToString(b[4:6]),
		hex.EncodeToString(b[6:8]),
		hex.EncodeToString(b[8:10]),
		hex.EncodeToString(b[10:16]),
	), nil
}
