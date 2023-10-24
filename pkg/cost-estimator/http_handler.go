package cost_estimator

import (
	"fmt"
	kaytuAzure "github.com/kaytu-io/kaytu-azure-describer/pkg/kaytu-es-sdk"
	"github.com/kaytu-io/kaytu-util/pkg/kaytu-es-sdk"
	"go.uber.org/zap"
)

type HttpHandler struct {
	client kaytu.Client
	//awsClient   kaytuAws.Client
	azureClient kaytuAzure.Client

	logger *zap.Logger
}

func InitializeHttpHandler(
	elasticSearchPassword, elasticSearchUsername, elasticSearchAddress string,
	logger *zap.Logger,
) (h *HttpHandler, err error) {
	h = &HttpHandler{}

	fmt.Println("Initializing http handler")

	h.client, err = kaytu.NewClient(kaytu.ClientConfig{
		Addresses: []string{elasticSearchAddress},
		Username:  &elasticSearchUsername,
		Password:  &elasticSearchPassword,
	})
	if err != nil {
		return nil, err
	}

	//h.awsClient = kaytuAws.Client{
	//	Client: h.client,
	//}
	h.azureClient = kaytuAzure.Client{
		Client: h.client,
	}
	fmt.Println("Initialized elasticSearch : ", h.client)

	h.logger = logger

	return h, nil
}
