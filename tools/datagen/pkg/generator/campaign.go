package generator

import (
	"encoding/hex"
	"fmt"
	"io"
	"math/rand"
	"os"
	"sync"
)

// Campaign represent campaign configuration with bid price
type Campaign struct {
	CampaignID Key   `dynamodbav:"campaign_id"`
	BidPrice   int64 `dynamodbav:"bid_price"`
}

// CampaignGenerator creates `Campaign` items a writes them the `out` channel
func CampaignGenerator(out chan<- Record, cfg *Config, enc *Encryptor) {
	defer close(out)

	for i := cfg.KeyLow; i <= cfg.KeyHigh; i++ {
		out <- &Campaign{
			CampaignID: enc.Encrypt(i),
			BidPrice:   (rand.Int63() % (cfg.MaxBidPrice + 1 - cfg.MinBidPrice)) + cfg.MinBidPrice,
		}
	}
}

// CampaignPrinter reads `Campaign` items from `in` channel,
// and writes them to the `io.Writer`
func CampaignPrinter(in <-chan Record, cw io.Writer) error {
	var err error
	for r := range in {
		b := r.(*Campaign)
		_, err = fmt.Fprintf(cw, "%s\t%d\n", hex.EncodeToString(b.CampaignID), b.BidPrice)
		if err != nil {
			return err
		}
	}
	return nil
}

// GenerateCampaigns builds and runs a pipeline that generates campaigns
func GenerateCampaigns(cfg *Config) error {
	enc, err := NewDefaultEncryptor()
	if err != nil {
		return err
	}

	ch := make(chan Record)

	go CampaignGenerator(ch, cfg, enc)

	switch cfg.Output {
	case OutputStdout:
		return CampaignPrinter(ch, os.Stdout)
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
