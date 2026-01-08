package nats

import "time"

// Config represents NATS client configuration
type Config struct {
	// URL is the NATS server URL
	URL string `json:"url" yaml:"url"`

	// Enabled controls whether NATS is enabled
	Enabled bool `json:"enabled" yaml:"enabled"`

	// Connection options
	ReconnectWait       time.Duration `json:"reconnect_wait" yaml:"reconnect_wait"`
	MaxReconnects       int           `json:"max_reconnects" yaml:"max_reconnects"`
	ReconnectBufferSize int           `json:"reconnect_buffer_size" yaml:"reconnect_buffer_size"`
	ConnectTimeout      time.Duration `json:"connect_timeout" yaml:"connect_timeout"`

	// TLS options
	TLSEnabled bool   `json:"tls_enabled" yaml:"tls_enabled"`
	CertFile   string `json:"cert_file" yaml:"cert_file"`
	KeyFile    string `json:"key_file" yaml:"key_file"`
	CAFile     string `json:"ca_file" yaml:"ca_file"`

	// Authentication
	Username string `json:"username" yaml:"username"`
	Password string `json:"password" yaml:"password"`
	Token    string `json:"token" yaml:"token"`
	NKeyFile string `json:"nkey_file" yaml:"nkey_file"`
}

// DefaultConfig returns default NATS configuration
func DefaultConfig() *Config {
	return &Config{
		URL:                 "nats://localhost:4222",
		Enabled:             false,
		ReconnectWait:       2 * time.Second,
		MaxReconnects:       -1, // Infinite reconnects
		ReconnectBufferSize: 8 * 1024 * 1024, // 8MB
		ConnectTimeout:      10 * time.Second,
	}
}

// JetStreamConfig represents JetStream configuration
type JetStreamConfig struct {
	// Domain is the JetStream domain (optional)
	Domain string `json:"domain" yaml:"domain"`

	// Streams is a list of stream configurations to create
	Streams []StreamConfig `json:"streams" yaml:"streams"`
}

// StreamConfig represents a JetStream stream configuration
type StreamConfig struct {
	Name        string        `json:"name" yaml:"name"`
	Subjects    []string      `json:"subjects" yaml:"subjects"`
	Retention   string        `json:"retention" yaml:"retention"` // limits, interest, workqueue
	MaxAge      time.Duration `json:"max_age" yaml:"max_age"`
	MaxBytes    int64         `json:"max_bytes" yaml:"max_bytes"`
	MaxMsgs     int64         `json:"max_msgs" yaml:"max_msgs"`
	Storage     string        `json:"storage" yaml:"storage"` // file, memory
	Replicas    int           `json:"replicas" yaml:"replicas"`
	Description string        `json:"description" yaml:"description"`
}

// DefaultJetStreamConfig returns default JetStream configuration
func DefaultJetStreamConfig() *JetStreamConfig {
	return &JetStreamConfig{
		Streams: []StreamConfig{
			{
				Name:      "LLM_EVENTS",
				Subjects:  []string{"llm.>"},
				Retention: "limits",
				MaxAge:    7 * 24 * time.Hour, // 7 days
				Storage:   "file",
				Replicas:  1,
			},
			{
				Name:      "LOG_EVENTS",
				Subjects:  []string{"log.write"},
				Retention: "limits",
				MaxAge:    24 * time.Hour, // 1 day
				Storage:   "file",
				Replicas:  1,
			},
			{
				Name:      "BILLING_EVENTS",
				Subjects:  []string{"billing.>"},
				Retention: "limits",
				MaxAge:    30 * 24 * time.Hour, // 30 days
				Storage:   "file",
				Replicas:  1,
			},
			{
				Name:      "SYNC_EVENTS",
				Subjects:  []string{"sync.>", "chat.>", "user.>"},
				Retention: "limits",
				MaxAge:    3 * 24 * time.Hour, // 3 days
				Storage:   "file",
				Replicas:  1,
			},
		},
	}
}
