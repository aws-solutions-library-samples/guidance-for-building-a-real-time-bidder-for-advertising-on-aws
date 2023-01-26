package loadgenerator

import (
	"io/ioutil"
	"math/rand"
	"testing"
	"time"

	bmizerany "github.com/bmizerany/perks/quantile"
	"github.com/dgryski/go-gk"
	streadway "github.com/streadway/quantile"
	"github.com/stretchr/testify/assert"
)

var expectedMetrics = &Metrics{
	Latencies: LatencyMetrics{
		Total: mustParseDuration("50.005s"),
		Mean:  mustParseDuration("5.0005ms"),
		P50:   mustParseDuration("5.0005ms"),
		P90:   mustParseDuration("9.0005ms"),
		P95:   mustParseDuration("9.5005ms"),
		P99:   mustParseDuration("9.9005ms"),
		Max:   mustParseDuration("10ms"),
		Min:   mustParseDuration("1us"),
	},
	BytesIn:     ByteMetrics{Total: 10240000, Mean: 1024},
	BytesOut:    ByteMetrics{Total: 5120000, Mean: 512},
	Earliest:    time.Unix(0, 0),
	Latest:      time.Unix(9999, 0),
	End:         time.Unix(9999, 0).Add(10000 * time.Microsecond),
	Duration:    mustParseDuration("2h46m39s"),
	Wait:        mustParseDuration("10ms"),
	Requests:    10000,
	Rate:        1.000100010001,
	Throughput:  0.6667660098349737,
	Success:     0.6667,
	StatusCodes: map[int]int{500: 3333, 200: 3334, 302: 3333},
	Errors:      []string{"Internal server error"},
}

func TestMetrics_Add(t *testing.T) {
	codes := []uint16{500, 200, 302}
	errors := []string{"Internal server error", ""}

	actual, err := NewMetrics()
	assert.NoError(t, err)

	for i := 1; i <= 10000; i++ {
		assert.NoError(t, actual.Add(&Result{
			Code:      codes[i%len(codes)],
			Timestamp: time.Unix(int64(i-1), 0),
			Latency:   time.Duration(i) * time.Microsecond,
			BytesIn:   1024,
			BytesOut:  512,
			Error:     errors[i%len(errors)],
		}))
	}
	actual.Close()

	expectedMetrics.success = 6667
	expectedMetrics.errors = map[string]struct{}{"Internal server error": {}}
	expectedMetrics.Latencies.estimator = actual.Latencies.estimator

	assert.InDelta(t, expectedMetrics.Latencies.P50, actual.Latencies.P50, float64(10*time.Microsecond))
	assert.InDelta(t, expectedMetrics.Latencies.P90, actual.Latencies.P90, float64(10*time.Microsecond))
	assert.InDelta(t, expectedMetrics.Latencies.P95, actual.Latencies.P95, float64(10*time.Microsecond))
	assert.InDelta(t, expectedMetrics.Latencies.P99, actual.Latencies.P99, float64(10*time.Microsecond))

	expectedMetrics.Latencies.P50 = actual.Latencies.P50
	expectedMetrics.Latencies.P90 = actual.Latencies.P90
	expectedMetrics.Latencies.P95 = actual.Latencies.P95
	expectedMetrics.Latencies.P99 = actual.Latencies.P99

	assert.EqualValues(t, expectedMetrics, actual)
}

func TestMetrics_Merge(t *testing.T) {
	codes := []uint16{500, 200, 302}
	errors := []string{"Internal server error", ""}

	m := []*Metrics(nil)
	for i := 0; i < 100; i++ {
		newMetric, err := NewMetrics()
		assert.NoError(t, err)
		m = append(m, newMetric)
	}

	for i := 1; i <= 10000; i++ {
		assert.NoError(t, m[i%len(m)].Add(&Result{
			Code:      codes[i%len(codes)],
			Timestamp: time.Unix(int64(i-1), 0),
			Latency:   time.Duration(i) * time.Microsecond,
			BytesIn:   1024,
			BytesOut:  512,
			Error:     errors[i%len(errors)],
		}))
	}

	actual, err := NewMetrics()
	assert.NoError(t, err)

	for i := 0; i < len(m); i++ {
		assert.NoError(t, actual.Merge(m[i]))
	}

	actual.Close()

	expectedMetrics.success = 6667
	expectedMetrics.errors = map[string]struct{}{"Internal server error": {}}
	expectedMetrics.Latencies.estimator = actual.Latencies.estimator

	assert.InDelta(t, expectedMetrics.Latencies.P50, actual.Latencies.P50, float64(10*time.Microsecond))
	assert.InDelta(t, expectedMetrics.Latencies.P90, actual.Latencies.P90, float64(10*time.Microsecond))
	assert.InDelta(t, expectedMetrics.Latencies.P95, actual.Latencies.P95, float64(10*time.Microsecond))
	assert.InDelta(t, expectedMetrics.Latencies.P99, actual.Latencies.P99, float64(10*time.Microsecond))

	expectedMetrics.Latencies.P50 = actual.Latencies.P50
	expectedMetrics.Latencies.P90 = actual.Latencies.P90
	expectedMetrics.Latencies.P95 = actual.Latencies.P95
	expectedMetrics.Latencies.P99 = actual.Latencies.P99

	assert.Equal(t, expectedMetrics, actual)
}

