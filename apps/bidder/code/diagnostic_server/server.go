package diagnosticserver

import (
	"context"
	"net/http"
	"net/http/pprof"
	"time"

	"emperror.dev/errors"
	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Server is the diagnostic server that is not accessible publicly.
// Server struct is mostly a convenience wrapper around http.Server.
type Server struct {
	server http.Server
}

// New creates new diagnostic server.
func New(cfg Config) *Server {
	router := mux.NewRouter()

	// Metrics endpoint
	router.Handle(cfg.MetricsPath, promhttp.Handler()).Methods("GET")

	// Trace endpoint
	router.HandleFunc(cfg.TracePath, traceHandler).Methods("GET")

	// Profiler endpoints
	router.HandleFunc(cfg.ProfilerPath+"/", pprof.Index)
	router.HandleFunc(cfg.ProfilerPath+"/{name:allocs|block|goroutine|heap|mutex|threadcreate}", pprof.Index)
	router.HandleFunc(cfg.ProfilerPath+"/cmdline", pprof.Cmdline)
	router.HandleFunc(cfg.ProfilerPath+"/profile", pprof.Profile)
	router.HandleFunc(cfg.ProfilerPath+"/symbol", pprof.Symbol)
	router.HandleFunc(cfg.ProfilerPath+"/trace", pprof.Trace)

	return &Server{http.Server{
		Addr:    cfg.Address,
		Handler: router,
	}}
}

// AsyncListenAndServe starts the server in its own goroutine.
func (s *Server) AsyncListenAndServe(errCallback func(error)) {
	go func() {
		if err := s.server.ListenAndServe(); err != nil {
			if !errors.Is(err, http.ErrServerClosed) && errCallback != nil {
				errCallback(err)
			}
		}
	}()
}

// Shutdown shutdowns the server.
func (s *Server) Shutdown(timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return s.server.Shutdown(ctx)
}
