package auction

import (
	"testing"

	"bidder/code/cache"
	"bidder/code/database/api"
	"bidder/code/id"
	"bidder/code/price"
	apiMocks "bidder/tests/mocks/code/database/api"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

var campaigns = []api.Campaign{
	{
		ID:     id.FromByteSlice("A000000000000000"),
		MaxCPM: price.ToInt(1.0),
	},
	{
		ID:     id.FromByteSlice("B000000000000000"),
		MaxCPM: price.ToInt(0.75),
	},
	{
		ID:     id.FromByteSlice("C000000000000000"),
		MaxCPM: price.ToInt(0.5),
	},
	{
		ID:     id.FromByteSlice("D000000000000000"),
		MaxCPM: price.ToInt(10),
	},
	{
		ID:     id.FromByteSlice("Equal1000000000"),
		MaxCPM: price.ToInt(1),
	},
	{
		ID:     id.FromByteSlice("Equal2000000000"),
		MaxCPM: price.ToInt(1),
	},
}

var budgets = []api.Budget{
	{
		CampaignID: id.FromByteSlice("A000000000000000"),
		Available:  price.ToInt(1.5),
	},
	{
		CampaignID: id.FromByteSlice("B000000000000000"),
		Available:  price.ToInt(1.5),
	},
	{
		CampaignID: id.FromByteSlice("C000000000000000"),
		Available:  price.ToInt(0.7),
	},
	{
		CampaignID: id.FromByteSlice("D000000000000000"),
		Available:  price.ToInt(20),
	},
	{
		CampaignID: id.FromByteSlice("Equal1000000000"),
		Available:  price.ToInt(1000),
	},
	{
		CampaignID: id.FromByteSlice("Equal2000000000"),
		Available:  price.ToInt(1000),
	},
}

var audiences = []api.Audience{
	{
		AudienceID:  id.FromByteSlice("X000000000000000"),
		CampaignIDs: nil,
	},
	{
		AudienceID:  id.FromByteSlice("A000000000000000"),
		CampaignIDs: id.FromByteSlices("A000000000000000"),
	},
	{
		AudienceID:  id.FromByteSlice("ABC0000000000000"),
		CampaignIDs: id.FromByteSlices("A000000000000000", "B000000000000000", "C000000000000000"),
	},
	{
		AudienceID:  id.FromByteSlice("CD00000000000000"),
		CampaignIDs: id.FromByteSlices("C000000000000000", "D000000000000000"),
	},
}

// Test if campaign indices set is constructed properly for given set of audience IDs.
func TestGetAudienceCampaigns(t *testing.T) {
	c := newMockCache(t)

	auction := New(c)
	pd := newPersistentData()

	// Empty argument list.
	assert.Nil(t, auction.getAudienceCampaigns(nil, pd))

	// Device containing no campaigns.
	assert.Nil(t, auction.getAudienceCampaigns(id.FromByteSlices("X000000000000000"), pd))

	assert.Equal(t, []int{0}, auction.getAudienceCampaigns(id.FromByteSlices("A000000000000000"), pd))
	assert.Equal(t, []int{2, 3}, auction.getAudienceCampaigns(id.FromByteSlices("CD00000000000000"), pd))
	assert.Equal(t, []int{0, 1, 2}, auction.getAudienceCampaigns(id.FromByteSlices("X000000000000000", "ABC0000000000000"), pd))
	assert.Equal(t, []int{0, 1, 2, 3}, auction.getAudienceCampaigns(id.FromByteSlices("ABC0000000000000", "CD00000000000000"), pd))
}

// Test if campaigns are chosen according to
// their MaxCPM and if their budget is spent.
func TestChooseCampaign(t *testing.T) {
	c := newMockCache(t)

	A, _ := c.Campaign.GetIndex(id.FromByteSlice("A000000000000000"))
	B, _ := c.Campaign.GetIndex(id.FromByteSlice("B000000000000000"))
	C, _ := c.Campaign.GetIndex(id.FromByteSlice("C000000000000000"))
	D, _ := c.Campaign.GetIndex(id.FromByteSlice("D000000000000000"))
	Equal1, _ := c.Campaign.GetIndex(id.FromByteSlice("Equal1000000000"))
	Equal2, _ := c.Campaign.GetIndex(id.FromByteSlice("Equal2000000000"))

	auction := New(c)
	pd := newPersistentData()

	// Empty argument list.
	assert.Nil(t, auction.chooseCampaign(nil, pd))

	// Campaigns with successive lower MaxCPM are chosen as budgets are spent.
	assert.Equal(t, id.FromByteSlice("A000000000000000"), auction.chooseCampaign([]int{A, B, C}, pd).ID)
	assert.Equal(t, id.FromByteSlice("B000000000000000"), auction.chooseCampaign([]int{A, B, C}, pd).ID)
	assert.Equal(t, id.FromByteSlice("B000000000000000"), auction.chooseCampaign([]int{A, B, C}, pd).ID)
	assert.Equal(t, id.FromByteSlice("C000000000000000"), auction.chooseCampaign([]int{A, B, C}, pd).ID)
	assert.Nil(t, auction.chooseCampaign([]int{A, B, C}, pd))
	assert.Equal(t, id.FromByteSlice("D000000000000000"), auction.chooseCampaign([]int{A, B, C, D}, pd).ID)
	assert.Equal(t, id.FromByteSlice("D000000000000000"), auction.chooseCampaign([]int{D}, pd).ID)
	assert.Nil(t, auction.chooseCampaign([]int{D}, pd))

	// If MaxCPM is equal, a campaign is chosen at random.
	randomResults := []id.ID(nil)
	for i := 0; i < 1000; i++ {
		randomResults = append(randomResults, auction.chooseCampaign([]int{Equal1, Equal2}, pd).ID)
	}
	assert.Contains(t, randomResults, id.FromByteSlice("Equal1000000000"))
	assert.Contains(t, randomResults, id.FromByteSlice("Equal2000000000"))
}

func newMockCache(t *testing.T) *cache.Cache {
	audienceRepoMock := &apiMocks.AudienceRepository{}
	audienceRepoMock.On("Scan", mock.Anything).Return(nil).
		Run(func(args mock.Arguments) {
			fn := args.Get(0).(func(api.Audience) error)
			for _, audience := range audiences {
				assert.NoError(t, fn(audience))
			}
		})

	budgetRepoMock := &apiMocks.BudgetRepository{}
	budgetRepoMock.On("FetchAll", mock.Anything).Return(nil).
		Run(func(args mock.Arguments) {
			fn := args.Get(0).(func(budget api.Budget) error)
			for _, budget := range budgets {
				assert.NoError(t, fn(budget))
			}
		})

	campaignRepoMock := &apiMocks.CampaignRepository{}
	campaignRepoMock.On("Scan", mock.Anything).Return(nil).
		Run(func(args mock.Arguments) {
			fn := args.Get(0).(func(campaign api.Campaign) error)
			for _, campaign := range campaigns {
				assert.NoError(t, fn(campaign))
			}
		})

	c, err := cache.New(
		cache.Config{},
		audienceRepoMock,
		budgetRepoMock,
		campaignRepoMock,
		nil,
	)
	assert.NoError(t, err)

	return c
}
