package inventory

import (
	"fmt"
	"gitlab.com/keibiengine/keibi-engine/pkg/keibi-es-sdk"
)

type HttpHandler struct {
	client keibi.Client
}

func InitializeHttpHandler(
	elasticSearchAddress string,
	elasticSearchUsername string,
	elasticSearchPassword string,
) (h *HttpHandler, err error) {

	h = &HttpHandler{}

	fmt.Println("Initializing http handler")

	defaultAccountID := "default"
	h.client, err = keibi.NewClient(keibi.ClientConfig{
		Addresses: []string{elasticSearchAddress},
		Username:  &elasticSearchUsername,
		Password:  &elasticSearchPassword,
		AccountID: &defaultAccountID,
	})
	if err != nil {
		return nil, err
	}

	return h, nil
}
