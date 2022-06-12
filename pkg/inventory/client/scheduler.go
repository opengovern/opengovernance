package client

import (
	"fmt"
	"net/http"

	"gitlab.com/keibiengine/keibi-engine/pkg/httprequest"
	"gitlab.com/keibiengine/keibi-engine/pkg/onboard/api"
)

func GetSource(baseUrl string, sourceID string) (*api.Source, error) {
	url := fmt.Sprintf("%s/api/v1/sources/%s", baseUrl, sourceID)

	var source api.Source
	if err := httprequest.DoRequest(http.MethodGet, url, nil, nil, &source); err != nil {
		return nil, err
	}
	return &source, nil
}
