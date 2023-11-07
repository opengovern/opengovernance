package client

import (
	"encoding/json"
	"fmt"
	"github.com/kaytu-io/kaytu-engine/pkg/cost-estimator/es"
	"github.com/kaytu-io/kaytu-engine/pkg/internal/httpclient"
	"net/http"
)

type CostEstimatorPricesClient interface {
	GetEC2InstanceCost(ctx *httpclient.Context, req es.EC2InstanceResponse) (float64, error)
	GetEC2VolumeCost(ctx *httpclient.Context, req es.EC2VolumeResponse) (float64, error)
	GetRDSInstance(ctx *httpclient.Context, req es.RDSDBInstanceResponse) (float64, error)
}

type costEstimatorClient struct {
	baseURL string
}

func NewCostEstimatorClient(baseURL string) CostEstimatorPricesClient {
	return &costEstimatorClient{baseURL: baseURL}
}

func (s *costEstimatorClient) GetEC2InstanceCost(ctx *httpclient.Context, req es.EC2InstanceResponse) (float64, error) {
	url := fmt.Sprintf("%s/api/v1/cost_estimator/ec2instance", s.baseURL)

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

func (s *costEstimatorClient) GetEC2VolumeCost(ctx *httpclient.Context, req es.EC2VolumeResponse) (float64, error) {
	url := fmt.Sprintf("%s/api/v1/cost_estimator/ec2volume", s.baseURL)

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

func (s *costEstimatorClient) GetRDSInstance(ctx *httpclient.Context, req es.RDSDBInstanceResponse) (float64, error) {
	url := fmt.Sprintf("%s/api/v1/cost_estimator/rdsinstance", s.baseURL)

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
