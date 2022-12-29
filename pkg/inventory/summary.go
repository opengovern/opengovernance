package inventory

import (
	"context"

	"gitlab.com/keibiengine/keibi-engine/pkg/cloudservice"
	"gitlab.com/keibiengine/keibi-engine/pkg/inventory/api"
	"gitlab.com/keibiengine/keibi-engine/pkg/inventory/es"
	"gitlab.com/keibiengine/keibi-engine/pkg/keibi-es-sdk"
	"gitlab.com/keibiengine/keibi-engine/pkg/source"
)

func pointerAdd(x, y *int) *int {
	var v *int
	if x != nil && y != nil {
		t := *x + *y
		v = &t
	} else if x != nil {
		v = x
	} else if y != nil {
		v = y
	}
	return v
}

func GetServices(client keibi.Client, provider source.Type, sourceID *string) ([]api.TopServicesResponse, error) {
	serviceResponse := map[string]api.TopServicesResponse{}
	if sourceID == nil {
		hits, err := es.FetchProviderServicesSummaryPage(client, provider, nil, EsFetchPageSize)
		if err != nil {
			return nil, err
		}
		for _, hit := range hits {
			if v, ok := serviceResponse[hit.ServiceName]; ok {
				v.ResourceCount += hit.ResourceCount
				serviceResponse[hit.ServiceName] = v
			} else {
				serviceResponse[hit.ServiceName] = api.TopServicesResponse{
					ServiceName:      hit.ServiceName,
					Provider:         string(hit.SourceType),
					ResourceCount:    hit.ResourceCount,
					LastDayCount:     hit.LastDayCount,
					LastWeekCount:    hit.LastWeekCount,
					LastQuarterCount: hit.LastQuarterCount,
					LastYearCount:    hit.LastYearCount,
				}
			}
		}
	} else {
		hits, err := es.FetchConnectionServicesSummaryPage(client, sourceID, nil, EsFetchPageSize)
		if err != nil {
			return nil, err
		}
		for _, hit := range hits {
			if v, ok := serviceResponse[hit.ServiceName]; ok {
				v.ResourceCount += hit.ResourceCount
				serviceResponse[hit.ServiceName] = v
			} else {
				serviceResponse[hit.ServiceName] = api.TopServicesResponse{
					ServiceName:      hit.ServiceName,
					Provider:         string(hit.SourceType),
					ResourceCount:    hit.ResourceCount,
					LastDayCount:     hit.LastDayCount,
					LastWeekCount:    hit.LastWeekCount,
					LastQuarterCount: hit.LastQuarterCount,
					LastYearCount:    hit.LastYearCount,
				}
			}
		}
	}

	var res []api.TopServicesResponse
	for _, v := range serviceResponse {
		res = append(res, v)
	}
	return res, nil
}

func GetResources(client keibi.Client, provider source.Type, sourceID *string, resourceTypes []string) ([]api.ResourceTypeResponse, error) {
	resourceTypeResponse := map[string]api.ResourceTypeResponse{}

	if sourceID == nil {
		hits, err := es.FetchProviderResourceTypeSummaryPage(client, provider, resourceTypes, nil, EsFetchPageSize)
		if err != nil {
			return nil, err
		}

		for _, hit := range hits {
			if v, ok := resourceTypeResponse[hit.ResourceType]; ok {
				v.ResourceCount += hit.ResourceCount
				resourceTypeResponse[hit.ResourceType] = v
			} else {
				resourceTypeResponse[hit.ResourceType] = api.ResourceTypeResponse{
					ResourceType:     cloudservice.ResourceTypeName(hit.ResourceType),
					ResourceCount:    hit.ResourceCount,
					LastDayCount:     hit.LastDayCount,
					LastWeekCount:    hit.LastWeekCount,
					LastQuarterCount: hit.LastQuarterCount,
					LastYearCount:    hit.LastYearCount,
				}
			}
		}
	} else {
		hits, err := es.FetchConnectionResourceTypeSummaryPage(client, sourceID, resourceTypes, nil, EsFetchPageSize)
		if err != nil {
			return nil, err
		}

		for _, hit := range hits {
			if v, ok := resourceTypeResponse[hit.ResourceType]; ok {
				v.ResourceCount += hit.ResourceCount
				resourceTypeResponse[hit.ResourceType] = v
			} else {
				resourceTypeResponse[hit.ResourceType] = api.ResourceTypeResponse{
					ResourceType:     cloudservice.ResourceTypeName(hit.ResourceType),
					ResourceCount:    hit.ResourceCount,
					LastDayCount:     hit.LastDayCount,
					LastWeekCount:    hit.LastWeekCount,
					LastQuarterCount: hit.LastQuarterCount,
					LastYearCount:    hit.LastYearCount,
				}
			}
		}
	}

	var res []api.ResourceTypeResponse
	for _, v := range resourceTypeResponse {
		if v.ResourceCount == 0 {
			continue
		}

		res = append(res, v)
	}
	return res, nil
}

