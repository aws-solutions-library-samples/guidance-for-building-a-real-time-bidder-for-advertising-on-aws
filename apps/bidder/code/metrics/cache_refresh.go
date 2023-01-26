package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Cache refresh related metrics
var (
	CacheRefreshRequestRecurringN = promauto.NewCounter(prometheus.CounterOpts{
		Name: "cache_refresh_request_recurring",
		Help: "Number of cache refresh requests due to recurring schedule",
	})
	CacheRefreshRequestOnDemandN = promauto.NewCounter(prometheus.CounterOpts{
		Name: "cache_refresh_request_on_demand",
		Help: "Number of cache refresh requests due to no-bid caused by not sufficient budget of campaigns",
	})
)
