package client

import (
	"encoding/json"
	"fmt"
	"github.com/kaytu-io/kaytu-engine/pkg/httpclient"
	"github.com/kaytu-io/kaytu-engine/services/es-sink/api/entity"
	"github.com/kaytu-io/kaytu-util/pkg/es"
	"github.com/labstack/echo/v4"
	"net/http"
)

type EsSinkServiceClient interface {
	Ingest(ctx *httpclient.Context, docs []es.Doc) error
}

type esSinkServiceClient struct {
	baseUrl string
}

func NewEsSinkServiceClient(baseUrl string) EsSinkServiceClient {
	return &esSinkServiceClient{
		baseUrl: baseUrl,
	}
}

func (c *esSinkServiceClient) Ingest(ctx *httpclient.Context, docs []es.Doc) error {
	url := fmt.Sprintf("%s/api/v1/ingest", c.baseUrl)

	jsonDocs, err := json.Marshal(docs)
	if err != nil {
		return err
	}
	var baseDocs []es.DocBase
	err = json.Unmarshal(jsonDocs, &baseDocs)
	if err != nil {
		return err
	}

	req := entity.IngestRequest{
		Docs: baseDocs,
	}

	reqJson, err := json.Marshal(req)
	if err != nil {
		return err
	}

	var res string

	if statusCode, err := httpclient.DoRequest(http.MethodPost, url, ctx.ToHeaders(), reqJson, &res); err != nil {
		if 400 <= statusCode && statusCode < 500 {
			return echo.NewHTTPError(statusCode, err.Error())
		}
		return err
	}

	return nil
}
