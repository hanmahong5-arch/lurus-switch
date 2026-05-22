package conversation

import (
	"os"
	"path/filepath"
)

// ContextFile is a project-level instruction file discovered next to a
// session's working directory. CLAUDE.md / AGENTS.md / .cursorrules are
// the three most common patterns; we ship as a read-only viewer in v1.
type ContextFile struct {
	Path     string `json:"path"`
	Name     string `json:"name"` // CLAUDE.md, AGENTS.md, etc.
	Content  string `json:"content"`
	Size     int64  `json:"size"`
	Truncated bool  `json:"truncated"`
}

const maxContextFileSize = 256 * 1024 // 256 KB cap, viewer-friendly

var contextFileNames = []string{
	"CLAUDE.md",
	"AGENTS.md",
	".cursorrules",
	"GEMINI.md",
	".clauderules",
}

// FindContextFiles enumerates known context files under cwd. Missing
// files are skipped; the function never errors so the UI degrades to
// "no context" rather than blocking.
func FindContextFiles(cwd string) []ContextFile {
	if cwd == "" {
		return nil
	}
	if _, err := os.Stat(cwd); err != nil {
		return nil
	}
	var out []ContextFile
	for _, name := range contextFileNames {
		path := filepath.Join(cwd, name)
		info, err := os.Stat(path)
		if err != nil {
			continue
		}
		cf := ContextFile{Path: path, Name: name, Size: info.Size()}
		readSize := info.Size()
		if readSize > maxContextFileSize {
			readSize = maxContextFileSize
			cf.Truncated = true
		}
		data := make([]byte, readSize)
		f, err := os.Open(path)
		if err != nil {
			continue
		}
		n, _ := f.Read(data)
		f.Close()
		cf.Content = string(data[:n])
		out = append(out, cf)
	}
	return out
}
