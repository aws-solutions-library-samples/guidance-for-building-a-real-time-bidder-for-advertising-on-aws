package api

import (
	"bidder/code/id"
)

// BudgetRepository is budget repository interface.
type BudgetRepository interface {
	FetchAll(consume func(Budget) error) error
}

// Budget represents a campaign available budget.
type Budget struct {
	CampaignID id.ID
	Available  int64
}

// BudgetAttributeName is a short name of budget attribute in DynamoDB
const BudgetAttributeName = "b"

// BudgetByteSize is length of compressed budget item (len(ID) + 8 bytes for value).
const BudgetByteSize = id.Len + 8
