package producer

import (
	"bytes"
	"crypto/md5"
	"math/rand"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/ClearcodeHQ/aws-sdk-go/aws"
	k "github.com/ClearcodeHQ/aws-sdk-go/service/kinesis"
	"github.com/ClearcodeHQ/gozstd"
	"github.com/getlantern/deepcopy"
	"github.com/segmentio/ksuid"
	"github.com/stretchr/testify/assert"
	"go.uber.org/atomic"
	"google.golang.org/protobuf/proto"
)

var correctResponse = responseMock{
	Error: nil,
	Response: &k.PutRecordsOutput{
		FailedRecordCount: aws.Int64(0),
	},
}

var testMessage = makeMessage(8096)

// Test if producer can be closed after receiving no messages.
func TestNoMessages(t *testing.T) {
	client := &clientMock{
		incoming:  make(map[int][]*k.PutRecordsRequestEntry),
		responses: []responseMock{correctResponse, correctResponse},
	}

	p := New(Config{FlushInterval: time.Hour, MaxConnections: 1}, client)
	assert.NoError(t, p.Start())

	p.Close()

	assert.EqualValues(t, 0, client.calls.Load())
}

// Test if producer flushes messages after batch achieved correct size and
// during shutdown.
func TestReasonSizeAndDrain(t *testing.T) {
	client := &clientMock{
		incoming:  make(map[int][]*k.PutRecordsRequestEntry),
		responses: []responseMock{correctResponse, correctResponse},
	}

	p := New(Config{FlushInterval: time.Hour, MaxConnections: 1}, client)
	assert.NoError(t, p.Start())

	const maxMessageSize = 8096
	message := makeMessage(maxMessageSize)

	// Put enough messages to trigger batch size-related flush.
	wg := sync.WaitGroup{}
	const messageNum = 700
	wg.Add(messageNum)
	for i := 0; i < messageNum; i++ {
		go func() {
			assert.NoError(t, p.Put(message))
			wg.Done()
		}()
	}
	wg.Wait()

	runtime.Gosched()

	// -race flag increases required wait time significantly.
	assert.Eventually(t, func() bool { return client.calls.Load() == 1 },
		time.Second*5, time.Millisecond*100, "client should be called one time")

	p.Close()

	// We put enough messages to trigger drain during producer shutdown.
	assert.EqualValues(t, 2, client.calls.Load())
}

// Test if producer flushes messages periodically.
func TestReasonInterval(t *testing.T) {
	client := &clientMock{
		incoming:  make(map[int][]*k.PutRecordsRequestEntry),
		responses: []responseMock{correctResponse},
	}

	p := New(Config{FlushInterval: time.Millisecond * 100, MaxConnections: 1}, client)
	assert.NoError(t, p.Start())

	// Put one message and wait for interval flush.
	assert.NoError(t, p.Put(testMessage))

	assert.Eventually(t, func() bool { return client.calls.Load() == 1 },
		time.Second*5, time.Millisecond*100, "client should be called one time")

	// The messages we put should have been flushed on first interval.
	// Check if multiple intervals after that result in no flush.
	time.Sleep(time.Millisecond * 500)
	assert.EqualValues(t, 1, client.calls.Load())

	p.Close()

	// Drain shouldn't result in flush as well.
	assert.EqualValues(t, 1, client.calls.Load())
}

// Test if producer retries to send failed records.
func TestReasonRetry(t *testing.T) {
	client := &clientMock{
		incoming: make(map[int][]*k.PutRecordsRequestEntry),
		responses: []responseMock{
			{
				Error: nil,
				Response: &k.PutRecordsOutput{
					FailedRecordCount: aws.Int64(1),
					Records: []*k.PutRecordsResultEntry{
						{SequenceNumber: aws.String("3"), ShardId: aws.String("1")},
						{ErrorCode: aws.String("400"), ErrorMessage: aws.String("test-error")},
					},
				},
			},
			correctResponse,
		},
	}

	p := New(Config{FlushInterval: time.Millisecond * 100, MaxConnections: 1}, client)
	assert.NoError(t, p.Start())

	// Put two messages large enough to result in two separate
	// Kinesis records. Error is mocked for one of them.
	largeMessage := makeMessage(40000)
	assert.NoError(t, p.Put(largeMessage))
	assert.NoError(t, p.Put(largeMessage))

	// Flush should be called twice. Second time because of retry.
	time.Sleep(time.Millisecond * 400)
	assert.EqualValues(t, 2, client.calls.Load())

	p.Close()
}

