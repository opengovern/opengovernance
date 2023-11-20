package postgresql

import (
	"fmt"
	"github.com/kaytu-io/kaytu-engine/pkg/workspace/costestimator/price"
	"github.com/kaytu-io/kaytu-engine/pkg/workspace/costestimator/product"
	"github.com/kaytu-io/kaytu-engine/pkg/workspace/db"
)

// ProductRepository implements the product.Repository.
type ProductRepository struct {
	db *db.Database
}

// NewProductRepository returns an implementation of product.Repository.
func NewProductRepository(db *db.Database) *ProductRepository {
	return &ProductRepository{db: db}
}

// Filter returns all the product.Product that match the given product.Filter.
func (r *ProductRepository) Filter(filter *product.Filter) (*price.Price, error) {
	table := fmt.Sprintf("%s %s", *filter.Service, *filter.Family)
	query := fmt.Sprintf("SELECT sku, price_unit, price FROM %s WHERE %s;", tables[table], parseProductFilter(filter))

	var price price.Price
	err := r.db.Orm.Raw(query).Find(&price).Error
	if err != nil {
		return nil, err
	}
	return &price, nil
}

var tables = map[string]string{
	"AmazonEC2 Compute Instance":       "",
	"AmazonEC2 Storage":                "",
	"AmazonEC2 System Operation":       "",
	"AmazonCloudWatch Metric":          "",
	"AmazonEC2 CPU Credits":            "",
	"AWSELB Load Balancer-Network":     "",
	"AWSELB Load Balancer-Gateway":     "",
	"AWSELB Load Balancer":             "",
	"AWSELB Load Balancer-Application": "",
	"AmazonRDS Database Instance":      "",
	"AmazonRDS Database Storage":       "",
	"AmazonRDS Provisioned IOPS":       "",

	"Virtual Machines Compute":   "azure_virtualmachine_prices",
	"Storage Storage":            "azure_managedstorage_prices",
	"Load Balancer Networking":   "azure_loadbalancer_prices",
	"Virtual Network Networking": "azure_virtualnetwork_prices",
	"VPN Gateway Networking":     "azure_vpngateway_prices",
}
