package describe

import (
	"context"
	"fmt"
	"strings"

	"gitlab.com/keibiengine/keibi-engine/pkg/cloudservice"
	"gitlab.com/keibiengine/keibi-engine/pkg/describe/kafka"
	"gitlab.com/keibiengine/keibi-engine/pkg/keibi-es-sdk"
)

func ExtractServiceSummary(es keibi.Client, job DescribeJob, lookupResources []kafka.LookupResource) ([]kafka.DescribedResource, error) {
	var msgs []kafka.DescribedResource
	serviceCount := map[string]int{}
	for _, resource := range lookupResources {
		if s := cloudservice.ServiceNameByResourceType(resource.ResourceType); s != "" {
			serviceCount[s]++
		}
	}

	for name, count := range serviceCount {
		var lastDayValue, lastWeekValue, lastQuarterValue, lastYearValue *int
		for idx, jobID := range []uint{job.LastDaySourceJobID, job.LastWeekSourceJobID, job.LastQuarterSourceJobID,
			job.LastYearSourceJobID} {
			var searchAfter []interface{}
			for {
				var response ServiceQueryResponse
				query, err := FindOldServiceValue(jobID, name, EsFetchPageSize, searchAfter)
				if err != nil {
					return nil, fmt.Errorf("failed to build query for service: %v", err.Error())
				}

				err = es.Search(context.Background(), kafka.SourceResourcesSummaryIndex, query, &response)
				if err != nil {
					return nil, fmt.Errorf("failed to run query for service: %v", err.Error())
				}

				if len(response.Hits.Hits) == 0 {
					break
				}

				for _, hit := range response.Hits.Hits {
					switch idx {
					case 0:
						if lastDayValue != nil {
							hit.Source.ResourceCount += *lastDayValue
						}
						lastDayValue = &hit.Source.ResourceCount
					case 1:
						if lastWeekValue != nil {
							hit.Source.ResourceCount += *lastWeekValue
						}
						lastWeekValue = &hit.Source.ResourceCount
					case 2:
						if lastQuarterValue != nil {
							hit.Source.ResourceCount += *lastQuarterValue
						}
						lastQuarterValue = &hit.Source.ResourceCount
					case 3:
						if lastYearValue != nil {
							hit.Source.ResourceCount += *lastYearValue
						}
						lastYearValue = &hit.Source.ResourceCount
					}
					searchAfter = hit.Sort
				}
			}
		}

		msgs = append(msgs, kafka.SourceServicesSummary{
			ServiceName:      name,
			SourceID:         job.SourceID,
			ResourceType:     job.ResourceType,
			SourceType:       job.SourceType,
			SourceJobID:      job.ParentJobID,
			DescribedAt:      job.DescribedAt,
			ResourceCount:    count,
			LastDayCount:     lastDayValue,
			LastWeekCount:    lastWeekValue,
			LastQuarterCount: lastQuarterValue,
			LastYearCount:    lastYearValue,
			ReportType:       kafka.ResourceSummaryTypeLastServiceSummary,
		})

		msgs = append(msgs, kafka.SourceServicesSummary{
			ServiceName:      name,
			SourceID:         job.SourceID,
			ResourceType:     job.ResourceType,
			SourceType:       job.SourceType,
			SourceJobID:      job.ParentJobID,
			DescribedAt:      job.DescribedAt,
			ResourceCount:    count,
			LastDayCount:     lastDayValue,
			LastWeekCount:    lastWeekValue,
			LastQuarterCount: lastQuarterValue,
			LastYearCount:    lastYearValue,
			ReportType:       kafka.ResourceSummaryTypeServiceHistorySummary,
		})
	}

	return msgs, nil
}

func ExtractCategorySummary(es keibi.Client, job DescribeJob, lookupResources []kafka.LookupResource) ([]kafka.DescribedResource, error) {
	var msgs []kafka.DescribedResource
	categoryCount := map[string]int{}
	for _, resource := range lookupResources {
		if s := cloudservice.CategoryByResourceType(resource.ResourceType); s != "" {
			if cloudservice.IsCommonByResourceType(resource.ResourceType) {
				categoryCount[s]++
			}
		}
	}

	for name, count := range categoryCount {
		var lastDayValue, lastWeekValue, lastQuarterValue, lastYearValue *int
		for idx, jobID := range []uint{job.LastDaySourceJobID, job.LastWeekSourceJobID, job.LastQuarterSourceJobID,
			job.LastYearSourceJobID} {
			var searchAfter []interface{}
			for {
				var response CategoryQueryResponse
				query, err := FindOldCategoryValue(jobID, name, EsFetchPageSize, searchAfter)
				if err != nil {
					return nil, fmt.Errorf("failed to build query for category: %v", err.Error())
				}

				err = es.Search(context.Background(), kafka.SourceResourcesSummaryIndex, query, &response)
				if err != nil {
					return nil, fmt.Errorf("failed to run query for category: %v", err.Error())
				}

				if len(response.Hits.Hits) == 0 {
					break
				}

				for _, hit := range response.Hits.Hits {
					switch idx {
					case 0:
						if lastDayValue != nil {
							hit.Source.ResourceCount += *lastDayValue
						}
						lastDayValue = &hit.Source.ResourceCount
					case 1:
						if lastWeekValue != nil {
							hit.Source.ResourceCount += *lastWeekValue
						}
						lastWeekValue = &hit.Source.ResourceCount
					case 2:
						if lastQuarterValue != nil {
							hit.Source.ResourceCount += *lastQuarterValue
						}
						lastQuarterValue = &hit.Source.ResourceCount
					case 3:
						if lastYearValue != nil {
							hit.Source.ResourceCount += *lastYearValue
						}
						lastYearValue = &hit.Source.ResourceCount
					}
					searchAfter = hit.Sort
				}
			}
		}

		msgs = append(msgs, kafka.SourceCategorySummary{
			CategoryName:     name,
			SourceType:       job.SourceType,
			SourceJobID:      job.ParentJobID,
			SourceID:         job.SourceID,
			DescribedAt:      job.DescribedAt,
			ResourceType:     job.ResourceType,
			ResourceCount:    count,
			LastDayCount:     lastDayValue,
			LastWeekCount:    lastWeekValue,
			LastQuarterCount: lastQuarterValue,
			LastYearCount:    lastYearValue,
			ReportType:       kafka.ResourceSummaryTypeLastCategorySummary,
		})

		msgs = append(msgs, kafka.SourceCategorySummary{
			CategoryName:     name,
			SourceType:       job.SourceType,
			SourceID:         job.SourceID,
			SourceJobID:      job.ParentJobID,
			DescribedAt:      job.DescribedAt,
			ResourceType:     job.ResourceType,
			ResourceCount:    count,
			LastDayCount:     lastDayValue,
			LastWeekCount:    lastWeekValue,
			LastQuarterCount: lastQuarterValue,
			LastYearCount:    lastYearValue,
			ReportType:       kafka.ResourceSummaryTypeCategoryHistorySummary,
		})
	}

	return msgs, nil
}

