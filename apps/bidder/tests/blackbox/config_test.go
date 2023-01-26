package blackbox

import (
	"bidder/code/app"
)

// config is a struct for holding blackbox test specific configuration.
type config struct {
	App app.Config

	BidderHost string `envconfig:"TEST_BIDDER_HOST" required:"true"`
}
