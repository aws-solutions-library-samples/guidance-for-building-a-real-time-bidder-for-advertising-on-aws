package producer

import (
	"bidder/code/ksuid"
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/proto"
)

// Test if we can correctly estimate the byte size of compressed
// protobuf aggregated record consisting of one large message.
// Message size is such that it should produce largest allowed
// aggregated record.
func TestProtobufSizeEstimationBig(t *testing.T) {
	const maxMessageSize = 51141

	message := make([]byte, maxMessageSize)
	partitionKey := ksuid.NewSequence().Get()
	keyIdx := uint64(0)

	record := Record{
		PartitionKeyIndex: &keyIdx,
		Data:              message,
	}

	aggregate := &AggregatedRecord{
		PartitionKeyTable: []string{partitionKey.String()},
		Records:           []*Record{&record},
	}

	protoRecord, err := proto.Marshal(aggregate)
	assert.NoError(t, err)

	expectedSize := recordMetadataSize() + messageProtoSize(message)
	assert.Equal(t, expectedSize, len(protoRecord))
	assert.Equal(t, maxEntrySize, len(protoRecord)+aggregateMetadataSize())
}

// Test if we can correctly estimate the byte size of compressed
// protobuf aggregated record consisting of multiple small messages.
func TestProtobufSizeEstimationSmall(t *testing.T) {
	const smallMessageSize = 3

	message := make([]byte, smallMessageSize)
	partitionKey := ksuid.NewSequence().Get()
	keyIdx := uint64(0)

	record := Record{
		PartitionKeyIndex: &keyIdx,
		Data:              message,
	}

	aggregate := &AggregatedRecord{
		PartitionKeyTable: []string{partitionKey.String()},
		Records:           []*Record(nil),
	}

	// Add small messages to record without exceeding maxEntrySize.
	for i := 0; entryMetadataSize()+(i+1)*messageProtoSize(message) <= maxEntrySize; i++ {
		aggregate.Records = append(aggregate.Records, &record)
	}

	protoRecord, err := proto.Marshal(aggregate)
	assert.NoError(t, err)
	expectedSize := len(aggregate.Records)*messageProtoSize(message) + recordMetadataSize()
	assert.Equal(t, expectedSize, len(protoRecord))
	assert.True(t, maxEntrySize >= len(protoRecord)+aggregateMetadataSize())

	// Record should exceed maxEntrySize after adding one more message.
	aggregate.Records = append(aggregate.Records, &record)

	protoRecordTooLarge, err := proto.Marshal(aggregate)
	assert.NoError(t, err)
	expectedSize = len(aggregate.Records)*messageProtoSize(message) + recordMetadataSize()
	assert.Equal(t, expectedSize, len(protoRecordTooLarge))
	assert.True(t, maxEntrySize < len(protoRecordTooLarge)+aggregateMetadataSize())
}
