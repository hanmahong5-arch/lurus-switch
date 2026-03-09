package promptlib

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// Store manages prompt persistence
type Store struct {
	dir string
}

// NewStore creates a prompt store, creating the directory if needed
func NewStore() (*Store, error) {
	dir, err := promptsDir()
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create prompts directory: %w", err)
	}
	return &Store{dir: dir}, nil
}

// promptsDir returns the directory where prompts are stored
func promptsDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}

	var base string
	switch runtime.GOOS {
	case "windows":
		appData := os.Getenv("APPDATA")
		if appData == "" {
			appData = filepath.Join(home, "AppData", "Roaming")
		}
		base = filepath.Join(appData, "lurus-switch")
	case "darwin":
		base = filepath.Join(home, "Library", "Application Support", "lurus-switch")
	default:
		base = filepath.Join(home, ".lurus-switch")
	}

	return filepath.Join(base, "prompts"), nil
}

// ListPrompts returns all stored prompts, optionally filtered by category
func (s *Store) ListPrompts(category string) ([]Prompt, error) {
	entries, err := os.ReadDir(s.dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read prompts directory: %w", err)
	}

	var prompts []Prompt
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(s.dir, e.Name()))
		if err != nil {
			continue
		}
		var p Prompt
		if err := json.Unmarshal(data, &p); err != nil {
			continue
		}
		if category != "" && category != "all" && p.Category != category {
			continue
		}
		prompts = append(prompts, p)
	}
	return prompts, nil
}

// GetPrompt returns a single prompt by ID
func (s *Store) GetPrompt(id string) (*Prompt, error) {
	if err := validateID(id); err != nil {
		return nil, err
	}
	data, err := os.ReadFile(filepath.Join(s.dir, id+".json"))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("prompt not found: %s", id)
		}
		return nil, fmt.Errorf("failed to read prompt: %w", err)
	}
	var p Prompt
	if err := json.Unmarshal(data, &p); err != nil {
		return nil, fmt.Errorf("failed to parse prompt: %w", err)
	}
	return &p, nil
}

// SavePrompt persists a prompt; auto-generates ID and timestamps if missing
func (s *Store) SavePrompt(p Prompt) error {
	now := time.Now().Format(time.RFC3339)
	if p.ID == "" {
		p.ID = fmt.Sprintf("prompt-%d", time.Now().UnixMilli())
	}
	if err := validateID(p.ID); err != nil {
		return err
	}
	if p.CreatedAt == "" {
		p.CreatedAt = now
	}
	p.UpdatedAt = now

	data, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal prompt: %w", err)
	}

	if err := os.WriteFile(filepath.Join(s.dir, p.ID+".json"), data, 0644); err != nil {
		return fmt.Errorf("failed to write prompt: %w", err)
	}
	return nil
}

// DeletePrompt removes a prompt by ID
func (s *Store) DeletePrompt(id string) error {
	if err := validateID(id); err != nil {
		return err
	}
	path := filepath.Join(s.dir, id+".json")
	if err := os.Remove(path); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("prompt not found: %s", id)
		}
		return fmt.Errorf("failed to delete prompt: %w", err)
	}
	return nil
}

// ClearAllUser removes all user-created prompt files (builtin prompts are not stored on disk)
func (s *Store) ClearAllUser() (int, error) {
	entries, err := os.ReadDir(s.dir)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, fmt.Errorf("failed to read prompts directory: %w", err)
	}
	count := 0
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		if err := os.Remove(filepath.Join(s.dir, entry.Name())); err == nil {
			count++
		}
	}
	return count, nil
}

// ExportAll returns all prompts serialized as a JSON array string
func (s *Store) ExportAll() (string, error) {
	prompts, err := s.ListPrompts("")
	if err != nil {
		return "", err
	}
	data, err := json.MarshalIndent(prompts, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal prompts: %w", err)
	}
	return string(data), nil
}

// ImportFromJSON parses a JSON array of prompts and saves each one; returns count saved
func (s *Store) ImportFromJSON(data string) (int, error) {
	var prompts []Prompt
	if err := json.Unmarshal([]byte(data), &prompts); err != nil {
		return 0, fmt.Errorf("invalid JSON: %w", err)
	}
	count := 0
	for _, p := range prompts {
		// Reset ID so we don't collide with existing prompts
		p.ID = fmt.Sprintf("import-%d-%d", time.Now().UnixMilli(), count)
		if err := s.SavePrompt(p); err == nil {
			count++
		}
	}
	return count, nil
}

