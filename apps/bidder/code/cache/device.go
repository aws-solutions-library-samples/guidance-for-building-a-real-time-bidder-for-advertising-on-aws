package cache

import (
	"math/rand"
	"sync"
	"time"

	"bidder/code/database/api"
	"bidder/code/id"
	"bidder/code/metrics"
)

// Device is responsible for providing device ID to audiences mapping.
type Device struct {
	repository api.DeviceRepository
	audiences  []id.ID

	cfg     Config
	rngPool *sync.Pool
}

// newDevice initializes device cache.
func newDevice(cfg Config, repository api.DeviceRepository, audiences *Audience) *Device {
	c := &Device{
		repository: repository,
		cfg:        cfg,
	}

	if cfg.DeviceQueryDisable {
		dbAudiences := audiences.GetAll()
		c.audiences = make([]id.ID, 0, len(dbAudiences))
		for audienceID := range dbAudiences {
			c.audiences = append(c.audiences, audienceID)
		}

		c.rngPool = &sync.Pool{}
	}

	return c
}

// Get gets the audiences associated with device ID.
// Result is returned via `result` parameter to avoid heap allocation.
func (c *Device) Get(deadline time.Time, deviceID id.ID, result *api.Device) error {
	if deviceID == id.ZeroID {
		return api.ErrItemNotFound
	}

	if !c.cfg.DeviceQueryDisable {
		err := c.repository.Get(deadline, deviceID, result)
		if err != nil {
			metrics.DBDeviceGetErrorsN.Inc()
		}
		return err
	}

	// Generate random response if device query is disabled.
	c.getStub(result)
	return nil
}

// getStub returns randomly generated device if device
// query is disabled for benchmarking purposes.
func (c *Device) getStub(result *api.Device) {
	if c.cfg.MockDeviceQueryDelay != 0 {
		time.Sleep(c.cfg.MockDeviceQueryDelay)
	}

	if len(c.audiences) == 0 {
		return
	}

	// Get RNG from the pool.
	r := (*rand.Rand)(nil)
	v := c.rngPool.Get()
	if v == nil {
		r = rand.New(rand.NewSource(time.Now().Unix()))
	} else {
		r = v.(*rand.Rand)
	}
	defer c.rngPool.Put(r)

	if r.Float64() < c.cfg.MockDeviceNoBidFraction {
		// Returning no audiences to cause NoBid.
		return
	}

	const audienceNumber = 5
	for i := 0; i < audienceNumber; i++ {
		result.AudienceIDs = append(result.AudienceIDs, c.audiences[r.Intn(len(c.audiences))])
	}
}
