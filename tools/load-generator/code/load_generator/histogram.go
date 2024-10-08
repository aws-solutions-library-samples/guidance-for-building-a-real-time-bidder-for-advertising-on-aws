package loadgenerator

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
	"time"
)

// Buckets represents an Histogram's latency buckets.
type Buckets []time.Duration

// Histogram is a bucketed latency Histogram.
type Histogram struct {
	Buckets Buckets
	Counts  []uint64
	Total   uint64
}

// Add implements the Add method of the Report interface by finding the right
// Bucket for the given Result latency and increasing its count by one as well
// as the total count.
func (h *Histogram) Add(r *Result) error {
	if len(h.Counts) != len(h.Buckets) {
		h.Counts = make([]uint64, len(h.Buckets))
	}

	var i int
	for ; i < len(h.Buckets)-1; i++ {
		if r.Latency >= h.Buckets[i] && r.Latency < h.Buckets[i+1] {
			break
		}
	}

	h.Total++
	h.Counts[i]++

	return nil
}

// Merge implements the Merge method of the Report interface by adding the given
// Result to Metrics.
func (h *Histogram) Merge(r Report) error {
	if len(h.Counts) != len(h.Buckets) {
		h.Counts = make([]uint64, len(h.Buckets))
	}

	other := r.(*Histogram)
	if len(other.Counts) != len(other.Buckets) {
		other.Counts = make([]uint64, len(other.Buckets))
	}

	if len(h.Buckets) != len(other.Buckets) {
		return errors.New("error: merging histograms with different bucket count")
	}

	for i := 0; i < len(h.Buckets); i++ {
		if h.Buckets[i] != other.Buckets[i] {
			return errors.New("error: merging histograms with different bucket values")
		}

		h.Counts[i] += other.Counts[i]
	}

	h.Total = other.Total

	return nil
}

// MarshalJSON returns a JSON encoding of the buckets and their counts.
func (h *Histogram) MarshalJSON() ([]byte, error) {
	var buf bytes.Buffer

	// Custom marshaling to guarantee order.
	buf.WriteString("{")
	for i := range h.Buckets {
		if i > 0 {
			buf.WriteString(", ")
		}
		if _, err := fmt.Fprintf(&buf, "\"%d\": %d", h.Buckets[i], h.Counts[i]); err != nil {
			return nil, err
		}
	}
	buf.WriteString("}")

	return buf.Bytes(), nil
}

// Nth returns the nth bucket represented as a string.
func (bs Buckets) Nth(i int) (left, right string) {
	if i >= len(bs)-1 {
		return bs[i].String(), "+Inf"
	}
	return bs[i].String(), bs[i+1].String()
}

// UnmarshalText implements the encoding.TextUnmarshaler interface.
func (bs *Buckets) UnmarshalText(value []byte) error {
	if len(value) < 2 || value[0] != '[' || value[len(value)-1] != ']' {
		return fmt.Errorf("bad buckets: %s", value)
	}
	for _, v := range strings.Split(string(value[1:len(value)-1]), ",") {
		d, err := time.ParseDuration(strings.TrimSpace(v))
		if err != nil {
			return err
		}
		*bs = append(*bs, d)
	}
	if len(*bs) == 0 {
		return fmt.Errorf("bad buckets: %s", value)
	}
	return nil
}
