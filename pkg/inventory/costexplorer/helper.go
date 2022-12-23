package costexplorer

import (
	"gitlab.com/keibiengine/keibi-engine/pkg/inventory/api"
	summarizer "gitlab.com/keibiengine/keibi-engine/pkg/summarizer/es"
)

func AggregateServiceCosts(costs map[string][]summarizer.ServiceCostSummary) map[string][]api.CostWithUnit {
	aggregatedCosts := make(map[string][]api.CostWithUnit)
	for service, cost := range costs {
		dateFlagMap := make(map[int64]bool)
		for _, value := range cost {
			if _, ok := dateFlagMap[value.PeriodEnd]; ok {
				continue
			} else {
				dateFlagMap[value.PeriodEnd] = true
			}
			costValue, costUnit := value.GetCostAndUnit()
			if aggCosts, ok := aggregatedCosts[service]; !ok {
				aggregatedCosts[service] = []api.CostWithUnit{{
					Cost: costValue,
					Unit: costUnit,
				}}
				continue
			} else {
				doesUnitExist := false
				for i, aggCost := range aggCosts {
					if aggCost.Unit == costUnit {
						aggCosts[i].Cost += costValue
						doesUnitExist = true
					}
				}
				if !doesUnitExist {
					aggCosts = append(aggCosts, api.CostWithUnit{
						Cost: costValue,
						Unit: costUnit,
					})
				}
			}

		}
	}
	return aggregatedCosts
}

func MergeCostArrays(costs ...[]api.CostWithUnit) []api.CostWithUnit {
	mergedCostsMap := make(map[string]api.CostWithUnit)
	for _, costArr := range costs {
		if costArr == nil {
			continue
		}
		for _, cost := range costArr {
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

	mergedCosts := make([]api.CostWithUnit, 0, len(mergedCostsMap))
	for _, v := range mergedCostsMap {
		mergedCosts = append(mergedCosts, v)
	}

	return mergedCosts
}
