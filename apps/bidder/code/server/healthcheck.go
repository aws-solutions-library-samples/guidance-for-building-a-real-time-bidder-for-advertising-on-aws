package server

import (
	"github.com/rs/zerolog/log"
	"github.com/valyala/fasthttp"
)

func healthCheckHandler(ctx *fasthttp.RequestCtx) {
	log.Printf("received health check request")
	ctx.SetStatusCode(fasthttp.StatusOK)
}
