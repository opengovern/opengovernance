package client

import (
	"encoding/json"
	"fmt"
	"github.com/kaytu-io/kaytu-engine/pkg/internal/httpclient"
	"github.com/kaytu-io/kaytu-engine/pkg/workspace/api"
	"net/http"
)

type CostEstimatorPricesClient interface {
	GetEC2InstanceCost(ctx *httpclient.Context, req api.GetEC2InstanceCostRequest) (float64, error)
	GetEC2VolumeCost(ctx *httpclient.Context, req api.GetEC2VolumeCostRequest) (float64, error)
	GetLBCost(ctx *httpclient.Context, req api.GetLBCostRequest) (float64, error)
	GetRDSInstance(ctx *httpclient.Context, req api.GetRDSInstanceRequest) (float64, error)
}

type costEstimatorClient struct {
	baseURL string
}

func NewCostEstimatorClient(baseURL string) CostEstimatorPricesClient {
	return &costEstimatorClient{baseURL: baseURL}
}

func (s *costEstimatorClient) GetEC2InstanceCost(ctx *httpclient.Context, req api.GetEC2InstanceCostRequest) (float64, error) {
	url := fmt.Sprintf("%s/api/v1/costestimator/ec2instance", s.baseURL)

	payload, err := json.Marshal(req)
	if err != nil {
		return 0, err
	}
	var response float64
	if _, err := httpclient.DoRequest(http.MethodGet, url, ctx.ToHeaders(), payload, &response); err != nil {
		return 0, err
	}
	return response, nil
}

func (s *costEstimatorClient) GetEC2VolumeCost(ctx *httpclient.Context, req api.GetEC2VolumeCostRequest) (float64, error) {
	url := fmt.Sprintf("%s/api/v1/costestimator/ec2volume", s.baseURL)

	payload, err := json.Marshal(req)
	if err != nil {
		return 0, err
	}
	var response float64
	if _, err := httpclient.DoRequest(http.MethodGet, url, ctx.ToHeaders(), payload, &response); err != nil {
		return 0, err
	}
	return response, nil
}

func (s *costEstimatorClient) GetLBCost(ctx *httpclient.Context, req api.GetLBCostRequest) (float64, error) {
	url := fmt.Sprintf("%s/api/v1/costestimator/loadbalancer", s.baseURL)

	payload, err := json.Marshal(req)
	if err != nil {
		return 0, err
	}
	var response float64
	if _, err := httpclient.DoRequest(http.MethodGet, url, ctx.ToHeaders(), payload, &response); err != nil {
		return 0, err
	}
	return response, nil
}

func (s *costEstimatorClient) GetRDSInstance(ctx *httpclient.Context, req api.GetRDSInstanceRequest) (float64, error) {
	url := fmt.Sprintf("%s/api/v1/costestimator/rdsinstance", s.baseURL)

	payload, err := json.Marshal(req)
	if err != nil {
		return 0, err
	}
	var response float64
	if _, err := httpclient.DoRequest(http.MethodGet, url, ctx.ToHeaders(), payload, &response); err != nil {
		return 0, err
	}
	return response, nil
}
