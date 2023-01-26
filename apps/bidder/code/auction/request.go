package auction

import (
	"bidder/code/id"
	"bidder/code/openrtb"
)

// Item contains all elements of a bid request item that are used by the bidder.
type Item struct {
	ID []byte
}

// Request contains all elements of bid request that are used by the bidder.
type Request struct {
	ID             []byte
	Item           []Item
	DeviceID       id.ID
	OpenRTBVersion openrtb.Version
}
