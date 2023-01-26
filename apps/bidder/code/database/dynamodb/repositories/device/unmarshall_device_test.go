package device

import (
	"testing"

	"bidder/code/database/api"
	"bidder/code/id"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/stretchr/testify/assert"
)

func TestUnmarshallAudienceAttributesV1(t *testing.T) {
	deviceMap := map[string]*dynamodb.AttributeValue{
		"d": {
			B: []byte("\x7c\x77\x79\xcf\x98\xdd\xfd\xea\xa4\x06\xd1\x46\xb7\x4e\x0f\x14"),
		},
		"a": {
			B: []byte("\x7c\x77\x79\xcf\x98\xdd\xfd\xea\xa4\x06\xd1\x46\xb7\x4e\x0f\x14" +
				"\x49\x82\x3c\x44\x95\xa7\x93\x43\xe8\x97\x7b\x30\xe5\xaf\xc1\x50"),
		},
	}

	actual := api.Device{}
	assert.NoError(t, unmarshallDeviceAttributesV1(deviceMap, &actual))

	expected := api.Device{
		AudienceIDs: []id.ID{
			id.FromByteSlice("\x7c\x77\x79\xcf\x98\xdd\xfd\xea\xa4\x06\xd1\x46\xb7\x4e\x0f\x14"),
			id.FromByteSlice("\x49\x82\x3c\x44\x95\xa7\x93\x43\xe8\x97\x7b\x30\xe5\xaf\xc1\x50"),
		},
	}
	assert.Equal(t, expected, actual)
}

func TestUnmarshallAudienceAttributesV1Empty(t *testing.T) {
	actual := api.Device{}
	assert.NoError(t, unmarshallDeviceAttributesV1(
		map[string]*dynamodb.AttributeValue{"a": {}},
		&actual,
	))

	expected := api.Device{
		AudienceIDs: []id.ID(nil),
	}
	assert.Equal(t, expected, actual)
}

func TestUnmarshallAudienceAttributesV2(t *testing.T) {
	deviceMap := map[string]types.AttributeValue{
		"d": &types.AttributeValueMemberB{
			Value: []byte("\x7c\x77\x79\xcf\x98\xdd\xfd\xea\xa4\x06\xd1\x46\xb7\x4e\x0f\x14"),
		},
		"a": &types.AttributeValueMemberB{
			Value: []byte("\x7c\x77\x79\xcf\x98\xdd\xfd\xea\xa4\x06\xd1\x46\xb7\x4e\x0f\x14" +
				"\x49\x82\x3c\x44\x95\xa7\x93\x43\xe8\x97\x7b\x30\xe5\xaf\xc1\x50"),
		},
	}

	actual := api.Device{}
	assert.NoError(t, unmarshallDeviceAttributesV2(deviceMap, &actual))

	expected := api.Device{
		AudienceIDs: []id.ID{
			id.FromByteSlice("\x7c\x77\x79\xcf\x98\xdd\xfd\xea\xa4\x06\xd1\x46\xb7\x4e\x0f\x14"),
			id.FromByteSlice("\x49\x82\x3c\x44\x95\xa7\x93\x43\xe8\x97\x7b\x30\xe5\xaf\xc1\x50"),
		},
	}
	assert.Equal(t, expected, actual)
}

func TestUnmarshallAudienceAttributesV2Empty(t *testing.T) {
	actual := api.Device{}
	assert.NoError(t, unmarshallDeviceAttributesV2(
		map[string]types.AttributeValue{"a": &types.AttributeValueMemberB{}},
		&actual,
	))

	expected := api.Device{
		AudienceIDs: []id.ID(nil),
	}
	assert.Equal(t, expected, actual)
}
