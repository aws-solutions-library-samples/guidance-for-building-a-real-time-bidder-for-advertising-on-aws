package aerospike

import (
	"encoding/hex"

	"bidder/code/database/api"
	"bidder/code/id"

	"emperror.dev/errors"
	"github.com/rs/zerolog/log"
)

// CampaignRepository allows accessing campaign Client set.
type CampaignRepository struct {
	aerospike *Client
}

// NewCampaignRepository creates new instance of *AudienceRepository
func NewCampaignRepository(aerospikeClient *Client) (*CampaignRepository, error) {
	return &CampaignRepository{aerospikeClient}, nil
}

// Scan scans all campaigns entries stored in the database.
func (r *CampaignRepository) Scan(consume func(api.Campaign) error) error {
	log.Info().Msg("downloading campaign data")

	recordsChan, err := r.aerospike.ScanAll(CampaignSet)
	if err != nil {
		return err
	}

	for res := range recordsChan {
		if res.Err != nil {
			return errors.Wrap(res.Err, "error during scanning campaign set")
		}

		var campaignIDPlain = res.Record.Key.Value().GetObject().([]byte)

		var campaignID id.ID
		copy(campaignID[:], campaignIDPlain)

		err := consume(api.Campaign{
			ID:     campaignID,
			MaxCPM: int64(res.Record.Bins["bid_price"].(int)),
			HexID:  hex.EncodeToString(campaignIDPlain),
		})
		if err != nil {
			return errors.Wrap(err, "error during consuming")
		}
	}

	return nil
}
