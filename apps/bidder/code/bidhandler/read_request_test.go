package bidhandler

import (
	"bidder/code/auction"
	"bidder/code/openrtb"
	"bidder/code/stream"
	bidFixtures "bidder/tests/fixtures/bid"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/mxmCherry/openrtb/openrtb2"
	"github.com/mxmCherry/openrtb/openrtb3"
	"github.com/stretchr/testify/assert"
	"github.com/valyala/fasthttp"
)

// Test if bidRequestHandler properly decodes OpenRTB data.
func TestReadRequest(t *testing.T) {
	pd := newPersistentData()
	ctx := &fasthttp.RequestCtx{}
	dataStream, err := stream.NewStream(stream.Config{Disable: true})
	assert.NoError(t, err)

	ctx.Request.Header.SetMethod("POST")
	ctx.Request.SetRequestURI("/bidrequest")

	t.Run("OpenRTB 3.0", func(t *testing.T) {
		ctx.Request.SetBodyString(bidFixtures.BidRequest3)
		_, request := Handler{
			dataStream: dataStream,
			cfg: &Config{
				OpenRTBVersion: openrtb.OpenRTB3_0,
			},
		}.readRequest(ctx, pd)

		expectedRequestBody := &openrtb3.Body{}
		assert.NoError(t, json.Unmarshal([]byte(bidFixtures.BidRequest3), expectedRequestBody))
		assert.Equal(t, expectedRequestBody.OpenRTB.Request.ID, string(request.ID))

		assert.Len(t, expectedRequestBody.OpenRTB.Request.Item, 1)
		assert.Equal(t, []auction.Item{{ID: []byte(expectedRequestBody.OpenRTB.Request.Item[0].ID)}}, request.Item)

		actualDeviceID, err := uuid.FromBytes(request.DeviceID[:])
		assert.NoError(t, err)
		assert.Equal(t, "9c1ce5bd-3013-5c90-2598-cd744e1c96d6", actualDeviceID.String())
	})

	t.Run("OpenRTB 2.5", func(t *testing.T) {
		ctx.Request.SetBodyString(bidFixtures.BidRequest2)
		_, request := Handler{
			dataStream: dataStream,
			cfg: &Config{
				OpenRTBVersion: openrtb.OpenRTB2_5,
			},
		}.readRequest(ctx, pd)

		expectedRequestBody := &openrtb2.BidRequest{}
		assert.NoError(t, json.Unmarshal([]byte(bidFixtures.BidRequest2), expectedRequestBody))
		assert.Equal(t, expectedRequestBody.ID, string(request.ID))

		assert.Len(t, expectedRequestBody.Imp, 1)
		assert.Equal(t, []auction.Item{{ID: []byte(expectedRequestBody.Imp[0].ID)}}, request.Item)

		actualDeviceID, err := uuid.FromBytes(request.DeviceID[:])
		assert.NoError(t, err)
		assert.Equal(t, "9c1ce5bd-3013-5c90-2598-cd744e1c96d6", actualDeviceID.String())
	})
}

// Test if bidRequestHandler rejects requests declaring invalid versions.
func TestReadRequestInvalidVersion(t *testing.T) {
	pd := newPersistentData()
	ctx := &fasthttp.RequestCtx{}
	dataStream, err := stream.NewStream(stream.Config{Disable: true})
	assert.NoError(t, err)

	ctx.Request.Header.SetMethod("POST")
	ctx.Request.Header.Add("x-openrtb-version", "2.4")
	ctx.Request.SetRequestURI("/bidrequest")

	ctx.Request.SetBodyString(bidFixtures.BidRequest3)
	_, request := Handler{
		dataStream: dataStream,
		cfg: &Config{
			OpenRTBVersion: openrtb.OpenRTB2_5,
		},
	}.readRequest(ctx, pd)
	assert.Nil(t, request)
	assert.Equal(t, 400, ctx.Response.StatusCode())
}

func TestParseEmptyDeviceID(t *testing.T) {
	deviceUUID, err := parseDeviceID([]byte{})
	assert.NoError(t, err)

	actualDeviceID, err := uuid.FromBytes(deviceUUID[:])
	assert.NoError(t, err)
	assert.Equal(t, "00000000-0000-0000-0000-000000000000", actualDeviceID.String())
}
