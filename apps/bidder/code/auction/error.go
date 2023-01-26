package auction

import "emperror.dev/errors"

var (
	// ErrNoBid is returned by auction in case of no bid.
	ErrNoBid = errors.New("no bid")

	// ErrTimeout is returned by auction in case of timeout.
	ErrTimeout = errors.New("timeout")
)
