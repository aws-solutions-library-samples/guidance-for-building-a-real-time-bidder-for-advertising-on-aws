package app

import (
	diagnosticServer "bidder/code/diagnostic_server"
	bidserver "bidder/code/server"
	"sync"

	"emperror.dev/errors"
	"github.com/rs/zerolog/log"
)

// shutdown closes app servers.
func shutdown(
	cfg Config,
	server *bidserver.Server,
	diagServer *diagnosticServer.Server,
) (errReturn error) {
	log.Info().Msg("bidder shutting down...")

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()

		err := server.Shutdown()
		if err != nil {
			errReturn = errors.Wrap(err, "error during server shutdown")
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()

		err := diagServer.Shutdown(cfg.DiagnosticServer.ShutdownTimeout)
		if err != nil {
			errReturn = errors.Wrap(err, "error during diagnostic server shutdown")
		}
	}()

	wg.Wait()

	return
}
