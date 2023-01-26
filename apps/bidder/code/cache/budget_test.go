package cache

import (
	"testing"

	"bidder/code/database/api"
	"bidder/code/id"
	"bidder/code/price"
	apiMocks "bidder/tests/mocks/code/database/api"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// TestEmpty tests that building a budget cache for no campaigns does not panic.
//
// We cannot do anything else with it, since there are no campaigns with budgets to reserve.
func TestEmpty(t *testing.T) {
	dbMock := &apiMocks.BudgetRepository{}
	dbMock.On("FetchAll", mock.Anything).Return(nil)

	assert.NotPanics(t, func() {
		budget, err := newBudget(Config{}, dbMock, &Campaign{})
		assert.NoError(t, err)
		budget.GetCurrentState()
	})
}

// TestBudget tests that budget cache operations return correct remaining amounts.
func TestBudget(t *testing.T) {
	campaignRepoMock := &apiMocks.CampaignRepository{}
	campaignRepoMock.On("Scan", mock.Anything).Return(nil).
		Run(func(args mock.Arguments) {
			fn := args.Get(0).(func(campaign api.Campaign) error)
			assert.NoError(t, fn(api.Campaign{ID: id.FromByteSlice("b"), MaxCPM: 1.}))
			assert.NoError(t, fn(api.Campaign{ID: id.FromByteSlice("a"), MaxCPM: 1.}))
		})

	budgetRepoMock := &apiMocks.BudgetRepository{}
	budgetRepoMock.On("FetchAll", mock.Anything).Return(nil).
		Run(func(args mock.Arguments) {
			fn := args.Get(0).(func(budget api.Budget) error)
			assert.NoError(t, fn(api.Budget{CampaignID: id.FromByteSlice("b"), Available: price.ToInt(16.)}))
			assert.NoError(t, fn(api.Budget{CampaignID: id.FromByteSlice("a"), Available: price.ToInt(1.)}))
		})

	campaign, err := newCampaign(campaignRepoMock)
	assert.NoError(t, err)

	budget, err := newBudget(Config{}, budgetRepoMock, campaign)
	assert.NoError(t, err)
	state := budget.GetCurrentState()

	assert.True(t, state.Reserve(1, price.ToInt(1.)))
	assert.False(t, state.Reserve(1, price.ToInt(1.)))
	state.Free(1, price.ToInt(2.))
	assert.True(t, state.Reserve(1, price.ToInt(0.5)))
	assert.True(t, state.Reserve(1, price.ToInt(0.5)))
	assert.False(t, state.Reserve(1, price.ToInt(0.5)))
}
