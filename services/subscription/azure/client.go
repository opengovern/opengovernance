package azure

import (
	"encoding/json"
	"fmt"
	"github.com/kaytu-io/kaytu-engine/pkg/httpclient"
	"github.com/kaytu-io/kaytu-engine/services/subscription/azure/entities"
	"github.com/labstack/echo/v4"
	"net/http"
)

type SaaSClient struct {
	token string
}

func (c *SaaSClient) Resolve(marketplaceToken string) (*entities.AzureSaaSResolveResponse, error) {
	method := http.MethodPost
	url := "https://marketplaceapi.microsoft.com/api/saas/subscriptions/resolve?api-version=2018-08-31"
	headers := map[string]string{}
	headers["content-type"] = "application/json"
	headers["authorization"] = "Bearer " + c.token
	headers["x-ms-marketplace-token"] = marketplaceToken

	var res entities.AzureSaaSResolveResponse
	if statusCode, err := httpclient.DoRequest(method, url, headers, nil, &res); err != nil {
		if 400 <= statusCode && statusCode < 500 {
			return nil, echo.NewHTTPError(statusCode, err.Error())
		}
		return nil, err
	}
	return &res, nil
}

func (c *SaaSClient) Activate(subscrptionId string) error {
	method := http.MethodPost
	url := fmt.Sprintf("https://marketplaceapi.microsoft.com/api/saas/subscriptions/%s/activate?api-version=2018-08-31", subscrptionId)
	headers := map[string]string{}
	headers["content-type"] = "application/json"
	headers["authorization"] = "Bearer " + c.token

	if statusCode, err := httpclient.DoRequest(method, url, headers, nil, nil); err != nil {
		if 400 <= statusCode && statusCode < 500 {
			return echo.NewHTTPError(statusCode, err.Error())
		}
		return err
	}
	return nil
}

func (c *SaaSClient) ListAllSubscriptions() (*entities.AzureSaaSGetAllSubscriptionsResponse, error) {
	method := http.MethodGet
	url := "https://marketplaceapi.microsoft.com/api/saas/subscriptions?api-version=2018-08-31"
	headers := map[string]string{}
	headers["content-type"] = "application/json"
	headers["authorization"] = "Bearer " + c.token

	var res entities.AzureSaaSGetAllSubscriptionsResponse
	if statusCode, err := httpclient.DoRequest(method, url, headers, nil, &res); err != nil {
		if 400 <= statusCode && statusCode < 500 {
			return nil, echo.NewHTTPError(statusCode, err.Error())
		}
		return nil, err
	}
	return &res, nil
}

func (c *SaaSClient) GetSubscription(subscriptionId string) (*entities.AzureSaaSSubscription, error) {
	method := http.MethodGet
	url := fmt.Sprintf("https://marketplaceapi.microsoft.com/api/saas/subscriptions/%s?api-version=2018-08-31", subscriptionId)
	headers := map[string]string{}
	headers["content-type"] = "application/json"
	headers["authorization"] = "Bearer " + c.token

	var res entities.AzureSaaSSubscription
	if statusCode, err := httpclient.DoRequest(method, url, headers, nil, &res); err != nil {
		if 400 <= statusCode && statusCode < 500 {
			return nil, echo.NewHTTPError(statusCode, err.Error())
		}
		return nil, err
	}
	return &res, nil
}

func (c *SaaSClient) ListOutstandingOperations(subscriptionId string) (*entities.AzureSaaSListOutstandingOperations, error) {
	method := http.MethodGet
	url := fmt.Sprintf("https://marketplaceapi.microsoft.com/api/saas/subscriptions/%s/operations?api-version=2018-08-31", subscriptionId)
	headers := map[string]string{}
	headers["content-type"] = "application/json"
	headers["authorization"] = "Bearer " + c.token

	var res entities.AzureSaaSListOutstandingOperations
	if statusCode, err := httpclient.DoRequest(method, url, headers, nil, &res); err != nil {
		if 400 <= statusCode && statusCode < 500 {
			return nil, echo.NewHTTPError(statusCode, err.Error())
		}
		return nil, err
	}
	return &res, nil
}

func (c *SaaSClient) GetOperationStatus(subscriptionId, operationId string) (*entities.AzureSaaSGetOperationStatus, error) {
	method := http.MethodGet
	url := fmt.Sprintf("https://marketplaceapi.microsoft.com/api/saas/subscriptions/%s/operations/%s?api-version=2018-08-31", subscriptionId, operationId)
	headers := map[string]string{}
	headers["content-type"] = "application/json"
	headers["authorization"] = "Bearer " + c.token

	var res entities.AzureSaaSGetOperationStatus
	if statusCode, err := httpclient.DoRequest(method, url, headers, nil, &res); err != nil {
		if 400 <= statusCode && statusCode < 500 {
			return nil, echo.NewHTTPError(statusCode, err.Error())
		}
		return nil, err
	}
	return &res, nil
}

func (c *SaaSClient) Update(subscriptionId, operationId string) (*entities.AzureSaaSUpdateResponse, error) {
	method := http.MethodGet
	url := fmt.Sprintf("https://marketplaceapi.microsoft.com/api/saas/subscriptions/%s/operations/%s?api-version=2018-08-31", subscriptionId, operationId)
	headers := map[string]string{}
	headers["content-type"] = "application/json"
	headers["authorization"] = "Bearer " + c.token

	var res entities.AzureSaaSUpdateResponse
	if statusCode, err := httpclient.DoRequest(method, url, headers, nil, &res); err != nil {
		if 400 <= statusCode && statusCode < 500 {
			return nil, echo.NewHTTPError(statusCode, err.Error())
		}
		return nil, err
	}
	return &res, nil
}

func (c *SaaSClient) UsageEvent(req entities.AzureSaaSUsageEventRequest) (*entities.AzureSaaSUsageEventResponse, error) {
	method := http.MethodPost
	url := "https://marketplaceapi.microsoft.com/api/usageEvent?api-version=2018-08-31"
	headers := map[string]string{}
	headers["content-type"] = "application/json"
	headers["authorization"] = "Bearer " + c.token

	reqJson, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	var res entities.AzureSaaSUsageEventResponse
	if statusCode, err := httpclient.DoRequest(method, url, headers, reqJson, &res); err != nil {
		if 400 <= statusCode && statusCode < 500 {
			return nil, echo.NewHTTPError(statusCode, err.Error())
		}
		return nil, err
	}
	return &res, nil
}
