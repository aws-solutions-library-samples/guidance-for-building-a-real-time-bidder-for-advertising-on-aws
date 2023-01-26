package generator

import (
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAudienceGenerator(t *testing.T) {
	ch := make(chan *ReversedAudience)
	cfg := &Config{
		KeyLow: 1, KeyHigh: 1,
		MinAudiences: 1, MaxAudiences: 1,
		MaxAudienceID: 1,
	}
	enc, err := NewDefaultEncryptor()
	assert.NoError(t, err)

	go ReversedAudienceGenerator(ch, cfg, enc)

	res := <-ch
	assert.Equal(t, enc.Decrypt(res.CampaignID), uint64(1))
	assert.Len(t, res.AudienceIDs, 1)
	assert.Equal(t, enc.Decrypt(res.AudienceIDs[0]), uint64(1))
}

func TestReverseIndex_Process(t *testing.T) {
	cid1, cid2 := Key{0x01}, Key{0x02}
	aid1, aid2 := Key{0x03}, Key{0x04}
	in, out := make(chan *ReversedAudience), make(chan Record)
	ri := newReverseIndex()

	go ri.Process(in, out)

	in <- &ReversedAudience{CampaignID: cid1, AudienceIDs: []Key{aid1, aid2}}
	in <- &ReversedAudience{CampaignID: cid2, AudienceIDs: []Key{aid1, aid2}}
	close(in)

	for r := range out {
		res := r.(*Audience)
		assert.Contains(t, []Key{aid1, aid2}, res.AudienceID)
		for _, c := range res.CampaignIDs {
			assert.Contains(t, []Key{cid1, cid2}, c)
		}
	}
}

func TestAudiencePrinter(t *testing.T) {
	w := &strings.Builder{}
	var wg sync.WaitGroup
	wg.Add(1)
	ch := make(chan Record)

	go func() {
		err := AudiencePrinter(ch, w)
		assert.NoError(t, err)
		wg.Done()
	}()

	ch <- &Audience{AudienceID: Key{0x01}, CampaignIDs: []Key{{0x02}, {0x03}}}
	close(ch)
	wg.Wait()

	assert.Equal(t, "01\t02\t03\n", w.String())
}
