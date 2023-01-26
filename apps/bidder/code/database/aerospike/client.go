package aerospike

import (
	"bidder/code/database/api"
	"bidder/code/metrics"
	"time"

	"emperror.dev/errors"
	as "github.com/aerospike/aerospike-client-go"
	asTypes "github.com/aerospike/aerospike-client-go/types"
	"github.com/prometheus/client_golang/prometheus"
)

const (
	// DeviceSet is an name of aerospike set for devices
	DeviceSet = "device"
	// AudienceSet is an name of aerospike set for audience and campaign mapping
	AudienceSet = "audience_campaigns"
	// CampaignSet is an name of aerospike set for campaigns
	CampaignSet = "campaign"
	// BudgetSet is an name of aerospike set for budgets
	BudgetSet = "budget"
	// BudgetBatchesSet is a name of aerospike set for budget batches keys
	BudgetBatchesSet = "budget_batches"
)

const maxRetriesDefault = 2

// Config holds Aerospike Client config parameters
type Config struct {
	Host                    string        `envconfig:"AEROSPIKE_HOST" required:"true"`
	Port                    int           `envconfig:"AEROSPIKE_PORT" required:"true"`
	Namespace               string        `envconfig:"AEROSPIKE_NAMESPACE" required:"true"`
	WarmUpCount             int           `envconfig:"AEROSPIKE_WARM_UP_COUNT" required:"true"`
	ScanTotalTimeout        time.Duration `envconfig:"AEROSPIKE_SCAN_TOTAL_TIMEOUT" required:"true"`
	ScanMaxRetries          int           `envconfig:"AEROSPIKE_SCAN_MAX_RETRIES" required:"true"`
	ScanPriority            int           `envconfig:"AEROSPIKE_SCAN_PRIORITY" required:"true"`
	ScanConcurrentNodes     bool          `envconfig:"AEROSPIKE_SCAN_CONCURRENT_NODES" required:"true"`
	ScanSleepBetweenRetries time.Duration `envconfig:"AEROSPIKE_SCAN_SLEEP_BETWEEN_RETRIES" required:"true"`
	ScanSleepMultiplier     float64       `envconfig:"AEROSPIKE_SCAN_SLEEP_MULTIPLIER" required:"true"`
	GetTotalTimeout         time.Duration `envconfig:"BIDREQUEST_TIMEOUT" required:"true"`
	GetMaxRetries           int           `envconfig:"AEROSPIKE_GET_MAX_RETRIES" required:"true"`
	GetPriority             int           `envconfig:"AEROSPIKE_GET_PRIORITY" required:"true"`
	ClientLogLevel          string        `envconfig:"AEROSPIKE_CLIENT_LOG_LEVEL" required:"true"`
	DisableScan             bool          `envconfig:"AEROSPIKE_DISABLE_SCAN" required:"true"`
	BudgetBatchesKey        string        `envconfig:"AEROSPIKE_BUDGET_BATCHES_KEY" default:"budget_batches_keys"`
	BudgetGetTotalTimeout   time.Duration `envconfig:"AEROSPIKE_BUDGET_GET_TIMEOUT" required:"true"`
}

// Client allows to interact with Aerospike database
type Client struct {
	Client    *as.Client
	Namespace string
	config    Config
}

// NewAerospike creates new Aerospike Client instance
func NewAerospike(client *as.Client, cfg Config) *Client {
	return &Client{
		Client:    client,
		Namespace: cfg.Namespace,
		config:    cfg,
	}
}

// Close closes connection to Aerospike
func (c *Client) Close() {
	c.Client.Close()
}

// ScanAll scans all records within a set
func (c *Client) ScanAll(setName string) (<-chan *as.Result, error) {
	metric, err := metrics.DatabaseTime.GetMetricWithLabelValues("Scan", setName)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create Prometheus metric")
	}
	defer prometheus.NewTimer(metric).ObserveDuration()

	records, err := c.Client.ScanAll(c.scanPolicy(), c.config.Namespace, setName)
	if err != nil {
		return nil, errors.Wrapf(err, "error during scanning set '%s'", setName)
	}

	return records.Results(), nil
}

// Get returns records from given key
func (c *Client) Get(key *as.Key, policy *as.BasePolicy) (*as.Record, error) {
	record, err := c.Client.Get(policy, key)
	if err != nil {
		asErr, ok := err.(asTypes.AerospikeError)

		switch {
		// Aerospike package exports sentinel errors. Unfortunately
		// it doesn't always return them, instead returning ad-hoc error values.
		// That's why we can't use AS sentinel errors.
		case ok && asErr.ResultCode() == asTypes.TIMEOUT:
			return nil, api.ErrTimeout
		case ok && asErr.ResultCode() == asTypes.KEY_NOT_FOUND_ERROR:
			return nil, api.ErrItemNotFound
		default:
			return nil, err
		}
	}

	return record, nil
}

// WriteKey stores key and bins
func (c *Client) WriteKey(key *as.Key, bins ...*as.Bin) error {
	policy := as.NewWritePolicy(0, 0)
	policy.SendKey = true
	return c.Client.PutBins(policy, key, bins...)
}

func (c *Client) scanPolicy() *as.ScanPolicy {
	scanPolicy := as.NewScanPolicy()
	scanPolicy.IncludeBinData = true
	scanPolicy.ConcurrentNodes = c.config.ScanConcurrentNodes

	scanPolicy.Priority = as.HIGH
	if as.Priority(c.config.ScanPriority) != as.HIGH {
		scanPolicy.Priority = as.Priority(c.config.ScanPriority)
	}

	scanPolicy.MaxRetries = maxRetriesDefault
	if c.config.ScanMaxRetries != maxRetriesDefault {
		scanPolicy.MaxRetries = c.config.ScanMaxRetries
	}

	scanPolicy.TotalTimeout = 0
	if c.config.ScanTotalTimeout != 0 {
		scanPolicy.TotalTimeout = c.config.ScanTotalTimeout
	}

	if c.config.ScanSleepBetweenRetries != 0 {
		scanPolicy.SleepBetweenRetries = c.config.ScanSleepBetweenRetries
	}

	if c.config.ScanSleepMultiplier != 0 {
		scanPolicy.SleepMultiplier = c.config.ScanSleepMultiplier
	}

	return scanPolicy
}

// GetPolicy builds a GET policy based on the configuration passed to the client.
func (c *Client) GetPolicy() *as.BasePolicy {
	policy := as.NewPolicy()

	policy.Priority = as.HIGH
	if as.Priority(c.config.GetPriority) != as.HIGH {
		policy.Priority = as.Priority(c.config.GetPriority)
	}

	policy.MaxRetries = maxRetriesDefault
	if c.config.GetMaxRetries != maxRetriesDefault {
		policy.MaxRetries = c.config.GetMaxRetries
	}

	policy.TotalTimeout = 0
	if c.config.GetTotalTimeout != 0 {
		policy.TotalTimeout = c.config.GetTotalTimeout
	}

	return policy
}
