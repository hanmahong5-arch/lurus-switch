package conf

import (
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// Bootstrap is the root configuration
type Bootstrap struct {
	Server        Server        `yaml:"server"`
	Database      Database      `yaml:"database"`
	Redis         Redis         `yaml:"redis"`
	NATS          NATS          `yaml:"nats"`
	Casdoor       Casdoor       `yaml:"casdoor"`
	Billing       Billing       `yaml:"billing"`
	Observability Observability `yaml:"observability"`
}

// Server configuration
type Server struct {
	HTTP HTTP `yaml:"http"`
}

// HTTP server configuration
type HTTP struct {
	Addr    string        `yaml:"addr"`
	Timeout time.Duration `yaml:"timeout"`
}

// Database configuration
type Database struct {
	Driver          string        `yaml:"driver"`
	DSN             string        `yaml:"dsn"`
	MaxIdleConns    int           `yaml:"max_idle_conns"`
	MaxOpenConns    int           `yaml:"max_open_conns"`
	ConnMaxLifetime time.Duration `yaml:"conn_max_lifetime"`
}

// Redis configuration
type Redis struct {
	Addr     string `yaml:"addr"`
	Password string `yaml:"password"`
	DB       int    `yaml:"db"`
	PoolSize int    `yaml:"pool_size"`
}

// NATS configuration
type NATS struct {
	URL      string      `yaml:"url"`
	Subjects NATSSubject `yaml:"subjects"`
}

// NATSSubject defines NATS subjects
type NATSSubject struct {
	BillingUsage string `yaml:"billing_usage"`
	QuotaUpdate  string `yaml:"quota_update"`
}

// Casdoor OAuth2 configuration
type Casdoor struct {
	Enabled      bool   `yaml:"enabled"`
	Endpoint     string `yaml:"endpoint"`
	ClientID     string `yaml:"client_id"`
	ClientSecret string `yaml:"client_secret"`
	Organization string `yaml:"organization"`
	Application  string `yaml:"application"`
}

// Billing configuration
type Billing struct {
	DefaultQuota int64    `yaml:"default_quota"`
	ResetPeriod  string   `yaml:"reset_period"`
	FreeTier     FreeTier `yaml:"free_tier"`
	Pricing      Pricing  `yaml:"pricing"`
}

// FreeTier configuration
type FreeTier struct {
	Enabled      bool  `yaml:"enabled"`
	DailyLimit   int64 `yaml:"daily_limit"`
	MonthlyLimit int64 `yaml:"monthly_limit"`
}

// Pricing configuration (per 1M tokens in USD)
type Pricing struct {
	InputTokens       float64 `yaml:"input_tokens"`
	OutputTokens      float64 `yaml:"output_tokens"`
	CacheReadTokens   float64 `yaml:"cache_read_tokens"`
	CacheCreateTokens float64 `yaml:"cache_create_tokens"`
}

// Observability configuration
type Observability struct {
	Tracing Tracing `yaml:"tracing"`
	Metrics Metrics `yaml:"metrics"`
}

// Tracing configuration
type Tracing struct {
	Enabled    bool    `yaml:"enabled"`
	Endpoint   string  `yaml:"endpoint"`
	SampleRate float64 `yaml:"sample_rate"`
}

// Metrics configuration
type Metrics struct {
	Enabled bool   `yaml:"enabled"`
	Path    string `yaml:"path"`
}

// Load loads configuration from file
func Load(path string) (*Bootstrap, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	// Expand environment variables
	data = []byte(os.ExpandEnv(string(data)))

	var cfg Bootstrap
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	// Set defaults
	setDefaults(&cfg)

	return &cfg, nil
}

func setDefaults(cfg *Bootstrap) {
	if cfg.Server.HTTP.Addr == "" {
		cfg.Server.HTTP.Addr = ":18103"
	}
	if cfg.Server.HTTP.Timeout == 0 {
		cfg.Server.HTTP.Timeout = 30 * time.Second
	}
	if cfg.Database.MaxIdleConns == 0 {
		cfg.Database.MaxIdleConns = 10
	}
	if cfg.Database.MaxOpenConns == 0 {
		cfg.Database.MaxOpenConns = 100
	}
	if cfg.Database.ConnMaxLifetime == 0 {
		cfg.Database.ConnMaxLifetime = time.Hour
	}
	if cfg.Redis.PoolSize == 0 {
		cfg.Redis.PoolSize = 10
	}
	if cfg.Billing.DefaultQuota == 0 {
		cfg.Billing.DefaultQuota = 1000000
	}
	if cfg.Billing.ResetPeriod == "" {
		cfg.Billing.ResetPeriod = "monthly"
	}
	if cfg.Billing.Pricing.InputTokens == 0 {
		cfg.Billing.Pricing.InputTokens = 3.0
	}
	if cfg.Billing.Pricing.OutputTokens == 0 {
		cfg.Billing.Pricing.OutputTokens = 15.0
	}
	if cfg.NATS.Subjects.BillingUsage == "" {
		cfg.NATS.Subjects.BillingUsage = "billing.usage"
	}
}
