package audience

import (
	"bidder/code/database/api"
	"bidder/code/database/dynamodb"

	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/rs/zerolog/log"
)

// Repository allows accessing audience database table.
type Repository struct {
	cfg dynamodb.AudienceConfig
	db  *dynamodb.Client
}

// NewRepository initializes new Repository.
func NewRepository(cfg dynamodb.AudienceConfig, db *dynamodb.Client) (*Repository, error) {
	if err := db.VerifyTable(cfg.TableName); err != nil {
		return nil, err
	}

	return &Repository{
		cfg: cfg,
		db:  db,
	}, nil
}

// Scan scans all audience to
// campaigns entries stored in the database.
func (r *Repository) Scan(consume func(api.Audience) error) error {
	log.Info().Msgf("downloading '%s' table...", r.cfg.TableName)

	consumeAttributeMap := func(item dynamodb.AttributeMapV2) error {
		audience := api.Audience{}
		if err := attributevalue.UnmarshalMap(item, &audience); err != nil {
			return err
		}
		return consume(audience)
	}

	return r.db.ScanDynamo(r.cfg.TableName, dynamodb.EventuallyConsistent, consumeAttributeMap)
}
