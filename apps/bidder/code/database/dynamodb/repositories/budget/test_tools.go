package budget

import (
	"encoding/binary"

	"bidder/code/database/api"
)

// This file contains budget schemas and conversion functions useful in blackbox tests.

// ItemBatch represents batch of available campaign budgets.
type ItemBatch struct {
	ID    int
	Batch []api.Budget
}

// ItemBatchDB represents ItemBatch in database compressed form.
type ItemBatchDB struct {
	ID        int    `dynamodbav:"i"`
	BatchSize int    `dynamodbav:"s"`
	Batch     []byte `dynamodbav:"b"`
}

// ItemBatchCompress compresses ItemBatch into ItemBatchDB.
func ItemBatchCompress(bb *ItemBatch) ItemBatchDB {
	compressed := ItemBatchDB{
		ID:        bb.ID,
		BatchSize: len(bb.Batch),
	}

	buffer := [8]byte{}
	for _, b := range bb.Batch {
		compressed.Batch = append(compressed.Batch, b.CampaignID[:]...)
		binary.LittleEndian.PutUint64(buffer[:], uint64(b.Available))
		compressed.Batch = append(compressed.Batch, buffer[:]...)
	}

	return compressed
}

// ItemBatchDecompress decompresses ItemBatchDB into ItemBatch.
func ItemBatchDecompress(bb *ItemBatchDB) ItemBatch {
	if bb.BatchSize == 0 {
		return ItemBatch{ID: bb.ID}
	}

	decompressed := ItemBatch{
		ID:    bb.ID,
		Batch: make([]api.Budget, bb.BatchSize),
	}

	const budgetByteSize = 24
	const IDByteSize = 16

	offset := 0
	for i := 0; i < bb.BatchSize; i++ {
		copy(decompressed.Batch[i].CampaignID[:], bb.Batch[offset:])
		decompressed.Batch[i].Available = int64(binary.LittleEndian.Uint64(bb.Batch[offset+IDByteSize:]))
		offset += budgetByteSize
	}

	return decompressed
}
