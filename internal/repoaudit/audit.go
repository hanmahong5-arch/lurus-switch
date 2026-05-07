// Package repoaudit scans a project directory for AI-CLI config overrides
// that could be malicious — chiefly the CVE-2026-21852 family where a
// cloned repo's .claude/settings.json silently redirects ANTHROPIC_BASE_URL
// to an attacker-controlled host the moment Claude Code is launched there.
//
// Audit() returns a structured report; the GUI surfaces findings to the
// user with severity badges and lets them quarantine or delete suspect
// files before launching the CLI.
package repoaudit

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Severity string

const (
	SeverityInfo    Severity = "info"
	SeverityCaution Severity = "caution"
	SeverityRisky   Severity = "risky"
)

type Verdict string

const (
	VerdictSafe    Verdict = "safe"
	VerdictCaution Verdict = "caution"
	VerdictRisky   Verdict = "risky"
)

// Finding describes one suspect entry inside a config file. One audited
// file may produce many findings (e.g. base URL override AND a hardcoded
// apiKey AND an MCP server).
type Finding struct {
	File            string   `json:"file"`            // path relative to scanned root
	FullPath        string   `json:"fullPath"`        // absolute path on disk
	Tool            string   `json:"tool"`            // claude | codex | gemini | mcp | other
	Field           string   `json:"field"`           // dotted JSON path of the offending key
	Severity        Severity `json:"severity"`        // info | caution | risky
	IssueZh         string   `json:"issueZh"`         // human-friendly headline
	IssueEn         string   `json:"issueEn"`         //   "
	DetailValue     string   `json:"detailValue"`     // truncated raw value
	SuggestedAction string   `json:"suggestedAction"` // review | quarantine | delete
}

type AuditReport struct {
	Path        string    `json:"path"`        // root that was scanned
	ScannedAt   time.Time `json:"scannedAt"`
	Findings    []Finding `json:"findings"`
	FilesFound  []string  `json:"filesFound"`  // every config file we inspected (relative paths)
	Verdict     Verdict   `json:"verdict"`
}

// Audit walks `root` looking for known AI-CLI config files and flags
// fields that could be used to exfiltrate API keys, redirect traffic, or
// run arbitrary commands via MCP. It does NOT modify any files — call
// Quarantine() separately to act on findings.
func Audit(root string) (*AuditReport, error) {
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("resolve path: %w", err)
	}
	stat, err := os.Stat(abs)
	if err != nil {
		return nil, fmt.Errorf("stat path: %w", err)
	}
	if !stat.IsDir() {
		return nil, fmt.Errorf("not a directory: %s", abs)
	}

	report := &AuditReport{
		Path:      abs,
		ScannedAt: time.Now(),
	}

	// Each known config location → its inspector. Inspectors append both
	// to FilesFound (when the file exists at all) and to Findings (when
	// they spot something suspicious).
	checks := []struct {
		rel  string
		tool string
		fn   func(*AuditReport, string, string, string)
	}{
		{".claude/settings.json", "claude", inspectClaudeSettings},
		{".codex/config.toml", "codex", inspectCodexConfig},
		{".gemini/settings.json", "gemini", inspectGeminiSettings},
		{".picoclaw/config.json", "picoclaw", inspectGenericJSON},
		{".nullclaw/config.json", "nullclaw", inspectGenericJSON},
		{"CLAUDE.md", "claude", inspectMarkdownContext},
		{"AGENTS.md", "codex", inspectMarkdownContext},
		{".cursorrules", "cursor", inspectMarkdownContext},
	}
	for _, c := range checks {
		full := filepath.Join(abs, c.rel)
		if _, err := os.Stat(full); err == nil {
			report.FilesFound = append(report.FilesFound, c.rel)
			c.fn(report, full, c.rel, c.tool)
		}
	}

	report.Verdict = computeVerdict(report.Findings)
	return report, nil
}

func computeVerdict(findings []Finding) Verdict {
	hasRisky := false
	hasCaution := false
	for _, f := range findings {
		if f.Severity == SeverityRisky {
			hasRisky = true
		} else if f.Severity == SeverityCaution {
			hasCaution = true
		}
	}
	switch {
	case hasRisky:
		return VerdictRisky
	case hasCaution:
		return VerdictCaution
	default:
		return VerdictSafe
	}
}

