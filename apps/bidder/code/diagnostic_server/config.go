package diagnosticserver

import "time"

// Config is a struct for holding diagnostic server configuration.
type Config struct {
	Address         string        `envconfig:"DIAGNOSTIC_SERVER_ADDRESS" required:"true"`
	MetricsPath     string        `envconfig:"DIAGNOSTIC_SERVER_METRICS_PATH" required:"true"`
	ProfilerPath    string        `envconfig:"DIAGNOSTIC_SERVER_PROFILER_PATH" required:"true"`
	TracePath       string        `envconfig:"DIAGNOSTIC_SERVER_TRACE_PATH" required:"true"`
	ShutdownTimeout time.Duration `envconfig:"DIAGNOSTIC_SERVER_SHUTDOWN_TIMEOUT" required:"true"`
}
