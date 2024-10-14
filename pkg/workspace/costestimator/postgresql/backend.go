package postgresql

import (
	"github.com/opengovern/opengovernance/pkg/workspace/db"

	"github.com/opengovern/opengovernance/pkg/workspace/costestimator/product"
)

// Backend is the MySQL implementation of the costestimation.Backend, using repositories that connect
// to a MySQL database.
type Backend struct {
	Db          *db.Database
	productRepo *ProductRepository
}

// NewBackend returns a new Backend with a product.Repository and a price.Repository included.
func NewBackend(db *db.Database) *Backend {
	return &Backend{
		Db:          db,
		productRepo: NewProductRepository(db),
	}
}

// Products returns the product.Repository that uses the Backend's querier.
func (b *Backend) Products() product.Repository { return b.productRepo }
