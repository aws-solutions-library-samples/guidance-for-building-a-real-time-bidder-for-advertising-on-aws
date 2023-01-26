package stream

import (
	"bidder/code/stream/producer"
	"context"
	"time"

	"github.com/ClearcodeHQ/aws-sdk-go/aws"
	awsRequest "github.com/ClearcodeHQ/aws-sdk-go/aws/request"
	awsSession "github.com/ClearcodeHQ/aws-sdk-go/aws/session"
	awsKinesis "github.com/ClearcodeHQ/aws-sdk-go/service/kinesis"
)

// Stream is a service used to stream bidrequests and bidresponses.
type Stream struct {
	kinesis *awsKinesis.Kinesis
	cfg     Config

	producer *producer.Producer
}

// NewStream creates a new stream instance.
func NewStream(cfg Config) (*Stream, error) {
	if cfg.Disable {
		return &Stream{
			kinesis:  nil,
			cfg:      cfg,
			producer: nil,
		}, nil
	}

	session, err := awsSession.NewSession(&aws.Config{
		Endpoint: aws.String(cfg.KinesisEndpoint),
		LogLevel: aws.LogLevel(aws.LogLevelType(cfg.AWSLogLevel)),
	})

	if err != nil {
		return nil, err
	}
	client := awsKinesis.New(session)

	// We use a single producer for both requests and responses (forcing the use of a single stream),
	// since we don't need multiple streams now and the library wouldn't export Prometheus metrics for more than
	// one producer.
	kinesisProducer := producer.New(cfg.Producer, client)

	if err := kinesisProducer.Start(); err != nil {
		return nil, err
	}

	return &Stream{
		kinesis:  client,
		cfg:      cfg,
		producer: kinesisProducer,
	}, nil
}

// Close stops producer goroutines, flushing buffered data.
func (s *Stream) Close() {
	if s.producer == nil {
		return
	}
	s.producer.Close()
}

// WaitUntilStreamsExist waits until streams are initialized.
func (s *Stream) WaitUntilStreamsExist() error {
	if s.kinesis == nil {
		return nil
	}

	if err := s.waitUntilStreamsExist(s.cfg.Producer.StreamName); err != nil {
		return err
	}

	return nil
}

// PutRequest publishes bidrequest to the stream.
func (s *Stream) PutRequest(data []byte) error {
	if s.producer == nil {
		return nil
	}

	return s.producer.Put(data)
}

// PutResponse publishes bidresponse to the stream.
func (s *Stream) PutResponse(data []byte) error {
	if s.producer == nil {
		return nil
	}

	return s.producer.Put(data)
}

func (s *Stream) waitUntilStreamsExist(streamName string) error {
	ctx, cancel := context.WithTimeout(context.Background(), s.cfg.OperationTimeout)
	defer cancel()

	// Setting maxRetry to a large value, so it
	// doesn't interrupt wait before context times out.
	const maxRetry = 4294967295

	return s.kinesis.WaitUntilStreamExistsWithContext(
		ctx,
		&awsKinesis.DescribeStreamInput{
			StreamName: aws.String(streamName),
		},
		awsRequest.WithWaiterMaxAttempts(maxRetry),
		awsRequest.WithWaiterDelay(awsRequest.ConstantWaiterDelay(time.Second)))
}
