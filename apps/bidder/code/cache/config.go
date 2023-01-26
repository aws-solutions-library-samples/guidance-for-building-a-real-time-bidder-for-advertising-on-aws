package cache

import "time"

// Config holds auction related parameters
type Config struct {
	BudgetSyncPeriod  time.Duration `envconfig:"BUDGET_SYNC_PERIOD_SECONDS" required:"true"`
	BudgetSyncDisable bool          `envconfig:"BUDGET_SYNC_DISABLE" required:"true"`

	DeviceQueryDisable      bool          `envconfig:"DEVICE_QUERY_DISABLE" required:"true"`
	MockDeviceQueryDelay    time.Duration `envconfig:"MOCK_DEVICE_QUERY_DELAY" required:"true"`
	MockDeviceNoBidFraction float64       `envconfig:"MOCK_DEVICE_NO_BID_FRACTION" required:"true"`
}
