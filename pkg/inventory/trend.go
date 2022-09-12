package inventory

import (
	"gitlab.com/keibiengine/keibi-engine/pkg/inventory/es"
	"gitlab.com/keibiengine/keibi-engine/pkg/keibi-es-sdk"
	"gitlab.com/keibiengine/keibi-engine/pkg/source"
)

func ExtractTrend(client keibi.Client, provider source.Type, sourceID *string, fromTime, toTime int64) (map[int64]int, error) {
	datapoints := map[int64]int{}
	sortMap := []map[string]interface{}{
		{
			"described_at": "asc",
		},
	}
	if sourceID != nil {
		hits, err := es.FetchConnectionTrendSummaryPage(client, sourceID, fromTime, toTime, sortMap, EsFetchPageSize)
		if err != nil {
			return nil, err
		}
		for _, hit := range hits {
			datapoints[hit.DescribedAt] += hit.ResourceCount
		}
	} else {
		hits, err := es.FetchProviderTrendSummaryPage(client, provider, fromTime, toTime, sortMap, EsFetchPageSize)
		if err != nil {
			return nil, err
		}
		for _, hit := range hits {
			datapoints[hit.DescribedAt] += hit.ResourceCount
		}
	}
	return datapoints, nil
}
