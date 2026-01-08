package observability

import (
	"context"
	"os"

	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// LogConfig represents logging configuration
type LogConfig struct {
	// Level is the log level (debug, info, warn, error)
	Level string `json:"level" yaml:"level"`

	// Format is the log format (json, console)
	Format string `json:"format" yaml:"format"`

	// Output is the log output (stdout, stderr, file path)
	Output string `json:"output" yaml:"output"`

	// Development enables development mode (more verbose)
	Development bool `json:"development" yaml:"development"`

	// AddCaller adds caller information to logs
	AddCaller bool `json:"add_caller" yaml:"add_caller"`

	// Sampling enables log sampling
	Sampling *SamplingConfig `json:"sampling" yaml:"sampling"`
}

// SamplingConfig represents log sampling configuration
type SamplingConfig struct {
	Initial    int `json:"initial" yaml:"initial"`
	Thereafter int `json:"thereafter" yaml:"thereafter"`
}

// DefaultLogConfig returns default logging configuration
func DefaultLogConfig() *LogConfig {
	return &LogConfig{
		Level:       "info",
		Format:      "json",
		Output:      "stdout",
		Development: false,
		AddCaller:   true,
		Sampling: &SamplingConfig{
			Initial:    100,
			Thereafter: 100,
		},
	}
}

// InitLogger initializes a zap logger
func InitLogger(cfg *LogConfig) (*zap.Logger, error) {
	if cfg == nil {
		cfg = DefaultLogConfig()
	}

	// Parse log level
	var level zapcore.Level
	if err := level.UnmarshalText([]byte(cfg.Level)); err != nil {
		level = zapcore.InfoLevel
	}

	// Create encoder config
	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "time",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		FunctionKey:    zapcore.OmitKey,
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	// Create encoder
	var encoder zapcore.Encoder
	if cfg.Format == "console" {
		encoder = zapcore.NewConsoleEncoder(encoderConfig)
	} else {
		encoder = zapcore.NewJSONEncoder(encoderConfig)
	}

	// Create output
	var output zapcore.WriteSyncer
	switch cfg.Output {
	case "stdout":
		output = zapcore.AddSync(os.Stdout)
	case "stderr":
		output = zapcore.AddSync(os.Stderr)
	default:
		file, err := os.OpenFile(cfg.Output, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return nil, err
		}
		output = zapcore.AddSync(file)
	}

	// Create core
	core := zapcore.NewCore(encoder, output, level)

	// Apply sampling if enabled
	if cfg.Sampling != nil {
		core = zapcore.NewSamplerWithOptions(
			core,
			1, // tick duration (seconds)
			cfg.Sampling.Initial,
			cfg.Sampling.Thereafter,
		)
	}

	// Build logger options
	opts := []zap.Option{}
	if cfg.AddCaller {
		opts = append(opts, zap.AddCaller())
	}
	if cfg.Development {
		opts = append(opts, zap.Development())
	}

	return zap.New(core, opts...), nil
}

// ContextLogger returns a logger with trace context
func ContextLogger(ctx context.Context, logger *zap.Logger) *zap.Logger {
	span := trace.SpanFromContext(ctx)
	if span.SpanContext().HasTraceID() {
		return logger.With(
			zap.String("trace_id", span.SpanContext().TraceID().String()),
			zap.String("span_id", span.SpanContext().SpanID().String()),
		)
	}
	return logger
}

// WithTraceID adds trace ID to logger
func WithTraceID(logger *zap.Logger, traceID string) *zap.Logger {
	return logger.With(zap.String("trace_id", traceID))
}

// WithUserID adds user ID to logger
func WithUserID(logger *zap.Logger, userID string) *zap.Logger {
	return logger.With(zap.String("user_id", userID))
}

// WithRequestID adds request ID to logger
func WithRequestID(logger *zap.Logger, requestID string) *zap.Logger {
	return logger.With(zap.String("request_id", requestID))
}

// WithService adds service name to logger
func WithService(logger *zap.Logger, service string) *zap.Logger {
	return logger.With(zap.String("service", service))
}

// LogFields represents common log fields
type LogFields struct {
	TraceID   string
	SpanID    string
	UserID    string
	RequestID string
	Service   string
	Platform  string
	Provider  string
	Model     string
}

// ToZapFields converts LogFields to zap fields
func (f *LogFields) ToZapFields() []zap.Field {
	fields := make([]zap.Field, 0, 8)
	if f.TraceID != "" {
		fields = append(fields, zap.String("trace_id", f.TraceID))
	}
	if f.SpanID != "" {
		fields = append(fields, zap.String("span_id", f.SpanID))
	}
	if f.UserID != "" {
		fields = append(fields, zap.String("user_id", f.UserID))
	}
	if f.RequestID != "" {
		fields = append(fields, zap.String("request_id", f.RequestID))
	}
	if f.Service != "" {
		fields = append(fields, zap.String("service", f.Service))
	}
	if f.Platform != "" {
		fields = append(fields, zap.String("platform", f.Platform))
	}
	if f.Provider != "" {
		fields = append(fields, zap.String("provider", f.Provider))
	}
	if f.Model != "" {
		fields = append(fields, zap.String("model", f.Model))
	}
	return fields
}