// https://github.com/tsenart/vegeta/issues/208
func TestMetrics_NoInfiniteRate(t *testing.T) {
	t.Parallel()

	m, err := NewMetrics()
	assert.NoError(t, err)

	m.Requests = 1
	m.Duration = time.Microsecond

	m.Close()

	assert.Equal(t, 1.0, m.Rate)
}

// https://github.com/tsenart/vegeta/pull/277
func TestMetrics_NonNilErrorsOnClose(t *testing.T) {
	t.Parallel()

	m, err := NewMetrics()
	assert.NoError(t, err)

	m.Errors = nil

	m.Close()

	assert.Equal(t, []string(nil), m.Errors)
}

// https://github.com/tsenart/vegeta/issues/461
func TestMetrics_EmptyMetricsCanBeReported(t *testing.T) {
	t.Parallel()

	var m Metrics
	m.Close()

	reporter := NewJSONReporter(&m)
	if err := reporter(ioutil.Discard); err != nil {
		t.Error(err)
	}
}
func BenchmarkMetrics(b *testing.B) {
	b.StopTimer()
	b.ResetTimer()

	rng := rand.New(rand.NewSource(time.Now().UnixNano()))

	latencies := make([]time.Duration, 1000000)
	for i := range latencies {
		latencies[i] = time.Duration(1e6 + rng.Int63n(1e10-1e6)) // 1ms to 10s
	}

	td, err := newTdigestEstimator()
	assert.NoError(b, err)
	for _, tc := range []struct {
		name string
		estimator
	}{
		{"streadway/quantile", newStreadwayEstimator(streadway.New(
			streadway.Known(0.50, 0.01),
			streadway.Known(0.90, 0.005),
			streadway.Known(0.95, 0.001),
			streadway.Known(0.99, 0.0005),
		))},
		{"bmizerany/perks/quantile", newBmizeranyEstimator(
			0.50,
			0.90,
			0.95,
			0.99,
		)},
		{"dgrisky/go-gk", newDgriskyEstimator(0.5)},
		{"influxdata/tdigest", td},
	} {
		m := Metrics{Latencies: LatencyMetrics{estimator: tc.estimator}}
		b.Run("Add/"+tc.name, func(b *testing.B) {
			for i := 0; i <= b.N; i++ {
				_ = m.Add(&Result{
					Code:      200,
					Timestamp: time.Unix(int64(i), 0),
					Latency:   latencies[i%len(latencies)],
					BytesIn:   1024,
					BytesOut:  512,
				})
			}
		})

		b.Run("Close/"+tc.name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				m.Close()
			}
		})
	}
}

type streadwayEstimator struct {
	estimator *streadway.Estimator
}

func newStreadwayEstimator(e *streadway.Estimator) *streadwayEstimator {
	return &streadwayEstimator{estimator: e}
}

func (e *streadwayEstimator) Add(s float64) error {
	e.estimator.Add(s)
	return nil
}

func (e *streadwayEstimator) Get(q float64) float64 {
	return e.estimator.Get(q)
}

func (e *streadwayEstimator) Merge(estimator) error {
	return nil
}

type bmizeranyEstimator struct {
	*bmizerany.Stream
}

func newBmizeranyEstimator(qs ...float64) *bmizeranyEstimator {
	return &bmizeranyEstimator{Stream: bmizerany.NewTargeted(qs...)}
}

func (e *bmizeranyEstimator) Add(s float64) error {
	e.Insert(s)
	return nil
}

func (e *bmizeranyEstimator) Get(q float64) float64 {
	return e.Query(q)
}

func (e *bmizeranyEstimator) Merge(estimator) error {
	return nil
}

type dgryskiEstimator struct {
	*gk.Stream
}

func newDgriskyEstimator(epsilon float64) *dgryskiEstimator {
	return &dgryskiEstimator{Stream: gk.New(epsilon)}
}

func (e *dgryskiEstimator) Add(s float64) error {
	e.Insert(s)
	return nil
}

func (e *dgryskiEstimator) Get(q float64) float64 {
	return e.Query(q)
}

func (e *dgryskiEstimator) Merge(estimator) error {
	return nil
}

func mustParseDuration(s string) time.Duration {
	d, err := time.ParseDuration(s)
	if err != nil {
		panic(err)
	}
	return d
}
