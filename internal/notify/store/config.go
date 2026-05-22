// Package store persists notify subsystem preferences (transport
// credentials + rule toggles) to disk. Kept in its own package so
// `notify` itself can stay dependency-free of any concrete transport —
// the import cycle would otherwise be:
//
//	notify → feishu → notify
//
// store sits outside `notify`, depends on both `notify/feishu` and
// `notify/rules`, and nobody depends on store except app wiring.
package store

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"lurus-switch/internal/notify/feishu"
	"lurus-switch/internal/notify/rules"
)

// configFilename is what we persist on disk under appDataBaseDir. Keep it
// stable — the file path is the only contract between releases.
const configFilename = "notify.json"

// AppConfig is the on-disk shape of the user's notify preferences. The
// Settings UI loads/saves this whole document atomically.
//
// Rule durations live as integer seconds so the JSON renders trivially
// in a form ("show me 60") instead of the Go-marshalled "1m0s" string
// the React side would have to parse.
type AppConfig struct {
	// Enabled gates the entire notify subsystem. When false the Bus has
	// no transports registered and the Engine isn't ticking.
	Enabled bool `json:"enabled"`

	// Feishu is the per-transport block. Other transports get their own
	// nested struct here when they land.
	Feishu feishu.Config `json:"feishu"`

	// Rules mirrors rules.Config but with durations represented as
	// integer seconds for ergonomics in the form layer.
	Rules RulesPersist `json:"rules"`
}

// RulesPersist is the JSON projection of rules.Config. Durations as
// seconds so the form layer can edit them as plain numbers.
type RulesPersist struct {
	StuckAfterSec    int  `json:"stuckAfterSec"`
	StuckEscalateSec int  `json:"stuckEscalateSec"`
	IdleAfterSec     int  `json:"idleAfterSec"`
	NotifyStuck      bool `json:"notifyStuck"`
	NotifyDone       bool `json:"notifyDone"`
}

// DefaultAppConfig is what brand-new installations start with: feature off
// (so the user opts in deliberately), rules defaults from the engine.
func DefaultAppConfig() AppConfig {
	d := rules.DefaultConfig()
	return AppConfig{
		Enabled: false,
		Rules: RulesPersist{
			StuckAfterSec:    int(d.StuckAfter / time.Second),
			StuckEscalateSec: int(d.StuckEscalate / time.Second),
			IdleAfterSec:     int(d.IdleAfter / time.Second),
			NotifyStuck:      d.NotifyStuck,
			NotifyDone:       d.NotifyDone,
		},
	}
}

// ToRulesConfig projects the persistable shape back into the engine's
// duration-typed Config. Zero / negative seconds fall back to defaults
// so a hand-edited file with a missing field doesn't disable a rule.
func (r RulesPersist) ToRulesConfig() rules.Config {
	d := rules.DefaultConfig()
	out := d
	if r.StuckAfterSec > 0 {
		out.StuckAfter = time.Duration(r.StuckAfterSec) * time.Second
	}
	if r.StuckEscalateSec > 0 {
		out.StuckEscalate = time.Duration(r.StuckEscalateSec) * time.Second
	}
	if r.IdleAfterSec > 0 {
		out.IdleAfter = time.Duration(r.IdleAfterSec) * time.Second
	}
	out.NotifyStuck = r.NotifyStuck
	out.NotifyDone = r.NotifyDone
	return out
}

var configMu sync.Mutex

// Load reads notify.json from dir. Missing file → DefaultAppConfig
// (no error). Corrupt file → error so the caller can surface it instead
// of silently dropping the user's saved preferences.
func Load(dir string) (AppConfig, error) {
	path := filepath.Join(dir, configFilename)
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return DefaultAppConfig(), nil
		}
		return AppConfig{}, fmt.Errorf("read notify config: %w", err)
	}
	var cfg AppConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return AppConfig{}, fmt.Errorf("corrupt notify config: %w", err)
	}
	// Backfill zero values from defaults so partial files don't disable
	// a rule by omission.
	def := DefaultAppConfig()
	if cfg.Rules.StuckAfterSec == 0 {
		cfg.Rules.StuckAfterSec = def.Rules.StuckAfterSec
	}
	if cfg.Rules.StuckEscalateSec == 0 {
		cfg.Rules.StuckEscalateSec = def.Rules.StuckEscalateSec
	}
	if cfg.Rules.IdleAfterSec == 0 {
		cfg.Rules.IdleAfterSec = def.Rules.IdleAfterSec
	}
	return cfg, nil
}

// Save writes notify.json atomically (.tmp + rename) — same pattern as
// internal/toolmanifest/overrides.go so a crash mid-write can't leave a
// half-truncated file.
func Save(dir string, cfg AppConfig) error {
	configMu.Lock()
	defer configMu.Unlock()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	path := filepath.Join(dir, configFilename)
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
