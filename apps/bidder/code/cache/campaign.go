package cache

import (
	"bidder/code/database/api"
	"bidder/code/id"
	"bidder/code/metrics"

	"emperror.dev/errors"
	"github.com/rs/zerolog/log"
)

// Campaign contains cached campaigns.
type Campaign struct {
	data      []api.Campaign
	idToIndex map[id.ID]int
}

// Get gets the campaign associated with campaign ID.
func (c *Campaign) Get(campaignIndex int) *api.Campaign {
	return &c.data[campaignIndex]
}

// GetAll returns all cached campaigns.
func (c *Campaign) GetAll() []api.Campaign {
	return c.data
}

// GetIndex gets the index associated with campaign ID.
func (c *Campaign) GetIndex(campaignID id.ID) (int, bool) {
	idx, found := c.idToIndex[campaignID]
	return idx, found
}

// Size returns the number of campaigns stored in cache.
func (c *Campaign) Size() int {
	return len(c.data)
}

// newCampaign initializes campaign cache.
func newCampaign(repository api.CampaignRepository) (*Campaign, error) {
	invalid := 0
	cache := &Campaign{idToIndex: map[id.ID]int{}}
	consumer := func(campaign api.Campaign) error {
		if campaign.IsValid() {
			cache.idToIndex[campaign.ID] = len(cache.data)
			cache.data = append(cache.data, campaign)
		} else {
			invalid++
		}

		return nil
	}

	if err := repository.Scan(consumer); err != nil {
		metrics.DBCampaignScanErrorsN.Inc()
		return nil, errors.Wrap(err, "error while initializing campaign cache")
	}

	if invalid != 0 {
		log.Info().Msgf("discarded %d invalid 'campaign' items", invalid)
	}
	log.Info().Msgf("cached %d 'campaign' items", len(cache.data))

	return cache, nil
}
