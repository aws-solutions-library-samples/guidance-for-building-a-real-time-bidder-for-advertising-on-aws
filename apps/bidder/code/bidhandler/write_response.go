package bidhandler

import (
	"bidder/code/auction"
	"bidder/code/metrics"

	"emperror.dev/errors"
	"github.com/rs/zerolog/log"
	"github.com/valyala/fasthttp"
)

// writeResponse serializes, logs and writes http bidresponse.
func (h Handler) writeResponse(
	response []byte,
	ctx *fasthttp.RequestCtx,
) {
	metrics.SuccessN.Inc()

	if err := h.dataStream.PutResponse(response); err != nil {
		log.Error().Err(errors.Wrap(err, "error while streaming response")).Msg("")
	}

	ctx.SetContentType("application/json")
	ctx.SetBody(response)
}

// writeResponseError writes http bidresponse in case of failed auction.
func (h Handler) writeResponseError(
	auctionErr error,
	ctx *fasthttp.RequestCtx,
) {
	switch {
	case errors.Is(auctionErr, auction.ErrNoBid):
		metrics.NoBidN.Inc()
		ctx.SetStatusCode(fasthttp.StatusNoContent)
	case errors.Is(auctionErr, auction.ErrTimeout):
		metrics.RequestTimeoutN.Inc()
		ctx.SetStatusCode(h.cfg.TimeoutStatus)
	default:
		metrics.ServerErrorN.Inc()
		err := errors.Wrap(auctionErr, "error while performing auction")
		log.Error().Err(err).Msg("")
		ctx.Error(err.Error(), fasthttp.StatusInternalServerError)
	}
}
