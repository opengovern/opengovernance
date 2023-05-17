package client

import (
	"fmt"
	"gitlab.com/keibiengine/keibi-engine/pkg/internal/httpclient"
	"net/http"

	"gitlab.com/keibiengine/keibi-engine/pkg/inventory/api"
	"gitlab.com/keibiengine/keibi-engine/pkg/source"
)

type InventoryServiceClient interface {
	CountResources(ctx *httpclient.Context) (int64, error)
	GetAccountsResourceCount(ctx *httpclient.Context, provider source.Type, sourceId *string) ([]api.ConnectionResourceCountResponse, error)
}

type inventoryClient struct {
	baseURL string
}

func NewInventoryServiceClient(baseURL string) InventoryServiceClient {
	return &inventoryClient{baseURL: baseURL}
}

func (s *inventoryClient) CountResources(ctx *httpclient.Context) (int64, error) {
	url := fmt.Sprintf("%s/api/v1/resources/count", s.baseURL)

	var count int64
	if err := httpclient.DoRequest(http.MethodGet, url, ctx.ToHeaders(), nil, &count); err != nil {
		return 0, err
	}
	return count, nil
}

func (s *inventoryClient) GetAccountsResourceCount(ctx *httpclient.Context, provider source.Type, sourceId *string) ([]api.ConnectionResourceCountResponse, error) {
	url := fmt.Sprintf("%s/api/v1/accounts/resource/count?provider=%s", s.baseURL, provider.String())
	if sourceId != nil {
		url += "&sourceId=" + *sourceId
	}

	var response []api.ConnectionResourceCountResponse
	if err := httpclient.DoRequest(http.MethodGet, url, ctx.ToHeaders(), nil, &response); err != nil {
		return nil, err
	}
	return response, nil
}
