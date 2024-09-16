package product

import (
	"github.com/kaytu-io/open-governance/pkg/workspace/costestimator/price"
)

//go:generate mockgen -destination=../mock/product_repository.go -mock_names=Repository=ProductRepository -package mock github.com/kaytu-io/open-governance/pkg/workspace/costestimator/product Repository

// Repository describes interactions with a storage system to deal with Product entries.
type Repository interface {
	// Filter returns Products with attributes matching the Filter.
	Filter(filter *Filter) (*price.Price, error)
}
