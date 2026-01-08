package conf

import (
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// Bootstrap is the root configuration
type Bootstrap struct {
	Server   Server   `yaml:"server"`
	Provider Provider `yaml:"provider"`
	Billing  Billing  `yaml:"billing"`
	Log      Log      `yaml:"log"`
	NATS     NATS     `yaml:"nats"`
	Features Features `yaml:"features"`
	Proxy    Proxy    `yaml:"proxy"`
	Metrics  Metrics  `yaml:"metrics"`
	Tracing  Tracing  `yaml:"tracing"`
}

// Server configuration
type Server struct {
	HTTP HTTP   `yaml:"http"`
	Mode string `yaml:"mode"` // development, staging, production
}

// HTTP server configuration
type HTTP struct {
	Addr         string        `yaml:"addr"`
	Timeout      time.Duration `yaml:"timeout"`
	ReadTimeout  time.Duration `yaml:"read_timeout"`
	WriteTimeout time.Duration `yaml:"write_timeout"`
	IdleTimeout  time.Duration `yaml:"idle_timeout"`
}

// Provider service client configuration
type Provider struct {
	Endpoint string        `yaml:"endpoint"`
	Timeout  time.Duration `yaml:"timeout"`
	CacheTTL time.Duration `yaml:"cache_ttl"`
}

// Billing service client configuration
type Billing struct {
	Endpoint string        `yaml:"endpoint"`
	Timeout  time.Duration `yaml:"timeout"`
	Enabled  bool          `yaml:"enabled"`
}

// Log configuration
type Log struct {
	Enabled      bool          `yaml:"enabled"`
	BatchSize    int           `yaml:"batch_size"`
	BatchTimeout time.Duration `yaml:"batch_timeout"`
}

// NATS configuration
type NATS struct {
	URL      string      `yaml:"url"`
	Subjects NATSSubject `yaml:"subjects"`
}

// NATSSubject defines NATS subjects
type NATSSubject struct {
	LogWrite     string `yaml:"log_write"`
	BillingUsage string `yaml:"billing_usage"`
}

// Features for feature flags
type Features struct {
	NewAPIEnabled bool   `yaml:"new_api_enabled"`
	NewAPIURL     string `yaml:"new_api_url"`
	OfflineMode   bool   `yaml:"offline_mode"`
	BillingCheck  bool   `yaml:"billing_check"`
	AsyncLogging  bool   `yaml:"async_logging"`
}

// Proxy configuration
type Proxy struct {
	RoundRobin     bool          `yaml:"round_robin"`
	RetryCount     int           `yaml:"retry_count"`
	RequestTimeout time.Duration `yaml:"request_timeout"`
}

// Metrics configuration
type Metrics struct {
	Enabled bool   `yaml:"enabled"`
	Path    string `yaml:"path"`
}

// Tracing configuration
type Tracing struct {
	Enabled    bool    `yaml:"enabled"`
	Endpoint   string  `yaml:"endpoint"`
	SampleRate float64 `yaml:"sample_rate"`
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
		cfg.Server.HTTP.Addr = ":18100"
	}
	if cfg.Server.Mode == "" {
		cfg.Server.Mode = "development"
	}
	if cfg.Server.HTTP.Timeout == 0 {
		cfg.Server.HTTP.Timeout = 300 * time.Second
	}
	if cfg.Server.HTTP.ReadTimeout == 0 {
		cfg.Server.HTTP.ReadTimeout = 30 * time.Second
	}
	if cfg.Server.HTTP.WriteTimeout == 0 {
		cfg.Server.HTTP.WriteTimeout = 300 * time.Second
	}
	if cfg.Provider.Timeout == 0 {
		cfg.Provider.Timeout = 5 * time.Second
	}
	if cfg.Provider.CacheTTL == 0 {
		cfg.Provider.CacheTTL = 5 * time.Minute
	}
	if cfg.Billing.Timeout == 0 {
		cfg.Billing.Timeout = 3 * time.Second
	}
	if cfg.Log.BatchSize == 0 {
		cfg.Log.BatchSize = 10
	}
	if cfg.Log.BatchTimeout == 0 {
		cfg.Log.BatchTimeout = 100 * time.Millisecond
	}
	if cfg.Proxy.RetryCount == 0 {
		cfg.Proxy.RetryCount = 3
	}
	if cfg.Proxy.RequestTimeout == 0 {
		cfg.Proxy.RequestTimeout = 120 * time.Second
	}
	if cfg.NATS.Subjects.LogWrite == "" {
		cfg.NATS.Subjects.LogWrite = "log.write"
	}
	if cfg.NATS.Subjects.BillingUsage == "" {
		cfg.NATS.Subjects.BillingUsage = "billing.usage"
	}
	if cfg.Tracing.Endpoint == "" {
		cfg.Tracing.Endpoint = "localhost:4317"
	}
	if cfg.Tracing.SampleRate == 0 {
		cfg.Tracing.SampleRate = 1.0
	}
}
