// Package conversation discovers, parses, and indexes the JSONL session
// files that AI CLIs (Claude / Codex / Gemini) accumulate on disk, then
// joins them against Switch's existing audit + DLP rings so an admin can
// trace any DLP hit back to the exact message in the exact session.
//
// JSONL schemas drift between CLI versions, so the parser is permissive:
// unknown keys are preserved on Event.Raw, never error out. The index is
// rebuilt on mtime change and stored beside the rest of Switch's local
// state under appDataBaseDir().
package conversation

import (
	"os"
	"path/filepath"
	"regexp"
	goruntime "runtime"
	"strings"
)

// Tool identifiers used throughout the package. Matches the values used by
// the existing tool installer / runtime probes so frontend filters can
// reuse the same enum.
const (
	ToolClaude = "claude"
	ToolCodex  = "codex"
	ToolGemini = "gemini"
)

// SessionFile is a single JSONL transcript on disk.
type SessionFile struct {
	Tool      string // "claude" | "codex" | "gemini"
	SessionID string // file stem (e.g. UUID for claude, free-form for others)
	Cwd       string // working directory the session was run in (best-effort)
	Path      string // absolute path to the JSONL
	Size      int64
	ModTime   int64 // UnixNano; used by the index for mtime-based rebuild
}

// userHomeDir returns the platform's user home, mirroring the conventions
// the rest of Switch uses for tool-specific config lookup.
func userHomeDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return home
}

// DiscoverAll walks every supported tool's session directory and returns
// the union of files found. Errors from any single tool are non-fatal —
// each tool short-circuits to an empty slice on missing directories.
func DiscoverAll() []SessionFile {
	var out []SessionFile
	out = append(out, discoverClaude()...)
	out = append(out, discoverCodex()...)
	out = append(out, discoverGemini()...)
	return out
}

// claudeEncodedCwdRe matches the encoded-cwd pattern Claude Code writes:
// every non-alphanumeric character in the cwd is replaced with '-'.
//
// Example: cwd `C:\Users\Anita\Desktop\lurus\2c-gui-switch`
//          dir `C--Users-Anita-Desktop-lurus-2c-gui-switch`
var claudeEncodedCwdRe = regexp.MustCompile(`[^A-Za-z0-9]`)

func claudeProjectsDir() string {
	h := userHomeDir()
	if h == "" {
		return ""
	}
	return filepath.Join(h, ".claude", "projects")
}

// decodeClaudeCwd is the inverse of Claude's cwd encoding — but the encoding
// is lossy (multiple cwds can map to one folder), so we just turn the
// dashes back into a forward-slash path which is good enough for display.
func decodeClaudeCwd(encoded string) string {
	if encoded == "" {
		return ""
	}
	if goruntime.GOOS == "windows" {
		// Pattern "C--Users-Anita-..." → "C:/Users/Anita/..."
		if len(encoded) >= 3 && encoded[1] == '-' && encoded[2] == '-' {
			tail := strings.ReplaceAll(encoded[3:], "-", "/")
			return string(encoded[0]) + ":/" + tail
		}
	}
	return "/" + strings.ReplaceAll(encoded, "-", "/")
}

func discoverClaude() []SessionFile {
	root := claudeProjectsDir()
	if root == "" {
		return nil
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil
	}
	var out []SessionFile
	for _, projDir := range entries {
		if !projDir.IsDir() {
			continue
		}
		projPath := filepath.Join(root, projDir.Name())
		cwd := decodeClaudeCwd(projDir.Name())
		files, err := os.ReadDir(projPath)
		if err != nil {
			continue
		}
		for _, f := range files {
			if f.IsDir() || !strings.HasSuffix(f.Name(), ".jsonl") {
				continue
			}
			info, err := f.Info()
			if err != nil {
				continue
			}
			sid := strings.TrimSuffix(f.Name(), ".jsonl")
			out = append(out, SessionFile{
				Tool:      ToolClaude,
				SessionID: sid,
				Cwd:       cwd,
				Path:      filepath.Join(projPath, f.Name()),
				Size:      info.Size(),
				ModTime:   info.ModTime().UnixNano(),
			})
		}
	}
	return out
}

func discoverCodex() []SessionFile {
	h := userHomeDir()
	if h == "" {
		return nil
	}
	// Codex writes sessions under ~/.codex/sessions/<YYYY>/<MM>/<DD>/<sid>.jsonl
	// We walk shallowly to keep latency bounded — large session histories
	// can still surface, just no deeper than the daily folder.
	root := filepath.Join(h, ".codex", "sessions")
	return walkJsonlTree(ToolCodex, root, 4)
}

func discoverGemini() []SessionFile {
	h := userHomeDir()
	if h == "" {
		return nil
	}
	// Gemini stores sessions either flat under ~/.gemini/sessions/ or
	// nested similar to Codex; we try the flat layout first, then a
	// shallow tree walk.
	root := filepath.Join(h, ".gemini", "sessions")
	return walkJsonlTree(ToolGemini, root, 3)
}

// walkJsonlTree walks `root` up to maxDepth and returns every .jsonl file
// found. Missing root returns nil — calling code treats absence as "tool
// not used", which is the right behavior.
func walkJsonlTree(tool, root string, maxDepth int) []SessionFile {
	if _, err := os.Stat(root); err != nil {
		return nil
	}
	var out []SessionFile
	rootDepth := strings.Count(root, string(filepath.Separator))
	_ = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil || info == nil {
			return nil
		}
		depth := strings.Count(path, string(filepath.Separator)) - rootDepth
		if depth > maxDepth {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if info.IsDir() || !strings.HasSuffix(info.Name(), ".jsonl") {
			return nil
		}
		sid := strings.TrimSuffix(info.Name(), ".jsonl")
		out = append(out, SessionFile{
			Tool:      tool,
			SessionID: sid,
			Path:      path,
			Size:      info.Size(),
			ModTime:   info.ModTime().UnixNano(),
		})
		return nil
	})
	return out
}
