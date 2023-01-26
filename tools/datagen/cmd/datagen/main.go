package main

import (
	"datagen/pkg/generator"
	"flag"
	"fmt"
	"os"

	"github.com/kelseyhightower/envconfig"
)

// allowed operations
const (
	// generate data for the database
	generateData = "generate-data"
	// clear aerospike set
	clearSet = "clear-set"
)

var (
	operation = flag.String(
		"operation", generateData,
		fmt.Sprintf("one of %v", []string{generateData, clearSet}),
	)
	output = flag.String(
		"output", generator.OutputStdout,
		fmt.Sprintf("one of %v", []string{generator.OutputStdout, generator.OutputDynamodb, generator.OutputAerospike}),
	)
	dataType = flag.String("type", "", fmt.Sprintf("data type to generate: one of %v", []string{
		generator.TypeCampaigns,
		generator.TypeAudiences,
		generator.TypeDevices,
		generator.TypeBudgets,
	}))
	low  = flag.Uint64("low", generator.DefaultKeyLow, "lowest id of generated object")
	high = flag.Uint64("high", generator.DefaultKeyHigh, "highest id of generated object")

	dynamodbTable       = flag.String("table", "", "table name for DynamoDB output")
	dynamodbConcurrency = flag.Int(
		"concurrency", generator.DefaultConcurrency,
		"number of concurrent DynamoDB writers",
	)

	minAudiences  = flag.Int("min-audiences", generator.DefaultMinAudiences, "min number of audiences per device/campaign")
	maxAudiences  = flag.Int("max-audiences", generator.DefaultMaxAudiences, "max number of audiences per device/campaign")
	maxAudienceID = flag.Uint64(
		"max-audience-id", generator.DefaultMaxAudienceID,
		"max audience id (number of unique audience ids)",
	)

	asHost             = flag.String("aerospike-host", "aerospike", "Aerospike host")
	asPort             = flag.Int("aerospike-port", 3000, "Aerospike port")
	asNamespace        = flag.String("aerospike-namespace", "test", "Aerospike namespace")
	asBudgetBatchesKey = flag.String("aerospike-budget-batches-key", "budget_batches_keys", "Aerospike budget batches key")

	minBidPrice = flag.Int64("min-bid-price", generator.DefaultMinBidPrice, "min campaign bid price in microCPM")
	maxBidPrice = flag.Int64("max-bid-price", generator.DefaultMaxBidPrice, "max campaign bid price in microCPM")

	minBudget       = flag.Int64("min-budget", generator.DefaultMinBudget, "min campaign budget in microCPM")
	maxBudget       = flag.Int64("max-budget", generator.DefaultMaxBudget, "max campaign budget in microCPM")
	budgetBatchSize = flag.Int("budget-batch-size", 170, "Number of individual budgets batched into a single database item")
)

func handleError(err error) {
	fmt.Printf("datagen: %v\n\n", err)
	flag.PrintDefaults()
	os.Exit(1)
}

//nolint: gocyclo,funlen
func buildConfig() (*generator.Config, error) {
	var err error
	var cfg generator.Config

	if err = envconfig.Process("", &cfg); err != nil {
		handleError(err)
	}

	flag.Parse()

	if output == nil || *output == "" {
		return nil, fmt.Errorf("error: missing output")
	}
	switch *output {
	case generator.OutputStdout:
		cfg.Output = *output
	case generator.OutputDynamodb:
		cfg.Output = *output
		if *dynamodbTable == "" {
			return nil, fmt.Errorf("error: missing table name")
		}
		cfg.DynamodbTable = *dynamodbTable
		cfg.DynamodbConcurrency = *dynamodbConcurrency
	case generator.OutputAerospike:
		cfg.Output = *output
		cfg.ASHost = *asHost
		cfg.ASPort = *asPort
		cfg.ASNamespace = *asNamespace
		cfg.ASBudgetBatchesKey = *asBudgetBatchesKey

	default:
		return nil, fmt.Errorf("error: invalid output %s", *output)
	}

	if operation != nil && *operation == clearSet {
		cfg.Type = clearSet
		return &cfg, nil
	}

	if dataType == nil || *dataType == "" {
		return nil, fmt.Errorf("error: missing data type")
	}
	switch *dataType {
	case generator.TypeCampaigns:
		cfg.Type = *dataType
		cfg.MinBidPrice = *minBidPrice
		cfg.MaxBidPrice = *maxBidPrice
	case generator.TypeAudiences:
		cfg.Type = *dataType
		cfg.MinAudiences = *minAudiences
		cfg.MaxAudiences = *maxAudiences
		cfg.MaxAudienceID = *maxAudienceID
	case generator.TypeDevices:
		cfg.Type = *dataType
		cfg.MinAudiences = *minAudiences
		cfg.MaxAudiences = *maxAudiences
		cfg.MaxAudienceID = *maxAudienceID
	case generator.TypeBudgets:
		cfg.Type = *dataType
		cfg.MinBudget = *minBudget
		cfg.MaxBudget = *maxBudget
	default:
		return nil, fmt.Errorf("error: invalid data type %s", *dataType)
	}

	if low == nil || high == nil {
		return nil, fmt.Errorf("error: missing id range")
	}

	cfg.KeyLow = *low
	cfg.KeyHigh = *high
	cfg.BudgetBatchSize = *budgetBatchSize

	cfg.Resolve()

	return &cfg, nil
}

func main() {
	cfg, err := buildConfig()
	if err != nil {
		handleError(err)
	}

	switch cfg.Type {
	case generator.TypeCampaigns:
		if err := generator.GenerateCampaigns(cfg); err != nil {
			handleError(err)
		}
	case generator.TypeAudiences:
		if err := generator.GenerateAudiences(cfg); err != nil {
			handleError(err)
		}
	case generator.TypeDevices:
		if err := generator.GenerateDevices(cfg); err != nil {
			handleError(err)
		}
	case generator.TypeBudgets:
		if err := generator.GenerateBudgets(cfg); err != nil {
			handleError(err)
		}
	case clearSet:
		if err := generator.ClearBudgetBatches(cfg); err != nil {
			handleError(err)
		}
	default:
		handleError(fmt.Errorf("error: invalid data type %s", *dataType))
	}
}
