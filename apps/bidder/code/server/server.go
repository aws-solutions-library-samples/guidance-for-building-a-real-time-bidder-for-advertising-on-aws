package server

import (
	"bidder/code/bidhandler"

	"github.com/valyala/fasthttp"
	"github.com/valyala/fasthttp/reuseport"
)

// Server is mostly convenience wrapper around http.Server.
// But when/if we decide to change http framework, it may
// isolate rest of the app from code changes.
type Server struct {
	server fasthttp.Server
	cfg    Config
}

// NewServer creates new bidrequest server.
func NewServer(
	cfg Config,
	bidHandler bidhandler.Handler,
) *Server {
	requestHandler := func(ctx *fasthttp.RequestCtx) {
		switch string(ctx.Path()) {
		case cfg.BidRequestPath:
			if !ctx.IsPost() {
				ctx.Error("Unsupported method", fasthttp.StatusMethodNotAllowed)
				return
			}
			bidHandler.HandleRequest(ctx)
		case cfg.HealthCheckPath:
			if !ctx.IsGet() && !ctx.IsHead() {
				ctx.Error("Unsupported method", fasthttp.StatusMethodNotAllowed)
				return
			}
			healthCheckHandler(ctx)
		default:
			ctx.Error("Unsupported path", fasthttp.StatusNotFound)
		}
	}

	return &Server{server: fasthttp.Server{
		Handler:      requestHandler,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
		IdleTimeout:  cfg.IdleTimeout,
		LogAllErrors: cfg.LogAllErrors,
		Logger:       logger{},
		// Server header is not required, so omit it. (There are options to omit the Date and Content-Type
		// headers, but Date is required and we know what content types we use, so keep them.)
		NoDefaultServerHeader: true,
	},
		cfg: cfg}
}

// AsyncListenAndServe starts the server in its own goroutine.
func (s *Server) AsyncListenAndServe(errCallback func(error)) {
	go func() {
		listener, err := reuseport.Listen("tcp4", s.cfg.Address)
		if err != nil && errCallback != nil {
			errCallback(err)
			return
		}
		if err := s.server.Serve(listener); err != nil {
			if errCallback != nil {
				errCallback(err)
			}
		}
	}()
}

// Shutdown shutdowns the server.
func (s *Server) Shutdown() error {
	return s.server.Shutdown()
}