func GetResourcesFromPostgres(db Database, provider source.Type, sourceID *string, resourceTypes []string) ([]api.ResourceTypeResponse, error) {
	var m []Metric
	var err error

	if sourceID == nil {
		if provider.IsNull() {
			m, err = db.FetchMetrics(resourceTypes)
			if err != nil {
				return nil, err
			}
		} else {
			m, err = db.FetchProviderMetrics(provider, resourceTypes)
			if err != nil {
				return nil, err
			}
		}
	} else {
		m, err = db.FetchConnectionMetrics(*sourceID, resourceTypes)
		if err != nil {
			return nil, err
		}
	}

	resourceTypeResponse := map[string]api.ResourceTypeResponse{}
	for _, hit := range m {
		if v, ok := resourceTypeResponse[hit.ResourceType]; ok {
			v.ResourceCount += hit.Count
			v.LastDayCount = pointerAdd(v.LastDayCount, hit.LastDayCount)
			v.LastWeekCount = pointerAdd(v.LastWeekCount, hit.LastWeekCount)
			v.LastQuarterCount = pointerAdd(v.LastQuarterCount, hit.LastQuarterCount)
			v.LastYearCount = pointerAdd(v.LastYearCount, hit.LastYearCount)
			resourceTypeResponse[hit.ResourceType] = v
		} else {
			resourceTypeResponse[hit.ResourceType] = api.ResourceTypeResponse{
				ResourceType:     hit.ResourceType,
				ResourceTypeName: cloudservice.ResourceTypeName(hit.ResourceType),
				ResourceCount:    hit.Count,
				LastDayCount:     hit.LastDayCount,
				LastWeekCount:    hit.LastWeekCount,
				LastQuarterCount: hit.LastQuarterCount,
				LastYearCount:    hit.LastYearCount,
			}
		}
	}

	var res []api.ResourceTypeResponse
	for _, v := range resourceTypeResponse {
		if v.ResourceCount == 0 {
			continue
		}

		res = append(res, v)
	}
	return res, nil
}

func GetResourceTypeListFromFilters(filters []Filter) []string {
	result := map[string]struct{}{}
	for _, filter := range filters {
		switch filter.GetFilterType() {
		case FilterTypeCloudResourceType:
			f := filter.(*FilterCloudResourceTypeNode)
			result[f.ResourceType] = struct{}{}
		default:
			continue
		}
	}
	res := make([]string, 0, len(result))
	for k := range result {
		res = append(res, k)
	}
	return res
}

func GetServiceNameListFromFilters(filters []Filter) []string {
	result := map[string]struct{}{}
	for _, filter := range filters {
		switch filter.GetFilterType() {
		case FilterTypeCost:
			f := filter.(*FilterCostNode)
			result[f.ServiceName] = struct{}{}
		default:
			continue
		}
	}
	res := make([]string, 0, len(result))
	for k := range result {
		res = append(res, k)
	}
	return res
}

func GetMetricResourceTypeSummaryIndexByResourceType(metrics []MetricResourceTypeSummary) map[string]MetricResourceTypeSummary {
	metricIndex := map[string]MetricResourceTypeSummary{}
	for _, metric := range metrics {
		metricIndex[metric.ResourceType] = metric
	}
	return metricIndex
}