func ExtractResourceTrend(es keibi.Client, job DescribeJob, lookupResources []kafka.LookupResource) ([]kafka.DescribedResource, error) {
	var msgs []kafka.DescribedResource
	var lastDayValue, lastWeekValue, lastQuarterValue, lastYearValue *int
	for idx, jobID := range []uint{job.LastDaySourceJobID, job.LastWeekSourceJobID, job.LastQuarterSourceJobID,
		job.LastYearSourceJobID} {
		var searchAfter []interface{}
		for {
			var response ResourceQueryResponse
			query, err := FindOldResourceValue(jobID, EsFetchPageSize, searchAfter)
			if err != nil {
				return nil, fmt.Errorf("failed to build query for category: %v", err.Error())
			}
			err = es.Search(context.Background(), kafka.SourceResourcesSummaryIndex, query, &response)
			if err != nil {
				return nil, fmt.Errorf("failed to run query for category: %v", err.Error())
			}

			if len(response.Hits.Hits) == 0 {
				break
			}

			for _, hit := range response.Hits.Hits {
				switch idx {
				case 0:
					if lastDayValue != nil {
						hit.Source.ResourceCount += *lastDayValue
					}
					lastDayValue = &hit.Source.ResourceCount
				case 1:
					if lastWeekValue != nil {
						hit.Source.ResourceCount += *lastWeekValue
					}
					lastWeekValue = &hit.Source.ResourceCount
				case 2:
					if lastQuarterValue != nil {
						hit.Source.ResourceCount += *lastQuarterValue
					}
					lastQuarterValue = &hit.Source.ResourceCount
				case 3:
					if lastYearValue != nil {
						hit.Source.ResourceCount += *lastYearValue
					}
					lastYearValue = &hit.Source.ResourceCount
				}
				searchAfter = hit.Sort
			}
		}
	}
	trend := kafka.SourceResourcesSummary{
		SourceID:         job.SourceID,
		SourceType:       job.SourceType,
		ResourceType:     job.ResourceType,
		SourceJobID:      job.JobID,
		DescribedAt:      job.DescribedAt,
		ResourceCount:    len(lookupResources),
		ReportType:       kafka.ResourceSummaryTypeResourceGrowthTrend,
		LastDayCount:     lastDayValue,
		LastWeekCount:    lastWeekValue,
		LastQuarterCount: lastQuarterValue,
		LastYearCount:    lastYearValue,
	}
	msgs = append(msgs, trend)

	last := kafka.SourceResourcesLastSummary{
		SourceResourcesSummary: trend,
	}
	last.ReportType = kafka.ResourceSummaryTypeLastSummary
	msgs = append(msgs, last)
	return msgs, nil
}

func ExtractDistribution(es keibi.Client, job DescribeJob, lookupResources []kafka.LookupResource) ([]kafka.DescribedResource, error) {
	var msgs []kafka.DescribedResource
	locationDistribution := map[string]int{}
	serviceDistribution := map[string]map[string]int{}
	for _, resource := range lookupResources {
		region := strings.TrimSpace(resource.Location)
		if region != "" {
			locationDistribution[region]++
			if s := cloudservice.ServiceNameByResourceType(resource.ResourceType); s != "" {
				if serviceDistribution[s] == nil {
					serviceDistribution[s] = make(map[string]int)
				}

				serviceDistribution[s][region]++
			}
		}
	}

	locDistribution := kafka.LocationDistributionResource{
		SourceID:             job.SourceID,
		SourceType:           job.SourceType,
		SourceJobID:          job.JobID,
		ResourceType:         job.ResourceType,
		LocationDistribution: locationDistribution,
		ReportType:           kafka.ResourceSummaryTypeLocationDistribution,
	}
	msgs = append(msgs, locDistribution)

	for serviceName, m := range serviceDistribution {
		msgs = append(msgs, kafka.SourceServiceDistributionResource{
			SourceID:             job.SourceID,
			ServiceName:          serviceName,
			ResourceType:         job.ResourceType,
			SourceType:           job.SourceType,
			SourceJobID:          job.JobID,
			LocationDistribution: m,
			ReportType:           kafka.ResourceSummaryTypeServiceDistributionSummary,
		})
	}
	return msgs, nil
}
