package bidhandler

import (
	"bidder/code/auction"
	"bidder/code/metrics"
	"bidder/code/stream"
	"time"

	"emperror.dev/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog/log"
	"github.com/valyala/fasthttp"
)

// HTTPStatus is the name of error detail used to optionally return
// HTTP status from auction logic.
const HTTPStatus = "HTTPStatus"

// Handler handles bidrequests by decoding them, passing to auction,
// encoding and sending bidresponse and streaming request and response.
type Handler struct {
	cfg        *Config
	auction    *auction.Auction
	dataStream *stream.Stream

	pool *pool
}

// New returns new Handler.
func New(
	cfg Config,
	auctionFn *auction.Auction,
	dataStream *stream.Stream,
) Handler {
	return Handler{
		cfg:        &cfg,
		auction:    auctionFn,
		dataStream: dataStream,
		pool:       &pool{},
	}
}

// HandleRequest handles bid requests and sets response in the context.
func (h Handler) HandleRequest(ctx *fasthttp.RequestCtx) {
	// Using deadline instead of context.WithTimeout to try to save
	// heap allocations. Possibly not all bidrequests will require async
	// operations and in those cases context allocation can be spared.
	log.Info().Msg("Received request")
	deadline := time.Now().Add(h.cfg.Timeout)

	metrics.BidRequestReceivedN.Inc()
	metrics.BidRequestActiveN.Inc()
	timer := prometheus.NewTimer(metrics.BidRequestTime)
	defer metrics.BidRequestActiveN.Dec()
	defer timer.ObserveDuration()

	// Get persistent data from the pool.
	pd := h.pool.Get()
	defer h.pool.Put(pd)

	byteRequest, request := h.readRequest(ctx, pd)
	if request == nil {
		return
	}

	// We have a valid bid request, so put it to the data stream. (We need its ID as a partition key; invalid bid
	// requests are unlikely to be useful when processing the stream.)
	if err := h.dataStream.PutRequest(byteRequest); err != nil {
		log.Error().Err(errors.Wrap(err, "error while streaming request")).Msg("Error streaming to kinesis")
	}

	log.Info().Msg("Received request")
	response := auction.Response{}
	err := h.auction.Run(deadline, request, &response)
	log.Info().Msg("Received request")
	if err != nil {
		h.writeResponseError(err, ctx)
		return
	}

	bidResponse := buildResponse(&response, pd)
	h.writeResponse(bidResponse, ctx)
}
