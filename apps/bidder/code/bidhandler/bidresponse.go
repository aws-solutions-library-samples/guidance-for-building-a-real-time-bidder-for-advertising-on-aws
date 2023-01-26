package bidhandler

import (
	"bidder/code/auction"
	"bidder/code/openrtb"
	"bidder/code/price"
	"net/url"
	"strconv"

	"gvisor.dev/gvisor/pkg/gohacks"
)

// buildResponse appends to response an OpenRTB response for the given request, item and campaign
// based on the version of the bid request.
func buildResponse(r *auction.Response, pd *persistentData) []byte {
	if r.Request.OpenRTBVersion == openrtb.OpenRTB2_5 {
		return buildResponse2(r, pd)
	}
	return buildResponse3(r, pd)
}

// buildResponse3 appends to response an OpenRTB 3.0 response for the given request, item and campaign.
//
// Since campaigns do not specify all the fields needed in an ad, many values are sample or fake. E.g. we pretend to
// return a display creative and none of the returned tracking URLs are working.
//
// The response is a byte slice of its JSON representation, containing a root object with the "openrtb" field.
func buildResponse3(r *auction.Response, pd *persistentData) []byte {
	pd.byteResponse = pd.byteResponse[:0]

	requestID := gohacks.StringFromImmutableBytes(r.Request.ID)

	// Bid response is just the bid request ID and an array of seat bids.
	//
	// We could also provide a BidID, but it's probably not useful, since each bid request has at most one response
	// and we don't need a separate ID to track the response.
	//
	// Non-USD bid responses would additionally specify the currency.
	pd.byteResponse = append(pd.byteResponse, `{"openrtb":{"ver":"3.0","domainspec":"adcom","domainver":"1.0","response":{"id":`...)
	pd.byteResponse = strconv.AppendQuote(pd.byteResponse, requestID)
	pd.byteResponse = append(pd.byteResponse, `,"seatbid":[{"seat":`...)
	// Seat ID, almost certainly needs a value provided by the exchange.
	pd.byteResponse = append(pd.byteResponse, `"XYZ"`...)
	pd.byteResponse = append(pd.byteResponse, `,"bid":[{"id":"`...)
	pd.byteResponse = pd.ksuidSequence.Get().Append(pd.byteResponse)
	pd.byteResponse = append(pd.byteResponse, `","item":"`...)
	pd.byteResponse = append(pd.byteResponse, r.Item.ID...)
	pd.byteResponse = append(pd.byteResponse, `","price":`...)
	pd.byteResponse = strconv.AppendFloat(pd.byteResponse, price.ToFloat(r.Price), 'f', -1, 64)
	// With PMP, we would put the deal ID in the bid and possibly have other bids on different deals.
	pd.byteResponse = append(pd.byteResponse, `,"cid":`...)
	pd.byteResponse = strconv.AppendQuote(pd.byteResponse, r.Campaign.HexID)
	pd.byteResponse = append(pd.byteResponse, `,"burl":"`...)
	// Billing notice URL. This one is fake, a real one would provide enough information to update a campaign's
	// budget when the auction is won.
	pd.byteResponse = append(pd.byteResponse, "https://t.ab.clearcode.cc/"...)
	escapedRID := url.PathEscape(requestID)
	escapedCID := url.PathEscape(r.Campaign.HexID)
	pd.byteResponse = append(pd.byteResponse, escapedRID...)
	pd.byteResponse = append(pd.byteResponse, "/"...)
	pd.byteResponse = append(pd.byteResponse, escapedCID...)
	pd.byteResponse = append(pd.byteResponse, "/${OPENRTB_PRICE}"...)
	pd.byteResponse = append(pd.byteResponse, `","media":`...)

	// Build the ad media object.
	pd.byteResponse = append(pd.byteResponse, `{"ad":{"id":`...)
	// Creative ID; let's assume that they are 1-on-1 with campaigns.
	pd.byteResponse = strconv.AppendQuote(pd.byteResponse, r.Campaign.HexID)
	pd.byteResponse = append(pd.byteResponse, `,"adomain":[`...)
	// Advertiser domain, should be obtained from campaign data.
	pd.byteResponse = append(pd.byteResponse, `"ford.com"`...)
	pd.byteResponse = append(pd.byteResponse, `],"secure":1,"display":{"mime":"image/jpeg","ctype":1,"w":`...)
	// Assume a 320x50 image/jpeg display creative.
	pd.byteResponse = append(pd.byteResponse, "320"...)
	pd.byteResponse = append(pd.byteResponse, `,"h":`...)
	pd.byteResponse = append(pd.byteResponse, "50"...)
	pd.byteResponse = append(pd.byteResponse, `,"curl":"`...)
	// URL providing the creative HTML. Needs campaign ID (which ad to show) and might take other parameters to
	// customize it depending on the bid request context.
	pd.byteResponse = append(pd.byteResponse, "https://t.ab.clearcode.cc/"...)
	pd.byteResponse = append(pd.byteResponse, escapedRID...)
	pd.byteResponse = append(pd.byteResponse, "/"...)
	pd.byteResponse = append(pd.byteResponse, escapedCID...)

	pd.byteResponse = append(pd.byteResponse, `"}}}`...)
	pd.byteResponse = append(pd.byteResponse, `}]}]}}}`...)

	return pd.byteResponse
}

