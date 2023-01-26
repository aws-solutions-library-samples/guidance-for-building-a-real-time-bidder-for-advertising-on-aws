package api

import (
	"encoding/base64"
	"encoding/json"
	"time"

	"bidder/code/id"
)

// DeviceRepository is device repo interface.
type DeviceRepository interface {
	Get(time.Time, id.ID, *Device) error
}

// Device represents a single device to audiences map entry.
type Device struct {
	AudienceIDs []id.ID
}

// MarshalJSON implements json.Marshaler interface.
// Used for pretty printing items when debugging.
func (d Device) MarshalJSON() ([]byte, error) {
	type jsonDevice struct {
		AudienceIDs []string
	}

	pretty := jsonDevice{
		AudienceIDs: make([]string, len(d.AudienceIDs)),
	}

	for i := 0; i < len(d.AudienceIDs); i++ {
		pretty.AudienceIDs[i] = base64.URLEncoding.EncodeToString(d.AudienceIDs[i][:])
	}

	return json.Marshal(pretty)
}
