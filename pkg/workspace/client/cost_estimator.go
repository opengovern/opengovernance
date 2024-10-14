package client

import (
	"encoding/json"
	"fmt"
	"github.com/opengovern/og-util/pkg/httpclient"
	"github.com/opengovern/opengovernance/pkg/workspace/api"
	"net/http"
)

type CostEstimatorPricesClient interface {
	GetAzure(ctx *httpclient.Context, req api.BaseRequest) (float64, error)
	GetAWS(ctx *httpclient.Context, req api.BaseRequest) (float64, error)
}

type costEstimatorClient struct {
	baseURL string
}

func NewCostEstimatorClient(baseURL string) CostEstimatorPricesClient {
	return &costEstimatorClient{baseURL: baseURL}
}

func (s *costEstimatorClient) GetAzure(ctx *httpclient.Context, req api.BaseRequest) (float64, error) {
	url := fmt.Sprintf("%s/api/v1/costestimator/azure", s.baseURL)

	payload, err := json.Marshal(req)
	if err != nil {
		return 0, err
	}
	fmt.Println("PAAYYYLOOOAADDD")
	fmt.Println(string(payload))
	var response float64
	if _, err := httpclient.DoRequest(ctx.Ctx, http.MethodGet, url, ctx.ToHeaders(), payload, &response); err != nil {
		return 0, err
	}
	return response, nil
}

func (s *costEstimatorClient) GetAWS(ctx *httpclient.Context, req api.BaseRequest) (float64, error) {
	url := fmt.Sprintf("%s/api/v1/costestimator/aws", s.baseURL)

	payload, err := json.Marshal(req)
	if err != nil {
		return 0, err
	}
	var response float64
	if _, err := httpclient.DoRequest(ctx.Ctx, http.MethodGet, url, ctx.ToHeaders(), payload, &response); err != nil {
		return 0, err
	}
	return response, nil
}
