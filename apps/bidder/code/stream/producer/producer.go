package producer

import (
	"bytes"
	"sync"
	"time"

	"emperror.dev/errors"
	// "github.com/ClearcodeHQ/aws-sdk-go" is a fork of "github.com/aws/aws-sdk-go/aws"
	// that introduces kinesis.PutRecordsBuffered function. The kinesis.PutRecordsBuffered
	// accepts a pre-allocated memory buffer which allows to significantly reduce the amount
	// of heap allocation performed by the SDK.
	//
	// Excessive heap allocation and associated GC runs are slow in themselves, but more
	// importantly they can interfere with sync.Pool-based allocation optimizations.
	// Such optimizations are used thorough the Bidder code and in some packages, eg fasthttp.
	// In worst case of interference, the sync.Pool internal critical section can become
	// congested and bottleneck the system.
	k "github.com/ClearcodeHQ/aws-sdk-go/service/kinesis"
	"github.com/ClearcodeHQ/gozstd"
	"github.com/valyala/bytebufferpool"
	"go.uber.org/atomic"
)

var (
	// ErrAlreadyStarted is returned when an attempt to start an already running Producer is made.
	ErrAlreadyStarted = errors.New("kinesis producer was already started")

	// ErrRecordSizeExceeded is returned when trying to Put an over-sized message.
	ErrRecordSizeExceeded = errors.New("data must be less than or equal to 51200 bytes in size")
)

// Putter is the interface that wraps the KinesisAPI.PutRecordsBuffered method.
type Putter interface {
	PutRecordsBuffered(*k.PutRecordsInput, *bytes.Buffer) (*k.PutRecordsOutput, error)
}

// Producer batches records.
//
// A KPL-like batch producer for Amazon Kinesis built on top of the official Go AWS SDK
// and using the same aggregation format that KPL use.
type Producer struct {
	cfg     Config
	client  Putter
	metrics metrics

	inputChan   chan *bytebufferpool.ByteBuffer
	outputChan  chan *batch
	flushTicker *time.Ticker

	started     atomic.Bool
	aggregateWg sync.WaitGroup
	flushWg     sync.WaitGroup

	messagePool bytebufferpool.Pool
	batchPool   batchPool
	entryPool   entryPool
}

// New initializes but not starts a Producer.
func New(cfg Config, client Putter) *Producer {
	return &Producer{
		cfg:     cfg,
		client:  client,
		metrics: newMetrics(cfg.StreamName),

		inputChan: make(chan *bytebufferpool.ByteBuffer, inputBufferSize),

		// Theoretically the buffer in inputChan should be enough to ensure
		// non-blocking operation of Put method. However if the outputChan
		// blocks the aggregate goroutine even for a brief periods of time,
		// the CPU load of aggregate goroutine rises several times, making
		// it unable to drain inputChan in time, bottlenecking the system.
		//
		// Interestingly the CPU profile indicates that all computations
		// performed by aggregate goroutine take more CPU time when the
		// outputChan is blocked. This includes computations completely
		// unrelated to channel operation!
		outputChan: make(chan *batch, outputBufferSize),
	}
}

// Put a message to stream. The message is compressed and then sent
// to a buffered channel. This allows the Put function call to be
// synchronous, as long as the message channel is not full or super
// congested.
func (p *Producer) Put(data []byte) error {
	if messageProtoSize(data)+entryMetadataSize() > maxEntrySize {
		return ErrRecordSizeExceeded
	}

	message := p.messagePool.Get()
	message.B = gozstd.Compress(message.B, data)

	p.metrics.userRecordsPutCnt.Inc()
	p.metrics.userRecordsDataPutSz.Observe(float64(len(message.B)))
	p.metrics.inputBufferCnt.Inc()

	p.inputChan <- message
	return nil
}

// Start starts Kinesis producer by launching aggregate, send and ticker
// goroutines. Starting previously closed producer is not supported.
func (p *Producer) Start() error {
	if !p.started.CAS(false, true) {
		return ErrAlreadyStarted
	}

	// Flush ticker.
	p.flushTicker = time.NewTicker(p.cfg.FlushInterval)

	// Aggregate.
	p.aggregateWg.Add(1)
	go func() {
		p.aggregate()
		p.aggregateWg.Done()
	}()

	// Flush workers.
	p.flushWg.Add(p.cfg.MaxConnections)
	for i := 0; i < p.cfg.MaxConnections; i++ {
		go func() {
			compiler := newBatchCompiler(p)
			requestBuffer := &bytes.Buffer{}
			for b := range p.outputChan {
				p.metrics.outputBufferCnt.Dec()
				p.flush(b, compiler, requestBuffer)
			}
			p.flushWg.Done()
		}()
	}

	return nil
}

// Close the producer by stopping aggregate goroutine and waiting for all
// ongoing Kinesis Put calls to finish. This method causes the remaining
// unsent messages to be flushed to Kinesis.
func (p *Producer) Close() {
	p.flushTicker.Stop()

	close(p.inputChan)
	p.aggregateWg.Wait()

	close(p.outputChan)
	p.flushWg.Wait()
}

func (p *Producer) aggregate() {
	currentBatch := p.batchPool.get()
	lastSend := time.Now()

	send := func(reason string) {
		fullBatch := currentBatch
		fullBatch.flushReason = reason
		currentBatch = p.batchPool.get()

		elapsed := float64(time.Since(lastSend)) / float64(time.Millisecond)
		p.metrics.bufferingTimeDur.Observe(elapsed)
		p.metrics.outputBufferCnt.Inc()

		p.outputChan <- fullBatch
		lastSend = time.Now()
	}

	for {
		select {
		// Message received.
		case msg, ok := <-p.inputChan:
			if !ok {
				send(reasonDrain)
				return
			}

			p.metrics.inputBufferCnt.Dec()

			// Add message to current batch.
			if currentBatch.tryAddMessage(msg) {
				continue
			}

			// Batch is full, send it.
			send(reasonSize)

			// Add message to new, empty batch.
			currentBatch.tryAddMessage(msg)

		// Periodic flush.
		case <-p.flushTicker.C:
			if time.Since(lastSend) > p.cfg.FlushInterval {
				send(reasonInterval)
			}
		}
	}
}
