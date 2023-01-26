package producer

import "time"

// Config is a struct for holding Kinesis configuration.
type Config struct {
	// StreamName is the name of the Kinesis stream where bid requests and responses are written.
	StreamName     string        `envconfig:"KINESIS_STREAM_NAME" required:"true"`
	MaxConnections int           `envconfig:"KINESIS_MAX_CONNECTIONS" required:"true"`
	FlushInterval  time.Duration `envconfig:"KINESIS_FLUSH_INTERVAL" required:"true"`
}
