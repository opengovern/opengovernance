package esSinkClient

import (
	"encoding/json"
	"fmt"
	"github.com/kaytu-io/kaytu-engine/pkg/httpclient"
	"github.com/kaytu-io/kaytu-engine/services/es-sink/api/entity"
	"github.com/kaytu-io/kaytu-util/pkg/es"
	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
	"net/http"
)

type EsSinkServiceClient interface {
	Ingest(ctx *httpclient.Context, docs []es.Doc) error
}

type esSinkServiceClient struct {
	logger  *zap.Logger
	baseUrl string
}

func NewEsSinkServiceClient(logger *zap.Logger, baseUrl string) EsSinkServiceClient {
	return &esSinkServiceClient{
		logger:  logger,
		baseUrl: baseUrl,
	}
}

func (c *esSinkServiceClient) Ingest(ctx *httpclient.Context, docs []es.Doc) error {
	url := fmt.Sprintf("%s/api/v1/ingest", c.baseUrl)

	jsonDocs, err := json.Marshal(docs)
	if err != nil {
		c.logger.Error("failed to marshal docs", zap.Error(err), zap.Any("docs", docs))
		return err
	}
	var baseDocs []es.DocBase
	err = json.Unmarshal(jsonDocs, &baseDocs)
	if err != nil {
		c.logger.Error("failed to unmarshal docs", zap.Error(err), zap.Any("docs", docs), zap.String("jsonDocs", string(jsonDocs)))
		return err
	}

	req := entity.IngestRequest{
		Docs: baseDocs,
	}

	reqJson, err := json.Marshal(req)
	if err != nil {
		c.logger.Error("failed to marshal request", zap.Error(err), zap.Any("request", req))
		return err
	}

	var res string
	if statusCode, err := httpclient.DoRequest(http.MethodPost, url, ctx.ToHeaders(), reqJson, &res); err != nil {
		if 400 <= statusCode && statusCode < 500 {
			return echo.NewHTTPError(statusCode, err.Error())
		}
		c.logger.Error("failed to do request", zap.Error(err), zap.String("url", url), zap.String("reqJson", string(reqJson)))
		return err
	}

	return nil
}