func GetCategoryNodeResourceCountInfo(categoryNode *CategoryNode, metrics map[string]int, filterCacheMap map[string]api.Filter) api.CategoryNode {
	resourceCount := 0
	directFilters := map[string]api.Filter{}
	for _, f := range categoryNode.Filters {
		switch f.GetFilterType() {
		case FilterTypeCloudResourceType:
			filter := f.(*FilterCloudResourceTypeNode)
			if v, ok := filterCacheMap[filter.ElementID]; ok {
				directFilters[filter.ElementID] = *v.(*api.FilterCloudResourceType)
			} else {
				directFilters[filter.ElementID] = api.FilterCloudResourceType{
					FilterID:      filter.ElementID,
					CloudProvider: filter.CloudProvider,
					ResourceName:  filter.ResourceName,
					ResourceType:  filter.ResourceType,
					ResourceCount: 0,
				}
			}
		default:
			continue
		}
	}
	for _, f := range categoryNode.SubTreeFilters {
		switch f.GetFilterType() {
		case FilterTypeCloudResourceType:
			filter := f.(*FilterCloudResourceTypeNode)
			if v, ok := filterCacheMap[filter.ElementID]; ok {
				m := v.(*api.FilterCloudResourceType)
				resourceCount += m.ResourceCount
			} else {
				filterWithCount := api.FilterCloudResourceType{
					FilterID:      filter.ElementID,
					CloudProvider: filter.CloudProvider,
					ResourceType:  filter.ResourceType,
					ResourceName:  filter.ResourceName,
					ResourceCount: 0,
				}
				if m, ok := metrics[filter.ResourceType]; ok {
					resourceCount += m
					filterWithCount.ResourceCount += m
					if _, ok := directFilters[filter.ElementID].(api.FilterCloudResourceType); ok {
						directFilters[filter.ElementID] = filterWithCount
					}
					filterCacheMap[filter.ElementID] = &filterWithCount
				}
			}
		default:
			continue
		}
	}
	result := api.CategoryNode{
		CategoryID:    categoryNode.ElementID,
		CategoryName:  categoryNode.Name,
		ResourceCount: &resourceCount,
		Subcategories: []api.CategoryNode{},
		Filters:       []api.Filter{},
	}
	for _, c := range categoryNode.Subcategories {
		result.Subcategories = append(result.Subcategories, api.CategoryNode{
			CategoryID:   c.ElementID,
			CategoryName: c.Name,
		})
	}
	for _, f := range directFilters {
		result.Filters = append(result.Filters, f)
	}

	return result
}

