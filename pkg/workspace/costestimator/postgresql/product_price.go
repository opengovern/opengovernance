package postgresql

import (
	"fmt"
	"github.com/kaytu-io/open-governance/pkg/workspace/costestimator/price"
	"github.com/kaytu-io/open-governance/pkg/workspace/costestimator/product"
	"github.com/kaytu-io/open-governance/pkg/workspace/db"
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
	"AmazonEC2 Compute Instance":       "aws_ec2instance_prices",
	"AmazonEC2 Storage":                "aws_ec2instancestorage_prices",
	"AmazonEC2 System Operation":       "aws_ec2instancesystemoperation_prices",
	"AmazonCloudWatch Metric":          "aws_cloudwatch_prices",
	"AmazonEC2 CPU Credits":            "aws_ec2cpucredits_prices",
	"AWSELB Load Balancer-Network":     "aws_elasticloadbalancing_prices",
	"AWSELB Load Balancer-Gateway":     "aws_elasticloadbalancing_prices",
	"AWSELB Load Balancer":             "aws_elasticloadbalancing_prices",
	"AWSELB Load Balancer-Application": "aws_elasticloadbalancing_prices",
	"AmazonRDS Database Instance":      "aws_rdsinstance_prices",
	"AmazonRDS Database Storage":       "aws_rdsstorage_prices",
	"AmazonRDS Provisioned IOPS":       "aws_rdsiops_prices",

	"Virtual Machines Compute":   "azure_virtualmachine_prices",
	"Storage Storage":            "azure_managedstorage_prices",
	"Load Balancer Networking":   "azure_loadbalancer_prices",
	"Virtual Network Networking": "azure_virtualnetwork_prices",
	"VPN Gateway Networking":     "azure_vpngateway_prices",
}
