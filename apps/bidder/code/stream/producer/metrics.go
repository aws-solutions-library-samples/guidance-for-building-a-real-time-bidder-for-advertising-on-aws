package producer

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

const systemName = "go_kinesis_producer"

var timeMillisecondBuckets = []float64{.01, .1, .25, .5, 1, 2.5, 5, 10, 100, 1000, 10000, 60000}
var sizeByteBuckets = []float64{1, 16, 64, 256, 512, 1024, 16384, 65536, 262144, 1048576, 4194304}

var (
	userRecordsPutCnt = promauto.NewCounterVec(prometheus.CounterOpts{
		Subsystem: systemName,
		Name:      "user_records_put_total",
		Help:      "Count of how many logical user records were received by the Kinesis producer for put operations.",
	}, []string{"stream"})

	userRecordsDataPutSz = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Subsystem: systemName,
		Name:      "user_records_data_put_bytes",
		Help:      "Bytes in the logical user records were received by the Kinesis producer for put operations.",
		Buckets:   sizeByteBuckets,
	}, []string{"stream"})

	kinesisRecordsPutCnt = promauto.NewCounterVec(prometheus.CounterOpts{
		Subsystem: systemName,
		Name:      "kinesis_records_put_total",
		Help: "Count of how many Kinesis Data Streams records were put successfully" +
			"(each Kinesis Data Streams record can contain multiple user records).",
	}, []string{"stream", "shard"})

	kinesisRecordsDataPutSz = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Subsystem: systemName,
		Name:      "kinesis_records_data_put_bytes",
		Help:      "Bytes in the Kinesis Data Streams records.",
		Buckets:   sizeByteBuckets,
	}, []string{"stream"})

	errorsByCodeCnt = promauto.NewCounterVec(prometheus.CounterOpts{
		Subsystem: systemName,
		Name:      "errors_by_code_total",
		Help:      "Count of each type of error code.",
	}, []string{"stream", "code"})

	allErrorsCnt = promauto.NewCounterVec(prometheus.CounterOpts{
		Subsystem: systemName,
		Name:      "errors_total",
		Help:      "This is triggered by the same errors as Errors by Code, but does not distinguish between types.",
	}, []string{"stream"})

	retriesPerRecordSum = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Subsystem: systemName,
		Name:      "retries_per_record",
		Help:      "Number of retries performed per kinesis record. Zero is emitted for records that succeed in one try.",
		Buckets:   sizeByteBuckets,
	}, []string{"stream"})

	bufferingTimeDur = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Subsystem: systemName,
		Name:      "buffering_time_milliseconds",
		Help:      "The time between a user record arriving at the Kinesis producer and leaving for the backend.",
		Buckets:   timeMillisecondBuckets,
	}, []string{"stream"})

	requestTimeDur = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Subsystem: systemName,
		Name:      "request_time_milliseconds",
		Help:      "The time it takes to perform PutRecordsRequests.",
		Buckets:   timeMillisecondBuckets,
	}, []string{"stream"})

	userRecordsPerKinesisRecordSum = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Subsystem: systemName,
		Name:      "user_records_per_kinesis_record",
		Help:      "The number of logical user records aggregated into a single Kinesis Data Streams record.",
		Buckets:   sizeByteBuckets,
	}, []string{"stream"})

	kinesisRecordsPerPutRecordsRequestSum = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Subsystem: systemName,
		Name:      "kinesis_records_per_put_records_request",
		Help:      "The number of Kinesis Data Streams records aggregated into a single PutRecordsRequest.",
		Buckets:   sizeByteBuckets,
	}, []string{"stream"})

	inputBufferCnt = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Subsystem: systemName,
		Name:      "input_buffer_size",
		Help:      "Amount of messages stored in Kinesis producer input buffer.",
	}, []string{"stream"})

	outputBufferCnt = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Subsystem: systemName,
		Name:      "output_buffer_size",
		Help:      "Amount of AWS SDK requests stored in Kinesis producer output buffer.",
	}, []string{"stream"})
)

type metrics struct {
	userRecordsPutCnt                     prometheus.Counter
	userRecordsDataPutSz                  prometheus.Observer
	kinesisRecordsPutCnt                  *prometheus.CounterVec
	kinesisRecordsDataPutSz               prometheus.Observer
	errorsByCodeCnt                       *prometheus.CounterVec
	allErrorsCnt                          prometheus.Counter
	retriesPerRecordSum                   prometheus.Observer
	bufferingTimeDur                      prometheus.Observer
	requestTimeDur                        prometheus.Observer
	userRecordsPerKinesisRecordSum        prometheus.Observer
	kinesisRecordsPerPutRecordsRequestSum prometheus.Observer
	inputBufferCnt                        prometheus.Gauge
	outputBufferCnt                       prometheus.Gauge
}

func newMetrics(streamName string) metrics {
	return metrics{
		userRecordsPutCnt:                     userRecordsPutCnt.WithLabelValues(streamName),
		userRecordsDataPutSz:                  userRecordsDataPutSz.WithLabelValues(streamName),
		kinesisRecordsPutCnt:                  kinesisRecordsPutCnt,
		kinesisRecordsDataPutSz:               kinesisRecordsDataPutSz.WithLabelValues(streamName),
		errorsByCodeCnt:                       errorsByCodeCnt,
		allErrorsCnt:                          allErrorsCnt.WithLabelValues(streamName),
		retriesPerRecordSum:                   retriesPerRecordSum.WithLabelValues(streamName),
		bufferingTimeDur:                      bufferingTimeDur.WithLabelValues(streamName),
		requestTimeDur:                        requestTimeDur.WithLabelValues(streamName),
		userRecordsPerKinesisRecordSum:        userRecordsPerKinesisRecordSum.WithLabelValues(streamName),
		kinesisRecordsPerPutRecordsRequestSum: kinesisRecordsPerPutRecordsRequestSum.WithLabelValues(streamName),
		inputBufferCnt:                        inputBufferCnt.WithLabelValues(streamName),
		outputBufferCnt:                       outputBufferCnt.WithLabelValues(streamName),
	}
}
