package auction

import (
	"time"

	"bidder/code/cache"
	"bidder/code/database/api"
	"bidder/code/id"
	"bidder/code/metrics"

	"emperror.dev/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"math/rand"
	"strings"
	"encoding/json"
)

// Auction contains all bidder services necessary to perform the auction.
type Auction struct {
	cache *cache.Cache

	pool               *pool
	campaignLookupPool *campaignLookupPool
}

// New returns new auction object.
func New(dataCache *cache.Cache) *Auction {
	return &Auction{
		cache:              dataCache,
		pool:               &pool{},
		campaignLookupPool: &campaignLookupPool{},
	}
}

// Run wraps run method performing the auction.
// Run adds performance metrics and error checking,
// so that same type of errors don't have to be handled
// in multiple places.
// Result is returned via `response` parameter to avoid heap allocation.
func (a *Auction) Run(deadline time.Time, request *Request, response *Response) error {
	persistentData := a.pool.get()
	err := a.run(deadline, request, response, persistentData)
	a.pool.put(persistentData)

	if err != nil {
		// Check if error is caused by database timeout.
		if errors.Is(err, api.ErrTimeout) {
			log.Debug().Str("request ID", string(request.ID)).Msg("request timed out")
			return ErrTimeout
		}

		return err
	}

	return nil
}

// auction runs the auction itself. The method is intended to be called from Run wrapper.
// Result is returned via `response` parameter to avoid heap allocation.
func (a *Auction) run(
	deadline time.Time,
	request *Request,
	response *Response,
	pd *persistentData,
) error {
	rand.Seed(time.Now().UnixNano())
    
	httpposturl := "http://model-internal:8094/v2/models/ctr-lgbm/versions/v0.1.0/infer"
	
	features := []string{}
	for i:=0;i<17;i++ {
		features = append(features, fmt.Sprintf("%v", rand.Intn(2)))
	}
	
	str_features := strings.Join(features, ", ")
	
	jsonStr := `{"inputs": [{"name": "predict-prob", "shape": [1, 17], "datatype": "FP32", "data": [[` + str_features + `]]}]}`

	jsonData := []byte(jsonStr)

	httpReq, error := http.NewRequest("POST", httpposturl, bytes.NewBuffer(jsonData))
	httpReq.Header.Set("Content-Type", "application/json; charset=UTF-8")

	client := &http.Client{}
	httpResp, error := client.Do(httpReq)
	if error != nil {
		panic(error)
	}
	defer httpResp.Body.Close()

	// fmt.Println("response Status:", response.Status)
	// fmt.Println("response Headers:", response.Header)
	body, _ := ioutil.ReadAll(httpResp.Body)
	
	var result map[string][]interface{}
	json.Unmarshal([]byte(body), &result)
	outputs := result["outputs"]
	output := outputs[0]
	arr := output.(map[string]interface{})["data"]
	
	fmt.Println("predict: ", arr)
	
	if zerolog.GlobalLevel() == zerolog.TraceLevel {
		// IDs escape on heap even if log level is higher than trace.
		// That's why the log is wrapped in a 'if' statement.
		log.Trace().Msgf("received bidrequest with ID %s item %s", request.ID, request.Item[0].ID)
	}

	device, err := getDevice(deadline, a.cache, request.DeviceID, pd)
	if err != nil {
		if errors.Is(err, api.ErrItemNotFound) {
			// Return no bid if the device was not found in the db.
			log.Printf("device with id %x not found in the database", request.DeviceID)
			return ErrNoBid
		}
		return err
	}

	log.Trace().Hex("DeviceID", request.DeviceID[:]).Interface("item", device).Msg("got Device item from the db")

	campaignIndices := a.getAudienceCampaigns(device.AudienceIDs, pd)
	if len(campaignIndices) == 0 {
		// Return no bid if the device is not targeted by any campaign.
		log.Printf("device with id %x is not targeted by any campaign", request.DeviceID)
		return ErrNoBid
	}

	campaign := a.chooseCampaign(campaignIndices, pd)
	if campaign == nil {
		metrics.CacheRefreshRequestOnDemandN.Inc()
		a.cache.Budget.AsyncUpdate()

		log.Printf("all campaigns targeting device %x ran out of budget", request.DeviceID)
		return ErrNoBid
	}

	// Build a response for the matched campaign.
	*response = Response{
		Request:  request,
		Item:     &request.Item[0],
		Campaign: campaign,
		Price:    campaign.MaxCPM,
	}
	return nil
}

// getDevice is a convenience wrapper around Get db query.
func getDevice(deadline time.Time, c *cache.Cache, deviceID id.ID, pd *persistentData) (
	*api.Device, error) {
	timer := prometheus.NewTimer(metrics.DeviceQueryTime)
	defer timer.ObserveDuration()

	pd.device.AudienceIDs = pd.device.AudienceIDs[:0]
	if err := c.Device.Get(deadline, deviceID, &pd.device); err != nil {
		return nil, err
	}

	return &pd.device, nil
}

// getAudienceCampaigns reads IDs of campaigns based on a list of audiences.
func (a *Auction) getAudienceCampaigns(audienceIDs []id.ID, pd *persistentData) []int {
	// Initialize lookup table.
	l := a.campaignLookupPool.get()
	l.runID++
	if len(l.lookup) < a.cache.Campaign.Size() {
		l.lookup = make([]int, a.cache.Campaign.Size())
	}

	pd.campaignIndices = pd.campaignIndices[:0]
	for i := 0; i < len(audienceIDs); i++ {
		audienceCampaigns := a.cache.Audience.Get(audienceIDs[i])
		for _, c := range audienceCampaigns {
			if l.lookup[c] != l.runID {
				pd.campaignIndices = append(pd.campaignIndices, c)
				l.lookup[c] = l.runID
			}
		}
	}

	a.campaignLookupPool.put(l)

	return pd.campaignIndices
}

// chooseCampaign returns a campaign of the highest MaxCPM and spends its budget.
func (a *Auction) chooseCampaign(campaignIndices []int, pd *persistentData) *api.Campaign {
	budgets := a.cache.Budget.GetCurrentState()

	winningCPM := int64(0)

	// Find winning campaigns and reserve budget of all
	// campaigns participating in the auction.
	pd.winningAuctions = pd.winningAuctions[:0]
	for _, campaignIndex := range campaignIndices {
		campaignCPM := a.cache.Campaign.Get(campaignIndex).MaxCPM
		budgetLeft := budgets.Reserve(campaignIndex, campaignCPM)

		if !budgetLeft {
			continue
		}

		if campaignCPM > winningCPM {
			winningCPM = campaignCPM
			pd.winningAuctions = pd.winningAuctions[:0]
		}

		if campaignCPM == winningCPM {
			pd.winningAuctions = append(pd.winningAuctions, campaignIndex)
		}
	}

	// Select winning campaign from among campaigns with same MaxCPM.
	winningCampaignID := -1
	if len(pd.winningAuctions) > 0 {
		winningCampaignID = pd.winningAuctions[pd.rng.Intn(len(pd.winningAuctions))]
	}

	// Free	budget of all campaigns participating in the
	// auction, except the wining one.
	for _, campaignIndex := range campaignIndices {
		CPM := a.cache.Campaign.Get(campaignIndex).MaxCPM

		if campaignIndex != winningCampaignID {
			budgets.Free(campaignIndex, CPM)
		}
	}

	if winningCampaignID < 0 {
		return nil
	}

	return a.cache.Campaign.Get(winningCampaignID)
}
