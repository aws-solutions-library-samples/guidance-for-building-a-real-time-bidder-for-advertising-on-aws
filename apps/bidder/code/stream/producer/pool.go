package producer

import (
	"sync"

	"github.com/ClearcodeHQ/aws-sdk-go/service/kinesis"
)

// batchPool may be used for pooling batch data structures.
type batchPool struct {
	pool sync.Pool
}

// get returns AggregatedRecord from pool.
// The AggregatedRecord must be put to pool after use.
func (pp *batchPool) get() *batch {
	v := pp.pool.Get()
	if v == nil {
		return &batch{}
	}

	return v.(*batch)
}

// put returns p to pool.
func (pp *batchPool) put(p *batch) {
	p.reset()

	pp.pool.Put(p)
}

// entryPool may be used for pooling kinesis.PutRecordsRequestEntry data structures.
type entryPool struct {
	pool sync.Pool
}

// get returns AggregatedRecord from pool.
// The AggregatedRecord must be put to pool after use.
func (pp *entryPool) get() *kinesis.PutRecordsRequestEntry {
	v := pp.pool.Get()
	if v == nil {
		return &kinesis.PutRecordsRequestEntry{}
	}

	return v.(*kinesis.PutRecordsRequestEntry)
}

// put returns p to pool.
func (pp *entryPool) put(p *kinesis.PutRecordsRequestEntry) {
	p.Data = p.Data[:0]
	p.PartitionKey = nil

	pp.pool.Put(p)
}
