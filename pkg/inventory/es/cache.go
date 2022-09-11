package es

import (
	"context"
	"fmt"

	"gitlab.com/keibiengine/keibi-engine/pkg/source"

	"github.com/go-redis/cache/v8"
	"github.com/go-redis/redis/v8"
	"gitlab.com/keibiengine/keibi-engine/pkg/describe/kafka"
)

func FetchResourceLastSummaryCached(rcache *redis.Client, cs *cache.Cache,
	provider source.Type, sourceID *string, resourceType *string) ([]kafka.SourceResourcesSummary, error) {
	providerFilter := "*"
	if !provider.IsNull() {
		providerFilter = provider.String()
	}
	sourceIDFilter := "*"
	if sourceID != nil {
		sourceIDFilter = *sourceID
	}
	resourceTypeFilter := "*"
	if resourceType != nil {
		resourceTypeFilter = *resourceType
	}

	pattern := fmt.Sprintf("cache-%s-%s-%s-%s", kafka.ResourceSummaryTypeLastSummary,
		providerFilter, sourceIDFilter, resourceTypeFilter)

	var err error
	var cursor uint64 = 0
	var response []kafka.SourceResourcesSummary
	for {
		var keys []string
		keys, cursor, err = rcache.Scan(context.Background(), cursor, pattern, 0).Result()
		if err != nil {
			return nil, err
		}

		for _, key := range keys {
			var source kafka.SourceResourcesSummary
			if err := cs.Get(context.Background(), key, &source); err != nil {
				return nil, err
			}
			response = append(response, source)
		}

		if cursor == 0 {
			break
		}
	}
	return response, nil
}

func FetchLocationDistributionCached(rcache *redis.Client, cs *cache.Cache,
	provider source.Type, sourceID *string) ([]kafka.LocationDistributionResource, error) {
	providerFilter := "*"
	if !provider.IsNull() {
		providerFilter = provider.String()
	}
	sourceIDFilter := "*"
	if sourceID != nil {
		sourceIDFilter = *sourceID
	}

	pattern := fmt.Sprintf("cache-%s-%s-%s-%s", kafka.ResourceSummaryTypeLocationDistribution,
		providerFilter, sourceIDFilter, "*")

	var err error
	var cursor uint64 = 0
	var response []kafka.LocationDistributionResource
	for {
		var keys []string
		keys, cursor, err = rcache.Scan(context.Background(), cursor, pattern, 0).Result()
		if err != nil {
			return nil, err
		}

		for _, key := range keys {
			var source kafka.LocationDistributionResource
			if err := cs.Get(context.Background(), key, &source); err != nil {
				return nil, err
			}
			response = append(response, source)
		}

		if cursor == 0 {
			break
		}
	}
	return response, nil
}

func FetchCategoriesCached(rcache *redis.Client, cs *cache.Cache,
	provider source.Type, sourceID *string) ([]kafka.SourceCategorySummary, error) {
	providerFilter := "*"
	if !provider.IsNull() {
		providerFilter = provider.String()
	}
	sourceIDFilter := "*"
	if sourceID != nil {
		sourceIDFilter = *sourceID
	}
	resourceTypeFilter := "*"

	pattern := fmt.Sprintf("cache-%s-%s-%s-%s", kafka.ResourceSummaryTypeLastCategorySummary,
		providerFilter, sourceIDFilter, resourceTypeFilter)

	var err error
	var cursor uint64 = 0
	var response []kafka.SourceCategorySummary
	for {
		var keys []string
		keys, cursor, err = rcache.Scan(context.Background(), cursor, pattern, 0).Result()
		if err != nil {
			return nil, err
		}

		for _, key := range keys {
			var source kafka.SourceCategorySummary
			if err := cs.Get(context.Background(), key, &source); err != nil {
				return nil, err
			}
			response = append(response, source)
		}

		if cursor == 0 {
			break
		}
	}
	return response, nil
}

func FetchServicesCached(rcache *redis.Client, cs *cache.Cache,
	provider source.Type, sourceID *string) ([]kafka.SourceServicesSummary, error) {
	providerFilter := "*"
	if !provider.IsNull() {
		providerFilter = provider.String()
	}
	sourceIDFilter := "*"
	if sourceID != nil {
		sourceIDFilter = *sourceID
	}
	resourceTypeFilter := "*"

	pattern := fmt.Sprintf("cache-%s-%s-%s-%s", kafka.ResourceSummaryTypeLastServiceSummary,
		providerFilter, sourceIDFilter, resourceTypeFilter)

	var err error
	var cursor uint64 = 0
	var response []kafka.SourceServicesSummary
	for {
		var keys []string
		keys, cursor, err = rcache.Scan(context.Background(), cursor, pattern, 0).Result()
		if err != nil {
			return nil, err
		}

		for _, key := range keys {
			var source kafka.SourceServicesSummary
			if err := cs.Get(context.Background(), key, &source); err != nil {
				return nil, err
			}
			response = append(response, source)
		}

		if cursor == 0 {
			break
		}
	}
	return response, nil
}
