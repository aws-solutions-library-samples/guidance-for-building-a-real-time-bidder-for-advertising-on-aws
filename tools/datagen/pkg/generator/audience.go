package generator

import (
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"sync"
)

// ReversedAudience item represents a map from a campaign id to a list of targeted audiences
type ReversedAudience struct {
	CampaignID  Key
	AudienceIDs []Key
}

// Audience represents a map from an audience id to a list of campaigns
// that target the audience
type Audience struct {
	AudienceID  Key   `dynamodbav:"audience_id"`
	CampaignIDs []Key `dynamodbav:"campaign_ids"`
}

// AudiencesConcat creates an []byte of N audience keys based on `seed` value
func AudiencesConcat(seed uint64, cfg *Config, enc *Encryptor) Key {
	n := (seed % uint64(cfg.MaxAudiences-cfg.MinAudiences+1)) + uint64(cfg.MinAudiences)
	a := Key{}
	for i := uint64(0); i < n; i++ {
		a = append(a, enc.Encrypt(((seed+i)%cfg.MaxAudienceID)+1)...)
	}
	return a
}

// AudiencesSlice creates a list of N audience keys based on `seed` value
func AudiencesSlice(seed uint64, cfg *Config, enc *Encryptor) []Key {
	n := (seed % uint64(cfg.MaxAudiences-cfg.MinAudiences+1)) + uint64(cfg.MinAudiences)
	a := make([]Key, n)
	for i := uint64(0); i < n; i++ {
		a[i] = enc.Encrypt(((seed + i) % cfg.MaxAudienceID) + 1)
	}
	return a
}

// ReversedAudienceGenerator generates ReversedAudience items and writes them to the `out` channel
func ReversedAudienceGenerator(out chan<- *ReversedAudience, cfg *Config, enc *Encryptor) {
	defer close(out)

	for i := cfg.KeyLow; i <= cfg.KeyHigh; i++ {
		out <- &ReversedAudience{
			CampaignID:  enc.Encrypt(i),
			AudienceIDs: AudiencesSlice(i, cfg, enc),
		}
	}
}

type reverseIndex struct {
	data map[string]*Audience
}

func newReverseIndex() *reverseIndex {
	return &reverseIndex{data: map[string]*Audience{}}
}

func (ri reverseIndex) add(cid, aid Key) {
	rc, ok := ri.data[hex.EncodeToString(aid)]
	if !ok {
		ri.data[hex.EncodeToString(aid)] = &Audience{
			AudienceID:  aid,
			CampaignIDs: []Key{cid},
		}
		return
	}

	rc.CampaignIDs = append(rc.CampaignIDs, cid)
}

// Process collects ReversedAudience from `in` channel, constructs a reverse index,
// and writes the audience records to the `out` channel
func (ri reverseIndex) Process(in <-chan *ReversedAudience, out chan<- Record) {
	defer close(out)

	for c := range in {
		for _, a := range c.AudienceIDs {
			ri.add(c.CampaignID, a)
		}
	}

	for _, rc := range ri.data {
		out <- rc
	}
}

// AudiencePrinter receives the `Audience` items from `in` channel,
// a writes to `io.Writer` interface
func AudiencePrinter(in <-chan Record, cw io.Writer) error {
	var err error
	var rc *Audience
	for r := range in {
		rc = r.(*Audience)
		_, err = fmt.Fprintf(cw, "%s", hex.EncodeToString(rc.AudienceID))
		if err != nil {
			return err
		}
		for _, cid := range rc.CampaignIDs {
			_, err = fmt.Fprintf(cw, "\t%s", hex.EncodeToString(cid))
			if err != nil {
				return err
			}
		}
		_, err = fmt.Fprintln(cw)
		if err != nil {
			return err
		}
	}
	return nil
}

// GenerateAudiences builds and runs the audience generation pipeline
func GenerateAudiences(cfg *Config) error {
	enc, err := NewDefaultEncryptor()
	if err != nil {
		return err
	}

	c, d := make(chan *ReversedAudience), make(chan Record)

	ri := newReverseIndex()

	go ReversedAudienceGenerator(c, cfg, enc)
	go ri.Process(c, d)

	switch cfg.Output {
	case OutputStdout:
		return AudiencePrinter(d, os.Stdout)
	case OutputDynamodb, OutputAerospike:
		var last error
		worker := func(wg *sync.WaitGroup) {
			defer wg.Done()
			if err := writer(d, cfg); err != nil {
				fmt.Printf("error: %v\n", err)
				last = err
			}
		}
		wg := workGroup(worker, cfg.DynamodbConcurrency)
		wg.Wait()
		return last
	default:
		return fmt.Errorf("unknown output: %v", cfg.Output)
	}
}
