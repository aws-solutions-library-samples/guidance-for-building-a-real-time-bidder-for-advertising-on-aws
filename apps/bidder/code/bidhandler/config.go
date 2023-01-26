package bidhandler

import (
	"bidder/code/openrtb"
	"time"
)

// Config is a struct for holding bidhandler configuration.
type Config struct {
	Timeout        time.Duration   `envconfig:"BIDREQUEST_TIMEOUT" required:"true"`
	TimeoutStatus  int             `envconfig:"BIDREQUEST_TIMEOUT_STATUS" required:"true"`
	OpenRTBVersion openrtb.Version `envconfig:"BIDREQUEST_OPEN_RTB_VERSION" default:"3.0"`
}
