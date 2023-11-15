package product

import (
	"github.com/kaytu-io/kaytu-engine/pkg/workspace/costestimator/price"
)

//go:generate mockgen -destination=../mock/product_repository.go -mock_names=Repository=ProductRepository -package mock github.com/kaytu-io/terracost/product Repository

// Repository describes interactions with a storage system to deal with Product entries.
type Repository interface {
	// Filter returns Products with attributes matching the Filter.
	Filter(filter *Filter) (*price.Price, error)
}
