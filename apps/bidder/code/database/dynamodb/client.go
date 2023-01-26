package dynamodb

import (
	"context"
	"net/http"

	"bidder/code/database/api"
	"bidder/code/metrics"

	"emperror.dev/errors"
	"github.com/aws/aws-dax-go/dax"
	awsV2 "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/retry"
	awshttp "github.com/aws/aws-sdk-go-v2/aws/transport/http"
	configV2 "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go/aws"
	awsSession "github.com/aws/aws-sdk-go/aws/session"
	"github.com/prometheus/client_golang/prometheus"
)

// Client is a service used to communicate with DynamoDB.
type Client struct {
	// DynamoDB client for non-performance-critical queries performed during application startup. Use it when
	// reliability is more important than speed and there is no need to cache the data in DAX.
	dynamo *dynamodb.Client
	// DynamoDB client for use during bid request handling (with a different configuration); nil if DAX is
	// enabled. Use it (or DAX) if speed is more important and it's better to get a temporary error quickly than
	// slowly retrying for a complete answer.
	FastDynamo *dynamodb.Client
	// Optional DAX client: use FastDynamo if nil, otherwise use instead of FastDynamo.
	Dax *dax.Dax

	cfg       Config
	AWSConfig awsV2.Config
}

// NewClient creates a new DynamoDB Client service instance.
func NewClient(cfg Config) (*Client, error) {
	httpClient := awshttp.NewBuildableClient().WithTransportOptions(func(tr *http.Transport) {
		tr.MaxIdleConnsPerHost = cfg.MaxIdleConnsPerHost
	})

	cfgV2, err := configV2.LoadDefaultConfig(context.Background(),
		configV2.WithEndpointResolver(awsV2.EndpointResolverFunc(func(service, region string) (awsV2.Endpoint, error) {
			if cfg.DynamodbEndpoint != "" && service == dynamodb.ServiceID && region == cfg.AWSRegion {
				return awsV2.Endpoint{
					PartitionID:   "aws",
					URL:           cfg.DynamodbEndpoint,
					SigningRegion: cfg.AWSRegion,
					Source:        awsV2.EndpointSourceCustom,
				}, nil
			}
			return awsV2.Endpoint{}, &awsV2.EndpointNotFoundError{}
		})),
		configV2.WithClientLogMode(awsV2.ClientLogMode(cfg.AWSLogLevel)),
		configV2.WithHTTPClient(httpClient),
	)
	if err != nil {
		return nil, err
	}

	var daxClient *dax.Dax
	var fastDynamo *dynamodb.Client

	if cfg.DAXEnable && cfg.DAXEndpoint != "" {
		transport := http.Transport{
			MaxIdleConnsPerHost: cfg.MaxIdleConnsPerHost,
		}
		client := http.Client{
			Transport: &transport,
		}
		session, err := awsSession.NewSession(&aws.Config{
			Endpoint:   aws.String(cfg.DynamodbEndpoint),
			DisableSSL: aws.Bool(cfg.DisableSSL),
			LogLevel:   aws.LogLevel(aws.LogLevelType(cfg.AWSLogLevel)),
			HTTPClient: &client,
		})
		if err != nil {
			return nil, err
		}

		daxConfig := dax.DefaultConfig()
		daxConfig.RequestTimeout = cfg.DAXRequestTimeout
		daxConfig.ReadRetries = cfg.DAXReadRetries
		daxConfig.HostPorts = []string{cfg.DAXEndpoint}
		daxConfig.Region = *session.Config.Region
		daxConfig.Credentials = session.Config.Credentials
		daxConfig.LogLevel = aws.LogLevelType(cfg.DAXLogLevel)
		daxClient, err = dax.New(daxConfig)
		if err != nil {
			return nil, err
		}
	} else {
		fastDynamo = dynamodb.NewFromConfig(cfgV2,
			func(o *dynamodb.Options) {
				if cfg.MaxRetries != -1 {
					o.Retryer = retry.AddWithMaxAttempts(retry.NewStandard(), cfg.MaxRetries)
				}
				o.EndpointOptions.DisableHTTPS = cfg.DisableSSL
				o.DisableValidateResponseChecksum = cfg.DisableComputeChecksums
			},
		)
	}

	return &Client{
		dynamo: dynamodb.NewFromConfig(cfgV2,
			func(o *dynamodb.Options) {
				if cfg.ScanMaxRetries != -1 {
					o.Retryer = retry.AddWithMaxAttempts(retry.NewStandard(), cfg.ScanMaxRetries)
				}
				o.EndpointOptions.DisableHTTPS = cfg.DisableSSL
				o.DisableValidateResponseChecksum = cfg.DisableComputeChecksums
			},
		),
		FastDynamo: fastDynamo,
		Dax:        daxClient,
		cfg:        cfg,
		AWSConfig:  cfgV2,
	}, nil
}

// VerifyTable checks if a dynamoDB table exists and can be accessed.
func (d *Client) VerifyTable(tableName string) error {
	metric, err := metrics.DatabaseTime.GetMetricWithLabelValues("DescribeTable", tableName)
	if err != nil {
		return errors.Wrap(err, "getting Prometheus metric")
	}
	timer := prometheus.NewTimer(metric)
	defer timer.ObserveDuration()

	ctx, cancel := context.WithTimeout(context.Background(), d.cfg.OperationTimeout)
	defer cancel()

	input := &dynamodb.DescribeTableInput{TableName: &tableName}

	_, err = d.dynamo.DescribeTable(ctx, input)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return errors.WithDetails(api.ErrTimeout, "table name", tableName)
		}

		return err
	}

	return nil
}
