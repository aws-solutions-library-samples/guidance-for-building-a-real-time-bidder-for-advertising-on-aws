package producer

import "crypto/md5"

// entryTotalMetadataSize returns the total number of bytes in
// kinesis.PutRecordsRequestEntry struct required for metadata.
func entryTotalMetadataSize() int {
	return entryMetadataSize() + ksuidPartitionKeyStringSize
}

// entryMetadataSize returns the total number of bytes in
// kinesis.PutRecordsRequestEntry.Data field required for metadata.
func entryMetadataSize() int {
	return aggregateMetadataSize() + recordMetadataSize()
}

// aggregateMetadataSize returns total number of bytes in
// aggregate metadata.
func aggregateMetadataSize() int {
	return len(magicNumber) + md5.Size
}

// recordMetadataSize returns total number of bytes in Record
// struct required for metadata, under the assumption that
// a single KSUID is used as the partition key in
// AggregatedRecord.PartitionKeyTable.
func recordMetadataSize() int {
	const protobufOverhead = 9

	return protobufOverhead +
		ksuidPartitionKeySize
}

// messageProtoSize returns the byte length of the message
// when it's embedded into Record struct and compressed to
// protobuf format, under the assumption that
// Record.PartitionKeyIndex field points to int with value 0.
// Size of protobuf byte overhead was established experimentally.
//nolint:gomnd // magic numbers refer to message length thresholds
func messageProtoSize(msg []byte) int {
	l := len(msg)
	switch {
	case l <= 123:
		return l + 6
	case l <= 127:
		return l + 7
	case l <= 16378:
		return l + 8
	case l <= 16383:
		return l + 9
	default:
		return l + 10
	}
}
