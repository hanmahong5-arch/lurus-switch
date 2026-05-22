package livesession

import (
	"bufio"
	"io"
	"os"

	"lurus-switch/internal/conversation"
)

// readNew opens path, seeks to fromOffset, and returns every parsed event
// from there to EOF plus the new offset. A new session's first call
// passes fromOffset=0; subsequent polls reuse the returned offset to
// avoid re-parsing the same prefix.
//
// Lines that span the polling boundary (a partial line at the tail of the
// file because Claude was mid-write when we read) are detected and left
// in the offset — we re-read them on the next tick when they're complete.
// Without that guard a half-written assistant message would be silently
// dropped or, worse, treated as garbage.
func readNew(path string, fromOffset int64) ([]conversation.Event, int64, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fromOffset, err
	}
	defer f.Close()

	if fromOffset > 0 {
		if _, err := f.Seek(fromOffset, io.SeekStart); err != nil {
			return nil, fromOffset, err
		}
	}

	// Find total size so we know when we're in the trailing partial line.
	info, err := f.Stat()
	if err != nil {
		return nil, fromOffset, err
	}
	totalSize := info.Size()

	scanner := bufio.NewScanner(f)
	// Same buffer cap as conversation.Parse — tool args can be huge.
	buf := make([]byte, 0, 1024*1024)
	scanner.Buffer(buf, 16*1024*1024)

	// We need to track bytes-consumed-so-far including the newline. Scanner
	// strips the trailing newline from each token, so we add 1 per token
	// to match what we'd Seek past.
	consumed := fromOffset
	events := make([]conversation.Event, 0, 8)

	for scanner.Scan() {
		line := scanner.Bytes()
		consumed += int64(len(line)) + 1 // +1 for the '\n' Scanner ate
		// If the final "line" ran to EOF without a newline, it's a partial
		// write — don't consume it, let the next tick replay.
		if consumed > totalSize {
			break
		}
		if len(line) == 0 {
			continue
		}
		// Borrow conversation's permissive line parser so we agree on the
		// drift between Claude/Codex shapes for free.
		if ev, ok := parseConversationLine(line); ok {
			events = append(events, ev)
		}
	}
	if err := scanner.Err(); err != nil {
		return events, consumed, err
	}
	// Cap consumed at totalSize — a clean EOF with a final newline lands
	// exactly here; without one, the partial-line guard above kept us
	// short of totalSize, which is what we want.
	if consumed > totalSize {
		consumed = totalSize
	}
	return events, consumed, nil
}

// parseConversationLine is a thin wrapper that calls into conversation's
// (unexported) parseLine via the public Parse(io.Reader). We feed a
// 1-line buffer and pick the single result — cheaper than exporting a
// per-line API.
func parseConversationLine(line []byte) (conversation.Event, bool) {
	// Parse expects a reader; build one from the line plus a newline so
	// Scanner treats it as one record.
	r := &singleLineReader{buf: append(line, '\n')}
	out, err := conversation.Parse(r)
	if err != nil || len(out) == 0 {
		return conversation.Event{}, false
	}
	return out[0], true
}

// singleLineReader is a minimal io.Reader that yields a single byte slice
// and then EOF. Avoids pulling in bytes.NewReader just to call .Read once.
type singleLineReader struct {
	buf  []byte
	done bool
}

func (s *singleLineReader) Read(p []byte) (int, error) {
	if s.done {
		return 0, io.EOF
	}
	n := copy(p, s.buf)
	s.buf = s.buf[n:]
	if len(s.buf) == 0 {
		s.done = true
	}
	return n, nil
}
