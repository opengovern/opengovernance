package internal

import (
	"fmt"

	"gitlab.com/keibiengine/keibi-engine/pkg/inventory/api"
	summarizer "gitlab.com/keibiengine/keibi-engine/pkg/summarizer/es"
)

func AggregateServiceCosts(costs map[string][]summarizer.ServiceCostSummary) map[string]map[string]api.CostWithUnit {
	aggregatedCosts := make(map[string]map[string]api.CostWithUnit)
	for service, cost := range costs {
		dateFlagMap := make(map[string]bool)
		for _, value := range cost {
			if _, ok := dateFlagMap[fmt.Sprintf("%d:--:%s", value.PeriodEnd, value.SourceID)]; ok {
				continue
			} else {
				dateFlagMap[fmt.Sprintf("%d:--:%s", value.PeriodEnd, value.SourceID)] = true
			}
			costValue, costUnit := value.GetCostAndUnit()
			if aggCosts, ok := aggregatedCosts[service]; !ok {
				aggregatedCosts[service] = map[string]api.CostWithUnit{
					costUnit: {
						Cost: costValue,
						Unit: costUnit,
					},
				}
				continue
			} else {
				for i, aggCost := range aggCosts {
					if v, ok := aggCosts[aggCost.Unit]; ok {
						v.Cost += costValue
						aggCosts[i] = v
					} else {
						aggCosts[aggCost.Unit] = api.CostWithUnit{
							Cost: costValue,
							Unit: costUnit,
						}
					}
				}
			}

		}
	}
	return aggregatedCosts
}

func AggregateConnectionCosts(costs map[string][]summarizer.ConnectionCostSummary) map[string]map[string]api.CostWithUnit {
	aggregatedCosts := make(map[string]map[string]api.CostWithUnit)
	for service, cost := range costs {
		dateFlagMap := make(map[string]bool)
		for _, value := range cost {
			if _, ok := dateFlagMap[fmt.Sprintf("%d:--:%s", value.PeriodEnd, value.SourceID)]; ok {
				continue
			} else {
				dateFlagMap[fmt.Sprintf("%d:--:%s", value.PeriodEnd, value.SourceID)] = true
			}
			costValue, costUnit := value.GetCostAndUnit()
			if aggCosts, ok := aggregatedCosts[service]; !ok {
				aggregatedCosts[service] = map[string]api.CostWithUnit{
					costUnit: {
						Cost: costValue,
						Unit: costUnit,
					},
				}
				continue
			} else {
				for i, aggCost := range aggCosts {
					if v, ok := aggCosts[aggCost.Unit]; ok {
						v.Cost += costValue
						aggCosts[i] = v
					} else {
						aggCosts[aggCost.Unit] = api.CostWithUnit{
							Cost: costValue,
							Unit: costUnit,
						}
					}
				}
			}

		}
	}
	return aggregatedCosts
}

func MergeCostMaps(costs ...map[string]api.CostWithUnit) map[string]api.CostWithUnit {
	mergedCostsMap := make(map[string]api.CostWithUnit)
	for _, costMap := range costs {
		if costMap == nil {
			continue
		}
		for _, cost := range costMap {
			if v, ok := mergedCostsMap[cost.Unit]; !ok {
				mergedCostsMap[cost.Unit] = cost
			} else {
				v.Cost += cost.Cost
				mergedCostsMap[cost.Unit] = v
			}
		}
	}

	if len(mergedCostsMap) == 0 {
		return nil
	}

	return mergedCostsMap
}
