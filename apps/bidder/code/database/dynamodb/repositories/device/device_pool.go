package device

import (
	"sync"

	"bidder/code/database/dynamodb"
	lowlevel "bidder/code/database/dynamodb/low_level"

	"github.com/aws/aws-sdk-go-v2/aws"
)

// devicePool may be used for pooling low-level query objects.
type devicePool struct {
	pool sync.Pool
}

// persistentData contains data structures that are to persist
// between query calls, lessening malloc and GC load.
type persistentData struct {
	lowLevel *lowlevel.LowLevel
}

// Get returns persistentData from devicePool.
// The persistentData must be Put to devicePool after use.
func (pp *devicePool) Get(cfg dynamodb.DeviceConfig, tableName string, awsCfg aws.Config) (*persistentData, error) {
	v := pp.pool.Get()
	if v == nil {
		return newPersistentData(cfg, tableName, awsCfg)
	}

	return v.(*persistentData), nil
}

// Put returns p to pp.
func (pp *devicePool) Put(p *persistentData) {
	pp.pool.Put(p)
}

func newPersistentData(cfg dynamodb.DeviceConfig, tableName string, awsCfg aws.Config) (*persistentData, error) {
	lowLevel, err := lowlevel.New(cfg.LowLevel, tableName, awsCfg)
	if err != nil {
		return nil, err
	}

	return &persistentData{
		lowLevel: lowLevel,
	}, nil
}
