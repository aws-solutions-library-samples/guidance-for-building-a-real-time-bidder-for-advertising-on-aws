package generator

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"math/rand"
	"os"
	"sync"
)

// Budget represents a campaign available budget.
type Budget struct {
	CampaignID []byte
	Available  int64
}

// BudgetBatch represents batch of available campaign budgets.
type BudgetBatch struct {
	ID    int
	Batch []Budget
}

// BudgetBatchDB represents BudgetBatch in database compressed form.
type BudgetBatchDB struct {
	ID        int    `dynamodbav:"i"`
	BatchSize int    `dynamodbav:"s"`
	Batch     []byte `dynamodbav:"b"`
}

// BudgetGenerator creates `BudgetBatch` items a writes them to the `out` channel
func BudgetGenerator(out chan<- Record, cfg *Config, enc *Encryptor) {
	defer close(out)

	currentBatch := BudgetBatch{ID: 1}

	for i := cfg.KeyLow; i <= cfg.KeyHigh; i++ {
		budget := Budget{
			CampaignID: enc.Encrypt(i),
			Available:  (rand.Int63() % (cfg.MaxBudget + 1 - cfg.MinBudget)) + cfg.MinBudget,
		}

		currentBatch.Batch = append(currentBatch.Batch, budget)

		if len(currentBatch.Batch) == cfg.BudgetBatchSize {
			out <- budgetBatchCompress(&BudgetBatch{
				ID:    currentBatch.ID,
				Batch: currentBatch.Batch,
			})

			currentBatch.Batch = make([]Budget, 0)
			currentBatch.ID++
		}
	}

	if len(currentBatch.Batch) > 0 {
		out <- budgetBatchCompress(&currentBatch)
	}
}

// BudgetPrinter reads BudgetBatch items from `in` channel,
// and writes them to the `io.Writer`
func BudgetPrinter(in <-chan Record, cw io.Writer) error {
	var err error
	for r := range in {
		b := r.(BudgetBatchDB)
		_, err = fmt.Fprintf(cw, "%d\t%d\t%s\n", b.ID, b.BatchSize, hex.EncodeToString(b.Batch))
		if err != nil {
			return err
		}
	}
	return nil
}

// GenerateBudgets builds and runs a pipeline that generates budgets
func GenerateBudgets(cfg *Config) error {
	enc, err := NewDefaultEncryptor()
	if err != nil {
		return err
	}

	ch := make(chan Record)

	go BudgetGenerator(ch, cfg, enc)

	switch cfg.Output {
	case OutputStdout:
		return BudgetPrinter(ch, os.Stdout)
	case OutputDynamodb, OutputAerospike:
		var last error
		worker := func(wg *sync.WaitGroup) {
			defer wg.Done()
			if err := writer(ch, cfg); err != nil {
				fmt.Printf("error: %v\n", err)
				last = err
			}
		}
		wg := workGroup(worker, cfg.DynamodbConcurrency)
		wg.Wait()
		return last
	default:
		return fmt.Errorf("unexpected output: %v", cfg.Output)
	}
}

// ClearBudgetBatches clears the budget batches set in aerospike.
// This is useful when we are not sure if there is anything in the
// budget batches set when a new data generation for budget batches
// is triggered. This clears a list of keys of batches. Without that
// new keys could be appended to the old ones.
func ClearBudgetBatches(cfg *Config) error {
	if cfg.Output != OutputAerospike {
		return nil
	}
	asClient, err := connectAS(cfg)
	if err != nil {
		return err
	}
	return asClient.ClearSet(budgetBatchesSet)
}

// budgetBatchCompress compresses BudgetBatch into BudgetBatchDB.
func budgetBatchCompress(bb *BudgetBatch) BudgetBatchDB {
	compressed := BudgetBatchDB{
		ID:        bb.ID,
		BatchSize: len(bb.Batch),
	}

	buffer := [8]byte{}
	for _, b := range bb.Batch {
		compressed.Batch = append(compressed.Batch, b.CampaignID...)
		binary.LittleEndian.PutUint64(buffer[:], uint64(b.Available))
		compressed.Batch = append(compressed.Batch, buffer[:]...)
	}

	return compressed
}
