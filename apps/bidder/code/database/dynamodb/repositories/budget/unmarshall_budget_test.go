package budget

import (
	"testing"

	"bidder/code/database/api"
	"bidder/code/id"
	"bidder/code/price"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/stretchr/testify/assert"
)

func TestBudgetCompressionEmpty(t *testing.T) {
	expected := ItemBatch{}
	compressed := ItemBatchCompress(&expected)
	actual := ItemBatchDecompress(&compressed)

	assert.Equal(t, expected, actual)
}

func TestBudgetCompression(t *testing.T) {
	expected := ItemBatch{
		ID: 123456789,
		Batch: []api.Budget{
			{CampaignID: id.FromHex("0ec712516a1a9050a9d60e502b32eb2d"), Available: price.ToInt(2)},
			{CampaignID: id.FromHex("214f8627ae12428443673c314d0adaba"), Available: price.ToInt(0)},
			{CampaignID: id.FromHex("27a99a9da4b3e35e16a4f4a2b0ed6679"), Available: price.ToInt(123456789)},
		},
	}
	compressed := ItemBatchCompress(&expected)
	actual := ItemBatchDecompress(&compressed)

	assert.Equal(t, expected, actual)
}

func TestUnmarshallBudgetAttributesV1(t *testing.T) {
	deviceMap := map[string]*dynamodb.AttributeValue{
		"i": {N: aws.String("321")},
		"s": {N: aws.String("2")},
		"b": {
			B: []byte("\x7c\x77\x79\xcf\x98\xdd\xfd\xea\xa4\x06\xd1\x46\xb7\x4e\x0f\x14\x40\x78\x7d\x01\x00\x00\x00\x00" +
				"\x92\xa9\x95\xb5\xdc\x70\xe4\x7f\x2d\xc4\xa7\xf7\x51\x35\xce\x94\xc0\xf3\x5e\x01\x00\x00\x00\x00"),
		},
	}

	actual := []api.Budget(nil)
	assert.NoError(t, unmarshallBudgetAttributesV1(deviceMap, func(i api.Budget) error {
		actual = append(actual, i)
		return nil
	}))

	expected := []api.Budget{
		{
			CampaignID: id.FromByteSlice("\x7c\x77\x79\xcf\x98\xdd\xfd\xea\xa4\x06\xd1\x46\xb7\x4e\x0f\x14"),
			Available:  price.ToInt(25),
		},
		{
			CampaignID: id.FromByteSlice("\x92\xa9\x95\xb5\xdc\x70\xe4\x7f\x2d\xc4\xa7\xf7\x51\x35\xce\x94"),
			Available:  price.ToInt(23),
		},
	}
	assert.Equal(t, expected, actual)
}

func TestUnmarshallBudgetAttributesV2(t *testing.T) {
	deviceMap := map[string]types.AttributeValue{
		"i": &types.AttributeValueMemberN{Value: "321"},
		"s": &types.AttributeValueMemberN{Value: "2"},
		"b": &types.AttributeValueMemberB{
			Value: []byte("\x7c\x77\x79\xcf\x98\xdd\xfd\xea\xa4\x06\xd1\x46\xb7\x4e\x0f\x14\x40\x78\x7d\x01\x00\x00\x00\x00" +
				"\x92\xa9\x95\xb5\xdc\x70\xe4\x7f\x2d\xc4\xa7\xf7\x51\x35\xce\x94\xc0\xf3\x5e\x01\x00\x00\x00\x00"),
		},
	}

	actual := []api.Budget(nil)
	assert.NoError(t, unmarshallBudgetAttributesV2(deviceMap, func(i api.Budget) error {
		actual = append(actual, i)
		return nil
	}))

	expected := []api.Budget{
		{
			CampaignID: id.FromByteSlice("\x7c\x77\x79\xcf\x98\xdd\xfd\xea\xa4\x06\xd1\x46\xb7\x4e\x0f\x14"),
			Available:  price.ToInt(25),
		},
		{
			CampaignID: id.FromByteSlice("\x92\xa9\x95\xb5\xdc\x70\xe4\x7f\x2d\xc4\xa7\xf7\x51\x35\xce\x94"),
			Available:  price.ToInt(23),
		},
	}
	assert.Equal(t, expected, actual)
}
