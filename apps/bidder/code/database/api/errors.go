package api

import (
	"emperror.dev/errors"
)

var (
	// ErrItemNotFound is returned when requested item is not found in the database.
	ErrItemNotFound = errors.New("not found")

	// ErrTimeout is returned when database operation times out.
	ErrTimeout = errors.New("timeout")

	// ErrBadResponseFormat is returned when database returns response with unexpected format.
	ErrBadResponseFormat = errors.New("bad response format")
)
