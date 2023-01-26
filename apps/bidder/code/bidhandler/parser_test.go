// Test parsing bid requests.

package bidhandler

import (
	"bidder/code/auction"
	"bidder/code/id"
	bidFixtures "bidder/tests/fixtures/bid"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestParseBidRequestCorrect checks that the sample bid request is parsed into the correct structure.
func TestParseBidRequestCorrect(t *testing.T) {
	pd := newPersistentData()
	expectedRequest := auction.Request{
		ID: []byte("1nQSb41SvuYMALOiS1pslOvlHGf"),
		Item: []auction.Item{
			{ID: []byte("1nQSb4hpjPgxAINsLWdBhJZmkUu")},
		},
		DeviceID: id.FromByteSlice("\xa6\x41\x40\xcb\xff\x37\x1c\xdc\xd4\x7d\x70\xe1\x03\x99\x5b\x46"),
	}

	t.Run("Bid Request 3.x", func(t *testing.T) {
		byteRequest := []byte(bidFixtures.BenchmarkBidRequest3)
		request, err := parseBidRequest3(byteRequest, pd)
		assert.NoError(t, err)
		expectedRequest.OpenRTBVersion = "3.0"
		assert.Equal(t, &expectedRequest, request)
	})
	t.Run("Bid Request 2.x", func(t *testing.T) {
		byteRequest := []byte(bidFixtures.BenchmarkBidRequest2)
		request, err := parseBidRequest2(byteRequest, pd)
		assert.NoError(t, err)
		expectedRequest.OpenRTBVersion = "2.5"
		assert.Equal(t, &expectedRequest, request)
	})
}

// BenchmarkParseBidRequest benchmarks parseBidRequest on sample bid request.
func BenchmarkParseBidRequest(b *testing.B) {
	pd := newPersistentData()

	b.Run("Bid Request 3.x", func(b *testing.B) {
		byteRequest := []byte(bidFixtures.BenchmarkBidRequest3)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := parseBidRequest3(byteRequest, pd)
			assert.NoError(b, err)
		}
	})

	b.Run("Bid Request 2.x", func(b *testing.B) {
		byteRequest := []byte(bidFixtures.BenchmarkBidRequest2)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := parseBidRequest2(byteRequest, pd)
			assert.NoError(b, err)
		}
	})
}