func TestMessageValue(t *testing.T) {
	client := &clientMock{
		incoming:  make(map[int][]*k.PutRecordsRequestEntry),
		responses: []responseMock{correctResponse},
	}

	p := New(Config{FlushInterval: time.Millisecond * 100, MaxConnections: 1}, client)
	assert.NoError(t, p.Start())

	// Put one message and wait for interval flush.
	assert.NoError(t, p.Put(testMessage))

	assert.Eventually(t, func() bool { return client.calls.Load() == 1 },
		time.Second*5, time.Millisecond*100, "client should be called one time")
	p.Close()

	assert.Len(t, client.incoming[0], 1)
	actualEntry := client.incoming[0][0]

	// Explicit hash key is not used, Partition key should be a valid KSUID.
	assert.Nil(t, actualEntry.ExplicitHashKey)
	assert.NotZero(t, actualEntry.PartitionKey)
	_, err := ksuid.Parse(*actualEntry.PartitionKey)
	assert.NoError(t, err)

	// Check magic number.
	magicNumber = actualEntry.Data[0:4]
	assert.Equal(t, []byte{0xF3, 0x89, 0x9A, 0xC2}, magicNumber)

	// Check hash.
	data := actualEntry.Data[4 : len(actualEntry.Data)-16]
	hash := actualEntry.Data[len(actualEntry.Data)-16:]
	h := md5.New()
	_, _ = h.Write(data)
	assert.Equal(t, h.Sum(nil), hash)

	// Unmarshall aggregate.
	actualAggregate := &AggregatedRecord{}
	assert.NoError(t, proto.Unmarshal(data, actualAggregate))

	// Partition key should be a valid KSUID.
	assert.Len(t, actualAggregate.PartitionKeyTable, 1)
	_, err = ksuid.Parse(*actualEntry.PartitionKey)
	assert.NoError(t, err)

	// Record should reference the 0th partition key.
	assert.Len(t, actualAggregate.Records, 1)
	assert.EqualValues(t, 0, *actualAggregate.Records[0].PartitionKeyIndex)

	// Decompress and assert message.
	actualMessage, err := gozstd.Decompress(nil, actualAggregate.Records[0].Data)
	assert.NoError(t, err)
	assert.Equal(t, testMessage, actualMessage)
}

type responseMock struct {
	Response *k.PutRecordsOutput
	Error    error
}

func makeMessage(size int) []byte {
	message := make([]byte, size)
	for i := 0; i < len(message); i++ {
		message[i] = byte(rand.Int())
	}
	return message
}

type clientMock struct {
	calls     atomic.Int32
	responses []responseMock
	incoming  map[int][]*k.PutRecordsRequestEntry
}

func (c *clientMock) PutRecordsBuffered(input *k.PutRecordsInput, requestBuffer *bytes.Buffer) (*k.PutRecordsOutput, error) {
	calls := int(c.calls.Load())
	res := c.responses[calls]

	for _, r := range input.Records {
		// Perform a deep copy of input.Records because they may get deallocated
		// or reused by Producer after call to PutRecordsBuffered returns.
		rCopy := &k.PutRecordsRequestEntry{}
		if err := deepcopy.Copy(rCopy, r); err != nil {
			return nil, err
		}

		c.incoming[calls] = append(c.incoming[calls], rCopy)
	}

	c.calls.Inc()
	if res.Error != nil {
		return nil, res.Error
	}

	return res.Response, nil
}
