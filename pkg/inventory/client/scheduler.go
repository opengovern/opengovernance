package client

import (
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"gitlab.com/keibiengine/keibi-engine/pkg/httprequest"
	"gitlab.com/keibiengine/keibi-engine/pkg/onboard/api"
)

func GetSource(baseUrl string, sourceID uuid.UUID) (*api.Source, error) {
	url := fmt.Sprintf("%s/api/v1/sources/%s", baseUrl, sourceID.String())

	var source api.Source
	if err := httprequest.DoRequest(http.MethodGet, url, nil, nil, &source); err != nil {
		return nil, err
	}
	return &source, nil
}
