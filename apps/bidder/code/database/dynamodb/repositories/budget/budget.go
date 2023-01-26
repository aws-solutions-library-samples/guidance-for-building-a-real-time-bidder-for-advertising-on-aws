package budget

import (
	"bidder/code/database/api"
	"bidder/code/database/dynamodb"
)

// Repository allows accessing budget database table.
type Repository struct {
	cfg dynamodb.BudgetConfig
	db  *dynamodb.Client
}

// NewRepository initializes new Repository.
func NewRepository(cfg dynamodb.BudgetConfig, db *dynamodb.Client) (*Repository, error) {
	if err := db.VerifyTable(cfg.TableName); err != nil {
		return nil, err
	}

	return &Repository{
		cfg: cfg,
		db:  db,
	}, nil
}

// FetchAll reads all budget items stored in the database.
func (r *Repository) FetchAll(consume func(budget api.Budget) error) error {
	if r.db.Dax != nil {
		return r.db.ScanDAX(r.cfg.TableName, dynamodb.Consistent, func(item dynamodb.AttributeMapV1) error {
			return unmarshallBudgetAttributesV1(item, consume)
		})
	}

	return r.db.ScanDynamo(r.cfg.TableName, dynamodb.Consistent, func(item dynamodb.AttributeMapV2) error {
		return unmarshallBudgetAttributesV2(item, consume)
	})
}
