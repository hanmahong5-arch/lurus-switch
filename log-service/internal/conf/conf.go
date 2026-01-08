package conf

import (
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// Bootstrap is the root configuration
type Bootstrap struct {
	Server        Server        `yaml:"server"`
	Data          Data          `yaml:"data"`
	NATS          NATS          `yaml:"nats"`
	Observability Observability `yaml:"observability"`
}

// Server configuration
type Server struct {
	HTTP ServerConfig `yaml:"http"`
}

// ServerConfig for HTTP server
type ServerConfig struct {
	Addr    string        `yaml:"addr"`
	Timeout time.Duration `yaml:"timeout"`
}

// Data configuration
type Data struct {
	ClickHouse ClickHouse `yaml:"clickhouse"`
}

// ClickHouse configuration
type ClickHouse struct {
	Addr            string        `yaml:"addr"`
	Database        string        `yaml:"database"`
	Username        string        `yaml:"username"`
	Password        string        `yaml:"password"`
	Debug           bool          `yaml:"debug"`
	MaxOpenConns    int           `yaml:"max_open_conns"`
	MaxIdleConns    int           `yaml:"max_idle_conns"`
	ConnMaxLifetime time.Duration `yaml:"conn_max_lifetime"`
}

// NATS configuration
type NATS struct {
	URL          string        `yaml:"url"`
	Enabled      bool          `yaml:"enabled"`
	ConsumerName string        `yaml:"consumer_name"`
	Stream       string        `yaml:"stream"`
	Subject      string        `yaml:"subject"`
	BatchSize    int           `yaml:"batch_size"`
	BatchTimeout time.Duration `yaml:"batch_timeout"`
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

// Load loads configuration from a YAML file
func Load(path string) (*Bootstrap, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg Bootstrap
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	// Apply defaults
	if cfg.Server.HTTP.Timeout == 0 {
		cfg.Server.HTTP.Timeout = 60 * time.Second
	}
	if cfg.Data.ClickHouse.MaxOpenConns == 0 {
		cfg.Data.ClickHouse.MaxOpenConns = 10
	}
	if cfg.Data.ClickHouse.MaxIdleConns == 0 {
		cfg.Data.ClickHouse.MaxIdleConns = 5
	}
	if cfg.NATS.BatchSize == 0 {
		cfg.NATS.BatchSize = 100
	}
	if cfg.NATS.BatchTimeout == 0 {
		cfg.NATS.BatchTimeout = time.Second
	}

	return &cfg, nil
}
