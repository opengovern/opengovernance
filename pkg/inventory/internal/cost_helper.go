package internal

import (
	"fmt"

	summarizer "gitlab.com/keibiengine/keibi-engine/pkg/summarizer/es"
)

func AggregateServiceCosts(costs map[string][]summarizer.ServiceCostSummary) map[string]float64 {
	aggregatedCosts := make(map[string]float64)
	for service, cost := range costs {
		dateFlagMap := make(map[string]bool)
		for _, value := range cost {
			if _, ok := dateFlagMap[fmt.Sprintf("%d:--:%s", value.PeriodEnd, value.SourceID)]; ok {
				continue
			} else {
				dateFlagMap[fmt.Sprintf("%d:--:%s", value.PeriodEnd, value.SourceID)] = true
			}
			costValue, _ := value.GetCostAndUnit()
			if aggCosts, ok := aggregatedCosts[service]; !ok {
				aggregatedCosts[service] = costValue
				continue
			} else {
				aggregatedCosts[service] = aggCosts + costValue
			}
		}
	}
	return aggregatedCosts
}

func AggregateConnectionCosts(costs map[string][]summarizer.ConnectionCostSummary) map[string]float64 {
	aggregatedCosts := make(map[string]float64)
	for service, cost := range costs {
		dateFlagMap := make(map[string]bool)
		for _, value := range cost {
			if _, ok := dateFlagMap[fmt.Sprintf("%d:--:%s", value.PeriodEnd, value.SourceID)]; ok {
				continue
			} else {
				dateFlagMap[fmt.Sprintf("%d:--:%s", value.PeriodEnd, value.SourceID)] = true
			}
			costValue, _ := value.GetCostAndUnit()
			if aggCosts, ok := aggregatedCosts[service]; !ok {
				aggregatedCosts[service] = costValue
				continue
			} else {
				aggregatedCosts[service] = aggCosts + costValue
			}
		}
	}
	return aggregatedCosts
}