// buildResponse2 appends to response an OpenRTB 2.5 response for the given request, item and campaign.
//
// Since campaigns do not specify all the fields needed in an ad, many values are sample or fake. E.g. we pretend to
// return a display creative and none of the returned tracking URLs are working.
//
// The response is a byte slice of its JSON representation, containing a root object with the "openrtb" field.
func buildResponse2(r *auction.Response, pd *persistentData) []byte {
	pd.byteResponse = pd.byteResponse[:0]

	requestID := gohacks.StringFromImmutableBytes(r.Request.ID)

	// Bid response is just the bid request ID and an array of seat bids.
	//
	// We could also provide a BidID, but it's probably not useful, since each bid request has at most one response
	// and we don't need a separate ID to track the response.
	//
	// Non-USD bid responses would additionally specify the currency.

	pd.byteResponse = append(pd.byteResponse, `{"id":`...)
	pd.byteResponse = strconv.AppendQuote(pd.byteResponse, requestID)
	pd.byteResponse = append(pd.byteResponse, `,"seatbid":[{"seat":`...)
	// Seat ID, almost certainly needs a value provided by the exchange.
	pd.byteResponse = append(pd.byteResponse, `"XYZ"`...)
	pd.byteResponse = append(pd.byteResponse, `,"bid":[{"id":"`...)
	pd.byteResponse = pd.ksuidSequence.Get().Append(pd.byteResponse)
	pd.byteResponse = append(pd.byteResponse, `","price":`...)
	pd.byteResponse = strconv.AppendFloat(pd.byteResponse, price.ToFloat(r.Price), 'f', -1, 64)
	pd.byteResponse = append(pd.byteResponse, `,"impid":"`...)
	pd.byteResponse = append(pd.byteResponse, r.Item.ID...)
	// With PMP, we would put the deal ID in the bid and possibly have other bids on different deals.
	pd.byteResponse = append(pd.byteResponse, `","cid":`...)
	pd.byteResponse = strconv.AppendQuote(pd.byteResponse, r.Campaign.HexID)
	pd.byteResponse = append(pd.byteResponse, `,"burl":"`...)
	// Billing notice URL. This one is fake, a real one would provide enough information to update a campaign's
	// budget when the auction is won.
	pd.byteResponse = append(pd.byteResponse, "https://t.ab.clearcode.cc/"...)
	escapedRID := url.PathEscape(requestID)
	escapedCID := url.PathEscape(r.Campaign.HexID)
	pd.byteResponse = append(pd.byteResponse, escapedRID...)
	pd.byteResponse = append(pd.byteResponse, "/"...)
	pd.byteResponse = append(pd.byteResponse, escapedCID...)
	pd.byteResponse = append(pd.byteResponse, `/${OPENRTB_PRICE}",`...)
	pd.byteResponse = append(pd.byteResponse, `"crid":`...)
	// Creative ID; let's assume that they are 1-on-1 with campaigns.
	pd.byteResponse = strconv.AppendQuote(pd.byteResponse, r.Campaign.HexID)
	// Advertiser domain, should be obtained from campaign data.
	pd.byteResponse = append(pd.byteResponse, `,"adomain":["ford.com"],`...)
	pd.byteResponse = append(pd.byteResponse, `"ext": {"secire":1,"mime":"image/jpeg","ctype":1,"curl":"`...)
	// URL providing the creative HTML. Needs campaign ID (which ad to show) and might take other parameters to
	// customize it depending on the bid request context.
	pd.byteResponse = append(pd.byteResponse, "https://t.ab.clearcode.cc/"...)
	pd.byteResponse = append(pd.byteResponse, escapedRID...)
	pd.byteResponse = append(pd.byteResponse, "/"...)
	pd.byteResponse = append(pd.byteResponse, escapedCID...)
	pd.byteResponse = append(pd.byteResponse, `"},"w":320,"h":50}]}]}`...)

	return pd.byteResponse
}
