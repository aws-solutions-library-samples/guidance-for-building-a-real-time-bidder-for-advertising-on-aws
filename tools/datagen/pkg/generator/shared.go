package generator

import (
	"fmt"
	"sync"

	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

// Default settings for the Config
const (
	DefaultKeyLow        = 1
	DefaultKeyHigh       = 1_000_000
	DefaultMinAudiences  = 10
	DefaultMaxAudiences  = 20
	DefaultMaxAudienceID = 100_000
	DefaultOutput        = OutputStdout
	DefaultConcurrency   = 1
	// $0.10 to $10.00 per 1000 impressions.
	DefaultMinBidPrice int64 = 100_000
	DefaultMaxBidPrice int64 = 10_000_000
	// $500 to $1500 per campaign.
	DefaultMinBudget       int64 = 500_000_000
	DefaultMaxBudget       int64 = 1_500_000_000
	DefaultBudgetBatchSize int   = 170
)

// List of outputs that can be used for generated data
const (
	OutputStdout    = "stdout"
	OutputDynamodb  = "dynamodb"
	OutputAerospike = "aerospike"
)

// List of data types that can be generated
const (
	TypeCampaigns = "campaigns"
	TypeAudiences = "audiences"
	TypeDevices   = "devices"
	TypeBudgets   = "budgets"
)

// KeySize defines number of bytes of a
const KeySize = 16

// Key is an alias to shorten syntax
type Key = []byte

// Record represent a unit of generated data,
// used for constructing general purpose channels
type Record interface{}

// AWSConfig holds AWS SDK related parameters
type AWSConfig struct {
	Region              string `envconfig:"AWS_REGION"`
	DynamodbEndpointURL string `envconfig:"DYNAMODB_ENDPOINT"`
}

// Config stores parameters used for data generation
type Config struct {
	AWSConfig
	Output  string
	Type    string
	KeyLow  uint64
	KeyHigh uint64

	MinAudiences  int
	MaxAudiences  int
	MaxAudienceID uint64

	DynamodbTable       string
	DynamodbConcurrency int

	ASHost             string
	ASPort             int
	ASNamespace        string
	ASBudgetBatchesKey string

	// BudgetBatchSize is the number of individual budgets batched into
	// single database item.
	BudgetBatchSize int

	MinBidPrice int64
	MaxBidPrice int64
	MinBudget   int64
	MaxBudget   int64
}

// Resolve checks and corrects the configuration values
//nolint:gocyclo
func (c *Config) Resolve() {
	if c.KeyLow == 0 {
		c.KeyLow = DefaultKeyLow
	}
	if c.KeyHigh == 0 {
		c.KeyHigh = DefaultKeyHigh
	}
	//nolint:gocritic
	if c.MinAudiences == 0 && c.MaxAudiences == 0 {
		c.MinAudiences = DefaultMinAudiences
		c.MaxAudiences = DefaultMaxAudiences
	} else if c.MinAudiences == 0 {
		if c.MaxAudiences < DefaultMinAudiences {
			c.MinAudiences = c.MaxAudiences
		} else {
			c.MinAudiences = DefaultMinAudiences
		}
	} else if c.MinAudiences > c.MaxAudiences {
		c.MinAudiences = c.MaxAudiences
	}

	if c.MaxAudienceID == 0 {
		c.MaxAudienceID = DefaultMaxAudienceID
	}
	// fix max audience id: generator produces between `MinAudiences` and `MaxAudiences` of unique audience keys,
	// it ensures that there is at least `MaxAudiences` of possible audience ids.
	if c.MaxAudienceID < uint64(c.MaxAudiences) {
		c.MaxAudienceID = uint64(c.MaxAudiences)
	}
	if c.Output == "" {
		c.Output = DefaultOutput
	}
	if c.DynamodbConcurrency < 1 {
		c.DynamodbConcurrency = DefaultConcurrency
	}

	if c.MinBidPrice <= 0 {
		c.MinBidPrice = DefaultMinBidPrice
	}
	if c.MaxBidPrice <= 0 {
		c.MaxBidPrice = DefaultMaxBidPrice
	}
	if c.MaxBidPrice < c.MinBidPrice {
		c.MaxBidPrice = c.MinBidPrice
	}

	if c.MinBudget <= 0 {
		c.MinBudget = DefaultMinBudget
	}
	if c.MaxBudget <= 0 {
		c.MaxBudget = DefaultMaxBudget
	}
	if c.MaxBudget < c.MinBudget {
		c.MaxBudget = c.MinBudget
	}
	if c.BudgetBatchSize == 0 {
		c.BudgetBatchSize = DefaultBudgetBatchSize
	}
}

func workGroup(worker func(*sync.WaitGroup), size int) *sync.WaitGroup {
	wg := sync.WaitGroup{}
	wg.Add(size)
	for i := 0; i < size; i++ {
		go worker(&wg)
	}
	return &wg
}

func writer(in <-chan Record, cfg *Config) error {
	switch cfg.Output {
	case OutputDynamodb:
		return dynamodbWriter(in, cfg)
	case OutputAerospike:
		return aerospikeWriter(in, cfg)
	}
	return fmt.Errorf("unknown output: %v", cfg.Output)
}

func dynamodbWriter(in <-chan Record, cfg *Config) error {
	var av map[string]types.AttributeValue
	var err error

	conn := NewTableConn(cfg.DynamodbTable, &cfg.AWSConfig)

	for b := range in {
		av, err = attributevalue.MarshalMap(b)
		if err != nil {
			return err
		}
		err = conn.PutBuffered(av)
		if err != nil {
			return err
		}
	}

	if err := conn.Close(); err != nil {
		return err
	}

	return nil
}

func aerospikeWriter(in <-chan Record, cfg *Config) error {
	client, err := connectAS(cfg)
	if err != nil {
		return err
	}
	defer client.Close()

	for b := range in {
		err = client.PutRecord(b)
		if err != nil {
			return err
		}
	}

	return nil
}
