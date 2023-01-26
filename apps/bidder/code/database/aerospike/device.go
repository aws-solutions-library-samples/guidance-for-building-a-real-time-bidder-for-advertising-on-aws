package aerospike

import (
	"time"

	"bidder/code/database/api"
	"bidder/code/id"

	"emperror.dev/errors"
	"github.com/aerospike/aerospike-client-go"
)

// DeviceRepository allows accessing device Client se
type DeviceRepository struct {
	aerospike          *Client
	aerospikeGetPolicy *aerospike.BasePolicy
}

// NewDeviceRepository creates new instance of *AudienceRepository
func NewDeviceRepository(aerospikeClient *Client) (*DeviceRepository, error) {
	return &DeviceRepository{aerospikeClient, aerospikeClient.GetPolicy()}, nil
}

// Get reads audiences for given deviceID
func (r *DeviceRepository) Get(_ time.Time, deviceID id.ID, result *api.Device) error {
	var deviceIDBytes = deviceID[:]

	key, err := aerospike.NewKey(r.aerospike.Namespace, DeviceSet, deviceIDBytes)
	if err != nil {
		return errors.Wrap(err, "failed to create key")
	}

	record, err := r.aerospike.Get(key, r.aerospikeGetPolicy)
	if err != nil {
		return errors.Wrap(err, "failed to get key")
	}

	audiences := record.Bins["audience_id"].([]byte)
	ID := id.ID{}
	for offset := 0; offset < len(audiences); offset += id.Len {
		copy(ID[:], audiences[offset:])
		result.AudienceIDs = append(result.AudienceIDs, ID)
	}

	return nil
}
