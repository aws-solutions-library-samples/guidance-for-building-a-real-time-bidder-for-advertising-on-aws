package bidhandler

import (
	"bidder/code/auction"
	"bidder/code/ksuid"
	"sync"

	"github.com/rs/zerolog/log"
	"github.com/valyala/fastjson"
)

// pool may be used for pooling handler data structures.
type pool struct {
	pool sync.Pool
}

// persistentData contains data structures that are to persist
// between handler calls, lessening malloc and GC load.
type persistentData struct {
	parser        *fastjson.Parser
	ksuidSequence *ksuid.Sequence

	auctionRequest auction.Request
	byteResponse   []byte
}

// Get returns persistentData from pool.
// The persistentData must be Put to pool after use.
func (pp *pool) Get() *persistentData {
	v := pp.pool.Get()
	if v == nil {
		log.Trace().Msg("allocating bidhandler persistent data")
		return newPersistentData()
	}

	return v.(*persistentData)
}

// Put returns p to pp.
func (pp *pool) Put(p *persistentData) {
	pp.pool.Put(p)
}

func newPersistentData() *persistentData {
	return &persistentData{
		parser:        &fastjson.Parser{},
		ksuidSequence: ksuid.NewSequence(),
	}
}
