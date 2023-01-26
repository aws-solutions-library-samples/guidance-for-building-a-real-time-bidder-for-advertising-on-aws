package diagnosticserver

import "time"

// Config is a struct for holding diagnostic server configuration.
type Config struct {
	Address         string        `envconfig:"DIAGNOSTIC_SERVER_ADDRESS" default:":8092"`
	ProfilerPath    string        `envconfig:"DIAGNOSTIC_SERVER_PROFILER_PATH" default:"/debug/pprof"`
	TracePath       string        `envconfig:"DIAGNOSTIC_SERVER_TRACE_PATH" default:"/debug/trace"`
	ShutdownTimeout time.Duration `envconfig:"DIAGNOSTIC_SERVER_SHUTDOWN_TIMEOUT" default:"2s"`
}
