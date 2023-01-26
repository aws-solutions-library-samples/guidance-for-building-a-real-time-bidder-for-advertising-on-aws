package cache

import (
	"bidder/code/database/api"

	"runtime"
)

// Cache is a collection of database caches.
type Cache struct {
	Device   *Device
	Audience *Audience
	Campaign *Campaign
	Budget   *Budget

	cfg Config
}

// New initializes new collection of database caches.
func New(
	cfg Config,
	audienceRepo api.AudienceRepository,
	budgetRepo api.BudgetRepository,
	campaignRepo api.CampaignRepository,
	deviceRepo api.DeviceRepository,
) (*Cache, error) {
	campaignCache, err := newCampaign(campaignRepo)
	if err != nil {
		return nil, err
	}
	runtime.GC()

	audienceCache, err := newAudience(audienceRepo, campaignCache)
	if err != nil {
		return nil, err
	}
	runtime.GC()

	budgetCache, err := newBudget(cfg, budgetRepo, campaignCache)
	if err != nil {
		return nil, err
	}
	runtime.GC()

	c := &Cache{
		Device:   newDevice(cfg, deviceRepo, audienceCache),
		Audience: audienceCache,
		Campaign: campaignCache,
		Budget:   budgetCache,
		cfg:      cfg,
	}
	runtime.GC()

	c.start()

	return c, nil
}

// start periodic cache jobs, like syncing with databases.
func (c *Cache) start() {
	c.Budget.start()
}

// Stop all scheduled jobs.
func (c *Cache) Stop() {
	c.Budget.stop()
}
