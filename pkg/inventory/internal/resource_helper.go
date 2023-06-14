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
		if v, ok := compareTo[filterID]; ok {
			switch filterVal.GetFilterType() {
			case api.FilterTypeCloudResourceType:
				fv := filterVal.(*api.FilterCloudResourceType)
				vv := v.(*api.FilterCloudResourceType)
				if vv.ResourceCount != 0 {
					change := calculatePercentageGrowth(fv.ResourceCount, vv.ResourceCount)
					fv.ResourceCountChange = &change
					source[filterID] = filterVal
				}
			case api.FilterTypeInsightMetric:
				fv := filterVal.(*api.FilterInsightMetric)
				vv := v.(*api.FilterInsightMetric)
				if vv.Value != 0 {
					change := calculatePercentageGrowth(fv.Value, vv.Value)
					fv.ValueChange = &change
					source[filterID] = filterVal
				}
			}
		}
	}
	return source
}
