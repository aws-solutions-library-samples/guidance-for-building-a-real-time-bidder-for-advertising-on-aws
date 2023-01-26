package aerospike

import (
	"bidder/code/database/api"
	"bidder/code/id"

	"emperror.dev/errors"
	"github.com/rs/zerolog/log"
)

// AudienceRepository allows accessing audience Aerospike set.
type AudienceRepository struct {
	aerospike *Client
}

// NewAudienceRepository creates new instance of *AudienceRepository
func NewAudienceRepository(aerospikeClient *Client) (*AudienceRepository, error) {
	return &AudienceRepository{aerospikeClient}, nil
}

// Scan scans all audience to
// campaigns entries stored in the database.
func (r *AudienceRepository) Scan(consume func(api.Audience) error) error {
	log.Info().Msg("downloading audience data")

	recordsChan, err := r.aerospike.ScanAll(AudienceSet)
	if err != nil {
		return err
	}

	for res := range recordsChan {
		if res.Err != nil {
			return errors.Wrap(res.Err, "error during scanning audience_campaigns set")
		}

		keyValue := res.Record.Key.Value()
		if keyValue == nil {
			continue
		}

		var audienceID id.ID
		copy(audienceID[:], keyValue.GetObject().([]byte))

		campaignIdsBin := res.Record.Bins["campaign_ids"]
		if campaignIdsBin == nil {
			continue
		}

		campaignIdsPlain := campaignIdsBin.([]interface{})

		var campaignIds = make([]id.ID, len(campaignIdsPlain))
		for i, cID := range campaignIdsPlain {
			copy(campaignIds[i][:], cID.([]byte))
		}

		err := consume(api.Audience{
			AudienceID:  audienceID,
			CampaignIDs: campaignIds,
		})
		if err != nil {
			return errors.Wrap(err, "error during consuming")
		}
	}

	return nil
}
