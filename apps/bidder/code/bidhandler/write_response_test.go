package bidhandler

import (
	"bidder/code/auction"
	"bidder/code/openrtb"
	"bidder/code/stream"
	bidFixtures "bidder/tests/fixtures/bid"
	"encoding/json"
	"strings"
	"testing"

	"github.com/mxmCherry/openrtb/openrtb2"
	"github.com/mxmCherry/openrtb/openrtb3"
	"github.com/stretchr/testify/assert"
	"github.com/valyala/fasthttp"
)

// Test if auction returning no bid results in http.StatusNoContent
func TestNoBid(t *testing.T) {
	var ctx fasthttp.RequestCtx

	Handler{}.writeResponseError(auction.ErrNoBid, &ctx)

	assert.Equal(t, fasthttp.StatusNoContent, ctx.Response.StatusCode())
}

// Test if auction returning timeout results in configured timeout status
func TestTimeout(t *testing.T) {
	var ctx fasthttp.RequestCtx

	Handler{cfg: &Config{TimeoutStatus: 504}}.writeResponseError(auction.ErrTimeout, &ctx)

	assert.Equal(t, fasthttp.StatusGatewayTimeout, ctx.Response.StatusCode())
}

// Test if bidRequestHandler properly encodes OpenRTB data.
//nolint:dupl // tests are similar but different
func TestWriteResponse(t *testing.T) {
	var ctx fasthttp.RequestCtx
	dataStream, err := stream.NewStream(stream.Config{Disable: true})
	assert.NoError(t, err)

	t.Run("OpenRTB 3.0", func(t *testing.T) {
		response := &openrtb3.Body{}
		assert.NoError(t, json.Unmarshal([]byte(bidFixtures.BidResponse3), response))

		byteResponse, err := json.Marshal(response)
		assert.NoError(t, err)
		Handler{
			dataStream: dataStream,
			cfg: &Config{
				OpenRTBVersion: openrtb.OpenRTB3_0,
			},
		}.writeResponse(byteResponse, &ctx)

		// Check if http response contains bidResponse equal to fixture.
		assert.Equal(t, fasthttp.StatusOK, ctx.Response.StatusCode())
		assert.Equal(t,
			stripWhitespace(bidFixtures.BidResponse3),
			stripWhitespace(string(ctx.Response.Body())))
	})
	t.Run("OpenRTB 2.5", func(t *testing.T) {
		response := &openrtb2.BidResponse{}
		assert.NoError(t, json.Unmarshal([]byte(bidFixtures.BidResponse2), response))

		byteResponse, err := json.Marshal(response)
		assert.NoError(t, err)
		Handler{
			dataStream: dataStream,
			cfg: &Config{
				OpenRTBVersion: openrtb.OpenRTB2_5,
			},
		}.writeResponse(byteResponse, &ctx)

		// Check if http response contains bidResponse equal to fixture.
		assert.Equal(t, fasthttp.StatusOK, ctx.Response.StatusCode())
		assert.Equal(t,
			stripWhitespace(bidFixtures.BidResponse2),
			stripWhitespace(string(ctx.Response.Body())))
	})
}

var stripWhitespace = strings.NewReplacer(" ", "", "\t", "", "\n", "").Replace
