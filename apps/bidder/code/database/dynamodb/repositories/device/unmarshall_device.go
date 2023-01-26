package device

import (
	"bidder/code/database/api"
	"bidder/code/id"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/aws/aws-sdk-go/service/dynamodb"
)

// unmarshallDeviceAttributesV1 unmarshalls device item from AWS SDK v1 attribute map format.
func unmarshallDeviceAttributesV1(m map[string]*dynamodb.AttributeValue, result *api.Device) error {
	audiencesAttribute, ok := m[api.AudiencesAttributeName]
	if !ok {
		return api.ErrBadResponseFormat
	}

	audiences := audiencesAttribute.B

	ID := id.ID{}
	for offset := 0; offset < len(audiences); offset += id.Len {
		copy(ID[:], audiences[offset:])
		result.AudienceIDs = append(result.AudienceIDs, ID)
	}

	return nil
}

// unmarshallDeviceAttributesV2 unmarshalls device item from AWS SDK v2 attribute map format.
func unmarshallDeviceAttributesV2(m map[string]types.AttributeValue, result *api.Device) error {
	audiencesAttribute, ok := m[api.AudiencesAttributeName].(*types.AttributeValueMemberB)
	if !ok {
		return api.ErrBadResponseFormat
	}

	audiences := audiencesAttribute.Value

	ID := id.ID{}
	for offset := 0; offset < len(audiences); offset += id.Len {
		copy(ID[:], audiences[offset:])
		result.AudienceIDs = append(result.AudienceIDs, ID)
	}

	return nil
}
