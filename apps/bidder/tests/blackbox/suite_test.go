package blackbox

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsConfig "github.com/aws/aws-sdk-go-v2/config"
	awsKinesis "github.com/aws/aws-sdk-go-v2/service/kinesis"
	"github.com/aws/aws-sdk-go-v2/service/kinesis/types"
	"github.com/kelseyhightower/envconfig"
	"github.com/stretchr/testify/suite"
)

type blackboxSuite struct {
	suite.Suite
	cfg     config
	kinesis *awsKinesis.Client
}

func (s *blackboxSuite) SetupSuite() {
	err := envconfig.Process("", &s.cfg)
	s.Require().NoError(err)

	s.setupStreams()

	s.Require().Eventually(s.waitForBidder, time.Second*60, time.Second)
}

// TestBlackboxSuite runs testing suite
func TestBlackboxSuite(t *testing.T) {
	suite.Run(t, new(blackboxSuite))
}

func (s *blackboxSuite) waitUntilStreamsExist(streamName string) error {
	waiter := awsKinesis.NewStreamExistsWaiter(s.kinesis, func(o *awsKinesis.StreamExistsWaiterOptions) {
		o.MinDelay = time.Second
		o.MaxDelay = s.cfg.App.Stream.OperationTimeout
	})
	return waiter.Wait(context.Background(), &awsKinesis.DescribeStreamInput{
		StreamName: aws.String(streamName),
	}, s.cfg.App.Stream.OperationTimeout)
}

func (s *blackboxSuite) setupStreams() {
	cfg, err := awsConfig.LoadDefaultConfig(context.Background(),
		// Pass endpoint (for localstack) via our environment variables. SDK defaults obtain other credentials
		// from environment variables or other sources if these aren't set.
		awsConfig.WithEndpointResolver(aws.EndpointResolverFunc(func(service, region string) (aws.Endpoint, error) {
			return aws.Endpoint{
				PartitionID:   "aws",
				URL:           s.cfg.App.Stream.KinesisEndpoint,
				SigningRegion: region,
				Source:        aws.EndpointSourceCustom,
			}, nil
		})),
	)
	s.Require().NoError(err)
	s.kinesis = awsKinesis.NewFromConfig(cfg)
	s.Require().NoError(s.waitUntilStreamsExist(s.cfg.App.Stream.Producer.StreamName))
}

func (s *blackboxSuite) getStreamRecords(streamName string) []types.Record {
	kc := s.kinesis

	description, err := kc.DescribeStream(
		context.Background(),
		&awsKinesis.DescribeStreamInput{StreamName: aws.String(streamName)},
	)
	s.Require().NoError(err)

	iteratorOutput, err := kc.GetShardIterator(
		context.Background(),
		&awsKinesis.GetShardIteratorInput{
			ShardId:           description.StreamDescription.Shards[0].ShardId,
			ShardIteratorType: types.ShardIteratorTypeTrimHorizon,
			StreamName:        &streamName,
		})
	s.Require().NoError(err)

	records, err := kc.GetRecords(
		context.Background(),
		&awsKinesis.GetRecordsInput{
			ShardIterator: iteratorOutput.ShardIterator,
		})
	s.Require().NoError(err)

	return records.Records
}

// waitForBidder waits for bidder to start by requesting healthcheck endpoint.
func (s *blackboxSuite) waitForBidder() bool {
	client := &http.Client{}

	req, err := http.NewRequest(
		"GET",
		s.cfg.BidderHost+s.cfg.App.Server.Address+s.cfg.App.Server.HealthCheckPath,
		nil,
	)
	s.Require().NoError(err)

	resp, err := client.Do(req)
	if err != nil {
		return false
	}

	s.Require().NoError(resp.Body.Close())
	return resp.StatusCode == http.StatusOK
}
