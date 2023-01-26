package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"sort"
	"time"

	"emperror.dev/errors"
	vegeta "github.com/tsenart/vegeta/v12/lib"
)

type metricsWithBuckets struct {
	vegeta.Metrics
	Buckets map[time.Duration]uint64 `json:"buckets,omitempty"`
}

func loadFromLocalFiles(conf *config) ([]vegeta.Metrics, error) {
	inputs := make([]vegeta.Metrics, 0)

	if len(conf.InputFiles) < 1 {
		return nil, errors.New("at least one input file must be provided")
	}

	for _, filename := range conf.InputFiles {
		metrics, err := loadMetricsFromJSONFile(filename)

		if err != nil {
			return nil, errors.Wrap(err, fmt.Sprintf("unable to load file '%s'", filename))
		}

		inputs = append(inputs, *metrics)
	}

	return inputs, nil
}

func loadMetricsFromJSONFile(filename string) (*vegeta.Metrics, error) {
	data, err := ioutil.ReadFile(filename)

	if err != nil {
		log.Fatal(err)
	}

	return unmarshallMetrics(data)
}

func unmarshallMetrics(data []byte) (*vegeta.Metrics, error) {
	var mb metricsWithBuckets

	if err := json.Unmarshal(data, &mb); err != nil {
		return nil, err
	}

	if mb.Buckets != nil {
		unmarshallHistogram(&mb)
	}

	return &mb.Metrics, nil
}

func unmarshallHistogram(mb *metricsWithBuckets) {
	var histogram vegeta.Histogram
	mb.Metrics.Histogram = &histogram

	buckets := make([]time.Duration, 0, len(mb.Buckets))

	for k := range mb.Buckets {
		buckets = append(buckets, k)
	}

	sort.Slice(buckets, func(i, j int) bool { return buckets[i] < buckets[j] })
	histogram.Buckets = buckets

	for _, bucket := range buckets {
		value := mb.Buckets[bucket]

		histogram.Counts = append(histogram.Counts, value)
		histogram.Total += value
	}
}
