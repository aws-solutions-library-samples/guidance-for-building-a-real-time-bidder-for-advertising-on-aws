package requestbuilder

import (
	"encoding/hex"
	"math/rand"
	"time"
)

const (
	uuidByteLen   = 16
	uuidStringLen = 36
)

// uuidBuilder is used to Generate UUID look-alike
// strings for purpose of mocking http requests.
type uuidBuilder struct {
	rawBuffer     [uuidByteLen]byte
	encodedBuffer [uuidStringLen]byte

	rng *rand.Rand
}

func newUUIDBuilder() *uuidBuilder {
	return &uuidBuilder{
		rng: rand.New(rand.NewSource(time.Now().Unix())),
	}
}

// uuid generates a new uuid look-alike string using
// default random number generator. The string is returned as
// []byte pointing to underlying buffer.
func (b *uuidBuilder) uuid() []byte {
	for i := 0; i < len(b.rawBuffer); i++ {
		b.rawBuffer[i] = byte(b.rng.Int())
	}

	hex.Encode(b.encodedBuffer[0:8], b.rawBuffer[0:4])
	b.encodedBuffer[8] = '-'
	hex.Encode(b.encodedBuffer[9:13], b.rawBuffer[4:6])
	b.encodedBuffer[13] = '-'
	hex.Encode(b.encodedBuffer[14:18], b.rawBuffer[6:8])
	b.encodedBuffer[18] = '-'
	hex.Encode(b.encodedBuffer[19:23], b.rawBuffer[8:10])
	b.encodedBuffer[23] = '-'
	hex.Encode(b.encodedBuffer[24:], b.rawBuffer[10:])

	return b.encodedBuffer[:]
}
