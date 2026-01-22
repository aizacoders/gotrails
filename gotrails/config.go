package gotrails

// Config holds the configuration for gotrails
type Config struct {
	// Service identification
	ServiceName string
	Environment string

	// Trace header configuration
	TraceIDHeader   string
	RequestIDHeader string

	// Body size limits
	MaxRequestBodySize  int
	MaxResponseBodySize int

	// Masking configuration
	MaskFields    []string
	MaskValue     string
	EnableMasking bool

	// Header filtering
	ExcludeHeaders []string
	IncludeHeaders []string

	// Sink configuration
	EnableAsync    bool
	AsyncQueueSize int
}

// DefaultConfig returns the default configuration
func DefaultConfig() *Config {
	return &Config{
		ServiceName:         "unknown-service",
		Environment:         "development",
		TraceIDHeader:       "X-Trace-ID",
		RequestIDHeader:     "X-Request-ID",
		MaxRequestBodySize:  64 * 1024, // 64KB
		MaxResponseBodySize: 64 * 1024, // 64KB
		MaskFields: []string{
			"password",
			"token",
			"secret",
			"api_key",
			"apikey",
			"authorization",
			"credit_card",
			"creditcard",
			"cvv",
			"pin",
		},
		MaskValue:     "***MASKED***",
		EnableMasking: true,
		ExcludeHeaders: []string{
			"authorization",
			"cookie",
			"set-cookie",
			"x-api-key",
		},
		IncludeHeaders: nil, // nil means include all (except excluded)
		EnableAsync:    true,
		AsyncQueueSize: 1000,
	}
}

// ConfigOption is a function that modifies Config
type ConfigOption func(*Config)

// WithServiceName sets the service name
func WithServiceName(name string) ConfigOption {
	return func(c *Config) {
		c.ServiceName = name
	}
}

// WithEnvironment sets the environment
func WithEnvironment(env string) ConfigOption {
	return func(c *Config) {
		c.Environment = env
	}
}

// WithTraceIDHeader sets the trace ID header name
func WithTraceIDHeader(header string) ConfigOption {
	return func(c *Config) {
		c.TraceIDHeader = header
	}
}

// WithRequestIDHeader sets the request ID header name
func WithRequestIDHeader(header string) ConfigOption {
	return func(c *Config) {
		c.RequestIDHeader = header
	}
}

// WithMaxRequestBodySize sets the max request body size
func WithMaxRequestBodySize(size int) ConfigOption {
	return func(c *Config) {
		c.MaxRequestBodySize = size
	}
}

// WithMaxResponseBodySize sets the max response body size
func WithMaxResponseBodySize(size int) ConfigOption {
	return func(c *Config) {
		c.MaxResponseBodySize = size
	}
}

// WithMaskFields sets the fields to mask
func WithMaskFields(fields []string) ConfigOption {
	return func(c *Config) {
		c.MaskFields = fields
	}
}

// WithMaskValue sets the mask replacement value
func WithMaskValue(value string) ConfigOption {
	return func(c *Config) {
		c.MaskValue = value
	}
}

// WithMaskingEnabled enables or disables masking
func WithMaskingEnabled(enabled bool) ConfigOption {
	return func(c *Config) {
		c.EnableMasking = enabled
	}
}

// WithExcludeHeaders sets headers to exclude from logging
func WithExcludeHeaders(headers []string) ConfigOption {
	return func(c *Config) {
		c.ExcludeHeaders = headers
	}
}

// WithIncludeHeaders sets specific headers to include (whitelist)
func WithIncludeHeaders(headers []string) ConfigOption {
	return func(c *Config) {
		c.IncludeHeaders = headers
	}
}

// WithAsyncEnabled enables or disables async processing
func WithAsyncEnabled(enabled bool) ConfigOption {
	return func(c *Config) {
		c.EnableAsync = enabled
	}
}

// WithAsyncQueueSize sets the async queue size
func WithAsyncQueueSize(size int) ConfigOption {
	return func(c *Config) {
		c.AsyncQueueSize = size
	}
}

// NewConfig creates a new Config with the given options
func NewConfig(opts ...ConfigOption) *Config {
	cfg := DefaultConfig()
	for _, opt := range opts {
		opt(cfg)
	}
	return cfg
}
