package generator

import (
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCampaignGenerator(t *testing.T) {
	enc, err := NewDefaultEncryptor()
	assert.NoError(t, err)

	cfg := &Config{KeyLow: 1, KeyHigh: 1, MinBidPrice: 1000, MaxBidPrice: 1_000_000}

	ch := make(chan Record)

	go CampaignGenerator(ch, cfg, enc)

	res := <-ch
	budget := res.(*Campaign)

	assert.Equal(t, enc.Decrypt(budget.CampaignID), uint64(1))

	assert.GreaterOrEqual(t, budget.BidPrice, cfg.MinBidPrice)
	assert.LessOrEqual(t, budget.BidPrice, cfg.MaxBidPrice)
}

func TestCampaignPrinter(t *testing.T) {
	w := &strings.Builder{}
	var wg sync.WaitGroup
	wg.Add(1)
	ch := make(chan Record)

	go func() {
		err := CampaignPrinter(ch, w)
		assert.NoError(t, err)
		wg.Done()
	}()

	ch <- &Campaign{
		CampaignID: Key{0x01},
		BidPrice:   1,
	}
	close(ch)
	wg.Wait()

	assert.Equal(t, w.String(), "01\t1\n")
}