// truncate keeps long values short in the report to avoid bloating IPC
// payloads when settings.json embeds e.g. multi-KB customInstructions.
func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

// ─── Inspectors ─────────────────────────────────────────────────────

func inspectClaudeSettings(r *AuditReport, full, rel, tool string) {
	data, err := os.ReadFile(full)
	if err != nil {
		return
	}
	var raw map[string]interface{}
	if json.Unmarshal(data, &raw) != nil {
		return
	}

	// Risky: base URL overrides — the canonical CVE-2026-21852 vector.
	for _, key := range []string{"apiBaseUrl", "apiBaseURL", "anthropic_base_url", "anthropicBaseURL"} {
		if v, ok := raw[key]; ok {
			r.Findings = append(r.Findings, Finding{
				File: rel, FullPath: full, Tool: tool, Field: key,
				Severity:        SeverityRisky,
				IssueZh:         "API 端点被覆盖到自定义地址",
				IssueEn:         "API base URL overridden to a custom endpoint",
				DetailValue:     truncate(fmt.Sprint(v), 120),
				SuggestedAction: "quarantine",
			})
		}
	}

	// Risky: hardcoded API key in repo-level settings (would leak via git).
	if v, ok := raw["apiKey"]; ok {
		if s, _ := v.(string); s != "" {
			// Never echo any portion of a secret — the report can be
			// pasted into chat/email when reporting an incident, and
			// even partial leakage of an API key is a credential leak.
			r.Findings = append(r.Findings, Finding{
				File: rel, FullPath: full, Tool: tool, Field: "apiKey",
				Severity:        SeverityRisky,
				IssueZh:         "仓库级配置中硬编码了 API Key",
				IssueEn:         "Hardcoded API key in repo-scoped config",
				DetailValue:     "(redacted — secret never echoed)",
				SuggestedAction: "delete",
			})
		}
	}

	// Caution: per-repo MCP servers — each is an exec vector.
	if mcp, ok := raw["mcpServers"].(map[string]interface{}); ok {
		for name, cfg := range mcp {
			detail := name
			if c, _ := cfg.(map[string]interface{}); c != nil {
				if cmd, _ := c["command"].(string); cmd != "" {
					detail = fmt.Sprintf("%s (cmd=%s)", name, cmd)
				}
			}
			r.Findings = append(r.Findings, Finding{
				File: rel, FullPath: full, Tool: "mcp", Field: "mcpServers." + name,
				Severity:        SeverityCaution,
				IssueZh:         "仓库携带 MCP 服务器（可执行任意命令）",
				IssueEn:         "Repo ships an MCP server (runs arbitrary commands)",
				DetailValue:     truncate(detail, 120),
				SuggestedAction: "review",
			})
		}
	}

	// Info: customInstructions can contain prompt-injection payloads.
	if v, ok := raw["customInstructions"]; ok {
		if s, _ := v.(string); len(s) > 0 {
			r.Findings = append(r.Findings, Finding{
				File: rel, FullPath: full, Tool: tool, Field: "customInstructions",
				Severity:        SeverityInfo,
				IssueZh:         "仓库注入了自定义系统指令（可能含 prompt 注入）",
				IssueEn:         "Repo injects custom system instructions (potential prompt injection)",
				DetailValue:     truncate(s, 120),
				SuggestedAction: "review",
			})
		}
	}
}

func inspectCodexConfig(r *AuditReport, full, rel, tool string) {
	data, err := os.ReadFile(full)
	if err != nil {
		return
	}
	text := string(data)
	// TOML parsing without a dependency: just regex-scan for the
	// suspect lines. False positives are OK at this stage — the user
	// reviews findings before acting.
	if strings.Contains(text, "base_url") || strings.Contains(text, "api_base_url") {
		r.Findings = append(r.Findings, Finding{
			File: rel, FullPath: full, Tool: tool, Field: "provider.base_url",
			Severity:        SeverityRisky,
			IssueZh:         "仓库级 TOML 覆盖了 provider base_url",
			IssueEn:         "Repo-level TOML overrides provider base_url",
			DetailValue:     extractTomlValue(text, "base_url"),
			SuggestedAction: "quarantine",
		})
	}
	if strings.Contains(text, "api_key") {
		r.Findings = append(r.Findings, Finding{
			File: rel, FullPath: full, Tool: tool, Field: "provider.api_key",
			Severity:        SeverityRisky,
			IssueZh:         "仓库级 TOML 硬编码了 api_key",
			IssueEn:         "Repo-level TOML hardcodes api_key",
			DetailValue:     "(redacted)",
			SuggestedAction: "delete",
		})
	}
}

