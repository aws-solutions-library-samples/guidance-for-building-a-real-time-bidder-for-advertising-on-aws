package dynamodb

import (
	"context"
	"sync"
	"time"

	"bidder/code/database/api"

	"emperror.dev/errors"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/rs/zerolog/log"
)

// AttributeMapV1 is type alias for AWS SDK v1 attribute value map.
type AttributeMapV1 = map[string]*dynamodb.AttributeValue

// ScanDAX reads and returns all items in a Client table using DAX API.
func (d *Client) ScanDAX(tableName string, consistent bool, consume func(AttributeMapV1) error) error {
	return d.scan(tableName, consistent, consume, d.scanWorkerDAX)
}

// scanWorkerDAX scans single segment of db table using DAX API.
func (d *Client) scanWorkerDAX(
	tableName string,
	consistent bool,
	segment int,
	totalSegments int,
	consumerMutex sync.Locker,
	consume interface{},
) error {
	itemsRead := 0
	exclusiveStartKey := map[string]*dynamodb.AttributeValue(nil)

	start := time.Now()
	defer func() { log.Trace().Int("items", itemsRead).Dur("worker time", time.Since(start)).Msg("") }()

	// Repeat scan until all items from the segment are read.
	for {
		ctx, cancel := context.WithTimeout(context.Background(), d.cfg.OperationTimeout)

		scanInput := &dynamodb.ScanInput{
			Segment:                aws.Int64(int64(segment)),
			TotalSegments:          aws.Int64(int64(totalSegments)),
			ExclusiveStartKey:      exclusiveStartKey,
			ReturnConsumedCapacity: aws.String("NONE"),
			TableName:              aws.String(tableName),
			ConsistentRead:         aws.Bool(consistent),
		}

		response, err := d.Dax.ScanWithContext(ctx, scanInput)
		cancel()
		if err != nil {
			if errors.Is(err, context.DeadlineExceeded) {
				return errors.WithDetails(api.ErrTimeout, "table name", tableName)
			}

			return err
		}

		if err := callConsumerV1(response.Items, consumerMutex, consume.(func(AttributeMapV1) error)); err != nil {
			return err
		}

		itemsRead += len(response.Items)

		exclusiveStartKey = response.LastEvaluatedKey
		if exclusiveStartKey == nil {
			return nil
		}
	}
}

// callConsumerV1 passes scanned item to consume callback using AWS SDK v1 format.
func callConsumerV1(
	items []AttributeMapV1,
	consumerMutex sync.Locker,
	consume func(AttributeMapV1) error,
) error {
	consumerMutex.Lock()
	defer consumerMutex.Unlock()

	for _, item := range items {
		if err := consume(item); err != nil {
			return err
		}
	}

	return nil
}
