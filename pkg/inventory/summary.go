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

func GetCategories(client keibi.Client, provider source.Type, sourceID *string) ([]api.CategoriesResponse, error) {

	categoryMap := map[string]api.CategoriesResponse{}
	if sourceID == nil {
		hits, err := es.FetchProviderCategoriesSummaryPage(client, provider, nil, EsFetchPageSize)
		if err != nil {
			return nil, err
		}
		for _, hit := range hits {
			if v, ok := categoryMap[hit.CategoryName]; ok {
				v.ResourceCount += hit.ResourceCount
				categoryMap[hit.CategoryName] = v
			} else {
				categoryMap[hit.CategoryName] = api.CategoriesResponse{
					CategoryName:  hit.CategoryName,
					ResourceCount: hit.ResourceCount,
				}
			}
		}
	} else {
		hits, err := es.FetchConnectionCategoriesSummaryPage(client, sourceID, nil, EsFetchPageSize)
		if err != nil {
			return nil, err
		}
		for _, hit := range hits {
			if v, ok := categoryMap[hit.CategoryName]; ok {
				v.ResourceCount += hit.ResourceCount
				categoryMap[hit.CategoryName] = v
			} else {
				categoryMap[hit.CategoryName] = api.CategoriesResponse{
					CategoryName:  hit.CategoryName,
					ResourceCount: hit.ResourceCount,
				}
			}
		}
	}

	var res []api.CategoriesResponse
	for _, v := range categoryMap {
		res = append(res, v)
	}

	return res, nil
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

func GetCategoryNodeInfo(categoryNode *CategoryNode, metrics []Metric) api.CategoryNode {
	resourceCount := api.HistoryCount{}
	directFilters := map[string]api.Filter{}
	for _, f := range categoryNode.Filters {
		switch f.GetFilterType() {
		case FilterTypeCloudServiceCount:
			filter := f.(*FilterCloudServiceCountNode)
			directFilters[filter.ElementID] = api.FilterCloudResourceCount{
				FilterID:      filter.ElementID,
				SourceID:      filter.SourceID,
				CloudProvider: filter.CloudProvider,
				CloudService:  filter.CloudService,
				ResourceCount: api.HistoryCount{},
			}
		}
	}
	for _, f := range categoryNode.SubTreeFilters {
		switch f.GetFilterType() {
		case FilterTypeCloudServiceCount:
			filter := f.(*FilterCloudServiceCountNode)
			for _, m := range metrics {
				serviceName := cloudservice.ServiceNameByResourceType(m.ResourceType)
				if serviceName == filter.CloudService && (filter.SourceID == nil || *filter.SourceID == m.SourceID) {
					resourceCount.Count += m.Count
					resourceCount.LastDayValue = pointerAdd(resourceCount.LastDayValue, m.LastDayCount)
					resourceCount.LastWeekValue = pointerAdd(resourceCount.LastWeekValue, m.LastWeekCount)
					resourceCount.LastQuarterValue = pointerAdd(resourceCount.LastQuarterValue, m.LastQuarterCount)
					resourceCount.LastYearValue = pointerAdd(resourceCount.LastYearValue, m.LastYearCount)

					if directFilter, ok := directFilters[filter.ElementID].(api.FilterCloudResourceCount); ok {
						directFilter.ResourceCount.Count += m.Count
						directFilter.ResourceCount.LastDayValue = pointerAdd(directFilter.ResourceCount.LastDayValue, m.LastDayCount)
						directFilter.ResourceCount.LastWeekValue = pointerAdd(directFilter.ResourceCount.LastWeekValue, m.LastWeekCount)
						directFilter.ResourceCount.LastQuarterValue = pointerAdd(directFilter.ResourceCount.LastQuarterValue, m.LastQuarterCount)
						directFilter.ResourceCount.LastYearValue = pointerAdd(directFilter.ResourceCount.LastYearValue, m.LastYearCount)
						directFilters[filter.ElementID] = filter
					}
				}
			}
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

func RenderCategoryDFS(ctx context.Context, graphDb GraphDatabase, rootID string, metrics []Metric, depth int, cacheMap map[string]api.CategoryNode) (*api.CategoryNode, error) {
	if depth <= 0 {
		return nil, nil
	}
	categoryNode, err := graphDb.GetCategory(ctx, rootID)
	if err != nil {
		return nil, err
	}

	result := GetCategoryNodeInfo(categoryNode, metrics)
	for i, c := range result.Subcategories {
		if v, ok := cacheMap[c.CategoryID]; ok {
			result.Subcategories[i] = v
		} else {
			subResult, err := RenderCategoryDFS(ctx, graphDb, c.CategoryID, metrics, depth-1, cacheMap)
			if err != nil {
				return nil, err
			}
			if subResult != nil {
				cacheMap[c.CategoryID] = *subResult
				result.Subcategories[i] = *subResult
			}
		}
	}

	return &result, nil
}
