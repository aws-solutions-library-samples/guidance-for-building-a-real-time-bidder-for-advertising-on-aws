package lowlevel

import "time"

// Config is a struct for holding low-level device query configuration.
type Config struct {
	// Setting used to terminate request in case of timeout.
	// fasthttp doesn't terminate timed-out requests even if client.DoWithDeadline() method returns because of timeout.
	// A separate client setting is required to terminate such requests. BIDREQUEST_TIMEOUT is a reasonable timeout value
	// for hanging requests.
	ReadWriteTimeout time.Duration `envconfig:"BIDREQUEST_TIMEOUT" required:"true"`
	MaxConnsPerHost  int           `envconfig:"DYNAMODB_LOW_LEVEL_MAX_CONNECTIONS" required:"true"`
}
