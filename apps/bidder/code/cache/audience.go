package cache

import (
	"bidder/code/database/api"
	"bidder/code/id"
	"bidder/code/metrics"

	"emperror.dev/errors"
	"github.com/rs/zerolog/log"
)

// Audience contains cached audienceID to campaigns index map.
type Audience struct {
	data map[id.ID][]int
}

// Get gets the campaigns IDs associated with audienceID.
// IDs are passed as strings, as it's not possible to index
// map with byte slices.
func (c *Audience) Get(audienceID id.ID) []int {
	return c.data[audienceID]
}

// GetAll returns all cached audiences.
func (c *Audience) GetAll() map[id.ID][]int {
	return c.data
}

// newAudience initializes Audience cache.
// Map records are downloaded from database.
func newAudience(repository api.AudienceRepository, campaigns *Campaign) (*Audience, error) {
	cache := &Audience{data: map[id.ID][]int{}}
	consumer := func(audience api.Audience) error {
		campaignIndices := make([]int, 0, len(audience.CampaignIDs))
		for _, ID := range audience.CampaignIDs {
			if idx, found := campaigns.GetIndex(ID); found {
				campaignIndices = append(campaignIndices, idx)
			}
		}
		cache.data[audience.AudienceID] = campaignIndices

		return nil
	}

	if err := repository.Scan(consumer); err != nil {
		metrics.DBAudienceScanErrorsN.Inc()
		return nil, errors.Wrap(err, "error while initializing audience cache")
	}

	log.Info().Msgf("cached %d 'audience to campaigns' items", len(cache.data))

	return cache, nil
}
