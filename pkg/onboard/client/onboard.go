package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/go-redis/cache/v8"
	"github.com/go-redis/redis/v8"

	"gitlab.com/keibiengine/keibi-engine/pkg/source"

	"gitlab.com/keibiengine/keibi-engine/pkg/onboard/api"

	"gitlab.com/keibiengine/keibi-engine/pkg/internal/httpclient"
)

type OnboardServiceClient interface {
	GetSource(ctx *httpclient.Context, sourceID string) (*api.Source, error)
	GetSources(ctx *httpclient.Context, sourceID []string) ([]api.Source, error)
	ListSources(ctx *httpclient.Context) ([]api.Source, error)
	CountSources(ctx *httpclient.Context, provider *source.Type) (int64, error)
}

type onboardClient struct {
	baseURL string
	rdb     *redis.Client
	cache   *cache.Cache
}

func NewOnboardServiceClient(baseURL string, cache *cache.Cache) OnboardServiceClient {
	return &onboardClient{
		baseURL: baseURL,
		cache:   cache,
	}
}

func (s *onboardClient) GetSource(ctx *httpclient.Context, sourceID string) (*api.Source, error) {
	url := fmt.Sprintf("%s/api/v1/source/%s", s.baseURL, sourceID)

	var source api.Source
	if s.cache != nil {
		if err := s.cache.Get(context.Background(), "get-source-"+sourceID, &source); err == nil {
			return &source, nil
		}
	}
	if err := httpclient.DoRequest(http.MethodGet, url, ctx.ToHeaders(), nil, &source); err != nil {
		return nil, err
	}
	if s.cache != nil {
		_ = s.cache.Set(&cache.Item{
			Ctx:   context.Background(),
			Key:   "get-source-" + sourceID,
			Value: source,
			TTL:   5 * time.Minute, // dont increase it! for enabled or disabled!
		})
	}
	return &source, nil
}

func (s *onboardClient) GetSources(ctx *httpclient.Context, sourceIDs []string) ([]api.Source, error) {
	url := fmt.Sprintf("%s/api/v1/sources", s.baseURL)

	var req api.GetSourcesRequest
	var res []api.Source

	for _, sourceID := range sourceIDs {
		if s.cache != nil {
			var src api.Source
			if err := s.cache.Get(context.Background(), "get-source-"+sourceID, &src); err == nil {
				res = append(res, src)
				continue
			}
		}
		req.SourceIDs = append(req.SourceIDs, sourceID)
	}

	if len(req.SourceIDs) > 0 {
		payload, err := json.Marshal(req)
		if err != nil {
			return nil, err
		}

		var response []api.Source
		if err := httpclient.DoRequest(http.MethodPost, url, ctx.ToHeaders(), payload, &response); err != nil {
			return nil, err
		}
		if s.cache != nil {
			for _, src := range response {
				_ = s.cache.Set(&cache.Item{
					Ctx:   context.Background(),
					Key:   "get-source-" + src.ID.String(),
					Value: src,
					TTL:   5 * time.Minute, // dont increase it! for enabled or disabled!
				})
			}
		}
		res = append(res, response...)
	}

	return res, nil
}

func (s *onboardClient) ListSources(ctx *httpclient.Context) ([]api.Source, error) {
	url := fmt.Sprintf("%s/api/v1/sources", s.baseURL)

	var response []api.Source
	if err := httpclient.DoRequest(http.MethodGet, url, ctx.ToHeaders(), nil, &response); err != nil {
		return nil, err
	}
	if s.cache != nil {
		for _, src := range response {
			_ = s.cache.Set(&cache.Item{
				Ctx:   context.Background(),
				Key:   "get-source-" + src.ID.String(),
				Value: src,
				TTL:   5 * time.Minute, // dont increase it! for enabled or disabled!
			})
		}
	}
	return response, nil
}

func (s *onboardClient) CountSources(ctx *httpclient.Context, provider *source.Type) (int64, error) {
	var url string
	if provider != nil {
		url = fmt.Sprintf("%s/api/v1/sources/count?type=%s", s.baseURL, *provider)
	} else {
		url = fmt.Sprintf("%s/api/v1/sources/count", s.baseURL)
	}

	var count int64
	if err := httpclient.DoRequest(http.MethodGet, url, ctx.ToHeaders(), nil, &count); err != nil {
		return 0, err
	}
	return count, nil
}
