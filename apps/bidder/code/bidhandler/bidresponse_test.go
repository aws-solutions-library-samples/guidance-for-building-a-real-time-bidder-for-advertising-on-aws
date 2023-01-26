// Tests for building responses.

package bidhandler

import (
	"encoding/json"
	"testing"

	"github.com/mxmCherry/openrtb/openrtb2"

	"bidder/code/auction"
	"bidder/code/database/api"
	"bidder/code/id"
	"bidder/code/price"

	"github.com/mxmCherry/openrtb/openrtb3"
	"github.com/stretchr/testify/assert"
)

// TestCorrect checks that buildResponse builds a valid OpenRTB 3.0 response with a bid ID.
func TestCorrectResponse(t *testing.T) {
	request := auction.Request{ID: []byte("4c2f99c6-e326-447c-8351-6fe4456100a1"), Item: []auction.Item{{ID: []byte("1")}}}
	campaign := api.Campaign{ID: id.FromByteSlice("\xfa\x24"), MaxCPM: price.ToInt(1.5)}
	r := &auction.Response{Request: &request, Item: &request.Item[0], Campaign: &campaign, Price: price.ToInt(1.42)}

	t.Run("Bid response 3.x", func(t *testing.T) {
		bidResponse := buildResponse3(r, newPersistentData())

		var unmarshalled openrtb3.Body
		err := json.Unmarshal(bidResponse, &unmarshalled)
		assert.NoError(t, err)

		// Extract random bid ID.
		ID := unmarshalled.OpenRTB.Response.SeatBid[0].Bid[0].ID
		assert.Len(t, ID, 27)
	})

	t.Run("Bid response 2.x", func(t *testing.T) {
		bidResponse := buildResponse2(r, newPersistentData())
		var unmarshalled openrtb2.BidResponse
		err := json.Unmarshal(bidResponse, &unmarshalled)
		assert.NoError(t, err)

		// Extract random bid ID.
		ID := unmarshalled.SeatBid[0].Bid[0].ID
		assert.Len(t, ID, 27)
	})
}

// BenchmarkBuildResponse benchmarks building a response.
func BenchmarkBuildResponse(b *testing.B) {
	request := auction.Request{ID: []byte("4c2f99c6-e326-447c-8351-6fe4456100a1"), Item: []auction.Item{{ID: []byte("1")}}}
	campaign := api.Campaign{ID: id.FromByteSlice("\xfa\x24"), MaxCPM: price.ToInt(1.5)}
	r := &auction.Response{Request: &request, Item: &request.Item[0], Campaign: &campaign, Price: price.ToInt(1.42)}
	pd := newPersistentData()

	b.Run("Bid response 3.x", func(b *testing.B) {
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			response := buildResponse3(r, pd)
			if len(response) == 0 || len(response) > 2000 {
				b.Fatal("Empty response")
			}
		}
	})

	b.Run("Bid response 2.x", func(b *testing.B) {
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			response := buildResponse2(r, pd)
			if len(response) == 0 || len(response) > 2000 {
				b.Fatal("Empty response")
			}
		}
	})
}
