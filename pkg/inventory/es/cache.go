package es

import (
	"context"
	"fmt"

	"github.com/go-redis/cache/v8"
	"github.com/go-redis/redis/v8"
	"gitlab.com/keibiengine/keibi-engine/pkg/describe/kafka"
)

func ListAccountResourceCountCached(rcache *redis.Client, cs *cache.Cache, provider string) ([]kafka.SourceResourcesSummary, error) {
	pattern := fmt.Sprintf("cache-%s-%s-%s-%s", kafka.ResourceSummaryTypeLastSummary,
		provider, "*", "*")

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
