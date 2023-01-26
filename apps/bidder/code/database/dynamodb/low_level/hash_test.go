package lowlevel

import (
	"testing"
)

func BenchmarkHashAlloc(b *testing.B) {
	b.ReportAllocs()
	h := newHasher()

	data := []byte("asgfadgkuadkgadkjf")
	buf := []byte(nil)

	for n := 0; n < b.N; n++ {
		h.sha256(data, &buf)
	}
}
