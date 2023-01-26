package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Metrics for time spent in various application components.
var (
	defaultSummaryPercentiles = map[float64]float64{0.9: 0.01, 0.95: 0.05, 0.99: 0.001}

	BidRequestTime = NewBufferedSummary(promauto.NewSummary(prometheus.SummaryOpts{
		Name:       "bid_request_time",
		Help:       "The summary of time spent handling bidrequest",
		Objectives: defaultSummaryPercentiles,
	}))

	// DeviceQueryTime is called in hot path, so it's separate
	// from the more versatile but unoptimized DatabaseTime.
	DeviceQueryTime = NewBufferedSummary(promauto.NewSummary(prometheus.SummaryOpts{
		Name:       "device_query_time",
		Help:       "The summary of time spent querying database for device",
		Objectives: defaultSummaryPercentiles,
	}))

	KinesisTime = NewBufferedSummary(promauto.NewSummary(prometheus.SummaryOpts{
		Name:       "kinesis_time",
		Help:       "The summary of time spent putting records to kinesis",
		Objectives: defaultSummaryPercentiles,
	}))

	BudgetUpdateTime = promauto.NewSummary(prometheus.SummaryOpts{
		Name:       "budget_update_time",
		Help:       "The summary of time spent updating budget",
		Objectives: defaultSummaryPercentiles,
	})

	DatabaseTime = promauto.NewSummaryVec(prometheus.SummaryOpts{
		Name: "database_time",
		Help: "The summary of time spent running database queries other than device query" +
			" (including network latency, timed out requests and scan loops)",
		Objectives: defaultSummaryPercentiles,
	}, []string{"query_type", "table_name"})
)

// Start all buffered summaries.
func Start() {
	BidRequestTime.StartService()
	DeviceQueryTime.StartService()
	KinesisTime.StartService()
}

// Close all buffered summaries.
func Close() {
	BidRequestTime.CloseService()
	DeviceQueryTime.CloseService()
	KinesisTime.CloseService()
}
