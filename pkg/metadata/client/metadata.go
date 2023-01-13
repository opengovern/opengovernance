package client

import (
	"fmt"
	"net/http"

	"gitlab.com/keibiengine/keibi-engine/pkg/onboard/api"

	"gitlab.com/keibiengine/keibi-engine/pkg/internal/httpclient"
)

type MetadataServiceClient interface {
	GetSource(ctx *httpclient.Context, sourceID string) (*api.Source, error)
}

type onboardClient struct {
	baseURL string
}

func NewMetadataServiceClient(baseURL string) MetadataServiceClient {
	return &onboardClient{
		baseURL: baseURL,
	}
}

func (s *onboardClient) GetSource(ctx *httpclient.Context, sourceID string) (*api.Source, error) {
	url := fmt.Sprintf("%s/api/v1/source/%s", s.baseURL, sourceID)
	var source api.Source
	if err := httpclient.DoRequest(http.MethodGet, url, ctx.ToHeaders(), nil, &source); err != nil {
		return nil, err
	}

	return &source, nil
}