// validateID prevents path traversal
func validateID(id string) error {
	if id == "" {
		return fmt.Errorf("ID must not be empty")
	}
	if strings.ContainsAny(id, `/\`) || strings.Contains(id, "..") {
		return fmt.Errorf("invalid ID: %q", id)
	}
	return nil
}

// GetBuiltinPrompts returns the bundled prompt library
func GetBuiltinPrompts() []Prompt {
	now := time.Now().Format(time.RFC3339)
	return []Prompt{
		{
			ID: "builtin-code-review", Name: "Code Review", Category: "coding",
			Tags: []string{"review", "quality"}, TargetTools: []string{"all"},
			Content: "You are an expert code reviewer. Review the provided code for:\n1. Correctness and potential bugs\n2. Performance issues\n3. Security vulnerabilities\n4. Code style and readability\n5. Design patterns and architecture\n\nProvide specific, actionable feedback with examples.",
			CreatedAt: now, UpdatedAt: now,
		},
		{
			ID: "builtin-unit-test", Name: "Unit Test Generator", Category: "coding",
			Tags: []string{"testing", "tdd"}, TargetTools: []string{"all"},
			Content: "Generate comprehensive unit tests for the provided code. Include:\n- Happy path tests\n- Edge cases and boundary conditions\n- Error handling scenarios\n- Mocking of external dependencies\n\nUse the appropriate testing framework for the language. Follow TDD best practices.",
			CreatedAt: now, UpdatedAt: now,
		},
		{
			ID: "builtin-doc-gen", Name: "Documentation Generator", Category: "coding",
			Tags: []string{"docs", "comments"}, TargetTools: []string{"all"},
			Content: "Generate clear, comprehensive documentation for the provided code. Include:\n- Function/method descriptions\n- Parameter documentation\n- Return value descriptions\n- Usage examples\n- Any important caveats or notes\n\nUse the standard documentation format for the language (JSDoc, GoDoc, etc.).",
			CreatedAt: now, UpdatedAt: now,
		},
		{
			ID: "builtin-refactor", Name: "Refactoring Advisor", Category: "coding",
			Tags: []string{"refactor", "clean-code"}, TargetTools: []string{"all"},
			Content: "Analyze the provided code and suggest refactoring improvements:\n1. Extract repeated logic into functions\n2. Simplify complex conditionals\n3. Improve naming clarity\n4. Apply SOLID principles\n5. Reduce coupling and increase cohesion\n\nExplain the rationale for each suggestion and show before/after examples.",
			CreatedAt: now, UpdatedAt: now,
		},
		{
			ID: "builtin-architecture", Name: "Architecture Analysis", Category: "analysis",
			Tags: []string{"architecture", "design"}, TargetTools: []string{"all"},
			Content: "Analyze the system architecture and provide insights on:\n1. Component relationships and dependencies\n2. Potential bottlenecks and scaling concerns\n3. Security boundaries\n4. Data flow and consistency guarantees\n5. Improvement recommendations\n\nDiagram the architecture if helpful.",
			CreatedAt: now, UpdatedAt: now,
		},
		{
			ID: "builtin-security-scan", Name: "Security Scanner", Category: "coding",
			Tags: []string{"security", "owasp"}, TargetTools: []string{"all"},
			Content: "Perform a security analysis of the provided code focusing on:\n- OWASP Top 10 vulnerabilities\n- Injection attacks (SQL, command, XSS)\n- Authentication and authorization issues\n- Sensitive data exposure\n- Dependency vulnerabilities\n\nRate severity as Critical/High/Medium/Low and provide remediation steps.",
			CreatedAt: now, UpdatedAt: now,
		},
		{
			ID: "builtin-performance", Name: "Performance Optimizer", Category: "coding",
			Tags: []string{"performance", "optimization"}, TargetTools: []string{"all"},
			Content: "Analyze the code for performance issues and suggest optimizations:\n1. Algorithm complexity improvements (O(n) → O(log n))\n2. Memory allocation reductions\n3. Database query optimization\n4. Caching opportunities\n5. Concurrency improvements\n\nProvide benchmarks or estimates where possible.",
			CreatedAt: now, UpdatedAt: now,
		},
		{
			ID: "builtin-git-commit", Name: "Git Commit Message", Category: "coding",
			Tags: []string{"git", "commit"}, TargetTools: []string{"all"},
			Content: "Generate a clear, concise git commit message for the provided changes. Follow the Conventional Commits specification:\n- feat: new feature\n- fix: bug fix\n- docs: documentation\n- refactor: code restructuring\n- test: adding tests\n- chore: maintenance\n\nFormat: <type>(<scope>): <description>\n\nKeep the summary under 72 characters.",
			CreatedAt: now, UpdatedAt: now,
		},
	}
}
