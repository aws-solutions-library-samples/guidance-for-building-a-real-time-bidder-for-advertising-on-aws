package metrics

import (
	"sync"

	"github.com/prometheus/client_golang/prometheus"
)

const bufferSize = 1e5

// BufferedSummary is an optimization wrapper around prometheus.Summary.
// BufferedSummary uses buffered channel to minimize critical section
// congestion when writing observations to prometheus.Summary.
type BufferedSummary struct {
	summary prometheus.Summary
	buffer  chan float64

	wg sync.WaitGroup
}

// NewBufferedSummary creates new BufferedSummary.
func NewBufferedSummary(summary prometheus.Summary) *BufferedSummary {
	return &BufferedSummary{
		summary: summary,
		buffer:  make(chan float64, bufferSize),
	}
}

// Observe adds a single observation to the summary.
func (b *BufferedSummary) Observe(value float64) {
	b.buffer <- value
}

// StartService starts the observation gathering goroutine.
func (b *BufferedSummary) StartService() {
	b.wg.Add(1)

	go func() {
		for v := range b.buffer {
			b.summary.Observe(v)
		}
		b.wg.Done()
	}()
}

// CloseService closes the BufferedSummary.
func (b *BufferedSummary) CloseService() {
	close(b.buffer)
	b.wg.Wait()
}
