package producer

import (
	"github.com/valyala/bytebufferpool"
)

// batch of messages that will be sent as one request to Kinesis.
type batch struct {
	totalMessageSize int
	entryMessageSize int
	entryNumber      int
	messages         []*bytebufferpool.ByteBuffer

	flushReason string
}

// tryAddMessage add a message to batch if it won't exceed any size limit.
func (b *batch) tryAddMessage(message *bytebufferpool.ByteBuffer) bool {
	msgSize := messageProtoSize(message.B)

	// Check if message overflows batch.
	if msgSize+b.requestTotalSize() > maxRequestSize {
		return false
	}

	// Check if message overflows entry.
	if msgSize+b.entrySize() > maxEntrySize {
		// Check if new entry overflows batch.
		if b.entryNumber == maxEntriesPerRequest {
			return false
		}

		// Check if message in new entry overflows batch.
		if msgSize+b.requestTotalSize()+entryTotalMetadataSize() > maxRequestSize {
			return false
		}

		// Add new entry.
		b.entryNumber++
		b.entryMessageSize = 0
	}

	// Add message.
	b.totalMessageSize += msgSize
	b.entryMessageSize += msgSize
	b.messages = append(b.messages, message)

	return true
}

func (b *batch) requestTotalSize() int {
	return (b.entryNumber+1)*entryTotalMetadataSize() + b.totalMessageSize
}

func (b *batch) entrySize() int {
	return entryMetadataSize() + b.entryMessageSize
}

// reset batch to initial state, used when returning batch to sync.Pool
func (b *batch) reset() {
	b.totalMessageSize = 0
	b.entryMessageSize = 0
	b.entryNumber = 0
	b.messages = b.messages[:0]
	b.flushReason = ""
}
