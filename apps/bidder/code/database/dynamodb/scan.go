package dynamodb

import (
	"sync"

	"bidder/code/metrics"

	"emperror.dev/errors"
	"github.com/prometheus/client_golang/prometheus"
)

const (
	// Consistent database read
	Consistent = true

	// EventuallyConsistent database read
	EventuallyConsistent = false
)

// scan reads and returns all items in a Client table.
// Needs to be parametrized with worker function using specific database API (DynamoDB or DAX).
func (d *Client) scan(
	tableName string,
	consistent bool,
	consume interface{},
	scanWorker func(string, bool, int, int, sync.Locker, interface{}) error,
) error {
	metric, err := metrics.DatabaseTime.GetMetricWithLabelValues("FetchAll", tableName)
	if err != nil {
		return errors.Wrap(err, "getting Prometheus metric")
	}
	timer := prometheus.NewTimer(metric)
	defer timer.ObserveDuration()

	workers := d.cfg.ScanWorkers
	lastError := error(nil)

	// Run scan in parallel.
	consumerMutex := &sync.Mutex{}
	wg := sync.WaitGroup{}
	wg.Add(workers)

	for i := 0; i < workers; i++ {
		go func(segment int) {
			err := scanWorker(tableName, consistent, segment, workers, consumerMutex, consume)
			if err != nil {
				lastError = err
			}
			wg.Done()
		}(i)
	}

	wg.Wait()

	return lastError
}
