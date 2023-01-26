package server

import (
	"time"
)

// Config is a struct for holding http server configuration.
type Config struct {
	Address         string `envconfig:"SERVER_ADDRESS" required:"true"`
	BidRequestPath  string `envconfig:"SERVER_BIDREQUEST_PATH" required:"true"`
	HealthCheckPath string `envconfig:"SERVER_HEALTHCHECK_PATH" required:"true"`

	ReadTimeout  time.Duration `envconfig:"SERVER_READ_TIMEOUT" required:"true"`
	WriteTimeout time.Duration `envconfig:"SERVER_WRITE_TIMEOUT" required:"true"`
	IdleTimeout  time.Duration `envconfig:"SERVER_IDLE_TIMEOUT" required:"true"`

	LogAllErrors bool `envconfig:"SERVER_LOG_ALL_FASTHTTP_ERRORS" required:"true"`
}
