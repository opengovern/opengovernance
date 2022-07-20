package client

import (
	"fmt"
	"net/http"

	"gitlab.com/keibiengine/keibi-engine/pkg/source"

	"gitlab.com/keibiengine/keibi-engine/pkg/onboard/api"

	"gitlab.com/keibiengine/keibi-engine/pkg/internal/httpclient"
)

type OnboardServiceClient interface {
	GetSource(ctx *httpclient.Context, sourceID string) (*api.Source, error)
	CountSources(ctx *httpclient.Context, provider *source.Type) (int64, error)
}

type onboardClient struct {
	baseURL string
}

func NewOnboardServiceClient(baseURL string) OnboardServiceClient {
	return &onboardClient{baseURL: baseURL}
}

func (s *onboardClient) GetSource(ctx *httpclient.Context, sourceID string) (*api.Source, error) {
	url := fmt.Sprintf("%s/api/v1/source/%s", s.baseURL, sourceID)

	var source api.Source
	if err := httpclient.DoRequest(http.MethodGet, url, ctx.ToHeaders(), nil, &source); err != nil {
		return nil, err
	}
	return &source, nil
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
