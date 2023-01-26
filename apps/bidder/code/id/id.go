package id

import (
	"encoding/base64"
	"encoding/hex"
)

// Len is length of ID in bytes.
const Len = 16

// ID of database items.
type ID [Len]byte

// ZeroID is the all-zero ID used for unknown/missing objects.
var ZeroID = ID{}

// FromByteSlice builds ID from byte slice passed as string.
// Should be used only in tests.
func FromByteSlice(b string) ID {
	id := [Len]byte{}
	copy(id[:], b)
	return id
}

// FromByteSlices builds IDs from byte slices passed as strings.
// Should be used only in tests.
func FromByteSlices(bs ...string) []ID {
	ids := make([]ID, 0, len(bs))
	for _, s := range bs {
		ids = append(ids, FromByteSlice(s))
	}
	return ids
}

// FromHex builds ID from hex encoded string.
// Should be used only in tests.
func FromHex(hx string) ID {
	b, err := hex.DecodeString(hx)
	if err != nil {
		panic(err)
	}

	id := [Len]byte{}
	copy(id[:], b)
	return id
}

// FromHexes builds IDs from hex encoded strings.
// Should be used only in tests.
func FromHexes(hxs []string) []ID {
	ids := make([]ID, 0, len(hxs))
	for _, s := range hxs {
		ids = append(ids, FromHex(s))
	}
	return ids
}

// FromBase64 builds ID from base64 encoded string.
// Should be used only in tests.
func FromBase64(base string) ID {
	b, err := base64.StdEncoding.DecodeString(base)
	if err != nil {
		panic(err)
	}

	id := [Len]byte{}
	copy(id[:], b)
	return id
}

// FromBase64s builds IDs from base64 encoded strings.
// Should be used only in tests.
func FromBase64s(hxs []string) []ID {
	ids := make([]ID, 0, len(hxs))
	for _, s := range hxs {
		ids = append(ids, FromBase64(s))
	}
	return ids
}
