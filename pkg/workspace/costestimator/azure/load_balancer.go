package azure

import (
	"github.com/kaytu-io/kaytu-engine/pkg/workspace/api"
	"github.com/kaytu-io/kaytu-engine/pkg/workspace/costestimator"
	"github.com/kaytu-io/kaytu-engine/pkg/workspace/db"
	"strings"
)

func LbCostByResource(db *db.Database, request api.GetAzureLoadBalancerRequest) (float64, error) {
	if string(*request.LoadBalancer.LoadBalancer.SKU.Name) == "Basic" {
		return 0, nil // Basic Load Balancer is free of charge
	} else if string(*request.LoadBalancer.LoadBalancer.SKU.Name) == "Gateway" {
		return getGatewayCost(db, request) // Not supported
	} else if string(*request.LoadBalancer.LoadBalancer.SKU.Name) == "Standard" {
		return getStandardLoadBalancerCost(db, request)
	}
	return 0, nil
}

// getGatewayCost Not supported
func getGatewayCost(db *db.Database, request api.GetAzureLoadBalancerRequest) (float64, error) {
	var cost float64
	// Not supported
	return cost, nil
}

func getStandardLoadBalancerCost(db *db.Database, request api.GetAzureLoadBalancerRequest) (float64, error) {
	var cost float64
	regionCode := convertRegion(request.RegionCode)
	rulesNumber := len(request.LoadBalancer.LoadBalancer.Properties.LoadBalancingRules) +
		len(request.LoadBalancer.LoadBalancer.Properties.OutboundRules)
	if string(*request.LoadBalancer.LoadBalancer.SKU.Tier) == "Regional" {
		includeRules, err := db.FindAzureLoadBalancerPrice(regionCode, "Standard Included LB Rules and Outbound Rules")
		if err != nil {
			return 0, err
		}
		cost += includeRules.Price
	} else if string(*request.LoadBalancer.LoadBalancer.SKU.Tier) == "Global" {
		includeRules, err := db.FindAzureLoadBalancerPrice(regionCode, "Global Included LB Rules and Outbound Rules")
		if err != nil {
			return 0, err
		}
		cost += includeRules.Price
	}
	if rulesNumber > 5 {
		overageRules := rulesNumber - 5
		if string(*request.LoadBalancer.LoadBalancer.SKU.Tier) == "Regional" {
			overagePrice, err := db.FindAzureLoadBalancerPrice(regionCode, "Standard Overage LB Rules and Outbound Rules")
			if err != nil {
				return 0, err
			}
			cost += overagePrice.Price * float64(overageRules)
		} else if string(*request.LoadBalancer.LoadBalancer.SKU.Tier) == "Global" {
			overagePrice, err := db.FindAzureLoadBalancerPrice(regionCode, "Global Overage LB Rules and Outbound Rules")
			if err != nil {
				return 0, err
			}
			cost += overagePrice.Price * float64(overageRules)
		}
	}

	// NAT rules are free.
	var dataProceeded int64 // GBs
	if request.DailyDataProceeded != nil {
		dataProceeded = *request.DailyDataProceeded
	} else {
		dataProceeded = 1000
	}

	if string(*request.LoadBalancer.LoadBalancer.SKU.Tier) == "Regional" {
		overagePrice, err := db.FindAzureLoadBalancerPrice(regionCode, "Standard Data Processed")
		if err != nil {
			return 0, err
		}
		cost += overagePrice.Price * float64(dataProceeded)
	} else if string(*request.LoadBalancer.LoadBalancer.SKU.Tier) == "Global" {
		overagePrice, err := db.FindAzureLoadBalancerPrice(regionCode, "Global Data Processed")
		if err != nil {
			return 0, err
		}
		cost += overagePrice.Price * float64(dataProceeded)
	}
	return cost * costestimator.TimeInterval, nil
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
