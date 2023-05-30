package client

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"time"

	"gitlab.com/keibiengine/keibi-engine/pkg/internal/httpclient"

	"github.com/go-redis/cache/v8"
	"github.com/go-redis/redis/v8"

	"github.com/kaytu-io/kaytu-util/pkg/source"

	"gitlab.com/keibiengine/keibi-engine/pkg/onboard/api"
)

type OnboardServiceClient interface {
	GetSource(ctx *httpclient.Context, sourceID string) (*api.Source, error)
	GetSourceFullCred(ctx *httpclient.Context, sourceID string) (*api.AWSCredential, *api.AzureCredential, error)
	GetSources(ctx *httpclient.Context, sourceID []string) ([]api.Source, error)
	ListSources(ctx *httpclient.Context, t []source.Type) ([]api.Source, error)
	CountSources(ctx *httpclient.Context, provider source.Type) (int64, error)
	GetSourceHealthcheck(ctx *httpclient.Context, sourceID string) (*api.Source, error)
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

func (s *onboardClient) GetSourceFullCred(ctx *httpclient.Context, sourceID string) (*api.AWSCredential, *api.AzureCredential, error) {
	url := fmt.Sprintf("%s/api/v1/source/%s/credentials/full", s.baseURL, sourceID)

	var awsCred api.AWSCredential
	var azureCred api.AzureCredential

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("new request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	for k, v := range ctx.ToHeaders() {
		req.Header.Add(k, v)
	}
	t := http.DefaultTransport.(*http.Transport)
	t.MaxIdleConns = 100
	t.MaxConnsPerHost = 100
	t.MaxIdleConnsPerHost = 100
	client := http.Client{
		Timeout:   15 * time.Second,
		Transport: t,
	}
	res, err := client.Do(req)
	if err != nil {
		return nil, nil, fmt.Errorf("do request: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		d, err := ioutil.ReadAll(res.Body)
		if err != nil {
			return nil, nil, fmt.Errorf("read body: %w", err)
		}
		return nil, nil, fmt.Errorf("http status: %d: %s", res.StatusCode, d)
	}
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, nil, err
	}

	if err := json.Unmarshal(body, &awsCred); err == nil {
		return &awsCred, nil, nil
	}
	if err := json.Unmarshal(body, &azureCred); err == nil {
		return nil, &azureCred, nil
	}
	return nil, nil, err
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

func (s *onboardClient) ListSources(ctx *httpclient.Context, t []source.Type) ([]api.Source, error) {
	url := fmt.Sprintf("%s/api/v1/sources", s.baseURL)
	for i, v := range t {
		if i == 0 {
			url += "?"
		} else {
			url += "&"
		}
		url += "connector=" + string(v)
	}

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

func (s *onboardClient) CountSources(ctx *httpclient.Context, provider source.Type) (int64, error) {
	var url string
	if !provider.IsNull() {
		url = fmt.Sprintf("%s/api/v1/sources/count?connector=%s", s.baseURL, provider.String())
	} else {
		url = fmt.Sprintf("%s/api/v1/sources/count", s.baseURL)
	}

	var count int64
	if err := httpclient.DoRequest(http.MethodGet, url, ctx.ToHeaders(), nil, &count); err != nil {
		return 0, err
	}
	return count, nil
}

func (s *onboardClient) GetSourceHealthcheck(ctx *httpclient.Context, sourceID string) (*api.Source, error) {
	url := fmt.Sprintf("%s/api/v1/source/%s/healthcheck", s.baseURL, sourceID)

	var source api.Source
	if s.cache != nil {
		if err := s.cache.Get(context.Background(), "get-source-healthcheck-"+sourceID, &source); err == nil {
			return &source, nil
		}
	}
	if err := httpclient.DoRequest(http.MethodGet, url, ctx.ToHeaders(), nil, &source); err != nil {
		return nil, err
	}
	if s.cache != nil {
		_ = s.cache.Set(&cache.Item{
			Ctx:   context.Background(),
			Key:   "get-source-healthcheck-" + sourceID,
			Value: source,
			TTL:   5 * time.Minute, // dont increase it! for enabled or disabled!
		})
	}
	return &source, nil
}
