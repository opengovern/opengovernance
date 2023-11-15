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
	q := r.db.Orm.Model(tables[table])
	q = parseProductFilter(r.db.Orm, filter)

	var price price.Price
	err := q.Select("sku, price_unit, price").Scan(&price).Error
	if err != nil {
		return nil, err
	}
	return &price, nil
}

var tables = map[string]any{
	"AmazonEC2 Compute Instance":       &db.AwsEC2InstancePrice{},
	"AmazonEC2 Storage":                &db.AwsEC2InstanceStoragePrice{},
	"AmazonEC2 System Operation":       &db.AwsEC2InstanceSystemOperationPrice{},
	"AmazonCloudWatch Metric":          &db.AwsCloudwatchPrice{},
	"AmazonEC2 CPU Credits":            &db.AwsEC2CpuCreditsPrice{},
	"AWSELB Load Balancer-Network":     &db.AwsElasticLoadBalancingPrice{},
	"AWSELB Load Balancer-Gateway":     &db.AwsElasticLoadBalancingPrice{},
	"AWSELB Load Balancer":             &db.AwsElasticLoadBalancingPrice{},
	"AWSELB Load Balancer-Application": &db.AwsElasticLoadBalancingPrice{},
	"AmazonRDS Database Instance":      &db.AwsRdsInstancePrice{},
	"AmazonRDS Database Storage":       &db.AwsRdsStoragePrice{},
	"AmazonRDS Provisioned IOPS":       &db.AwsRdsIopsPrice{},

	"Virtual Machines Compute": &db.AzureVirtualMachinePrice{},
	"Storage Storage":          &db.AzureManagedStoragePrice{},
	"Load Balancer Networking": &db.AzureLoadBalancerPrice{},
}
