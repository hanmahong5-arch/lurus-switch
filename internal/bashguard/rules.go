// Package bashguard intercepts shell commands the AI CLI is about to
// execute and blocks the ones that match a deny-list. Defaults cover
// the most-cited 2025-2026 horror stories — Reddit "rm -rf ~/" wipe,
// Wolak Incident (issue #10077), Replit prod-DB deletion, etc.
//
// The engine is intentionally regex-based rather than a full shell
// parser: it covers the common accident patterns and leaves the
// remaining 10% of edge cases to the user's allow-list. We trade
// completeness for "shipping today" — most damage in the wild comes
// from a small set of stereotyped commands, not from clever attackers.
package bashguard

import (
	"fmt"
	"regexp"
	"strings"
)

type Severity string

const (
	SeverityCritical Severity = "critical" // hard-block, never run
	SeverityHigh     Severity = "high"     // block by default, allow-list-able
	SeverityMedium   Severity = "medium"   // warn-only by default
)

// Rule is a single deny pattern. ID lets the UI/log refer to a rule
// stably even if its Pattern is later refined.
type Rule struct {
	ID        string   `json:"id"`
	Pattern   string   `json:"pattern"`
	Severity  Severity `json:"severity"`
	ReasonZh  string   `json:"reasonZh"`
	ReasonEn  string   `json:"reasonEn"`
	Reference string   `json:"reference,omitempty"`

	compiled *regexp.Regexp // populated by Compile()
}

// MatchResult describes what an evaluation found. Empty Rule = allow.
type MatchResult struct {
	Allowed   bool   `json:"allowed"`
	Rule      *Rule  `json:"rule,omitempty"`
	Reason    string `json:"reason,omitempty"`
	NormalizedCommand string `json:"normalizedCommand"`
}

// Engine evaluates commands against an ordered set of rules. First
// match wins (most-specific rules should be earlier).
type Engine struct {
	rules []*Rule
}

func NewEngine(rules []*Rule) (*Engine, error) {
	for _, r := range rules {
		re, err := regexp.Compile(r.Pattern)
		if err != nil {
			return nil, fmt.Errorf("rule %s: %w", r.ID, err)
		}
		r.compiled = re
	}
	return &Engine{rules: rules}, nil
}

func (e *Engine) Rules() []*Rule { return e.rules }

// Evaluate normalizes the command and runs every rule against it.
// Returns the first matching rule, or Allowed=true when none match.
func (e *Engine) Evaluate(cmd string) MatchResult {
	norm := normalizeCommand(cmd)
	for _, r := range e.rules {
		if r.compiled != nil && r.compiled.MatchString(norm) {
			return MatchResult{
				Allowed:           false,
				Rule:              r,
				Reason:            r.ReasonEn,
				NormalizedCommand: norm,
			}
		}
	}
	return MatchResult{Allowed: true, NormalizedCommand: norm}
}

// normalizeCommand collapses whitespace and trims comments so a rule
// like `rm\s+-rf\s+/` matches `rm  -rf  /  # cleanup` too.
//
// We deliberately don't expand $vars or ~ here — those are kept literal
// in the matched string so rules can target either the literal `~/`
// (the Reddit tilde-trick incident) and the expanded home path with
// separate patterns. Letting the AI CLI see a literal `~` *and* the
// shell expand it differently is precisely the bug class we're guarding.
func normalizeCommand(cmd string) string {
	// Strip line comments (# to EOL), preserving content inside quotes.
	// Cheap implementation: only strip when # is the first non-space
	// char or preceded by whitespace and not inside quotes (best-effort).
	norm := stripUnquotedComments(cmd)
	// Collapse whitespace runs to single spaces; trim ends.
	norm = strings.Join(strings.Fields(norm), " ")
	return norm
}

func stripUnquotedComments(s string) string {
	var b strings.Builder
	inSingle := false
	inDouble := false
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch c {
		case '\'':
			if !inDouble {
				inSingle = !inSingle
			}
		case '"':
			if !inSingle {
				inDouble = !inDouble
			}
		case '#':
			if !inSingle && !inDouble {
				// Comment to end of line
				if j := strings.IndexByte(s[i:], '\n'); j >= 0 {
					i += j // re-enter the loop after the newline
					continue
				}
				return b.String()
			}
		}
		b.WriteByte(c)
	}
	return b.String()
}

