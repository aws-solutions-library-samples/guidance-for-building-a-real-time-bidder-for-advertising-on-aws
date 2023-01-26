package dynamodb

import (
	"context"
	"sync"
	"time"

	"bidder/code/database/api"

	"emperror.dev/errors"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/rs/zerolog/log"
)

// AttributeMapV2 is type alias for AWS SDK v2 attribute value map.
type AttributeMapV2 = map[string]types.AttributeValue

// ScanDynamo reads and returns all items in a Client table using DynamoDB API.
func (d *Client) ScanDynamo(tableName string, consistent bool, consume func(AttributeMapV2) error) error {
	return d.scan(tableName, consistent, consume, d.scanWorkerDynamo)
}

// scanWorkerDynamo scans single segment of db table using DynamoDB API.
func (d *Client) scanWorkerDynamo(
	tableName string,
	consistent bool,
	segment int,
	totalSegments int,
	consumerMutex sync.Locker,
	consume interface{},
) error {
	itemsRead := 0
	exclusiveStartKey := map[string]types.AttributeValue(nil)

	start := time.Now()
	defer func() { log.Trace().Int("items", itemsRead).Dur("worker time", time.Since(start)).Msg("") }()

	// Repeat scan until all items from the segment are read.
	for {
		ctx, cancel := context.WithTimeout(context.Background(), d.cfg.OperationTimeout)

		scanInput := &dynamodb.ScanInput{
			Segment:                aws.Int32(int32(segment)),
			TotalSegments:          aws.Int32(int32(totalSegments)),
			ExclusiveStartKey:      exclusiveStartKey,
			ReturnConsumedCapacity: types.ReturnConsumedCapacityNone,
			TableName:              aws.String(tableName),
			ConsistentRead:         aws.Bool(consistent),
		}

		response, err := d.dynamo.Scan(ctx, scanInput)
		cancel()
		if err != nil {
			if errors.Is(err, context.DeadlineExceeded) {
				return errors.WithDetails(api.ErrTimeout, "table name", tableName)
			}

			return err
		}

		if err := callConsumerV2(response.Items, consumerMutex, consume.(func(AttributeMapV2) error)); err != nil {
			return err
		}

		itemsRead += len(response.Items)

		exclusiveStartKey = response.LastEvaluatedKey
		if exclusiveStartKey == nil {
			return nil
		}
	}
}

// callConsumerV2 passes scanned item to consume callback using AWS SDK v1 format.
func callConsumerV2(
	items []AttributeMapV2,
	consumerMutex sync.Locker,
	consume func(AttributeMapV2) error,
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
