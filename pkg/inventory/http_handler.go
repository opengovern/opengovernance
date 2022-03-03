package inventory

import (
	"fmt"
	"gitlab.com/keibiengine/keibi-engine/pkg/keibi-es-sdk"

	"gitlab.com/keibiengine/keibi-engine/pkg/internal/vault"
)

type HttpHandler struct {
	vault  vault.SourceConfig
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

	return h, nil
}
