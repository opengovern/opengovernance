package internal

import "gitlab.com/keibiengine/keibi-engine/pkg/inventory/api"

func calculatePercentageGrowth[T int | float64](current, previous T) float64 {
	if previous == 0 {
		return 0
	}
	return float64(current-previous) / float64(previous) * 100
}

func CalculateResourceTypeCountPercentChanges(root *api.CategoryNode, compareTo map[string]api.CategoryNode) *api.CategoryNode {
	if compareTo == nil || root.ResourceCount == nil {
		return root
	}
	if v, ok := compareTo[root.CategoryID]; ok {
		if v.ResourceCount != nil && *v.ResourceCount != 0 {
			change := calculatePercentageGrowth(*root.ResourceCount, *v.ResourceCount)
			root.ResourceCountChange = &change
		}
	}
	for i := range root.Subcategories {
		subcat := CalculateResourceTypeCountPercentChanges(&root.Subcategories[i], compareTo)
		root.Subcategories[i] = *subcat
	}
	return root
}

func CalculateMetricResourceTypeCountPercentChanges(source map[string]api.Filter, compareTo map[string]api.Filter) map[string]api.Filter {
	if compareTo == nil || source == nil {
		return source
	}
	for filterID, filterVal := range source {
		if v, ok := compareTo[filterID]; !ok {
			switch filterVal.GetFilterType() {
			case api.FilterTypeCloudResourceType:
				fv := filterVal.(*api.FilterCloudResourceType)
				vv := v.(*api.FilterCloudResourceType)
				if vv.ResourceCount != 0 {
					change := calculatePercentageGrowth(fv.ResourceCount, vv.ResourceCount)
					fv.ResourceCountChange = &change
					source[filterID] = filterVal
				}
			}
		}
	}
	return source
}

func CalculateCostPercentChanges(root *api.CategoryNode, compareTo map[string]api.CategoryNode) *api.CategoryNode {
	if compareTo == nil || root.Cost == nil {
		return root
	}
	if v, ok := compareTo[root.CategoryID]; ok {
		if v.Cost != nil {
			for currency, cost := range v.Cost {
				if rootCost, ok := root.Cost[currency]; ok {
					change := calculatePercentageGrowth(rootCost.Cost, cost.Cost)
					if root.CostChange == nil {
						root.CostChange = make(map[string]float64)
					}
					root.CostChange[currency] = change
				}
			}
		}
	}
	for i := range root.Subcategories {
		subcat := CalculateResourceTypeCountPercentChanges(&root.Subcategories[i], compareTo)
		root.Subcategories[i] = *subcat
	}
	return root
}

func CalculateMetricCostPercentChanges(source map[string]api.Filter, compareTo map[string]api.Filter) map[string]api.Filter {
	if compareTo == nil || source == nil {
		return source
	}
	for filterID, filterVal := range source {
		if v, ok := compareTo[filterID]; !ok {
			switch filterVal.GetFilterType() {
			case api.FilterTypeCost:
				fv := filterVal.(*api.FilterCost)
				vv := v.(*api.FilterCost)
				for currency, cost := range fv.Cost {
					if vvCost, ok := vv.Cost[currency]; ok {
						change := calculatePercentageGrowth(cost.Cost, vvCost.Cost)
						if fv.CostChange == nil {
							fv.CostChange = make(map[string]float64)
						}
						fv.CostChange[currency] = change
					}
				}
			}
		}
	}
	return source
}
