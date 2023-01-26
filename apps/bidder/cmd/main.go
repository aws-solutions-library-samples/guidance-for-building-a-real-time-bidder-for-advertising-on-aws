package main

import (
	"bidder/code/app"
	"os"

	"github.com/rs/zerolog/log"
)

func main() {
	log.Info().Msg("bidder starting...")
	if err := app.App(); err != nil {
		log.Error().Err(err).Msg("")
		log.Info().Msg("bidder exit")
		os.Exit(1)
	}
	log.Info().Msg("bidder exit")
	os.Exit(0)
}
