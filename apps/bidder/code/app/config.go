package app

import (
	"bidder/code/bidhandler"
	"bidder/code/cache"
	"bidder/code/database/aerospike"
	"bidder/code/database/dynamodb"
	diagnosticServer "bidder/code/diagnostic_server"
	"bidder/code/server"
	"bidder/code/stream"
)

const (
	clientDynamodb  = "dynamodb"
	clientAerospike = "aerospike"
)

// Config is a struct for holding application configuration.
type Config struct {
	Server           server.Config
	BidHandlerCfg    bidhandler.Config
	DiagnosticServer diagnosticServer.Config
	Stream           stream.Config
	Dynamodb         dynamodb.Config
	Aerospike        aerospike.Config
	Cache            cache.Config

	LogLevel       string `envconfig:"LOG_LEVEL" required:"true"`
	DatabaseClient string `envconfig:"DATABASE_CLIENT" required:"true"`
}
