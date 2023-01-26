package auction

import (
	"bidder/code/database/api"
)

// Response contains all the information needed to build a bid response.
//
// We assume exactly one bid per bid response (if the bidder handles multiple PMP deals, multiple seats, multiple items
// in a bid request, then it needs multiple values for all fields other than Request).
//
// None of the pointers are nil. If not bidding, nil Response pointers might be used.
type Response struct {
	Request  *Request
	Item     *Item
	Campaign *api.Campaign
	Price    int64
}
