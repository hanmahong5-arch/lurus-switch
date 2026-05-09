package dlp

// DefaultPatterns returns the curated pattern library that ships with
// Switch. These are conservative defaults — operators are expected to
// review and tighten policies (warn → redact → block) over time as
// they understand their org's risk profile.
//
// Categories:
//   - secrets:       API keys, JWTs, AWS access keys
//   - pii:           SSN, credit card, email, phone, government IDs
//   - internal:      placeholder for org-specific patterns operators add
//
// Adding a new default? Three rules:
//   1. The regex MUST work on the byte-offset returned by FindAllStringIndex
//      (i.e., it can't rely on lookahead/lookbehind that Go's regexp
//      doesn't support).
//   2. Default Policy should be PolicyWarn for false-positive-prone
//      patterns and PolicyRedact for unambiguous ones (CC numbers).
//   3. Add a corresponding test case in patterns_test.go.
func DefaultPatterns() []Pattern {
	return []Pattern{
		// === Secrets / API keys ===
		{
			Name:        "api_key.openai",
			Description: "OpenAI sk-... API key",
			Regex:       `\bsk-[A-Za-z0-9]{20,}\b`,
			Severity:    SeverityCritical,
			Policy:      PolicyBlock,
			Tags:        []string{"secrets", "api_key"},
		},
		{
			Name:        "api_key.anthropic",
			Description: "Anthropic sk-ant-... API key",
			Regex:       `\bsk-ant-[A-Za-z0-9_\-]{20,}\b`,
			Severity:    SeverityCritical,
			Policy:      PolicyBlock,
			Tags:        []string{"secrets", "api_key"},
		},
		{
			Name:        "api_key.aws_access",
			Description: "AWS access-key id (AKIA…)",
			Regex:       `\bAKIA[0-9A-Z]{16}\b`,
			Severity:    SeverityCritical,
			Policy:      PolicyBlock,
			Tags:        []string{"secrets", "aws"},
		},
		{
			Name:        "api_key.aws_secret",
			Description: "AWS-style 40-char secret key (heuristic)",
			// Heuristic — AWS secrets are 40 chars of [A-Za-z0-9/+]. False
			// positives possible on long opaque strings; default policy is
			// warn so admins decide if they want to tighten.
			Regex:    `\b[A-Za-z0-9/+]{40}\b`,
			Severity: SeverityWarning,
			Policy:   PolicyWarn,
			Tags:     []string{"secrets", "aws"},
		},
		{
			Name:        "secret.jwt",
			Description: "JSON Web Token",
			// Three base64-url segments separated by dots — header.payload.signature.
			Regex:    `\beyJ[A-Za-z0-9_-]{10,}\.[A-Za-z0-9_-]{10,}\.[A-Za-z0-9_-]{10,}\b`,
			Severity: SeverityCritical,
			Policy:   PolicyRedact,
			Tags:     []string{"secrets", "token"},
		},
		{
			Name:        "secret.github_pat",
			Description: "GitHub personal access token (ghp_…/github_pat_…)",
			Regex:       `\b(ghp_[A-Za-z0-9]{36}|github_pat_[A-Za-z0-9_]{82})\b`,
			Severity:    SeverityCritical,
			Policy:      PolicyBlock,
			Tags:        []string{"secrets", "github"},
		},
		{
			Name:        "secret.private_key",
			Description: "PEM-encoded private key header",
			Regex:       `-----BEGIN (RSA |EC |DSA |OPENSSH |ENCRYPTED |PGP )?PRIVATE KEY-----`,
			Severity:    SeverityCritical,
			Policy:      PolicyBlock,
			Tags:        []string{"secrets", "crypto"},
		},

		// === PII ===
		{
			Name:        "pii.credit_card",
			Description: "13-19 digit number that looks like a credit card",
			// Group of 13-19 digits, optionally separated by spaces or
			// dashes. Caller may want a Luhn check downstream — we keep
			// the regex simple and rely on Luhn at policy-tightening time.
			Regex:    `\b(?:\d[ -]?){13,19}\b`,
			Severity: SeverityCritical,
			Policy:   PolicyRedact,
			Tags:     []string{"pii", "financial"},
		},
		{
			Name:        "pii.ssn_us",
			Description: "US Social Security Number (XXX-XX-XXXX)",
			Regex:       `\b\d{3}-\d{2}-\d{4}\b`,
			Severity:    SeverityCritical,
			Policy:      PolicyRedact,
			Tags:        []string{"pii", "us"},
		},
		{
			Name:        "pii.cn_id",
			Description: "中国大陆居民身份证号（18位）",
			Regex:       `\b\d{17}[0-9Xx]\b`,
			Severity:    SeverityCritical,
			Policy:      PolicyRedact,
			Tags:        []string{"pii", "cn"},
		},
		{
			Name:        "pii.email",
			Description: "Email address (RFC-ish)",
			Regex:       `\b[A-Za-z0-9._%+\-]+@[A-Za-z0-9.\-]+\.[A-Za-z]{2,}\b`,
			Severity:    SeverityInfo,
			// Most prompts contain emails as references; default to warn so
			// the audit log shows it without breaking workflows. Operators
			// who care about email confidentiality move it to redact.
			Policy: PolicyWarn,
			Tags:   []string{"pii", "contact"},
		},
		{
			Name:        "pii.phone",
			Description: "Generic phone number (10+ digits with separators)",
			Regex:       `\b(?:\+?\d{1,3}[ -]?)?(?:\(\d{2,4}\)[ -]?)?\d{3}[ -]?\d{4,}\b`,
			Severity:    SeverityInfo,
			Policy:      PolicyWarn,
			Tags:        []string{"pii", "contact"},
		},

		// === Internal — placeholder, not enabled by default ===
		// Real deployments add custom patterns matching their internal
		// customer ID format, support ticket prefix, etc.
	}
}
