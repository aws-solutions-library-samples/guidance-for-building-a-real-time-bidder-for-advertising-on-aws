package loadgenerator

import (
	"time"

	"github.com/caio/go-tdigest"
)

// Metrics holds metrics computed out of a slice of Results which are used
// in some of the Reporters
type Metrics struct {
	// Latencies holds computed request latency metrics.
	Latencies LatencyMetrics `json:"latencies"`
	// Histogram, only if requested
	Histogram *Histogram `json:"buckets,omitempty"`
	// BytesIn holds computed incoming byte metrics.
	BytesIn ByteMetrics `json:"bytes_in"`
	// BytesOut holds computed outgoing byte metrics.
	BytesOut ByteMetrics `json:"bytes_out"`
	// Earliest is the earliest timestamp in a Result set.
	Earliest time.Time `json:"earliest"`
	// Latest is the latest timestamp in a Result set.
	Latest time.Time `json:"latest"`
	// End is the latest timestamp in a Result set plus its latency.
	End time.Time `json:"end"`
	// Duration is the duration of the attack.
	Duration time.Duration `json:"duration"`
	// Wait is the extra time waiting for responses from targets.
	Wait time.Duration `json:"wait"`
	// Requests is the total number of requests executed.
	Requests uint64 `json:"requests"`
	// Rate is the rate of sent requests per second.
	Rate float64 `json:"rate"`
	// Throughput is the rate of successful requests per second.
	Throughput float64 `json:"throughput"`
	// Success is the percentage of non-error responses.
	Success float64 `json:"success"`
	// StatusCodes is a histogram of the responses' status codes.
	StatusCodes map[int]int `json:"status_codes"`
	// Errors is a set of unique errors returned by the targets during the attack.
	Errors []string `json:"errors"`

	errors  map[string]struct{}
	success uint64
}

// NewMetrics initialize new metrics.
func NewMetrics() (*Metrics, error) {
	latencies, err := NewLatencyMetrics()
	if err != nil {
		return nil, err
	}

	return &Metrics{
		Latencies:   *latencies,
		StatusCodes: map[int]int{},
		errors:      map[string]struct{}{},
		Errors:      make([]string, 0),
	}, nil
}

// Add implements the Add method of the Report interface by adding the given
// Result to Metrics.
func (m *Metrics) Add(r *Result) error {
	m.Requests++
	m.StatusCodes[int(r.Code)]++
	m.BytesOut.Total += r.BytesOut
	m.BytesIn.Total += r.BytesIn

	if err := m.Latencies.Add(r.Latency); err != nil {
		return err
	}

	if m.Earliest.IsZero() || m.Earliest.After(r.Timestamp) {
		m.Earliest = r.Timestamp
	}

	if r.Timestamp.After(m.Latest) {
		m.Latest = r.Timestamp
	}

	if end := r.End(); end.After(m.End) {
		m.End = end
	}

	if r.Code >= 200 && r.Code < 400 {
		m.success++
	}

	if r.Error != "" {
		if _, ok := m.errors[r.Error]; !ok {
			m.errors[r.Error] = struct{}{}
			m.Errors = append(m.Errors, r.Error)
		}
	}

	if m.Histogram != nil {
		if err := m.Histogram.Add(r); err != nil {
			return err
		}
	}

	return nil
}

// Merge implements the Merge method of the Report interface by adding the given
// Result to Metrics.
func (m *Metrics) Merge(r Report) error {
	other := r.(*Metrics)

	if err := m.Latencies.Merge(&other.Latencies); err != nil {
		return err
	}

	if m.Histogram != nil && other.Histogram != nil {
		if err := m.Histogram.Merge(other.Histogram); err != nil {
			return err
		}
	}

	m.BytesOut.Total += other.BytesOut.Total
	m.BytesIn.Total += other.BytesIn.Total

	if m.Earliest.IsZero() || m.Earliest.After(other.Earliest) && !other.Earliest.IsZero() {
		m.Earliest = other.Earliest
	}

	if other.Latest.After(m.Latest) {
		m.Latest = other.Latest
	}

	if other.End.After(m.End) {
		m.End = other.End
	}

	m.Requests += other.Requests

	for k, v := range other.StatusCodes {
		m.StatusCodes[k] += v
	}

	for _, e := range other.Errors {
		if _, ok := m.errors[e]; !ok {
			m.errors[e] = struct{}{}
			m.Errors = append(m.Errors, e)
		}
	}

	m.success += other.success

	return nil
}

