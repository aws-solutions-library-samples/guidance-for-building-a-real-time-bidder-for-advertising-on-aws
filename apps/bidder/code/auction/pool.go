package auction

import (
	"bidder/code/database/api"

	"math/rand"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
)

// pool may be used for pooling auction data structures.
type pool struct {
	pool sync.Pool
}

// persistentData contains data structures that are to persist
// between auction calls, lessening malloc and GC load.
type persistentData struct {
	device          api.Device
	campaignIndices []int
	winningAuctions []int
	rng             *rand.Rand
}

// get returns persistentData from pool.
// The persistentData must be put to pool after use.
func (pp *pool) get() *persistentData {
	v := pp.pool.Get()
	if v == nil {
		log.Trace().Msg("allocating auction persistent data")
		return newPersistentData()
	}

	return v.(*persistentData)
}

// put returns p to pool.
func (pp *pool) put(p *persistentData) {
	pp.pool.Put(p)
}

func newPersistentData() *persistentData {
	return &persistentData{
		rng: rand.New(rand.NewSource(time.Now().Unix())),
	}
}

// campaignLookup is table used to unique a set of campaign IDs.
type campaignLookup struct {
	lookup []int
	runID  int
}

// campaignLookupPool may be used for pooling campaignLookup tables.
// campaignLookup are kept in a separate pool to reduce memory usage,
// as the tables are large but can be owned by auction for only a short
// amount of time.
type campaignLookupPool struct {
	pool sync.Pool
}

// get returns campaignLookup from pool.
// The campaignLookup must be put to pool after use.
func (pp *campaignLookupPool) get() *campaignLookup {
	v := pp.pool.Get()
	if v == nil {
		log.Trace().Msg("allocating auction campaign lookup table")
		return &campaignLookup{}
	}

	return v.(*campaignLookup)
}

// put returns p to pool.
func (pp *campaignLookupPool) put(p *campaignLookup) {
	pp.pool.Put(p)
}
