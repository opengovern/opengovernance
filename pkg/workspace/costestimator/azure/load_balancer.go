package azure

import (
	"github.com/kaytu-io/open-governance/pkg/workspace/api"
	"github.com/kaytu-io/open-governance/pkg/workspace/costestimator/product"
	"github.com/kaytu-io/open-governance/pkg/workspace/costestimator/query"
	"github.com/kaytu-io/open-governance/pkg/workspace/costestimator/util"
	"github.com/shopspring/decimal"
	"strings"
)

// LoadBalancer is the entity that holds the logic to calculate price
// of the azurerm_load_balancer
type LoadBalancer struct {
	provider *Provider

	location         string
	skuName          string
	rulesNumber      int32
	skuTier          string
	dailyDataProceed int64
}

// loadBalancerValues is holds the values that we need to be able
// to calculate the price of the ComputeInstance
type loadBalancerValues struct {
	SkuName          string `mapstructure:"sku_name"`
	Location         string `mapstructure:"location"`
	RulesNumber      int32  `mapstructure:"rules_number"`
	SkuTier          string `mapstructure:"sku_tier"`
	DailyDataProceed int64  `mapstructure:"daily_data_proceed"`
}

// decodeLoadBalancerValues decodes and returns computeInstanceValues from a Terraform values map.
func decodeLoadBalancerValues(request api.GetAzureLoadBalancerRequest) loadBalancerValues {
	regionCode := convertRegion(request.RegionCode)
	dailyDataProceed := int64(1000)
	if request.DailyDataProceed != nil {
		dailyDataProceed = *request.DailyDataProceed
	}
	return loadBalancerValues{
		SkuName:          request.SkuName,
		Location:         regionCode,
		RulesNumber:      request.RulesNumber,
		SkuTier:          request.SkuTier,
		DailyDataProceed: dailyDataProceed,
	}
}

// newManagedStorage initializes a new LoadBalancer from the provider
func (p *Provider) newLoadBalancer(vals loadBalancerValues) *LoadBalancer {
	inst := &LoadBalancer{
		provider: p,

		location:         vals.Location,
		skuName:          vals.SkuName,
		rulesNumber:      vals.RulesNumber,
		skuTier:          vals.SkuTier,
		dailyDataProceed: vals.DailyDataProceed,
	}

	return inst
}

func (inst *LoadBalancer) Components() []query.Component {
	var components []query.Component

	if inst.skuTier == "Regional" {
		components = append(components, inst.regionalIncludedRulesComponent(inst.provider.key, inst.location))
		// NAT rules are free.

		if inst.rulesNumber > 5 {
			components = append(components, inst.regionalOverageRulesComponent(inst.provider.key, inst.location, int(inst.rulesNumber)-5))
		}

		components = append(components, inst.regionalDataProceedComponent(inst.provider.key, inst.location, inst.dailyDataProceed))
	} else if inst.skuTier == "Global" {
		components = append(components, inst.globalIncludedRulesComponent(inst.provider.key, inst.location))
		// NAT rules are free.

		if inst.rulesNumber > 5 {
			components = append(components, inst.globalOverageRulesComponent(inst.provider.key, inst.location, int(inst.rulesNumber)-5))
		}

		components = append(components, inst.globalDataProceedComponent(inst.provider.key, inst.location, inst.dailyDataProceed))
	}

	return components
}

func (inst *LoadBalancer) regionalIncludedRulesComponent(key, location string) query.Component {
	return query.Component{
		Name:           "Regional Included Rules",
		HourlyQuantity: decimal.NewFromInt(1),
		ProductFilter: &product.Filter{
			Provider: util.StringPtr(key),
			Service:  util.StringPtr("Load Balancer"),
			Family:   util.StringPtr("Networking"),
			Location: util.StringPtr(location),
			AttributeFilters: []*product.AttributeFilter{
				{Key: "meter_name", Value: util.StringPtr("Standard Included LB Rules and Outbound Rules")},
			},
		},
	}
}

func (inst *LoadBalancer) globalIncludedRulesComponent(key, location string) query.Component {
	return query.Component{
		Name:           "Global Included Rules",
		HourlyQuantity: decimal.NewFromInt(1),
		ProductFilter: &product.Filter{
			Provider: util.StringPtr(key),
			Service:  util.StringPtr("Load Balancer"),
			Family:   util.StringPtr("Networking"),
			Location: util.StringPtr(location),
			AttributeFilters: []*product.AttributeFilter{
				{Key: "meter_name", Value: util.StringPtr("Global Included LB Rules and Outbound Rules")},
			},
		},
	}
}

func (inst *LoadBalancer) regionalOverageRulesComponent(key, location string, overageRules int) query.Component {
	return query.Component{
		Name:           "Regional Overage Rules",
		HourlyQuantity: decimal.NewFromInt(int64(overageRules)),
		ProductFilter: &product.Filter{
			Provider: util.StringPtr(key),
			Service:  util.StringPtr("Load Balancer"),
			Family:   util.StringPtr("Networking"),
			Location: util.StringPtr(location),
			AttributeFilters: []*product.AttributeFilter{
				{Key: "meter_name", Value: util.StringPtr("Standard Overage LB Rules and Outbound Rules")},
			},
		},
	}
}

func (inst *LoadBalancer) globalOverageRulesComponent(key, location string, overageRules int) query.Component {
	return query.Component{
		Name:           "Global Overage Rules",
		HourlyQuantity: decimal.NewFromInt(int64(overageRules)),
		ProductFilter: &product.Filter{
			Provider: util.StringPtr(key),
			Service:  util.StringPtr("Load Balancer"),
			Family:   util.StringPtr("Networking"),
			Location: util.StringPtr(location),
			AttributeFilters: []*product.AttributeFilter{
				{Key: "meter_name", Value: util.StringPtr("Global Overage LB Rules and Outbound Rules")},
			},
		},
	}
}

func (inst *LoadBalancer) regionalDataProceedComponent(key, location string, dataProceed int64) query.Component {
	return query.Component{
		Name:           "Regional Data Proceed",
		HourlyQuantity: decimal.NewFromInt(dataProceed),
		ProductFilter: &product.Filter{
			Provider: util.StringPtr(key),
			Service:  util.StringPtr("Load Balancer"),
			Family:   util.StringPtr("Networking"),
			Location: util.StringPtr(location),
			AttributeFilters: []*product.AttributeFilter{
				{Key: "meter_name", Value: util.StringPtr("Standard Data Processed")},
			},
		},
	}
}

func (inst *LoadBalancer) globalDataProceedComponent(key, location string, dataProceed int64) query.Component {
	return query.Component{
		Name:           "Global Data Proceed",
		HourlyQuantity: decimal.NewFromInt(dataProceed),
		ProductFilter: &product.Filter{
			Provider: util.StringPtr(key),
			Service:  util.StringPtr("Load Balancer"),
			Family:   util.StringPtr("Networking"),
			Location: util.StringPtr(location),
			AttributeFilters: []*product.AttributeFilter{
				{Key: "meter_name", Value: util.StringPtr("Global Data Processed")},
			},
		},
	}
}

func convertRegion(region string) string {
	if strings.Contains(strings.ToLower(region), "usgov") {
		return "US Gov"
	} else if strings.Contains(strings.ToLower(region), "china") {
		return "Сhina"
	} else {
		return "Global"
	}
}
