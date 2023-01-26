package budget

import (
	"encoding/binary"

	"bidder/code/database/api"
	"bidder/code/database/dynamodb"
	"bidder/code/id"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

// unmarshallBudgetAttributesV1 unmarshalls budget items from AWS SDK v1 attribute map format.
func unmarshallBudgetAttributesV1(m dynamodb.AttributeMapV1, consume func(api.Budget) error) error {
	budgetAttribute, ok := m[api.BudgetAttributeName]
	if !ok {
		return api.ErrBadResponseFormat
	}

	budgets := budgetAttribute.B

	if len(budgets)%api.BudgetByteSize != 0 {
		return api.ErrBadResponseFormat
	}

	item := api.Budget{}
	for offset := 0; offset < len(budgets); offset += api.BudgetByteSize {
		copy(item.CampaignID[:], budgets[offset:])
		item.Available = int64(binary.LittleEndian.Uint64(budgets[offset+id.Len:]))

		if err := consume(item); err != nil {
			return err
		}
	}

	return nil
}

// unmarshallBudgetAttributesV2 unmarshalls budget items from AWS SDK v2 attribute map format.
func unmarshallBudgetAttributesV2(m dynamodb.AttributeMapV2, consume func(api.Budget) error) error {
	budgetAttribute, ok := m[api.BudgetAttributeName].(*types.AttributeValueMemberB)
	if !ok {
		return api.ErrBadResponseFormat
	}

	budgets := budgetAttribute.Value

	if len(budgets)%api.BudgetByteSize != 0 {
		return api.ErrBadResponseFormat
	}

	item := api.Budget{}
	for offset := 0; offset < len(budgets); offset += api.BudgetByteSize {
		copy(item.CampaignID[:], budgets[offset:])
		item.Available = int64(binary.LittleEndian.Uint64(budgets[offset+id.Len:]))

		if err := consume(item); err != nil {
			return err
		}
	}

	return nil
}
