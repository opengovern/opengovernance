package client

import (
	"fmt"
	"github.com/kaytu-io/kaytu-engine/pkg/alerting/api"
	"github.com/kaytu-io/kaytu-util/pkg/httpclient"
	"github.com/labstack/echo/v4"
	"net/http"
	"time"
)

type AlertingServiceClient interface {
	ListRules(ctx *httpclient.Context) ([]api.Rule, error)
	CountTriggersByDate(ctx *httpclient.Context, startDate, endDate time.Time) (int64, error)
}

type alertingClient struct {
	baseURL string
}

func NewAlertingServiceClient(baseURL string) AlertingServiceClient {
	return &alertingClient{baseURL: baseURL}
}

func (s *alertingClient) ListRules(ctx *httpclient.Context) ([]api.Rule, error) {
	url := fmt.Sprintf("%s/api/v1/rule/list", s.baseURL)

	var resp []api.Rule
	if statusCode, err := httpclient.DoRequest(ctx.Ctx, http.MethodGet, url, ctx.ToHeaders(), nil, &resp); err != nil {
		if 400 <= statusCode && statusCode < 500 {
			return nil, echo.NewHTTPError(statusCode, err.Error())
		}
		return nil, err
	}
	return resp, nil
}

func (s *alertingClient) CountTriggersByDate(ctx *httpclient.Context, startDate, endDate time.Time) (int64, error) {
	url := fmt.Sprintf("%s/api/v1/trigger/bydate?startDate=%d&endDate=%d", s.baseURL, startDate.UnixMilli(), endDate.UnixMilli())

	var resp int64
	if statusCode, err := httpclient.DoRequest(ctx.Ctx, http.MethodGet, url, ctx.ToHeaders(), nil, &resp); err != nil {
		if 400 <= statusCode && statusCode < 500 {
			return 0, echo.NewHTTPError(statusCode, err.Error())
		}
		return 0, err
	}
	return resp, nil
}
