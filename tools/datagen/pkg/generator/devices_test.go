package generator

import (
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUserGenerator(t *testing.T) {
	cfg := &Config{
		KeyLow: 1, KeyHigh: 1,
		MinAudiences: 2, MaxAudiences: 2,
		MaxAudienceID: 2,
	}

	ch := make(chan Record)

	enc, err := NewDefaultEncryptor()
	assert.NoError(t, err)

	go DeviceGenerator(ch, cfg, enc)

	d := (<-ch).(*Device)

	assert.Equal(t, enc.Decrypt(d.DeviceID), uint64(1))
	assert.Len(t, d.AudienceIds, 32)
	assert.Equal(t, enc.Decrypt(d.AudienceIds[0:16]), uint64(2))
	assert.Equal(t, enc.Decrypt(d.AudienceIds[16:32]), uint64(1))
}

func TestUserPrinter(t *testing.T) {
	w := &strings.Builder{}
	var wg sync.WaitGroup
	wg.Add(1)
	ch := make(chan Record)

	go func() {
		err := DevicePrinter(ch, w)
		assert.NoError(t, err)
		wg.Done()
	}()

	ch <- &Device{DeviceID: Key{0x01}, AudienceIds: Key{0x02, 0x03}}
	close(ch)
	wg.Wait()

	assert.Equal(t, "01\t0203\n", w.String())
}
