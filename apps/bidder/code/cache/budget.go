package cache

import (
	"sync"
	"time"

	"bidder/code/database/api"
	"bidder/code/metrics"

	"emperror.dev/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog/log"
	"go.uber.org/atomic"
)

// Budget contains cached campaign budgets.
type Budget struct {
	state      *BudgetState
	stateMutex *sync.RWMutex

	updateInProgress atomic.Bool
	updateTicker     *time.Ticker
	updateChannel    chan bool
	updateWg         sync.WaitGroup

	cfg        Config
	repository api.BudgetRepository
	campaign   *Campaign
}

// BudgetState contains budget state obtained during each cache update.
// The containing struct Budget atomically updates a pointer to BudgetState
// after each update.
type BudgetState struct {
	available []atomic.Int64
}

// GetCurrentState returns current state of the budget.
// This method should be used by auction to obtain snapshot of budget cache state.
// To maintain cache consistency, a single auction should perform all budget-related
// operations on a single budget snapshot.
func (b *Budget) GetCurrentState() *BudgetState {
	b.stateMutex.RLock()
	defer b.stateMutex.RUnlock()
	return b.state
}

// AsyncUpdate starts budget cache update, if another update is not already in progress.
func (b *Budget) AsyncUpdate() {
	b.updateChannel <- true
}

// Reserve increases campaign spent budget by 'value' and returns false if
// campaign went over budget.
func (b *BudgetState) Reserve(campaignIndex int, value int64) bool {
	return b.available[campaignIndex].Sub(value) >= 0
}

// Free decreases campaign spent budget by 'value'.
func (b *BudgetState) Free(campaignIndex int, value int64) {
	b.available[campaignIndex].Add(value)
}

// newBudget initializes campaign budget cache.
func newBudget(cfg Config, repository api.BudgetRepository, campaign *Campaign) (*Budget, error) {
	cache := &Budget{
		stateMutex: &sync.RWMutex{},
		cfg:        cfg,
		repository: repository,
		campaign:   campaign,
	}

	newState, err := cache.scanState()
	if err != nil {
		return nil, errors.Wrap(err, "error while initializing budget cache")
	}

	cache.state = newState

	return cache, nil
}

// start periodic and on request budget update. stop needs to
// be called to free resources allocated by start.
func (b *Budget) start() {
	b.updateWg = sync.WaitGroup{}
	b.updateChannel = make(chan bool)

	updatePeriod := b.cfg.BudgetSyncPeriod
	if updatePeriod == 0 || b.cfg.BudgetSyncDisable {
		const never = time.Duration(1<<63 - 1)
		updatePeriod = never
	}

	b.updateTicker = time.NewTicker(updatePeriod)

	b.updateWg.Add(1)
	go func() {
		defer b.updateWg.Done()
		for {
			select {
			case run := <-b.updateChannel:
				if run {
					b.asyncUpdate()
				} else {
					return
				}
			case <-b.updateTicker.C:
				metrics.CacheRefreshRequestRecurringN.Inc()
				b.asyncUpdate()
			}
		}
	}()
}

// stop budget update and free associated resources.
func (b *Budget) stop() {
	b.updateTicker.Stop()
	b.updateChannel <- false
	b.updateWg.Wait()
	close(b.updateChannel)
}

// asyncUpdate initializes new budget values based on data downloaded from database.
// It does nothing if update is already in progress.
func (b *Budget) asyncUpdate() {
	if !b.updateInProgress.CAS(false, true) {
		// Update already in progress.
		return
	}

	// Launching update in goroutine to not block update channel.
	b.updateWg.Add(1)
	go func() {
		timer := prometheus.NewTimer(metrics.BudgetUpdateTime)
		defer timer.ObserveDuration()

		defer b.updateWg.Done()
		defer b.updateInProgress.Store(false)

		log.Debug().Msg("budget cache update")

		newState, err := b.scanState()
		if err != nil {
			log.Error().Err(errors.Wrap(err, "error while updating budget")).Msg("")
			return
		}

		b.stateMutex.Lock()
		b.state = newState
		b.stateMutex.Unlock()
	}()
}

// scanState initialized new budget values based on data obtained from database.
func (b *Budget) scanState() (*BudgetState, error) {
	state := BudgetState{
		available: make([]atomic.Int64, b.campaign.Size()),
	}

	consumer := func(budget api.Budget) error {
		campaignIndex, found := b.campaign.GetIndex(budget.CampaignID)

		if !found {
			log.Trace().Hex("campaignID", budget.CampaignID[:]).Msg("unknown campaign during budget update")
			return nil
		}

		state.available[campaignIndex].Add(budget.Available)
		return nil
	}

	if err := b.repository.FetchAll(consumer); err != nil {
		metrics.DBBudgetScanErrorsN.Inc()
		return nil, err
	}

	return &state, nil
}
