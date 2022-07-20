package inventory

import (
	"context"

	"gitlab.com/keibiengine/keibi-engine/pkg/describe"
	"gitlab.com/keibiengine/keibi-engine/pkg/inventory/api"
	"gitlab.com/keibiengine/keibi-engine/pkg/inventory/es"
	"gitlab.com/keibiengine/keibi-engine/pkg/keibi-es-sdk"
	"gitlab.com/keibiengine/keibi-engine/pkg/source"
)

func GetCategories(client keibi.Client, provider source.Type, sourceID *string) ([]api.CategoriesResponse, error) {
	var searchAfter []interface{}
	categoryMap := map[string]api.CategoriesResponse{}
	for {
		query, err := es.GetCategoriesQuery(string(provider), sourceID, EsFetchPageSize, searchAfter)
		if err != nil {
			return nil, err
		}

		var response es.CategoriesQueryResponse
		err = client.Search(context.Background(), describe.SourceResourcesSummary, query, &response)
		if err != nil {
			return nil, err
		}

		if len(response.Hits.Hits) == 0 {
			break
		}

		for _, hit := range response.Hits.Hits {
			if v, ok := categoryMap[hit.Source.CategoryName]; ok {
				v.ResourceCount += hit.Source.ResourceCount
			} else {
				categoryMap[hit.Source.CategoryName] = api.CategoriesResponse{
					CategoryName:     hit.Source.CategoryName,
					ResourceCount:    hit.Source.ResourceCount,
					LastDayCount:     hit.Source.LastDayCount,
					LastWeekCount:    hit.Source.LastWeekCount,
					LastQuarterCount: hit.Source.LastQuarterCount,
					LastYearCount:    hit.Source.LastYearCount,
				}
			}
			searchAfter = hit.Sort
		}
	}

	var res []api.CategoriesResponse
	for _, v := range categoryMap {
		res = append(res, v)
	}

	return res, nil
}

func GetServices(client keibi.Client, provider source.Type, sourceID *string) ([]api.TopServicesResponse, error) {
	var searchAfter []interface{}
	serviceResponse := map[string]api.TopServicesResponse{}
	for {
		query, err := es.FetchServicesQuery(string(provider), sourceID, EsFetchPageSize, searchAfter)
		if err != nil {
			return nil, err
		}

		var response es.FetchServicesQueryResponse
		err = client.Search(context.Background(), describe.SourceResourcesSummary, query, &response)
		if err != nil {
			return nil, err
		}

		if len(response.Hits.Hits) == 0 {
			break
		}

		for _, hit := range response.Hits.Hits {
			if v, ok := serviceResponse[hit.Source.ServiceName]; ok {
				v.ResourceCount += hit.Source.ResourceCount
			} else {
				serviceResponse[hit.Source.ServiceName] = api.TopServicesResponse{
					ServiceName:      hit.Source.ServiceName,
					ResourceCount:    hit.Source.ResourceCount,
					LastDayCount:     hit.Source.LastDayCount,
					LastWeekCount:    hit.Source.LastWeekCount,
					LastQuarterCount: hit.Source.LastQuarterCount,
					LastYearCount:    hit.Source.LastYearCount,
				}
			}
			searchAfter = hit.Sort
		}
	}

	var res []api.TopServicesResponse
	for _, v := range serviceResponse {
		res = append(res, v)
	}
	return res, nil
}
