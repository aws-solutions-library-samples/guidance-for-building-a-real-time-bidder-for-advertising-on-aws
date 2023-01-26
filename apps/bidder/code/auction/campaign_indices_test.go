package auction

import (
	"sort"
	"testing"

	"github.com/mpvl/unique"
	"golang.org/x/exp/rand"
)

// Bench campaign indices merging based on sort.
func BenchmarkCampaignIndicesSort(b *testing.B) {
	b.ReportAllocs()

	for n := 0; n < b.N; n++ {
		b.StopTimer()
		inputCampaignIndices := prepareCampaignIndices()
		b.StartTimer()

		campaigns := []int(nil)
		for _, batch := range inputCampaignIndices {
			campaigns = append(campaigns, batch...)
		}

		sort.Ints(campaigns)
		unique.Unique(unique.IntSlice{P: &campaigns})
	}
}

// Bench campaign indices merging based on map.
//nolint:staticcheck // Not using the campaigns doesn't influence result of the benchmark.
func BenchmarkCampaignIndicesMap(b *testing.B) {
	b.ReportAllocs()

	for n := 0; n < b.N; n++ {
		b.StopTimer()
		inputCampaignIndices := prepareCampaignIndices()
		b.StartTimer()

		campaigns := []int(nil)
		alreadyFound := map[int]bool{}
		for _, batch := range inputCampaignIndices {
			for c := range batch {
				if !alreadyFound[c] {
					campaigns = append(campaigns, c)
					alreadyFound[c] = true
				}
			}
		}
	}
}

// Bench campaign indices merging based on slice used as map.
// This approach is valid because we know the max campaign index beforehand.
// The `alreadyFound` map will have to be stored in sync.Pool.
//nolint:staticcheck // Not using the campaigns doesn't influence result of the benchmark.
func BenchmarkCampaignIndicesPoolSlice(b *testing.B) {
	b.ReportAllocs()
	alreadyFound := make([]int, 1e6)

	for n := 0; n < b.N; n++ {
		b.StopTimer()
		inputCampaignIndices := prepareCampaignIndices()
		b.StartTimer()

		runID := rand.Int()
		campaigns := []int(nil)
		for _, batch := range inputCampaignIndices {
			for c := range batch {
				if alreadyFound[c] != runID {
					campaigns = append(campaigns, c)
					alreadyFound[c] = runID
				}
			}
		}
	}
}

// prepareCampaignIndices prepares 20*20 int array representing
// campaign indices from 20 audiences, each with 20 campaigns.
func prepareCampaignIndices() [][]int {
	inputCampaignIndices := make([][]int, 20)
	for i := range inputCampaignIndices {
		inputCampaignIndices[i] = make([]int, 20)
		for k := range inputCampaignIndices[i] {
			inputCampaignIndices[i][k] = rand.Intn(200)
		}
	}
	return inputCampaignIndices
}
