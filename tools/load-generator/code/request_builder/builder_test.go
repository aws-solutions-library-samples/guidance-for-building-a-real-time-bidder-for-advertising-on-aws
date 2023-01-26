package requestbuilder

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/mxmCherry/openrtb/openrtb3"
	"github.com/stretchr/testify/assert"
)

func Benchmark_generateBidRequest(b *testing.B) {
	for is3, testCase := range []string{"Bid Request 2.x", "Bid Request 3.x"} {
		b.Run(testCase, func(b *testing.B) {
			builder, err := New()
			assert.NoError(b, err)

			for i := 0; i < b.N; i++ {
				_, _, err := builder.Generate(10, 0.5, float64(is3))
				assert.NoError(b, err)
			}
		})
	}
}

func Test_generateBidRequest(t *testing.T) {
	for is3, testCase := range []string{"Bid Request 2.x", "Bid Request 3.x"} {
		t.Run(testCase, func(t *testing.T) {
			builder, err := New()
			assert.NoError(t, err)

			bidRequestRaw, _, err := builder.Generate(10, 0.1, float64(is3))
			assert.NoError(t, err)

			if err := json.Unmarshal(bidRequestRaw, &openrtb3.Body{}); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func Test_generateBidRequestIsCompacted(t *testing.T) {
	for is3, testCase := range []string{"Bid Request 2.x", "Bid Request 3.x"} {
		t.Run(testCase, func(t *testing.T) {
			builder, err := New()
			assert.NoError(t, err)

			bidRequestRaw, _, err := builder.Generate(10, 0.1, float64(is3))
			assert.NoError(t, err)

			var compactBidRequest bytes.Buffer
			if err := json.Compact(&compactBidRequest, bidRequestRaw); err != nil {
				t.Fatal(err)
			}

			assert.Equal(t, compactBidRequest.String(), string(bidRequestRaw))
		})
	}
}
