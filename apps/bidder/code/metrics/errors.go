package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Metrics for number of database errors.
var (
	DBDeviceGetErrorsN = promauto.NewCounter(prometheus.CounterOpts{
		Name: "db_device_get_errors_number",
		Help: "The total number of database errors when getting a device",
	})
	DBAudienceScanErrorsN = promauto.NewCounter(prometheus.CounterOpts{
		Name: "db_audience_scan_errors_number",
		Help: "The total number of database errors when scanning audiences",
	})
	DBBudgetScanErrorsN = promauto.NewCounter(prometheus.CounterOpts{
		Name: "db_budget_scan_errors_number",
		Help: "The total number of database errors when scanning budgets",
	})
	DBCampaignScanErrorsN = promauto.NewCounter(prometheus.CounterOpts{
		Name: "db_campaign_scan_errors_number",
		Help: "The total number of database errors when scanning campaigns",
	})
)
