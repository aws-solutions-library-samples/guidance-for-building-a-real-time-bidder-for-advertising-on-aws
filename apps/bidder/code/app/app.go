package app

import (
	"bidder/code/auction"
	"bidder/code/bidhandler"
	diagnosticServer "bidder/code/diagnostic_server"
	bidserver "bidder/code/server"
	"os"
	"os/signal"
	"runtime"
	"time"

	"emperror.dev/errors"
	"github.com/kelseyhightower/envconfig"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// App is the majority of bidder code. The function purpose is to keep
// main() as small as possible and keep all the code in the /code directory.
func App() (errReturn error) {
	cfg := Config{}
	if err := envconfig.Process("", &cfg); err != nil {
		return errors.Wrap(err, "error during config initialization")
	}

	logLevel, err := zerolog.ParseLevel(cfg.LogLevel)
	if err != nil {
		return err
	}
	zerolog.SetGlobalLevel(logLevel)
	zerolog.TimeFieldFormat = time.Stamp

	log.Debug().Msgf(
		"bidder will be using %d of %d available threads for goroutine calls",
		runtime.GOMAXPROCS(0),
		runtime.NumCPU())

	services, err := initializeServices(cfg)
	if err != nil {
		return err
	}
	defer services.close()

	auctionFn := auction.New(services.cache)
	bidHandler := bidhandler.New(cfg.BidHandlerCfg, auctionFn, services.stream)

	server := bidserver.NewServer(cfg.Server, bidHandler)
	diagServer := diagnosticServer.New(cfg.DiagnosticServer)

	stop := make(chan os.Signal, 1)

	server.AsyncListenAndServe(func(err error) {
		errReturn = errors.Wrap(err, "error during server operation")
		stop <- os.Interrupt
	})

	diagServer.AsyncListenAndServe(func(err error) {
		errReturn = errors.Wrap(err, "error during diagnostic server operation")
		stop <- os.Interrupt
	})

	log.Info().Msg("bidder ready")
	signal.Notify(stop, os.Interrupt)
	<-stop

	err = shutdown(cfg, server, diagServer)
	if err != nil {
		if errReturn == nil {
			return err
		}
	}

	return
}
