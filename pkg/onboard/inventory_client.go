package onboard

import (
	"encoding/json"
	"gitlab.com/keibiengine/keibi-engine/pkg/inventory/api"
	"io/ioutil"
	"net/http"
	"time"
)

type InventoryClient interface {
	ListAccountsResourceCount() ([]api.TopAccountResponse, error)
}

type InventoryHttpClient struct {
	Address string
}

func NewInventoryClient(address string) InventoryClient {
	return &InventoryHttpClient{Address: address}
}

func (c InventoryHttpClient) ListAccountsResourceCount() ([]api.TopAccountResponse, error) {
	url := c.Address + "/api/v1/accounts/resource/count"
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

	var response []api.TopAccountResponse
	err = json.Unmarshal(body, &response)
	if err != nil {
		return nil, err
	}

	return response, nil
}