// Close implements the Close method of the Report interface by computing
// derived summary metrics which don't need to be run on every Add call.
//nolint:gomnd // magic numbers describe quantile percentiles
func (m *Metrics) Close() {
	if m.Requests == 0 {
		return
	}

	m.Rate = float64(m.Requests)
	m.Throughput = float64(m.success)
	m.Duration = m.Latest.Sub(m.Earliest)
	m.Wait = m.End.Sub(m.Latest)
	if secs := m.Duration.Seconds(); secs > 0 {
		m.Rate /= secs
		// No need to check for zero because we know m.Duration > 0
		m.Throughput /= (m.Duration + m.Wait).Seconds()
	}

	m.BytesIn.Mean = float64(m.BytesIn.Total) / float64(m.Requests)
	m.BytesOut.Mean = float64(m.BytesOut.Total) / float64(m.Requests)
	m.Success = float64(m.success) / float64(m.Requests)
	m.Latencies.Mean = time.Duration(float64(m.Latencies.Total) / float64(m.Requests))
	m.Latencies.P50 = m.Latencies.Quantile(0.50)
	m.Latencies.P90 = m.Latencies.Quantile(0.90)
	m.Latencies.P95 = m.Latencies.Quantile(0.95)
	m.Latencies.P99 = m.Latencies.Quantile(0.99)
}

// LatencyMetrics holds computed request latency metrics.
type LatencyMetrics struct {
	// Total is the total latency sum of all requests in an attack.
	Total time.Duration `json:"total"`
	// Mean is the mean request latency.
	Mean time.Duration `json:"mean"`
	// P50 is the 50th percentile request latency.
	P50 time.Duration `json:"50th"`
	// P90 is the 90th percentile request latency.
	P90 time.Duration `json:"90th"`
	// P95 is the 95th percentile request latency.
	P95 time.Duration `json:"95th"`
	// P99 is the 99th percentile request latency.
	P99 time.Duration `json:"99th"`
	// Max is the maximum observed request latency.
	Max time.Duration `json:"max"`
	// Min is the minimum observed request latency.
	Min time.Duration `json:"min"`

	estimator estimator
}

// NewLatencyMetrics initializes new LatencyMetrics.
func NewLatencyMetrics() (*LatencyMetrics, error) {
	estimator, err := newTdigestEstimator()
	if err != nil {
		return nil, err
	}

	return &LatencyMetrics{
		estimator: estimator,
	}, nil
}

// Add adds the given latency to the latency metrics.
func (l *LatencyMetrics) Add(latency time.Duration) error {
	if l.Total += latency; latency > l.Max {
		l.Max = latency
	}

	if latency < l.Min || l.Min == 0 {
		l.Min = latency
	}

	return l.estimator.Add(float64(latency))
}

// Merge implements the Merge method of the Report interface by adding the given
// Result to LatencyMetrics.
func (l *LatencyMetrics) Merge(other *LatencyMetrics) error {
	l.Total += other.Total

	if l.Max < other.Max {
		l.Max = other.Max
	}

	if l.Min == 0 || l.Min > other.Min && other.Min != 0 {
		l.Min = other.Min
	}

	return l.estimator.Merge(other.estimator)
}

// Quantile returns the nth quantile from the latency summary.
func (l LatencyMetrics) Quantile(nth float64) time.Duration {
	return time.Duration(l.estimator.Get(nth))
}

// ByteMetrics holds computed byte flow metrics.
type ByteMetrics struct {
	// Total is the total number of flowing bytes in an attack.
	Total uint64 `json:"total"`
	// Mean is the mean number of flowing bytes per hit.
	Mean float64 `json:"mean"`
}

type estimator interface {
	Add(sample float64) error
	Get(quantile float64) float64
	Merge(estimator) error
}

type tdigestEstimator struct {
	*tdigest.TDigest
}

func newTdigestEstimator() (*tdigestEstimator, error) {
	digest, err := tdigest.New()
	if err != nil {
		return nil, err
	}
	return &tdigestEstimator{TDigest: digest}, nil
}

func (e *tdigestEstimator) Add(s float64) error {
	return e.TDigest.Add(s)
}

func (e *tdigestEstimator) Get(q float64) float64 {
	return e.TDigest.Quantile(q)
}

func (e *tdigestEstimator) Merge(other estimator) error {
	return e.TDigest.Merge(other.(*tdigestEstimator).TDigest)
}
