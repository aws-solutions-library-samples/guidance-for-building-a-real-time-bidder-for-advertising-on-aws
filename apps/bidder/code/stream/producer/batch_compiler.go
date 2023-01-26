package producer

import (
	"bidder/code/ksuid"
	"crypto/md5"
	"hash"
	"unsafe"

	"github.com/ClearcodeHQ/aws-sdk-go/service/kinesis"
	"github.com/valyala/bytebufferpool"
	"google.golang.org/protobuf/proto"
	"gvisor.dev/gvisor/pkg/gohacks"
)

// batchCompiler contains state necessary to compile a slice
// of raw messages into *kinesis.PutRecordsInput
type batchCompiler struct {
	parent *Producer

	keyIdx uint64
	hash   hash.Hash
	keys   *ksuid.Sequence

	// Fields used to optimize heap allocation.
	records   []Record
	aggregate AggregatedRecord
}

func newBatchCompiler(parent *Producer) *batchCompiler {
	return &batchCompiler{
		parent: parent,
		hash:   md5.New(),
		keys:   ksuid.NewSequence(),

		aggregate: AggregatedRecord{
			// Assume all records use the same partition key,
			// so just one slice element is required.
			PartitionKeyTable: make([]string, 1),
		},
	}
}

// compileBatch compiles a collection of raw messages
// stored in batch struct to kinesis.PutRecordsInput.
//nolint:gosec // unsafe.Pointer is used to optimize heap allocation.
func (c *batchCompiler) compileBatch(b *batch) (*kinesis.PutRecordsInput, error) {
	if len(b.messages) == 0 {
		return nil, nil
	}

	request := &kinesis.PutRecordsInput{
		StreamName: &c.parent.cfg.StreamName,
	}

	messagesCompressed := 0
	for messagesCompressed < len(b.messages) {
		// For some reason compiler insists on moving 'c' to heap.
		this := (*batchCompiler)(gohacks.Noescape(unsafe.Pointer(c)))
		n, entry, err := this.compileMessages(b.messages[messagesCompressed:])

		if err != nil {
			// Release buffers, as the batch is corrupted anyway.
			for _, e := range request.Records {
				c.parent.entryPool.put(e)
			}

			return nil, err
		}

		c.parent.metrics.userRecordsPerKinesisRecordSum.Observe(float64(n))
		c.parent.metrics.kinesisRecordsDataPutSz.Observe(float64(len(entry.Data)))

		messagesCompressed += n
		request.Records = append(request.Records, entry)
	}

	c.parent.metrics.kinesisRecordsPerPutRecordsRequestSum.Observe(float64(len(request.Records)))

	return request, nil
}

// compileMessages compiles a slice of raw messages to kinesis.PutRecordsRequestEntry.
//nolint:gosec // unsafe.Pointer is used to optimize heap allocation.
func (c *batchCompiler) compileMessages(messages []*bytebufferpool.ByteBuffer,
) (int, *kinesis.PutRecordsRequestEntry, error,
) {
	partitionKey := c.keys.Get().String()
	c.aggregate.PartitionKeyTable[0] = partitionKey

	entrySize := entryMetadataSize()
	msgIdx := 0

	// Aggregate records.
	c.records = c.records[:0]
	c.aggregate.Records = c.aggregate.Records[:0]
	for msgIdx < len(messages) {
		msgSize := messageProtoSize(messages[msgIdx].B)

		if msgSize+entrySize > maxEntrySize {
			break
		}

		// Making sure the compiler won't move 'messages' function argument to heap.
		// We can do that because we keep elements of 'messages' in Producer.messagePool explicitly.
		msg := (*bytebufferpool.ByteBuffer)(gohacks.Noescape(unsafe.Pointer(messages[msgIdx])))

		c.records = append(c.records, Record{
			PartitionKeyIndex: &c.keyIdx,
			Data:              msg.B,
		})

		c.aggregate.Records = append(c.aggregate.Records, &c.records[len(c.records)-1])

		entrySize += msgSize
		msgIdx++
	}

	// Compress aggregate.
	entry := c.parent.entryPool.get()
	err := error(nil)
	entry.Data = append(entry.Data, magicNumber...)
	entry.Data, err = proto.MarshalOptions{}.MarshalAppend(entry.Data, &c.aggregate)
	if err != nil {
		c.parent.entryPool.put(entry) // Return buffer, as the entry is corrupted anyway.
		return 0, nil, err
	}

	// Add checksum.
	c.hash.Reset()
	_, _ = c.hash.Write(entry.Data[len(magicNumber):]) // md5.Write never returns a error.
	entry.Data = c.hash.Sum(entry.Data)

	entry.PartitionKey = &partitionKey

	return msgIdx, entry, nil
}
