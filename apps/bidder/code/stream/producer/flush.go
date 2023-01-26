package producer

import (
	"bytes"
	"time"

	"github.com/ClearcodeHQ/aws-sdk-go/service/kinesis"
	"github.com/jpillora/backoff"
	"github.com/rs/zerolog/log"
)

// flush compiles a message batch, sends it to Kinesis and
// releases associated resource.
func (p *Producer) flush(b *batch, c *batchCompiler, requestBuffer *bytes.Buffer) {
	if len(b.messages) == 0 {
		log.Debug().Str("reason", b.flushReason).Msg("kinesis Producer: no messages to send")
		return
	}

	request, err := c.compileBatch(b)

	// Release message and batch buffers, regardless of err returned by compileBatch.
	flushReason := b.flushReason
	for _, b := range b.messages {
		p.messagePool.Put(b)
	}
	p.batchPool.put(b)

	if err != nil {
		log.Error().Err(err).Msg("kinesis Producer: compiling batch")
		return
	}

	// Release entry buffers.
	defer func() {
		for _, e := range request.Records {
			p.entryPool.put(e)
		}
	}()

	if err := p.sendRequest(request, flushReason, requestBuffer); err != nil {
		log.Error().Err(err).Msg("kinesis Producer: flush")
		return
	}
}

// sendRequest sends Kinesis request and retries failures if necessary.
// for example: when we get "ProvisionedThroughputExceededException"
func (p *Producer) sendRequest(
	request *kinesis.PutRecordsInput,
	reason string,
	requestBuffer *bytes.Buffer,
) error {
	b := &backoff.Backoff{
		Jitter: true,
	}

	numRetries := 0
	numRecords := len(request.Records)

	for {
		log.Debug().
			Str("reason", reason).
			Int("records", len(request.Records)).
			Msg("kinesis Producer: flushing records")

		start := time.Now()

		out, err := p.client.PutRecordsBuffered(request, requestBuffer)

		elapsed := float64(time.Since(start)) / float64(time.Millisecond)
		p.metrics.requestTimeDur.Observe(elapsed)

		if err != nil {
			return err
		}

		for _, r := range out.Records {
			if r.ErrorCode != nil {
				log.Trace().
					Str("ErrorCode", *r.ErrorCode).
					Str("ErrorMessage", *r.ErrorMessage).
					Msg("kinesis Producer put failure")
				p.metrics.errorsByCodeCnt.WithLabelValues(p.cfg.StreamName, *r.ErrorCode).Inc()
				p.metrics.allErrorsCnt.Inc()
			} else {
				log.Trace().
					Str("ShardId", *r.ShardId).
					Str("SequenceNumber", *r.SequenceNumber).
					Msg("kinesis Producer put failure")
				p.metrics.kinesisRecordsPutCnt.WithLabelValues(p.cfg.StreamName, *r.ShardId).Inc()
			}
		}

		failed := *out.FailedRecordCount
		if failed == 0 {
			if numRetries != 0 {
				p.metrics.retriesPerRecordSum.Observe(float64(numRecords) / float64(numRetries))
			} else {
				p.metrics.retriesPerRecordSum.Observe(0)
			}
			return nil
		}

		duration := b.Duration()

		log.Debug().
			Int64("failures", failed).
			Str("backoff", duration.String()).
			Msg("kinesis Producer put failures")

		time.Sleep(duration)

		// change the logging state for the next iteration
		reason = reasonRetry
		request.Records = failures(request.Records, out.Records)
		numRetries++
	}
}

// failures returns the failed records as indicated in the response.
func failures(records []*kinesis.PutRecordsRequestEntry,
	response []*kinesis.PutRecordsResultEntry) (out []*kinesis.PutRecordsRequestEntry) {
	for i, record := range response {
		if record.ErrorCode != nil {
			out = append(out, records[i])
		}
	}
	return
}
