package campaign

import (
	"encoding/hex"

	"bidder/code/database/api"
	"bidder/code/database/dynamodb"

	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/rs/zerolog/log"
)

// Repository allows accessing campaign database table.
type Repository struct {
	cfg dynamodb.CampaignConfig
	db  *dynamodb.Client
}

// NewRepository initializes new Repository.
func NewRepository(cfg dynamodb.CampaignConfig, db *dynamodb.Client) (*Repository, error) {
	if err := db.VerifyTable(cfg.TableName); err != nil {
		return nil, err
	}

	return &Repository{
		cfg: cfg,
		db:  db,
	}, nil
}

// Scan scans all campaigns items stored in the database.
func (r *Repository) Scan(consume func(api.Campaign) error) error {
	log.Info().Msgf("downloading '%s' table...", r.cfg.TableName)

	consumeAttributeMap := func(item dynamodb.AttributeMapV2) error {
		campaign := api.Campaign{}
		if err := attributevalue.UnmarshalMap(item, &campaign); err != nil {
			return err
		}

		campaign.HexID = hex.EncodeToString(campaign.ID[:])

		return consume(campaign)
	}

	return r.db.ScanDynamo(r.cfg.TableName, dynamodb.EventuallyConsistent, consumeAttributeMap)
}
