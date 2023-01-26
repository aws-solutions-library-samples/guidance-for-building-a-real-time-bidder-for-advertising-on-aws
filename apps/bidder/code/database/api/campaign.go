package api

import (
	"bidder/code/id"

	"emperror.dev/errors"

	validation "github.com/go-ozzo/ozzo-validation"
	"github.com/rs/zerolog/log"
)

// CampaignRepository is campaign repository interface.
type CampaignRepository interface {
	Scan(consume func(Campaign) error) error
}

// Campaign represents a single campaign item.
type Campaign struct {
	HexID  string // Hexadecimal representation of ID.
	ID     id.ID  `dynamodbav:"campaign_id"`
	MaxCPM int64  `dynamodbav:"bid_price"`
}

// IsValid checks if budget related fields have
// values within acceptable ranges.
func (c *Campaign) IsValid() bool {
	err := validation.ValidateStruct(c,
		validation.Field(&c.ID, validation.Required),
		validation.Field(&c.MaxCPM, validation.Required),
	)

	if err != nil {
		log.Debug().Err(errors.Wrap(err, "invalid campaign")).Msg("")
	}

	return err == nil
}
