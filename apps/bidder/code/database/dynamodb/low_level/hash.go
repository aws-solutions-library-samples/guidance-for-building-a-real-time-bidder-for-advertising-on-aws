package lowlevel

import (
	"crypto/sha256"
	"hash"
)

// hasher performs sha256 and hmacsha256 hashes without heap allocations.
// This class methods are NOT safe for concurrent calls.
type hasher struct {
	inner hash.Hash
	outer hash.Hash

	ipad [sha256.BlockSize]byte
	opad [sha256.BlockSize]byte

	sumBuffer *[]byte
}

func newHasher() *hasher {
	return &hasher{
		inner: sha256.New(),
		outer: sha256.New(),

		sumBuffer: new([]byte),
	}
}

// sha256 encrypts data and puts them in resultBuffer.
//nolint:errcheck // sha256.Write doesn't return any error, so we can ignore it.
func (h *hasher) sha256(data []byte, resultBuffer *[]byte) []byte {
	h.inner.Reset()
	_, _ = h.inner.Write(data)

	return sumToBuffer(h.inner, resultBuffer)
}

// hmacsha256 encrypts data and puts them in resultBuffer.
// Implementation is based on https://golang.org/pkg/crypto/hmac/
//nolint:errcheck,gosec // sha256.Write doesn't return any error, so we can ignore it.
func (h *hasher) hmacsha256(key, data []byte, resultBuffer *[]byte) []byte {
	h.outer.Reset()
	h.inner.Reset()

	zero := [sha256.BlockSize]byte{}
	copy(h.ipad[:], zero[:])
	copy(h.opad[:], zero[:])

	if len(key) > sha256.BlockSize {
		// If key is too big, hash it.
		h.outer.Write(key)
		key = sumToBuffer(h.outer, h.sumBuffer)
	}
	copy(h.ipad[:], key)
	copy(h.opad[:], key)
	for i := range h.ipad {
		h.ipad[i] ^= 0x36
	}
	for i := range h.opad {
		h.opad[i] ^= 0x5c
	}
	h.inner.Write(h.ipad[:])
	h.inner.Write(data)

	h.outer.Reset()
	h.outer.Write(h.opad[:])

	h.outer.Write(sumToBuffer(h.inner, h.sumBuffer))
	return sumToBuffer(h.outer, resultBuffer)
}

// sumToBuffer wraps hash.Hash.Sum.
// Sum result is written to resultBuffer to avoid allocations.
// resultBuffer is reallocated if it's too small to contain the result.
func sumToBuffer(h hash.Hash, resultBuffer *[]byte) []byte {
	*resultBuffer = (*resultBuffer)[:0]
	result := h.Sum(*resultBuffer)
	*resultBuffer = result

	return result
}
