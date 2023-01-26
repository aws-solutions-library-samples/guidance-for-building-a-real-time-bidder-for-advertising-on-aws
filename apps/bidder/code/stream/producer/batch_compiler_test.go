package producer

import (
	"math/rand"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/valyala/bytebufferpool"
)

// Test if size of the compiled kinesis batch was estimated correctly.
func TestCompileSize(t *testing.T) {
	const maxMessageSize = 4096
	const minMessageSize = 64

	source := make([]byte, maxMessageSize)
	for i := 0; i < len(source); i++ {
		source[i] = 1
	}

	messagePool := bytebufferpool.Pool{}
	r := batch{}

	// Generate batch with raw messages.
	for {
		msg := messagePool.Get()
		msgSize := minMessageSize + rand.Intn(maxMessageSize-minMessageSize)
		msg.B = append(msg.B, source[:msgSize]...)
		if !r.tryAddMessage(msg) {
			break
		}
	}

	// Compile batch.
	p := New(Config{StreamName: "test-stream"}, nil)
	actual, err := newBatchCompiler(p).compileBatch(&r)
	assert.NoError(t, err)

	actualSize := 0
	for _, entry := range actual.Records {
		actualSize += len(entry.Data)
		actualSize += len(*entry.PartitionKey)
	}

	assert.Equal(t, r.requestTotalSize(), actualSize)
	assert.True(t, maxRequestSize >= r.requestTotalSize())
}

// Test if size of the compiled kinesis batch was estimated correctly for small messages.
func TestCompileSizeSmall(t *testing.T) {
	messagePool := bytebufferpool.Pool{}
	r := batch{}

	// Generate batch with raw messages.
	for {
		msg := messagePool.Get()
		msg.B = append(msg.B, 1)
		if !r.tryAddMessage(msg) {
			break
		}
	}

	// Compile batch.
	p := New(Config{StreamName: "test-stream"}, nil)
	actual, err := newBatchCompiler(p).compileBatch(&r)
	assert.NoError(t, err)

	actualSize := 0
	for _, entry := range actual.Records {
		actualSize += len(entry.Data)
		actualSize += len(*entry.PartitionKey)
	}

	assert.Equal(t, r.requestTotalSize(), actualSize)
	assert.True(t, maxRequestSize >= r.requestTotalSize())
}
