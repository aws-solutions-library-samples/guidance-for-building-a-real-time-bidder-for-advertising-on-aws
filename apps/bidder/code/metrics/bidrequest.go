package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Bidrequest summary metrics.
var (
	BidRequestReceivedN = promauto.NewCounter(prometheus.CounterOpts{
		Name: "bid_request_received_number",
		Help: "The total number of bidrequests received by the server",
	})

	BidRequestActiveN = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "bid_request_active_number",
		Help: "The number of bidrequests being handled by the server at any given time",
	})

	SuccessN = promauto.NewCounter(prometheus.CounterOpts{
		Name: "request_success_number",
		Help: "The total number of successfully served requests",
	})

	BadRequestN = promauto.NewCounter(prometheus.CounterOpts{
		Name: "bad_request_number",
		Help: "The total number of bad requests received",
	})

	RequestTimeoutN = promauto.NewCounter(prometheus.CounterOpts{
		Name: "request_timeout_number",
		Help: "The total of bidrequest timeouts",
	})

	ServerErrorN = promauto.NewCounter(prometheus.CounterOpts{
		Name: "server_error_number",
		Help: "The total number of bidrequests failed due to internal server errors",
	})

	NoBidN = promauto.NewCounter(prometheus.CounterOpts{
		Name: "no_bid_number",
		Help: "The total number of no_bid auctions",
	})
)
