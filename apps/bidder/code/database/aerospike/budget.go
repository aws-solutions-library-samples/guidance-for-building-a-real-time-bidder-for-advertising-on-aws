package aerospike

import (
	"bidder/code/database/api"
	"bidder/code/id"
	"encoding/binary"

	"emperror.dev/errors"
	"github.com/aerospike/aerospike-client-go"
)

// BudgetRepository allows accessing budget Client set.
type BudgetRepository struct {
	aerospike        *Client
	disableScan      bool
	budgetBatchesKey string
	asGetPolicy      *aerospike.BasePolicy
	// E2E tests need bidder to access different keys in the same namespace
	// when querying for the budget batches.
}

// NewBudgetRepository creates new instance of *AudienceRepository.
func NewBudgetRepository(aerospikeClient *Client, cfg Config) (*BudgetRepository, error) {
	asGetPolicy := aerospikeClient.GetPolicy()
	asGetPolicy.TotalTimeout = cfg.BudgetGetTotalTimeout
	return &BudgetRepository{
		aerospikeClient,
		cfg.DisableScan,
		cfg.BudgetBatchesKey,
		asGetPolicy,
	}, nil
}

// FetchAll reads all budgets records stored in the database.
func (r *BudgetRepository) FetchAll(consume func(api.Budget) error) error {
	if r.disableScan {
		return r.getAll(consume)
	}
	return r.scan(consume)
}

// scan reads all budgets records stored in the database using scan operation.
func (r *BudgetRepository) scan(consume func(api.Budget) error) error {
	recordsChan, err := r.aerospike.ScanAll(BudgetSet)
	if err != nil {
		return err
	}

	for res := range recordsChan {
		if res.Err != nil {
			return errors.Wrap(res.Err, "error during scanning budget set")
		}

		batchBin := res.Record.Bins["batch"].([]byte)
		if batchBin == nil {
			continue
		}

		for offset := 0; offset < len(batchBin); offset += api.BudgetByteSize {
			campaignID := id.ID{}
			copy(campaignID[:], batchBin[offset:])
			err := consume(api.Budget{
				CampaignID: campaignID,
				Available:  int64(binary.LittleEndian.Uint64(batchBin[offset+id.Len:])),
			})
			if err != nil {
				return errors.Wrap(err, "error during consuming")
			}
		}
	}

	return nil
}

// getAll reads all budgets records stored in the database using get operation.
// First it reads a list of budget batches keys so that we know what to iterate
// over when looking for budget batches. Then, fetches the batches one by one
// and for every batch it splits the batch bytes into campaigns data.
func (r *BudgetRepository) getAll(consume func(api.Budget) error) error {
	// First, fetch a list of current batches keys
	asKey, err := aerospike.NewKey(r.aerospike.Namespace, BudgetBatchesSet, r.budgetBatchesKey)
	if err != nil {
		return errors.Wrap(err, "failed to create batches key")
	}
	result, err := r.aerospike.Get(asKey, r.asGetPolicy)
	if err != nil {
		return errors.Wrap(err, "failed to get budget batches keys")
	}

	// Fetch budget batches
	keysToFetch := result.Bins["keys"].([]interface{})
	for _, key := range keysToFetch {
		asKey, err = aerospike.NewKey(r.aerospike.Namespace, BudgetSet, key.(int))
		if err != nil {
			return err
		}

		result, err = r.aerospike.Get(asKey, r.asGetPolicy)
		if err != nil {
			return errors.Wrap(err, "failed to get budget batches")
		}

		batchBin := result.Bins["batch"].([]byte)
		if batchBin == nil {
			continue
		}

		// Extract campaigns data from the batch.
		for offset := 0; offset < len(batchBin); offset += api.BudgetByteSize {
			campaignID := id.ID{}
			copy(campaignID[:], batchBin[offset:])
			err := consume(api.Budget{
				CampaignID: campaignID,
				Available:  int64(binary.LittleEndian.Uint64(batchBin[offset+id.Len:])),
			})
			if err != nil {
				return errors.Wrap(err, "error during consuming")
			}
		}
	}

	return nil
}
