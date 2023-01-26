package server

import "github.com/rs/zerolog/log"

// logger implements fasthttp Logger interface using zerolog.
type logger struct{}

// Printf logs fasthttp errors.
func (logger) Printf(format string, args ...interface{}) {
	log.Error().Msgf(format, args...)
}
