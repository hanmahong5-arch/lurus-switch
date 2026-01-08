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
	Consul        Consul        `yaml:"consul"`
	Observability Observability `yaml:"observability"`
}

// Server configuration
type Server struct {
	HTTP ServerConfig `yaml:"http"`
	GRPC ServerConfig `yaml:"grpc"`
}

// ServerConfig for HTTP/gRPC servers
type ServerConfig struct {
	Network string        `yaml:"network"`
	Addr    string        `yaml:"addr"`
	Timeout time.Duration `yaml:"timeout"`
}

// Data configuration
type Data struct {
	Database Database `yaml:"database"`
	Redis    Redis    `yaml:"redis"`
}

// Database configuration
type Database struct {
	Driver          string        `yaml:"driver"`
	Source          string        `yaml:"source"`
	MaxIdleConns    int           `yaml:"max_idle_conns"`
	MaxOpenConns    int           `yaml:"max_open_conns"`
	ConnMaxLifetime time.Duration `yaml:"conn_max_lifetime"`
}

// Redis configuration
type Redis struct {
	Addr         string        `yaml:"addr"`
	Password     string        `yaml:"password"`
	DB           int           `yaml:"db"`
	ReadTimeout  time.Duration `yaml:"read_timeout"`
	WriteTimeout time.Duration `yaml:"write_timeout"`
}

// Consul configuration
type Consul struct {
	Address     string `yaml:"address"`
	Scheme      string `yaml:"scheme"`
	ServiceName string `yaml:"service_name"`
	HealthCheck bool   `yaml:"health_check"`
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
	if cfg.Server.GRPC.Timeout == 0 {
		cfg.Server.GRPC.Timeout = 60 * time.Second
	}
	if cfg.Data.Database.MaxIdleConns == 0 {
		cfg.Data.Database.MaxIdleConns = 10
	}
	if cfg.Data.Database.MaxOpenConns == 0 {
		cfg.Data.Database.MaxOpenConns = 100
	}

	return &cfg, nil
}
