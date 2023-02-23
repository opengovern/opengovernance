package client

import (
	"fmt"
	"net/http"

	"gitlab.com/keibiengine/keibi-engine/pkg/metadata/models"

	"gitlab.com/keibiengine/keibi-engine/pkg/internal/httpclient"
)

type MetadataServiceClient interface {
	GetConfigMetadata(ctx *httpclient.Context, key models.MetadataKey) (models.IConfigMetadata, error)
}

type onboardClient struct {
	baseURL string
}

func NewMetadataServiceClient(baseURL string) MetadataServiceClient {
	return &onboardClient{
		baseURL: baseURL,
	}
}

func (s *onboardClient) GetConfigMetadata(ctx *httpclient.Context, key models.MetadataKey) (models.IConfigMetadata, error) {
	url := fmt.Sprintf("%s/api/v1/metadata/%s", s.baseURL, string(key))
	var cnf models.IConfigMetadata
	if err := httpclient.DoRequest(http.MethodGet, url, ctx.ToHeaders(), nil, &cnf); err != nil {
		return nil, err
	}

	return cnf, nil
}
