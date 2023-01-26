package api

import (
	"bidder/code/id"
)

const (
	// AudiencesAttributeName is a short name of audience attribute in DynamoDB
	AudiencesAttributeName = "a"
	// IDAttributeName is a short name of id attribute in DynamoDB
	IDAttributeName = "d"
)

// Audience represents a single audience to campaigns map entry.
type Audience struct {
	AudienceID  id.ID   `dynamodbav:"audience_id"`
	CampaignIDs []id.ID `dynamodbav:"campaign_ids"`
}

// AudienceRepository is audience repository interface.
type AudienceRepository interface {
	Scan(consume func(Audience) error) error
}
