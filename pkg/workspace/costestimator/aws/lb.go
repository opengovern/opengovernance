package aws

import (
	"github.com/kaytu-io/kaytu-engine/pkg/workspace/api"
	"github.com/kaytu-io/kaytu-engine/pkg/workspace/costestimator/price"
	"github.com/kaytu-io/kaytu-engine/pkg/workspace/costestimator/product"
	"github.com/kaytu-io/kaytu-engine/pkg/workspace/costestimator/query"
	"github.com/kaytu-io/kaytu-engine/pkg/workspace/costestimator/util"
	"github.com/shopspring/decimal"

	"github.com/kaytu-io/kaytu-engine/pkg/workspace/costestimator/aws/region"
)

// LB represents a Load Balancer definition that can be cost-estimated.
type LB struct {
	provider *Provider
	region   region.Code

	// lbType describes the type of the Load Balancer.
	// Valid values: "application", "gateway", "network".
	// A special value of "classic" is allowed to represent a Classic Load Balancer.
	lbType string
}

// lbValues represents the structure of Terraform values for aws_lb/aws_alb resource.
type lbValues struct {
	Region           string
	LoadBalancerType string `mapstructure:"load_balancer_type"`
}

// decodeLBValues decodes and returns lbValues from a Terraform values map.
func decodeLBValues(request api.GetLBCostRequest) lbValues {
	return lbValues{
		Region:           request.RegionCode,
		LoadBalancerType: request.LBType,
	}
}

// newLB created a new LB from lbValues.
func (p *Provider) newLB(vals lbValues) *LB {
	return &LB{
		provider: p,
		region:   region.Code(vals.Region),
		lbType:   vals.LoadBalancerType,
	}
}

// Components returns the price component queries that make up this LB.
func (lb *LB) Components() []query.Component {
	return []query.Component{lb.loadBalancerComponent()}
}

func (lb *LB) loadBalancerComponent() query.Component {
	var family, name string
	switch lb.lbType {
	case "network":
		name = "Network Load Balancer"
		family = "Load Balancer-Network"
	case "gateway":
		name = "Gateway Load Balancer"
		family = "Load Balancer-Gateway"
	case "classic":
		name = "Classic Load Balancer"
		family = "Load Balancer"
	default:
		name = "Application Load Balancer"
		family = "Load Balancer-Application"
	}

	return query.Component{
		Name:           name,
		HourlyQuantity: decimal.NewFromInt(1),
		ProductFilter: &product.Filter{
			Provider: util.StringPtr(lb.provider.key),
			Service:  util.StringPtr("AWSELB"),
			Family:   util.StringPtr(family),
			Location: util.StringPtr(lb.region.String()),
			AttributeFilters: []*product.AttributeFilter{
				{Key: "UsageType", ValueRegex: util.StringPtr("LoadBalancerUsage")},
			},
		},
		PriceFilter: &price.Filter{
			Unit: util.StringPtr("Hrs"),
		},
	}
}
