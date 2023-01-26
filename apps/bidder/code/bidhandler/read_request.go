package bidhandler

import (
	"bidder/code/auction"
	"bidder/code/id"
	"bidder/code/metrics"
	"bidder/code/openrtb"

	"emperror.dev/errors"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"github.com/valyala/fasthttp"
)

var errBadVersion = errors.New("unsupported x-openrtb-version")

// readRequest reads and deserializes bid request, returning both its raw body and auction.Request pointer.
// Eventual errors are written to response writer, so main
// handler doesn't need to handle them. In case of an error
// the method returns nil pointer.
func (h Handler) readRequest(
	ctx *fasthttp.RequestCtx,
	pd *persistentData,
) ([]byte, *auction.Request) {
	byteRequest := ctx.PostBody()
	openRTBVersion := openrtb.Version(ctx.Request.Header.Peek("x-openrtb-version"))

	if openRTBVersion == "" {
		openRTBVersion = h.cfg.OpenRTBVersion
	}

	var err error
	var request *auction.Request
	switch openRTBVersion {
	case openrtb.OpenRTB3_0:
		request, err = parseBidRequest3(byteRequest, pd)
	case openrtb.OpenRTB2_5:
		request, err = parseBidRequest2(byteRequest, pd)
	default:
		err = errBadVersion
	}

	if err != nil {
		log.Error().Err(err).Msg("")
		metrics.BadRequestN.Inc()
		ctx.Error(err.Error(), fasthttp.StatusBadRequest)
		return byteRequest, nil
	}
	return byteRequest, request
}

// parseBidRequest3 parses a bid request 3.x JSON byte slice into an Request
func parseBidRequest3(byteRequest []byte, pd *persistentData) (*auction.Request, error) {
	v, err := pd.parser.ParseBytes(byteRequest)
	if err != nil {
		return nil, errors.Wrap(err, "error while parsing request")
	}

	v = v.Get("openrtb", "request")
	if v == nil {
		return nil, errors.New("empty request body")
	}

	ID := v.GetStringBytes("id")
	if len(ID) == 0 {
		return nil, errors.New("empty request ID")
	}

	deviceID, err := parseDeviceID(v.GetStringBytes("context", "device", "ifa"))
	if err != nil {
		return nil, errors.Wrap(err, "malformed device ID")
	}

	itemValues := v.GetArray("item")
	if len(itemValues) == 0 {
		return nil, errors.New("bidrequest must contain at least one item")
	}

	pd.auctionRequest.Item = pd.auctionRequest.Item[:0]
	for _, item := range itemValues {
		pd.auctionRequest.Item = append(pd.auctionRequest.Item, auction.Item{ID: item.GetStringBytes("id")})
	}

	pd.auctionRequest.ID = ID
	pd.auctionRequest.DeviceID = deviceID
	pd.auctionRequest.OpenRTBVersion = openrtb.OpenRTB3_0

	return &pd.auctionRequest, nil
}

// parseDeviceID parsed UUID device ID from string to []byte.
func parseDeviceID(encoded []byte) (id.ID, error) {
	if len(encoded) == 0 {
		// No device ID is present (or it's empty).
		return id.ID{}, nil
	}

	deviceUUID, err := uuid.ParseBytes(encoded)
	if err != nil {
		return id.ID{}, err
	}

	bytes, err := deviceUUID.MarshalBinary()
	if err != nil {
		return id.ID{}, err
	}

	deviceID := id.ID{}
	copy(deviceID[:], bytes)
	return deviceID, nil
}

// parseBidRequest2 parses a bid request 2.x JSON byte slice into an Request
func parseBidRequest2(byteRequest []byte, pd *persistentData) (*auction.Request, error) {
	v, err := pd.parser.ParseBytes(byteRequest)
	if err != nil {
		return nil, errors.Wrap(err, "error while parsing request")
	}

	ID := v.GetStringBytes("id")
	if len(ID) == 0 {
		return nil, errors.New("empty request ID")
	}

	deviceID, err := parseDeviceID(v.GetStringBytes("device", "ifa"))
	if err != nil {
		return nil, errors.Wrap(err, "malformed device ID")
	}

	// Item information for 2.x is in the Imp object
	impValues := v.GetArray("imp")
	if len(impValues) == 0 {
		return nil, errors.New("bidrequest must contain at least one imp object")
	}

	pd.auctionRequest.Item = pd.auctionRequest.Item[:0]
	for _, item := range impValues {
		pd.auctionRequest.Item = append(pd.auctionRequest.Item, auction.Item{ID: item.GetStringBytes("id")})
	}

	pd.auctionRequest.ID = ID
	pd.auctionRequest.DeviceID = deviceID
	pd.auctionRequest.OpenRTBVersion = openrtb.OpenRTB2_5

	return &pd.auctionRequest, nil
}
