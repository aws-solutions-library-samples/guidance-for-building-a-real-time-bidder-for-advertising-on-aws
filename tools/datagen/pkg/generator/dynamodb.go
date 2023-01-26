package generator

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/retry"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

const (
	bufferTimeout     = 1_000 * time.Millisecond
	listTableLimit    = 10
	putManyBatchLimit = 25
)

// DynamoTableConn represents a connection to the DynamoDB table.
// It's bound to a concrete table name, and contains own buffer for writes.
type DynamoTableConn struct {
	conn         *dynamodb.Client
	table        *string
	buffer       []map[string]types.AttributeValue
	bufferMu     *sync.Mutex
	bufferTicker *time.Ticker
}

// NewTableConn constructs an instance of `DynamoTableConn`
func NewTableConn(table string, cfg *AWSConfig) *DynamoTableConn {
	awsConfig, err := config.LoadDefaultConfig(context.Background(),
		config.WithEndpointResolver(aws.EndpointResolverFunc(func(service, region string) (aws.Endpoint, error) {
			if cfg.DynamodbEndpointURL == "" {
				return aws.Endpoint{}, &aws.EndpointNotFoundError{}
			}
			return aws.Endpoint{
				PartitionID:   "aws",
				URL:           cfg.DynamodbEndpointURL,
				SigningRegion: region,
				Source:        aws.EndpointSourceCustom,
			}, nil
		})),
		config.WithRegion(cfg.Region),
	)
	if err != nil {
		log.Fatal(err)
	}

	conn := &DynamoTableConn{
		conn: dynamodb.NewFromConfig(awsConfig,
			func(o *dynamodb.Options) {
				// Have a high maximum number of retries.
				o.Retryer = retry.NewStandard(func(o *retry.StandardOptions) {
					o.MaxAttempts = 10
				})
			},
		),
		table:        &table,
		buffer:       []map[string]types.AttributeValue{},
		bufferMu:     &sync.Mutex{},
		bufferTicker: time.NewTicker(bufferTimeout),
	}

	//nolint
	go func() {
		for range conn.bufferTicker.C {
			if err := conn.flushBuffer(); err != nil {
				fmt.Printf("error: %v\n", err)
			}
		}
	}()

	return conn
}

// Close finalizes the use of the table connection
func (c *DynamoTableConn) Close() error {
	return c.flushBuffer()
}

// DescribeTable calls the DynamoDB for the table description
func (c *DynamoTableConn) DescribeTable() (*types.TableDescription, error) {
	res, err := c.conn.DescribeTable(context.Background(), &dynamodb.DescribeTableInput{TableName: c.table})
	if err != nil {
		return nil, err
	}
	return res.Table, nil
}

// CreateTable calls the DynamoDB to create a table with specified partition key,
// and a table name bound in the table connection
func (c *DynamoTableConn) CreateTable(pk string, pkType types.ScalarAttributeType) (*types.TableDescription, error) {
	if pk == "" {
		return nil, errors.New("dynamodb: partition key is required to create a table")
	}
	if pkType == "" {
		pkType = types.ScalarAttributeTypeB
	}
	attrDefs := []types.AttributeDefinition{
		{AttributeName: &pk, AttributeType: pkType},
	}

	keySchema := []types.KeySchemaElement{
		{AttributeName: &pk, KeyType: types.KeyTypeHash},
	}
	res, err := c.conn.CreateTable(context.Background(), &dynamodb.CreateTableInput{
		AttributeDefinitions:   attrDefs,
		BillingMode:            types.BillingModePayPerRequest,
		GlobalSecondaryIndexes: nil,
		KeySchema:              keySchema,
		LocalSecondaryIndexes:  nil,
		ProvisionedThroughput:  nil,
		SSESpecification:       nil,
		StreamSpecification:    nil,
		TableName:              c.table,
		Tags:                   nil,
	})
	if err != nil {
		return nil, err
	}
	return res.TableDescription, nil
}

// DeleteTable calls the DynamoDB to delete a table
func (c *DynamoTableConn) DeleteTable() (*types.TableDescription, error) {
	res, err := c.conn.DeleteTable(context.Background(), &dynamodb.DeleteTableInput{TableName: c.table})
	if err != nil {
		return nil, err
	}
	return res.TableDescription, nil
}

// ListTable calls the DynamoDB to list items in a table
func (c *DynamoTableConn) ListTable(limit int32) (*dynamodb.ScanOutput, error) {
	if limit < 1 {
		limit = listTableLimit
	}

	return c.conn.Scan(context.Background(), &dynamodb.ScanInput{
		Limit:                  &limit,
		ReturnConsumedCapacity: types.ReturnConsumedCapacityTotal,
		TableName:              c.table,
	})
}

// putMany calls the DynamoDB to write a list of items
func (c *DynamoTableConn) putMany(items []map[string]types.AttributeValue) (*dynamodb.BatchWriteItemOutput, error) {
	writeRequests := make([]types.WriteRequest, len(items))
	for i, item := range items {
		writeRequests[i] = types.WriteRequest{PutRequest: &types.PutRequest{Item: item}}
	}
	requestItems := map[string][]types.WriteRequest{
		*(c.table): writeRequests,
	}
	return c.conn.BatchWriteItem(context.Background(), &dynamodb.BatchWriteItemInput{
		RequestItems:                requestItems,
		ReturnConsumedCapacity:      types.ReturnConsumedCapacityNone,
		ReturnItemCollectionMetrics: types.ReturnItemCollectionMetricsNone,
	})
}

// PutBuffered stores the item in a buffer before write.
// The buffer is written to the DynamoDB once it reaches several times the maximal size of a batch,
// or after defined timeout period since last call.
func (c *DynamoTableConn) PutBuffered(item map[string]types.AttributeValue) error {
	c.addToBuffer(item)
	c.bufferTicker.Reset(bufferTimeout)
	if c.bufferSize() >= putManyBatchLimit {
		return c.flushBatchFromBuffer()
	}
	return nil
}

func (c *DynamoTableConn) bufferSize() int {
	c.bufferMu.Lock()
	defer c.bufferMu.Unlock()

	return len(c.buffer)
}

func (c *DynamoTableConn) addToBuffer(item map[string]types.AttributeValue) {
	c.bufferMu.Lock()
	defer c.bufferMu.Unlock()

	c.buffer = append(c.buffer, item)
}

func (c *DynamoTableConn) flushBuffer() error {
	for c.bufferSize() > 0 {
		if err := c.flushBatchFromBuffer(); err != nil {
			return err
		}
	}
	return nil
}

func (c *DynamoTableConn) flushBatchFromBuffer() error {
	c.bufferMu.Lock()
	defer c.bufferMu.Unlock()

	if len(c.buffer) == 0 {
		return nil
	}

	cut := putManyBatchLimit
	if len(c.buffer) < putManyBatchLimit {
		cut = len(c.buffer)
	}

	out, err := c.putMany(c.buffer[:cut])
	if err != nil {
		return err
	}
	c.buffer = c.buffer[cut:]

	for key, value := range out.UnprocessedItems {
		if key != *c.table {
			fmt.Printf("unexpected table unprocessed items: %v %v\n", key, value)
			continue
		}
		// If a put request failed due to throttling or internal errors, retry it with the next batch.
		for _, item := range value {
			c.buffer = append(c.buffer, item.PutRequest.Item)
		}
	}

	return err
}
