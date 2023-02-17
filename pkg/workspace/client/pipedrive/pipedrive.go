package pipedrive

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"go.uber.org/zap"
)

type PipedriveServiceClient interface {
	GetPipedriveOrganization(ctx context.Context, id int) (*Organization, error)
}

type pipedriveServiceClient struct {
	Logger   *zap.Logger
	BaseUrl  string
	ApiToken string
}

func NewPipedriveServiceClient(logger *zap.Logger, baseURL string, apiToken string) PipedriveServiceClient {
	return &pipedriveServiceClient{
		Logger:   logger,
		BaseUrl:  baseURL,
		ApiToken: apiToken,
	}
}

func (p *pipedriveServiceClient) GetPipedriveOrganization(ctx context.Context, id int) (*Organization, error) {
	url := fmt.Sprintf("%s/v1/organizations/%d?api_token=%s", p.BaseUrl, id, p.ApiToken)

	client := &http.Client{
		Timeout: time.Minute,
	}
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		p.Logger.Error("failed to create get organization details request", zap.Error(err))
		return nil, err
	}
	req.Header.Add("Accept", "application/json")

	res, err := client.Do(req)
	if err != nil {
		p.Logger.Error("failed to get organization details", zap.Error(err))
		return nil, err
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		p.Logger.Error("failed to read response body", zap.Error(err))
		return nil, err
	}
	if res.StatusCode != http.StatusOK {
		p.Logger.Error("failed to get organization details", zap.Int("statusCode", res.StatusCode), zap.String("response", string(body)))
		return nil, fmt.Errorf("failed to get organization details, status: %d body: %s", res.StatusCode, string(body))
	}

	var result GetOrganizationDetailsResponse
	if err := json.Unmarshal(body, &result); err != nil {
		p.Logger.Error("failed to unmarshal get organization response body", zap.Error(err))
		return nil, err
	}

	if result.Success == false {
		p.Logger.Error("failed to get organization details", zap.String("response", string(body)))
		return nil, fmt.Errorf("failed to get organization details, status: %d body: %s", res.StatusCode, string(body))
	}

	return &result.Data, nil
}
