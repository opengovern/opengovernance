package client

import (
	"fmt"
	"net/http"

	"gitlab.com/keibiengine/keibi-engine/pkg/internal/httpclient"
)

type InventoryServiceClient interface {
	CountResources(ctx *httpclient.Context) (int64, error)
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
