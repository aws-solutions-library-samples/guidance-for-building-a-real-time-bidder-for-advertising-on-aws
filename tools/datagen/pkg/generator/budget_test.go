package generator

import (
	"encoding/binary"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBudgetGenerator(t *testing.T) {
	ch := make(chan Record)
	cfg := &Config{KeyLow: 1, KeyHigh: 200, MinBudget: 1, MaxBudget: 1, BudgetBatchSize: 170}
	enc, err := NewDefaultEncryptor()
	assert.NoError(t, err)

	go BudgetGenerator(ch, cfg, enc)

	compressed := (<-ch).(BudgetBatchDB)
	actual := budgetBatchDecompress(&compressed)

	assert.Equal(t, 1, actual.ID)
	assert.Len(t, actual.Batch, 170)
	assert.Equal(t, enc.Decrypt(actual.Batch[0].CampaignID), uint64(1))
	assert.NotZero(t, actual.Batch[0].Available)

	compressed = (<-ch).(BudgetBatchDB)
	actual = budgetBatchDecompress(&compressed)

	assert.Equal(t, 2, actual.ID)
	assert.Len(t, actual.Batch, 30)
	assert.Equal(t, enc.Decrypt(actual.Batch[0].CampaignID), uint64(171))
	assert.NotZero(t, actual.Batch[0].Available)
}

// budgetBatchDecompress decompresses BudgetBatchDB into BudgetBatch.
func budgetBatchDecompress(bb *BudgetBatchDB) BudgetBatch {
	if bb.BatchSize == 0 {
		return BudgetBatch{ID: bb.ID}
	}

	decompressed := BudgetBatch{
		ID:    bb.ID,
		Batch: make([]Budget, bb.BatchSize),
	}

	const budgetByteSize = 24
	const IDByteSize = 16

	offset := 0
	for i := 0; i < bb.BatchSize; i++ {
		decompressed.Batch[i].CampaignID = make([]byte, 16)
		copy(decompressed.Batch[i].CampaignID, bb.Batch[offset:])
		decompressed.Batch[i].Available = int64(binary.LittleEndian.Uint64(bb.Batch[offset+IDByteSize:]))
		offset += budgetByteSize
	}

	return decompressed
}
