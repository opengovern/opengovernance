package onboard

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"time"

	"gitlab.com/keibiengine/keibi-engine/pkg/describe/api"
)

type DescribeSchedulerClient interface {
	ListSources() ([]api.Source, error)
}

type DescribeSchedulerHTTPClient struct {
	Address string
}

func NewDescribeSchedulerClient(address string) DescribeSchedulerClient {
	return &DescribeSchedulerHTTPClient{Address: address}
}

func (c DescribeSchedulerHTTPClient) ListSources() ([]api.Source, error) {
	url := c.Address + "/api/v1/sources"
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	client := http.Client{
		Timeout: 5 * time.Second,
	}
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	var response []api.Source
	err = json.Unmarshal(body, &response)
	if err != nil {
		return nil, err
	}

	return response, nil
}
