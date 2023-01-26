package dynamodb

import (
	lowlevel "bidder/code/database/dynamodb/low_level"

	"time"
)

// Config holds DynamoDB related parameters
type Config struct {
	DynamodbEndpoint        string        `envconfig:"DYNAMODB_ENDPOINT" required:"true"`
	OperationTimeout        time.Duration `envconfig:"DYNAMODB_OPERATION_TIMEOUT" required:"true"`
	ScanWorkers             int           `envconfig:"DYNAMODB_SCAN_WORKERS" required:"true"`
	DisableSSL              bool          `envconfig:"DYNAMODB_DISABLE_SSL" required:"true"`
	AWSLogLevel             uint          `envconfig:"DYNAMODB_LOG_LEVEL" required:"true"`
	DisableParamValidation  bool          `envconfig:"DYNAMODB_DISABLE_PARAM_VALIDATION" required:"true"`
	DisableComputeChecksums bool          `envconfig:"DYNAMODB_DISABLE_COMPUTE_CHECKSUMS" required:"true"`
	MaxRetries              int           `envconfig:"DYNAMODB_MAX_RETRIES" required:"true"`
	ScanMaxRetries          int           `envconfig:"DYNAMODB_SCAN_MAX_RETRIES" required:"true"`
	MaxIdleConnsPerHost     int           `envconfig:"DYNAMODB_MAX_IDLE_CONNS_PER_HOST" required:"true"`
	AWSRegion               string        `envconfig:"AWS_REGION" required:"true"`

	DAXEnable         bool          `envconfig:"DAX_ENABLE" required:"true"`
	DAXEndpoint       string        `envconfig:"DAX_ENDPOINT" required:"true"`
	DAXRequestTimeout time.Duration `envconfig:"DAX_REQUEST_TIMEOUT" required:"true"`
	DAXReadRetries    int           `envconfig:"DAX_READ_RETRIES" required:"true"`
	DAXLogLevel       uint          `envconfig:"DAX_LOG_LEVEL" required:"true"`

	DeviceRepository   DeviceConfig
	AudienceRepository AudienceConfig
	CampaignRepository CampaignConfig
	BudgetRepository   BudgetConfig
}

// DeviceConfig holds device table related parameters
type DeviceConfig struct {
	DeviceTableName string        `envconfig:"DYNAMODB_DEVICE_TABLE" required:"true"`
	SlowLogDuration time.Duration `envconfig:"DYNAMODB_SLOW_LOG_DURATION" required:"true"`

	EnableLowLevelDynamo bool `envconfig:"DYNAMODB_ENABLE_LOW_LEVEL" required:"true"`
	LowLevel             lowlevel.Config
}

// AudienceConfig holds audience table related parameters
type AudienceConfig struct {
	TableName string `envconfig:"DYNAMODB_AUDIENCE_TABLE" required:"true"`
}

// CampaignConfig holds campaign table related parameters
type CampaignConfig struct {
	TableName string `envconfig:"DYNAMODB_CAMPAIGN_TABLE" required:"true"`
}

// BudgetConfig holds budget table related parameters
type BudgetConfig struct {
	TableName string `envconfig:"DYNAMODB_BUDGET_TABLE" required:"true"`
}
