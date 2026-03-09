package docmgr

import (
	"fmt"
	"os"
	"path/filepath"
)

// ContextFile represents a tool's context/system prompt file on disk
type ContextFile struct {
	Tool    string `json:"tool"`   // "claude" | "gemini" | "picoclaw" | "nullclaw"
	Scope   string `json:"scope"`  // "global" | "project"
	Path    string `json:"path"`   // absolute path on disk
	Content string `json:"content"`
	Exists  bool   `json:"exists"`
}

// Manager provides read/write access to tool context files
type Manager struct{}

// NewManager creates a new document manager
func NewManager() *Manager {
	return &Manager{}
}

// GetContextFile reads (or templates) the context file for a tool/scope
func (m *Manager) GetContextFile(tool, scope, projectDir string) (*ContextFile, error) {
	p, err := resolveContextPath(tool, scope, projectDir)
	if err != nil {
		return nil, err
	}

	cf := &ContextFile{Tool: tool, Scope: scope, Path: p}

	data, err := os.ReadFile(p)
	if err != nil {
		if os.IsNotExist(err) {
			cf.Exists = false
			cf.Content = defaultContextContent(tool)
			return cf, nil
		}
		return nil, fmt.Errorf("failed to read context file %s: %w", p, err)
	}

	cf.Exists = true
	cf.Content = string(data)
	return cf, nil
}

// SaveContextFile writes content to a context file, creating parent directories as needed
func (m *Manager) SaveContextFile(f *ContextFile) error {
	if f == nil {
		return fmt.Errorf("context file must not be nil")
	}
	if f.Path == "" {
		return fmt.Errorf("context file path must not be empty")
	}

	if err := os.MkdirAll(filepath.Dir(f.Path), 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	if err := os.WriteFile(f.Path, []byte(f.Content), 0644); err != nil {
		return fmt.Errorf("failed to write context file: %w", err)
	}
	return nil
}

// ScanProjectDir finds all context files inside a project directory
func (m *Manager) ScanProjectDir(dir string) ([]ContextFile, error) {
	if dir == "" {
		return nil, fmt.Errorf("project directory must not be empty")
	}

	candidates := []struct{ rel, tool string }{
		{"CLAUDE.md", "claude"},
		{".gemini/GEMINI.md", "gemini"},
		{".picoclaw/SYSTEM.md", "picoclaw"},
		{".nullclaw/SYSTEM.md", "nullclaw"},
	}

	var files []ContextFile
	for _, c := range candidates {
		p := filepath.Join(dir, c.rel)
		data, err := os.ReadFile(p)
		if err != nil {
			if os.IsNotExist(err) {
				files = append(files, ContextFile{
					Tool: c.tool, Scope: "project", Path: p, Exists: false,
					Content: defaultContextContent(c.tool),
				})
				continue
			}
			continue
		}
		files = append(files, ContextFile{
			Tool: c.tool, Scope: "project", Path: p, Exists: true, Content: string(data),
		})
	}
	return files, nil
}

// resolveContextPath maps (tool, scope, projectDir) to a filesystem path
func resolveContextPath(tool, scope, projectDir string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}

	switch tool {
	case "claude":
		if scope == "global" {
			return filepath.Join(home, "CLAUDE.md"), nil
		}
		if projectDir == "" {
			return "", fmt.Errorf("projectDir required for project scope")
		}
		return filepath.Join(projectDir, "CLAUDE.md"), nil

	case "gemini":
		if scope == "global" {
			return filepath.Join(home, ".gemini", "GEMINI.md"), nil
		}
		if projectDir == "" {
			return "", fmt.Errorf("projectDir required for project scope")
		}
		return filepath.Join(projectDir, ".gemini", "GEMINI.md"), nil

	case "picoclaw":
		return filepath.Join(home, ".picoclaw", "SYSTEM.md"), nil

	case "nullclaw":
		return filepath.Join(home, ".nullclaw", "SYSTEM.md"), nil

	default:
		return "", fmt.Errorf("unknown tool: %s", tool)
	}
}

// defaultContextContent returns a starter template for tools that don't have a context file yet
func defaultContextContent(tool string) string {
	switch tool {
	case "claude":
		return "# Project Context\n\nDescribe your project, coding standards, and any relevant context here.\n"
	case "gemini":
		return "# Gemini Context\n\nDescribe your project context for Gemini CLI here.\n"
	case "picoclaw":
		return "# PicoClaw System Prompt\n\nDefine the system-level instructions for PicoClaw here.\n"
	case "nullclaw":
		return "# NullClaw System Prompt\n\nDefine the system-level instructions for NullClaw here.\n"
	default:
		return ""
	}
}