func inspectGeminiSettings(r *AuditReport, full, rel, tool string) {
	data, err := os.ReadFile(full)
	if err != nil {
		return
	}
	var raw map[string]interface{}
	if json.Unmarshal(data, &raw) != nil {
		return
	}
	if adv, ok := raw["advanced"].(map[string]interface{}); ok {
		if v, ok := adv["apiEndpoint"]; ok {
			r.Findings = append(r.Findings, Finding{
				File: rel, FullPath: full, Tool: tool, Field: "advanced.apiEndpoint",
				Severity:        SeverityRisky,
				IssueZh:         "仓库覆盖了 Gemini API 端点",
				IssueEn:         "Repo overrides Gemini API endpoint",
				DetailValue:     truncate(fmt.Sprint(v), 120),
				SuggestedAction: "quarantine",
			})
		}
	}
	if auth, ok := raw["auth"].(map[string]interface{}); ok {
		if v, ok := auth["serviceAccountPath"]; ok {
			r.Findings = append(r.Findings, Finding{
				File: rel, FullPath: full, Tool: tool, Field: "auth.serviceAccountPath",
				Severity:        SeverityCaution,
				IssueZh:         "仓库指向了 service account JSON（凭据文件路径）",
				IssueEn:         "Repo points at a service-account JSON path",
				DetailValue:     truncate(fmt.Sprint(v), 120),
				SuggestedAction: "review",
			})
		}
	}
}

func inspectGenericJSON(r *AuditReport, full, rel, tool string) {
	data, err := os.ReadFile(full)
	if err != nil {
		return
	}
	var raw map[string]interface{}
	if json.Unmarshal(data, &raw) != nil {
		return
	}
	for _, key := range []string{"apiKey", "api_key", "apiEndpoint", "api_endpoint", "baseUrl", "base_url"} {
		if v, ok := raw[key]; ok {
			sev := SeverityCaution
			if strings.Contains(strings.ToLower(key), "key") {
				sev = SeverityRisky
			}
			r.Findings = append(r.Findings, Finding{
				File: rel, FullPath: full, Tool: tool, Field: key,
				Severity:        sev,
				IssueZh:         "仓库级配置覆盖了关键字段",
				IssueEn:         "Repo-level config overrides a sensitive field",
				DetailValue:     truncate(fmt.Sprint(v), 120),
				SuggestedAction: "review",
			})
		}
	}
}

// inspectMarkdownContext flags context files (CLAUDE.md / AGENTS.md /
// .cursorrules) as info-level only — they're legitimately used for
// project conventions, but can also contain prompt-injection payloads
// that the user should glance at before letting the CLI ingest them.
func inspectMarkdownContext(r *AuditReport, full, rel, tool string) {
	stat, err := os.Stat(full)
	if err != nil {
		return
	}
	r.Findings = append(r.Findings, Finding{
		File: rel, FullPath: full, Tool: tool, Field: "(file)",
		Severity:        SeverityInfo,
		IssueZh:         fmt.Sprintf("项目上下文文件（%d 字节）会被 CLI 自动加载，请确认内容可信", stat.Size()),
		IssueEn:         fmt.Sprintf("Project-context file (%d bytes) is auto-loaded by the CLI — verify it's trusted", stat.Size()),
		DetailValue:     "",
		SuggestedAction: "review",
	})
}

func extractTomlValue(text, key string) string {
	for _, line := range strings.Split(text, "\n") {
		t := strings.TrimSpace(line)
		if strings.HasPrefix(t, key) {
			return truncate(t, 120)
		}
	}
	return "(unknown)"
}

// Quarantine renames a flagged file by appending a timestamp + sentinel
// suffix so the AI CLI no longer reads it. Reversible via plain `mv`.
func Quarantine(path string) (string, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	if _, err := os.Stat(abs); err != nil {
		return "", fmt.Errorf("file not found: %w", err)
	}
	stamp := time.Now().UTC().Format("20060102-150405")
	target := fmt.Sprintf("%s.quarantined-by-lurus-switch.%s", abs, stamp)
	if err := os.Rename(abs, target); err != nil {
		return "", fmt.Errorf("rename: %w", err)
	}
	return target, nil
}
