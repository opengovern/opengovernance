package inventory

import (
	"github.com/kaytu-io/kaytu-util/pkg/keibi-es-sdk"
	"github.com/kaytu-io/kaytu-util/pkg/source"
	"github.com/kaytu-io/kaytu-engine/pkg/inventory/es"
)

func ExtractTrend(client keibi.Client, provider source.Type, sourceID *string, fromTime, toTime int64) (map[int64]int, error) {
	datapoints := map[int64]int{}
	sortMap := []map[string]interface{}{
		{
			"described_at": "asc",
		},
	}
	if sourceID != nil {
		hits, err := es.FetchConnectionTrendSummaryPage(client, []string{*sourceID}, fromTime, toTime, sortMap, EsFetchPageSize)
		if err != nil {
			return nil, err
		}
		for _, hit := range hits {
			datapoints[hit.DescribedAt] += hit.ResourceCount
		}
	} else {
		hits, err := es.FetchProviderTrendSummaryPage(client, []source.Type{provider}, fromTime, toTime, sortMap, EsFetchPageSize)
		if err != nil {
			return nil, err
		}
		for _, hit := range hits {
			datapoints[hit.DescribedAt] += hit.ResourceCount
		}
	}
	return datapoints, nil
}