// DefaultRules returns the baked-in deny-list. The list is intentionally
// short and high-precision — every entry traces back to a documented
// incident, so we can defend the inclusion. Users can add custom rules
// via the UI but cannot remove these (they're part of the safety floor).
func DefaultRules() []*Rule {
	return []*Rule{
		// ── rm -rf wipes ────────────────────────────────────────────
		{
			ID: "rm-rf-root", Severity: SeverityCritical,
			// matches: rm -rf /, rm -rf /*, rm -fr /, etc.
			Pattern:  `^(?:\s*sudo\s+)?rm\s+(?:-[rRf]+\s+){1,2}(?:--\s+)?/(?:\s|$|\*)`,
			ReasonZh: "rm -rf 根目录会清空整台机器（参考 Wolak Incident issue #10077）",
			ReasonEn: "rm -rf on / wipes the entire machine (ref: Wolak Incident, issue #10077)",
			Reference: "https://github.com/anthropics/claude-code/issues/10077",
		},
		{
			ID: "rm-rf-tilde", Severity: SeverityCritical,
			// matches: rm -rf ~, rm -rf ~/, rm -rf ~/*
			Pattern:  `^(?:\s*sudo\s+)?rm\s+(?:-[rRf]+\s+){1,2}(?:--\s+)?~(?:/|\s|$|/\*)`,
			ReasonZh: "rm -rf ~ 会清空主目录（参考 Reddit byteiota 事故 + issue #12637）",
			ReasonEn: "rm -rf ~/ wipes your home directory (ref: byteiota Reddit incident, issue #12637)",
			Reference: "https://byteiota.com/claude-codes-rm-rf-bug-deleted-my-home-directory/",
		},
		{
			ID: "rm-rf-home-literal", Severity: SeverityCritical,
			// matches expanded forms: rm -rf /home/USER or /Users/USER
			Pattern:  `^(?:\s*sudo\s+)?rm\s+(?:-[rRf]+\s+){1,2}(?:--\s+)?(?:/home/[^/\s]+|/Users/[^/\s]+|/root)\b`,
			ReasonZh: "rm -rf 直接指向用户目录会丢失全部数据",
			ReasonEn: "rm -rf targeting a user home will destroy all personal data",
		},
		// ── overwrite block devices ─────────────────────────────────
		{
			ID: "dd-of-block-device", Severity: SeverityCritical,
			Pattern:  `\bdd\b[^|]*\bof=/dev/(?:sd[a-z]|nvme|disk|hd)`,
			ReasonZh: "dd 写入物理磁盘会覆盖整盘",
			ReasonEn: "dd writing to a block device will overwrite the entire disk",
		},
		{
			ID: "mkfs-non-temp", Severity: SeverityCritical,
			Pattern:  `\bmkfs(?:\.\w+)?\s+/dev/(?:sd[a-z]|nvme|disk|hd)`,
			ReasonZh: "mkfs 会格式化整个分区",
			ReasonEn: "mkfs will format an entire partition",
		},
		{
			ID: "format-c", Severity: SeverityCritical,
			Pattern:  `(?i)\bformat\s+[cC]:`,
			ReasonZh: "format C: 会清空 Windows 系统盘",
			ReasonEn: "format C: wipes the Windows system drive",
		},
		// ── permission disasters ────────────────────────────────────
		{
			ID: "chmod-777-root", Severity: SeverityHigh,
			Pattern:  `\bchmod\s+(?:-R\s+)?(?:0?777|a\+rwx)\s+/(?:\s|$)`,
			ReasonZh: "chmod 777 / 会向所有用户开放整个根目录",
			ReasonEn: "chmod 777 / opens the entire root directory to all users",
		},
		{
			ID: "chmod-recursive-root", Severity: SeverityHigh,
			Pattern:  `\bchmod\s+-R\s+\d+\s+/(?:\s|$)`,
			ReasonZh: "chmod -R 整个根目录会破坏系统权限",
			ReasonEn: "chmod -R on / will break system permissions",
		},
		// ── pipe-to-shell ───────────────────────────────────────────
		{
			ID: "curl-pipe-shell", Severity: SeverityHigh,
			Pattern:  `(?i)\b(?:curl|wget|fetch)\b[^|]+\|\s*(?:sudo\s+)?(?:bash|sh|zsh|fish|python|perl|ruby|node)\b`,
			ReasonZh: "curl 直接管道到 shell 会执行远端任意脚本",
			ReasonEn: "Piping curl/wget directly into a shell runs untrusted remote code",
			Reference: "OWASP supply-chain attacks",
		},
		{
			ID: "eval-curl", Severity: SeverityHigh,
			Pattern:  `\beval\b[^)]*(?:curl|wget|fetch)\b`,
			ReasonZh: "eval 远程 fetch 内容是经典 RCE 模式",
			ReasonEn: "eval'ing remotely-fetched content is a classic RCE pattern",
		},
		// ── fork bomb ───────────────────────────────────────────────
		{
			ID: "fork-bomb", Severity: SeverityCritical,
			Pattern:  `:\(\)\s*\{\s*:\s*\|\s*:\s*&\s*\}\s*;:`,
			ReasonZh: "Fork bomb 会立即耗尽系统资源",
			ReasonEn: "Fork bomb will exhaust system resources immediately",
		},
		// ── git destructive ─────────────────────────────────────────
		{
			ID: "git-push-force-protected", Severity: SeverityHigh,
			Pattern:  `\bgit\s+push\s+(?:--force|-f)\b[^&|;]*\b(?:main|master|prod|production|release)\b`,
			ReasonZh: "强制推送到主分支会覆盖他人提交",
			ReasonEn: "Force-pushing to a protected branch overwrites others' commits",
		},
		{
			ID: "git-clean-fdx", Severity: SeverityMedium,
			// Only flag when -d (cleans dirs) or -x (cleans .gitignored too).
			// Plain `git clean -f` is benign — only removes already-listed
			// untracked files matching the path argument.
			Pattern:  `\bgit\s+clean\b[^&|;]*-[fdx]*[dx][fdx]*\b`,
			ReasonZh: "git clean -d/-x 会删除所有未追踪文件和目录（含 .env、node_modules）",
			ReasonEn: "git clean -d/-x removes all untracked files and dirs (including .env, node_modules)",
		},
		// ── database destructive ────────────────────────────────────
		{
			ID: "drop-database", Severity: SeverityCritical,
			Pattern:  `(?i)DROP\s+DATABASE\b`,
			ReasonZh: "DROP DATABASE 删除整个数据库（参考 Replit SaaStr 事故）",
			ReasonEn: "DROP DATABASE deletes the whole DB (ref: Replit SaaStr incident, Jul 2025)",
		},
		{
			ID: "truncate-table", Severity: SeverityHigh,
			Pattern:  `(?i)TRUNCATE\s+TABLE\b`,
			ReasonZh: "TRUNCATE TABLE 不可回滚",
			ReasonEn: "TRUNCATE TABLE is not transactional in most engines",
		},
		// ── cloud destructive ───────────────────────────────────────
		{
			ID: "aws-s3-rb", Severity: SeverityCritical,
			Pattern:  `\baws\s+s3\s+rb\b.*--force\b`,
			ReasonZh: "aws s3 rb --force 强制删除存储桶及其全部对象",
			ReasonEn: "aws s3 rb --force deletes the bucket and all objects",
		},
		{
			ID: "gcloud-sql-delete", Severity: SeverityCritical,
			Pattern:  `\bgcloud\s+sql\s+(?:instances\s+)?delete\b`,
			ReasonZh: "gcloud sql delete 会删除托管数据库",
			ReasonEn: "gcloud sql delete removes a managed database instance",
		},
		// ── system kill ─────────────────────────────────────────────
		{
			ID: "kill-pid-1", Severity: SeverityHigh,
			Pattern:  `\bkill\s+(?:-9\s+)?1\b`,
			ReasonZh: "kill PID 1 会让 init 进程退出，整机不可用",
			ReasonEn: "Killing PID 1 (init) crashes the whole machine",
		},
	}
}