func GetCategoryNodeCostInfo(categoryNode *CategoryNode, costs map[string]map[string]api.CostWithUnit, filterCacheMap map[string]api.Filter) api.CategoryNode {
	apiCosts := make(map[string]api.CostWithUnit)
	directFilters := map[string]api.Filter{}
	for _, f := range categoryNode.Filters {
		switch f.GetFilterType() {
		case FilterTypeCost:
			filter := f.(*FilterCostNode)
			if v, ok := filterCacheMap[filter.ElementID]; ok {
				directFilters[filter.ElementID] = *v.(*api.FilterCost)
			} else {
				directFilters[filter.ElementID] = api.FilterCost{
					FilterID:      filter.ElementID,
					CloudProvider: filter.CloudProvider,
					Cost:          map[string]api.CostWithUnit{},
				}
			}
		default:
			continue
		}
	}

	for _, f := range categoryNode.SubTreeFilters {
		switch f.GetFilterType() {
		case FilterTypeCost:
			filter := f.(*FilterCostNode)
			if v, ok := filterCacheMap[filter.ElementID]; ok {
				filterCost := v.(*api.FilterCost)
				for key, costValue := range filterCost.Cost {
					if currentCostValue, ok := apiCosts[key]; ok {
						currentCostValue.Cost += costValue.Cost
						apiCosts[key] = currentCostValue
					} else {
						apiCosts[key] = api.CostWithUnit{
							Cost: costValue.Cost,
							Unit: costValue.Unit,
						}
					}
				}
				if _, ok := directFilters[filter.ElementID].(api.FilterCost); ok {
					directFilters[filter.ElementID] = *v.(*api.FilterCost)
				}
			} else {
				filterWithCost := api.FilterCost{
					FilterID:      filter.ElementID,
					CloudProvider: filter.CloudProvider,
					Cost:          map[string]api.CostWithUnit{},
				}
				if m, ok := costs[filter.ServiceName]; ok {
					for costUnit, costValue := range m {
						if currentCostValue, ok := apiCosts[costUnit]; ok {
							currentCostValue.Cost += costValue.Cost
							apiCosts[costUnit] = currentCostValue
						} else {
							apiCosts[costUnit] = costValue
						}
					}
				}

				for unit, costValue := range apiCosts {
					if v, ok := filterWithCost.Cost[unit]; ok {
						v.Cost += costValue.Cost
						filterWithCost.Cost[unit] = v
					} else {
						filterWithCost.Cost[unit] = api.CostWithUnit{
							Cost: costValue.Cost,
							Unit: unit,
						}
					}
				}

				if _, ok := directFilters[filter.ElementID].(api.FilterCost); ok {
					directFilters[filter.ElementID] = filterWithCost
				}
				filterCacheMap[filter.ElementID] = &filterWithCost
			}
		default:
			continue
		}
	}

	result := api.CategoryNode{
		CategoryID:    categoryNode.ElementID,
		CategoryName:  categoryNode.Name,
		Cost:          apiCosts,
		Subcategories: []api.CategoryNode{},
		Filters:       []api.Filter{},
	}
	for _, c := range categoryNode.Subcategories {
		result.Subcategories = append(result.Subcategories, api.CategoryNode{
			CategoryID:   c.ElementID,
			CategoryName: c.Name,
		})
	}
	for _, f := range directFilters {
		result.Filters = append(result.Filters, f)
	}
	return result
}

func RenderCategoryResourceCountDFS(ctx context.Context,
	graphDb GraphDatabase,
	rootNode *CategoryNode,
	metrics map[string]int,
	depth int,
	importanceArray []string,
	nodeCacheMap map[string]api.CategoryNode,
	filterCacheMap map[string]api.Filter) (*api.CategoryNode, error) {
	if depth <= 0 {
		return nil, nil
	}

	result := GetCategoryNodeResourceCountInfo(rootNode, metrics, filterCacheMap)
	for i, c := range result.Subcategories {
		if v, ok := nodeCacheMap[c.CategoryID]; ok {
			result.Subcategories[i] = v
		} else {
			subCategoryNode, err := graphDb.GetCategory(ctx, c.CategoryID, importanceArray)
			if err != nil {
				return nil, err
			}

			subResult, err := RenderCategoryResourceCountDFS(ctx, graphDb, subCategoryNode, metrics, depth-1, importanceArray, nodeCacheMap, filterCacheMap)
			if err != nil {
				return nil, err
			}
			if subResult != nil {
				nodeCacheMap[c.CategoryID] = *subResult
				result.Subcategories[i] = *subResult
			}
		}
	}

	return &result, nil
}

func RenderCategoryCostDFS(ctx context.Context,
	graphDb GraphDatabase,
	rootNode *CategoryNode,
	depth int,
	costs map[string]map[string]api.CostWithUnit,
	nodeCacheMap map[string]api.CategoryNode,
	filterCacheMap map[string]api.Filter) (*api.CategoryNode, error) {

	if depth <= 0 {
		return nil, nil
	}

	result := GetCategoryNodeCostInfo(rootNode, costs, filterCacheMap)
	for i, c := range result.Subcategories {
		if v, ok := nodeCacheMap[c.CategoryID]; ok {
			result.Subcategories[i] = v
		} else {
			subCategoryNode, err := graphDb.GetCategory(ctx, c.CategoryID, []string{"all"})
			if err != nil {
				return nil, err
			}

			subResult, err := RenderCategoryCostDFS(ctx, graphDb, subCategoryNode, depth, costs, nodeCacheMap, filterCacheMap)
			if err != nil {
				return nil, err
			}
			if subResult != nil {
				nodeCacheMap[c.CategoryID] = *subResult
				result.Subcategories[i] = *subResult
			}
		}
	}

	return &result, nil
}
