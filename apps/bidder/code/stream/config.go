package stream

import (
	"bidder/code/stream/producer"
	"time"
)

// Config is a struct for holding Kinesis configuration.
type Config struct {
	KinesisEndpoint  string        `envconfig:"KINESIS_ENDPOINT" required:"true"`
	OperationTimeout time.Duration `envconfig:"KINESIS_OPERATION_TIMEOUT" required:"true"`
	Disable          bool          `envconfig:"KINESIS_DISABLE" required:"true"`
	AWSLogLevel      uint          `envconfig:"KINESIS_LOG_LEVEL" required:"true"`

	Producer producer.Config
}
