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
	GetAzureVm(ctx *httpclient.Context, req api.GetAzureVmRequest) (float64, error)
	GetAzureManagedStorage(ctx *httpclient.Context, req api.GetAzureManagedStorageRequest) (float64, error)
	GetAzureLoadBalancer(ctx *httpclient.Context, req api.GetAzureLoadBalancerRequest) (float64, error)
	GetAzure(ctx *httpclient.Context, resourceType string, req any) (float64, error)
	GetAWS(ctx *httpclient.Context, resourceType string, req any) (float64, error)
	GetAzureSqlServerDatabase(ctx *httpclient.Context, req api.GetAzureSqlServersDatabasesRequest) (float64, error)
}

type costEstimatorClient struct {
	baseURL string
}

func NewCostEstimatorClient(baseURL string) CostEstimatorPricesClient {
	return &costEstimatorClient{baseURL: baseURL}
}

func (s *costEstimatorClient) GetEC2InstanceCost(ctx *httpclient.Context, req api.GetEC2InstanceCostRequest) (float64, error) {
	url := fmt.Sprintf("%s/api/v1/costestimator/aws/ec2instance", s.baseURL)

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
	url := fmt.Sprintf("%s/api/v1/costestimator/aws/ec2volume", s.baseURL)

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
	url := fmt.Sprintf("%s/api/v1/costestimator/aws/loadbalancer", s.baseURL)

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
	url := fmt.Sprintf("%s/api/v1/costestimator/aws/rdsinstance", s.baseURL)

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

func (s *costEstimatorClient) GetAzureVm(ctx *httpclient.Context, req api.GetAzureVmRequest) (float64, error) {
	url := fmt.Sprintf("%s/api/v1/costestimator/azure/virtualmachine", s.baseURL)

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

func (s *costEstimatorClient) GetAzureManagedStorage(ctx *httpclient.Context, req api.GetAzureManagedStorageRequest) (float64, error) {
	url := fmt.Sprintf("%s/api/v1/costestimator/azure/managedstorage", s.baseURL)

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

func (s *costEstimatorClient) GetAzureLoadBalancer(ctx *httpclient.Context, req api.GetAzureLoadBalancerRequest) (float64, error) {
	url := fmt.Sprintf("%s/api/v1/costestimator/azure/loadbalancer", s.baseURL)

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

func (s *costEstimatorClient) GetAzureSqlServerDatabase(ctx *httpclient.Context, req api.GetAzureSqlServersDatabasesRequest) (float64, error) {
	url := fmt.Sprintf("%s/api/v1/costestimator/azure/sqlserverdatabse", s.baseURL)
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

func (s *costEstimatorClient) GetAzure(ctx *httpclient.Context, resourceType string, req any) (float64, error) {
	url := fmt.Sprintf("%s/api/v1/costestimator/azure/%s", s.baseURL, resourceType)

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

func (s *costEstimatorClient) GetAWS(ctx *httpclient.Context, resourceType string, req any) (float64, error) {
	url := fmt.Sprintf("%s/api/v1/costestimator/aws/%s", s.baseURL, resourceType)

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
