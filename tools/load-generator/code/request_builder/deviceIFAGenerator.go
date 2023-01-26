package requestbuilder

import (
	"encoding/hex"
)

const maxDatabaseDeviceID = 10e10

// generateDeviceIFA generates random device IFA for specified number of devices
//nolint:gomnd // it forces us to declare 1. used in the calculation
func (b *Builder) generateDeviceIFA(devicesUsed int, nobidFraction float64) []byte {
	deviceID := int64(0)
	if makeNobid := b.rng.Float64() < nobidFraction; makeNobid {
		maxID := int(float64(devicesUsed) * (-1. + 1./(1.-nobidFraction)))
		if maxID <= 1 {
			deviceID = int64(maxDatabaseDeviceID + 1)
		} else {
			deviceID = b.randomInt64(maxDatabaseDeviceID+1, maxDatabaseDeviceID+maxID)
		}
	} else {
		if devicesUsed > 1 {
			deviceID = b.randomInt64(1, devicesUsed)
		} else {
			deviceID = 1
		}
	}

	deviceIDBytes := b.encryptor.Encrypt(uint64(deviceID))
	resizeBuffer(hex.EncodedLen(len(deviceIDBytes)), &b.formatBuffer)
	hex.Encode(b.formatBuffer, deviceIDBytes)

	return b.formatBuffer
}

func (b *Builder) randomInt64(min, max int) int64 {
	return int64(b.rng.Intn(max-min) + min)
}
