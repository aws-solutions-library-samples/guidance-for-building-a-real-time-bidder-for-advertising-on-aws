package generator

import (
	"fmt"
	"time"

	as "github.com/aerospike/aerospike-client-go"
)

const (
	deviceSet        = "device"
	audienceSet      = "audience_campaigns"
	campaignSet      = "campaign"
	budgetSet        = "budget"
	budgetBatchesSet = "budget_batches"
)

type asClient struct {
	client           *as.Client
	namespace        string
	budgetBatchesKey string
}

func connectAS(cfg *Config) (*asClient, error) {
	client, err := as.NewClient(cfg.ASHost, cfg.ASPort)
	if err != nil {
		return nil, err
	}
	return &asClient{
		client:           client,
		namespace:        cfg.ASNamespace,
		budgetBatchesKey: cfg.ASBudgetBatchesKey,
	}, nil
}

func (client *asClient) Close() {
	client.client.Close()
}

func (client *asClient) ClearSet(name string) error {
	now := time.Now().UTC()
	return client.client.Truncate(nil, client.namespace, name, &now)
}

func (client *asClient) PutRecord(record Record) error {
	policy := as.NewWritePolicy(0, 0)
	switch item := record.(type) {
	case *Device:
		key, err := as.NewKey(client.namespace, deviceSet, item.DeviceID)
		if err != nil {
			return err
		}
		return client.client.PutBins(policy, key, as.NewBin("audience_id", item.AudienceIds))
	case *Audience:
		policy.SendKey = true
		key, err := as.NewKey(client.namespace, audienceSet, item.AudienceID)
		if err != nil {
			return err
		}
		return client.client.PutBins(policy, key, as.NewBin("campaign_ids", item.CampaignIDs))
	case *Campaign:
		policy.SendKey = true
		key, err := as.NewKey(client.namespace, campaignSet, item.CampaignID)
		if err != nil {
			return err
		}
		return client.client.PutBins(policy, key, as.NewBin("bid_price", item.BidPrice))
	case BudgetBatchDB:
		key, err := as.NewKey(client.namespace, budgetBatchesSet, client.budgetBatchesKey)
		if err != nil {
			return err
		}
		appendToListOp := as.ListAppendOp("keys", item.ID)
		_, err = client.client.Operate(policy, key, appendToListOp)
		if err != nil {
			return err
		}

		key, err = as.NewKey(client.namespace, budgetSet, item.ID)
		if err != nil {
			return err
		}
		return client.client.PutBins(policy, key, as.NewBin("batch", item.Batch))
	}

	return fmt.Errorf("unknown record type: %#v", record)
}
