package producer

const (
	ksuidPartitionKeySize       = 20
	ksuidPartitionKeyStringSize = 27

	inputBufferSize  = 1e5
	outputBufferSize = 1e2

	maxEntriesPerRequest = 500

	// Max number of bytes in aggregated Kinesis record, including
	// magic number and checksum. It's possible to send larger records,
	// but they will be rejected by Kinesis backend.
	maxEntrySize = 51200 // 50k

	// Max number of bytes in kinesis.PutRecordsInput, including
	// record data and partition keys.
	maxRequestSize = 5 << 20 // 5MiB
)

const (
	reasonSize     = "batch size"
	reasonRetry    = "retry"
	reasonInterval = "interval"
	reasonDrain    = "drain"
)

var (
	magicNumber = []byte{0xF3, 0x89, 0x9A, 0xC2}
)
