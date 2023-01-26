package loadgenerator

import (
	"bytes"
	"net/http"
)

// Target is an HTTP request blueprint.
type Target struct {
	Method string      `json:"method"`
	URL    string      `json:"url"`
	Body   []byte      `json:"body,omitempty"`
	Header http.Header `json:"header,omitempty"`
}

// Equal returns true if the target is equal to the other given target.
func (t *Target) Equal(other *Target) bool {
	switch {
	case t == other:
		return true
	case t == nil || other == nil:
		return false
	default:
		equal := t.Method == other.Method &&
			t.URL == other.URL &&
			bytes.Equal(t.Body, other.Body) &&
			len(t.Header) == len(other.Header)

		if !equal {
			return false
		}

		for k := range t.Header {
			left, right := t.Header[k], other.Header[k]
			if len(left) != len(right) {
				return false
			}
			for i := range left {
				if left[i] != right[i] {
					return false
				}
			}
		}

		return true
	}
}

// A Targeter decodes a Target or returns an error in case of failure.
// Implementations must be safe for concurrent use.
type Targeter func(*Target) error

// Decode is a convenience method that calls the underlying Targeter function.
func (tr Targeter) Decode(t *Target) error {
	return tr(t)
}

// A TargetEncoder encodes a Target in a format that can be read by a Targeter.
type TargetEncoder func(*Target) error

// Encode is a convenience method that calls the underlying TargetEncoder function.
func (enc TargetEncoder) Encode(t *Target) error {
	return enc(t)
}
