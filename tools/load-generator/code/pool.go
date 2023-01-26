package app

import (
	requestbuilder "load-generator/code/request_builder"
	"sync"
)

// pool may be used for pooling bidRequestBuilders.
type pool struct {
	pool sync.Pool
}

// Get returns bidRequestBuilder from pool.
// The bidRequestBuilder must be Put to pool after use.
func (pp *pool) Get() (*requestbuilder.Builder, error) {
	v := pp.pool.Get()
	if v == nil {
		return requestbuilder.New()
	}

	return v.(*requestbuilder.Builder), nil
}

// Put returns p to pp.
func (pp *pool) Put(p *requestbuilder.Builder) {
	pp.pool.Put(p)
}
