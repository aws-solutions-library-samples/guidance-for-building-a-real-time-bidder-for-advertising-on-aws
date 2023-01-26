package device

import (
	"bidder/code/database/api"
	"bidder/code/database/dynamodb"
	"bidder/code/id"
	"context"
	"time"

	"emperror.dev/errors"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/middleware"
	dynamodbV2 "github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/aws/aws-sdk-go/aws/awserr"
	dynamodbV1 "github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/rs/zerolog/log"
)

// Repository allows accessing device database table.
type Repository struct {
	cfg dynamodb.DeviceConfig
	db  *dynamodb.Client

	pool *devicePool
}

// NewRepository initializes new repository.
func NewRepository(cfg dynamodb.DeviceConfig, db *dynamodb.Client) (api.DeviceRepository, error) {
	if err := db.VerifyTable(cfg.DeviceTableName); err != nil {
		return nil, err
	}

	return &Repository{
		cfg: cfg,
		db:  db,

		pool: &devicePool{},
	}, nil
}

// Get reads and returns single device to audiences map entry.
// If requested item does not exist, ErrItemNotFound is returned.
// Result is returned via `result` parameter to avoid heap allocation.
func (r *Repository) Get(deadline time.Time, deviceID id.ID, result *api.Device) error {
	if r.cfg.EnableLowLevelDynamo {
		pd, err := r.pool.Get(r.cfg, r.cfg.DeviceTableName, r.db.AWSConfig)
		if err != nil {
			return errors.Wrap(err, "error initializing low-level query object")
		}
		defer r.pool.Put(pd)

		return pd.lowLevel.GetDevice(deviceID, &result.AudienceIDs)
	}

	ctx, cancel := context.WithDeadline(context.Background(), deadline)
	defer cancel()

	if r.db.Dax != nil {
		return r.getDeviceDAX(ctx, deviceID, result)
	}

	return r.getDeviceDynamoDB(ctx, deviceID, result)
}

func (r *Repository) getDeviceDAX(ctx context.Context, deviceID id.ID, result *api.Device) error {
	getItemInput := dynamodbV1.GetItemInput{
		Key: map[string]*dynamodbV1.AttributeValue{
			api.IDAttributeName: {
				B: deviceID[:],
			},
		},
		TableName: aws.String(r.cfg.DeviceTableName),
	}
	record, err := r.db.Dax.GetItemWithContext(
		ctx,
		&getItemInput,
	)

	if err != nil {
		if awsErr, ok := err.(awserr.Error); ok && awsErr.OrigErr() == context.DeadlineExceeded {
			return api.ErrTimeout
		}

		return err
	}

	if record.Item == nil {
		return api.ErrItemNotFound
	}

	return unmarshallDeviceAttributesV1(record.Item, result)
}

func (r *Repository) getDeviceDynamoDB(ctx context.Context, deviceID id.ID, result *api.Device) error {
	// Start time of the request. We cannot reuse the Prometheus timer here: it won't expose the elapsed time
	// without observing the metric, while we want that only after unmarshalling the response or after any error.
	start := time.Now()

	getItemInput := dynamodbV2.GetItemInput{
		Key: map[string]types.AttributeValue{
			api.IDAttributeName: &types.AttributeValueMemberB{
				Value: deviceID[:],
			},
		},
		TableName: &r.cfg.DeviceTableName,
	}
	record, err := r.db.FastDynamo.GetItem(
		ctx,
		&getItemInput,
	)

	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return api.ErrTimeout
		}

		return err
	}

	dur := time.Since(start)
	if dur >= r.cfg.SlowLogDuration {
		message := log.Info()
		if requestID, ok := middleware.GetRequestIDMetadata(record.ResultMetadata); ok {
			message = message.Str("x-amzn-RequestId", requestID)
		}
		message.Hex("key", deviceID[:]).Dur("duration", dur).Bool("found", record.Item != nil).Msg("")
	}

	if record.Item == nil {
		return api.ErrItemNotFound
	}

	return unmarshallDeviceAttributesV2(record.Item, result)
}
