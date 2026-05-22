package store

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoad_MissingFileReturnsDefault(t *testing.T) {
	dir := t.TempDir()
	cfg, err := Load(dir)
	if err != nil {
		t.Fatalf("missing file should not error, got %v", err)
	}
	if cfg.Enabled {
		t.Errorf("default config should have Enabled=false (opt-in)")
	}
	if cfg.Rules.StuckAfterSec <= 0 {
		t.Errorf("default StuckAfterSec must be positive, got %d", cfg.Rules.StuckAfterSec)
	}
}

func TestSaveLoad_RoundTrips(t *testing.T) {
	dir := t.TempDir()
	in := AppConfig{
		Enabled: true,
		Rules: RulesPersist{
			StuckAfterSec:    120,
			StuckEscalateSec: 600,
			IdleAfterSec:     300,
			NotifyStuck:      true,
			NotifyDone:       false,
		},
	}
	in.Feishu.WebhookURL = "https://open.feishu.cn/open-apis/bot/v2/hook/test"
	in.Feishu.Secret = "shhh"

	if err := Save(dir, in); err != nil {
		t.Fatalf("Save: %v", err)
	}
	out, err := Load(dir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if out.Enabled != in.Enabled {
		t.Errorf("Enabled mismatch: want %v got %v", in.Enabled, out.Enabled)
	}
	if out.Feishu.WebhookURL != in.Feishu.WebhookURL {
		t.Errorf("WebhookURL mismatch")
	}
	if out.Feishu.Secret != in.Feishu.Secret {
		t.Errorf("Secret mismatch")
	}
	if out.Rules.NotifyDone != false {
		t.Errorf("NotifyDone should preserve false, got %v", out.Rules.NotifyDone)
	}
	if out.Rules.StuckAfterSec != 120 {
		t.Errorf("StuckAfterSec mismatch: want 120 got %d", out.Rules.StuckAfterSec)
	}
}

func TestSave_AtomicWriteLeavesNoTmp(t *testing.T) {
	dir := t.TempDir()
	cfg := DefaultAppConfig()
	cfg.Enabled = true
	if err := Save(dir, cfg); err != nil {
		t.Fatalf("Save: %v", err)
	}
	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		if filepath.Ext(e.Name()) == ".tmp" {
			t.Errorf("stale .tmp file left behind: %s", e.Name())
		}
	}
}

func TestRulesPersist_ToRulesConfig_BackfillsZero(t *testing.T) {
	r := RulesPersist{NotifyStuck: true, NotifyDone: true} // all durations zero
	cfg := r.ToRulesConfig()
	if cfg.StuckAfter <= 0 {
		t.Errorf("zero StuckAfterSec must fall back to default, got %s", cfg.StuckAfter)
	}
	if cfg.IdleAfter < time.Second {
		t.Errorf("zero IdleAfterSec must fall back to default, got %s", cfg.IdleAfter)
	}
}
