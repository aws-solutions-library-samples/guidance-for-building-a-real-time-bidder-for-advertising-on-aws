package main

import (
	"errors"
	"math"
	"strconv"
	"time"

	vegeta "github.com/tsenart/vegeta/v12/lib"
)

func aggregateMetrics(inputs []vegeta.Metrics) (*vegeta.Metrics, error) {
	var aggregated vegeta.Metrics

	if err := ensureHistogramCompatibility(inputs); err != nil {
		return nil, err
	}

	aggregateSimpleMetrics(&aggregated, inputs)
	aggregateHistograms(&aggregated, inputs)
	aggregateLatencies(&aggregated, inputs)

	return &aggregated, nil
}

func aggregateSimpleMetrics(aggregated *vegeta.Metrics, inputs []vegeta.Metrics) {
	aggregated.Errors = make([]string, 0)
	aggregated.StatusCodes = make(map[string]int)
	aggregated.Earliest = time.Now()

	var success uint64 = 0
	var weightedBytesInMeanSum float64 = 0
	var weightedBytesOutMeanSum float64 = 0

	for _, input := range inputs {
		aggregated.Requests += input.Requests
		aggregated.Wait += input.Wait
		aggregated.Duration += input.Duration
		aggregated.BytesIn.Total += input.BytesIn.Total
		aggregated.BytesOut.Total += input.BytesOut.Total
		aggregated.Rate += input.Rate
		aggregated.Throughput += input.Throughput

		weightedBytesInMeanSum += input.BytesIn.Mean * float64(input.Requests)
		weightedBytesOutMeanSum += input.BytesOut.Mean * float64(input.Requests)

		aggregated.Errors = append(aggregated.Errors, input.Errors...)

		if aggregated.Earliest.After(input.Earliest) {
			aggregated.Earliest = input.Earliest
		}

		if aggregated.Latest.Before(input.Latest) {
			aggregated.Latest = input.Latest
		}

		if aggregated.Latest.Before(input.End) {
			aggregated.End = input.End
		}

		for code, count := range input.StatusCodes {
			aggregated.StatusCodes[code] += count
			intCode, err := strconv.ParseInt(code, 10, 32)

			if err == nil && intCode >= 200 && intCode < 400 {
				success += uint64(count)
			}
		}
	}

	aggregated.Success = float64(success) / float64(aggregated.Requests)
	aggregated.BytesIn.Mean = weightedBytesInMeanSum / float64(aggregated.Requests)
	aggregated.BytesOut.Mean = weightedBytesOutMeanSum / float64(aggregated.Requests)

	aggregated.Duration /= time.Duration(len(inputs))
	aggregated.Wait /= time.Duration(len(inputs))
}

func aggregateLatencies(aggregated *vegeta.Metrics, inputs []vegeta.Metrics) {
	aggregated.Latencies.Min = time.Duration(math.MaxInt64)

	var weightedMeanSum uint64 = 0
	var sumOfRequests uint64 = 0

	for _, input := range inputs {
		aggregated.Latencies.Total += input.Latencies.Total

		if aggregated.Latencies.Max < input.Latencies.Max {
			aggregated.Latencies.Max = input.Latencies.Max
		}

		if aggregated.Latencies.Min > input.Latencies.Min {
			aggregated.Latencies.Min = input.Latencies.Min
		}

		weightedMeanSum += uint64(input.Latencies.Mean) * input.Requests
		sumOfRequests += input.Requests
	}

	aggregated.Latencies.Mean = time.Duration(weightedMeanSum / sumOfRequests)

	aggregated.Latencies.P50 = histogramPercentile(aggregated.Histogram, 50)
	aggregated.Latencies.P90 = histogramPercentile(aggregated.Histogram, 90)
	aggregated.Latencies.P95 = histogramPercentile(aggregated.Histogram, 95)
	aggregated.Latencies.P99 = histogramPercentile(aggregated.Histogram, 99)
}

func ensureHistogramCompatibility(inputs []vegeta.Metrics) error {
	firstInput := inputs[0]

	for _, input := range inputs[1:] {
		if len(firstInput.Histogram.Buckets) != len(input.Histogram.Buckets) {
			return errors.New("histogram buckets counts are not the same across all of the inputs")
		}

		for i := 0; i < len(firstInput.Histogram.Buckets); i++ {
			if firstInput.Histogram.Buckets[i] != input.Histogram.Buckets[i] {
				return errors.New("histogram buckets are not the same across all of the inputs")
			}
		}
	}

	return nil
}

func aggregateHistograms(aggregated *vegeta.Metrics, inputs []vegeta.Metrics) {
	var hist vegeta.Histogram
	aggregated.Histogram = &hist

	hist.Buckets = append(hist.Buckets, inputs[0].Histogram.Buckets...)
	hist.Counts = make([]uint64, len(hist.Buckets))

	for _, input := range inputs {
		for index, val := range input.Histogram.Counts {
			hist.Counts[index] += val
			hist.Total += val
		}
	}
}

//nolint:gomnd // it forces us to declare percentile divider
func histogramPercentile(hist *vegeta.Histogram, percentile float64) time.Duration {
	var totalToCurrentIndex uint64 = 0
	countAtPercentile := uint64(math.Ceil((percentile / 100) * float64(hist.Total)))

	for i := 0; i < len(hist.Buckets); i++ {
		totalToCurrentIndex += hist.Counts[i]

		if totalToCurrentIndex >= countAtPercentile {
			return hist.Buckets[i]
		}
	}

	return 0
}
