// Tests and benchmarks for compression code.
//
// They do not run our code: we reimplement benchmark code in the relevant stream functions.

package bidhandler

import (
	"bytes"
	"encoding/json"
	"testing"

	"bidder/code/auction"
	"bidder/code/database/api"
	"bidder/code/id"
	"bidder/code/price"
	bidFixtures "bidder/tests/fixtures/bid"

	"github.com/ClearcodeHQ/gozstd"
	"github.com/stretchr/testify/assert"
	"github.com/valyala/bytebufferpool"
)

// makeBidResponse returns a sample bid response.
//
// They behave like compacted JSON: we do not include unnecessary whitespace in it.
func makeBidResponse() []byte {
	request := auction.Request{ID: []byte("4c2f99c6-e326-447c-8351-6fe4456100a1"), Item: []auction.Item{{ID: []byte("1")}}}
	campaign := api.Campaign{ID: id.FromByteSlice("\xfa\x24"), MaxCPM: price.ToInt(1.5)}
	r := &auction.Response{Request: &request, Item: &request.Item[0], Campaign: &campaign, Price: price.ToInt(1.42)}
	pd := newPersistentData()

	response := buildResponse3(r, pd)
	return response
}

// compactJSON returns a copy of the input JSON object without extra whitespace.
func compactJSON(b *testing.B, input []byte) []byte {
	var compact bytes.Buffer
	err := json.Compact(&compact, input)
	if err != nil {
		b.Fatal(err)
	}
	return compact.Bytes()
}

// compressGoZstd benchmarks compression of the input buffer using gozstd.
func compressGoZstd(b *testing.B, input []byte) {
	var pool bytebufferpool.Pool

	buffer := pool.Get()
	buffer.B = gozstd.Compress(buffer.B, input)
	cSize := len(buffer.B)
	pool.Put(buffer)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buffer := pool.Get()
		buffer.B = gozstd.Compress(buffer.B, input)
		if cSize != len(buffer.B) {
			b.Fatal("wrong output size")
		}
		pool.Put(buffer)
	}
}

// TestCompressDecompress tests that gozstd can decompress properly a gozstd-compressed bid request.
func TestCompressDecompress(t *testing.T) {
	var pool bytebufferpool.Pool

	input := []byte(bidFixtures.BenchmarkBidRequest3)

	cBuffer := pool.Get()
	cBuffer.B = gozstd.Compress(cBuffer.B, input)

	decompressed, err := gozstd.Decompress(nil, cBuffer.B)
	assert.NoError(t, err)

	assert.Equal(t, input, decompressed)
	pool.Put(cBuffer)
}

// BenchmarkCompressGoZstdBidRequest benchmarks compressing a sample bid request.
func BenchmarkCompressGoZstdBidRequest(b *testing.B) {
	bidRequest := []byte(bidFixtures.BenchmarkBidRequest3)

	compressGoZstd(b, bidRequest)
}

// BenchmarkCompressGoZstdCompactBidRequest benchmarks compressing a sample compacted bid request.
func BenchmarkCompressGoZstdCompactBidRequest(b *testing.B) {
	bidRequest := []byte(bidFixtures.BenchmarkBidRequest3)
	compactBidRequest := compactJSON(b, bidRequest)

	compressGoZstd(b, compactBidRequest)
}

// BenchmarkCompressGoZstdBidResponse benchmarks compressing a sample bid response.
func BenchmarkCompressGoZstdBidResponse(b *testing.B) {
	bidResponse := makeBidResponse()

	compressGoZstd(b, bidResponse)
}
