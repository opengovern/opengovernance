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
	"AmazonEC2 Compute Instance":       &AwsEC2InstancePrice{},
	"AmazonEC2 Storage":                &AwsEC2InstanceStoragePrice{},
	"AmazonEC2 System Operation":       &AwsEC2InstanceSystemOperationPrice{},
	"AmazonCloudWatch Metric":          &AwsCloudwatchPrice{},
	"AmazonEC2 CPU Credits":            &AwsEC2CpuCreditsPrice{},
	"AWSELB Load Balancer-Network":     &AwsElasticLoadBalancingPrice{},
	"AWSELB Load Balancer-Gateway":     &AwsElasticLoadBalancingPrice{},
	"AWSELB Load Balancer":             &AwsElasticLoadBalancingPrice{},
	"AWSELB Load Balancer-Application": &AwsElasticLoadBalancingPrice{},
	"AmazonRDS Database Instance":      &AwsRdsInstancePrice{},
	"AmazonRDS Database Storage":       &AwsRdsStoragePrice{},
	"AmazonRDS Provisioned IOPS":       &AwsRdsIopsPrice{},

	"Virtual Machines Compute": &AzureVirtualMachinePrice{},
	"Storage Storage":          &AzureManagedStoragePrice{},
	"Load Balancer Networking": &AzureLoadBalancerPrice{},
}
