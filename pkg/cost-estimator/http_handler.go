package cost_estimator

import (
	"fmt"
	kaytuAws "github.com/kaytu-io/kaytu-aws-describer/pkg/kaytu-es-sdk"
	kaytuAzure "github.com/kaytu-io/kaytu-azure-describer/pkg/kaytu-es-sdk"
	"github.com/kaytu-io/kaytu-util/pkg/kaytu-es-sdk"
	"go.uber.org/zap"
)

type HttpHandler struct {
	client      kaytu.Client
	awsClient   kaytuAws.Client
	azureClient kaytuAzure.Client

	logger *zap.Logger
}

func InitializeHttpHandler(
	elasticSearchPassword, elasticSearchUsername, elasticSearchAddress string,
	logger *zap.Logger,
) (h *HttpHandler, err error) {
	h = &HttpHandler{}

	fmt.Println("Initializing http handler")

	defaultAccountID := "default"
	h.client, err = kaytu.NewClient(kaytu.ClientConfig{
		Addresses: []string{elasticSearchAddress},
		Username:  &elasticSearchUsername,
		Password:  &elasticSearchPassword,
		AccountID: &defaultAccountID,
	})
	if err != nil {
		return nil, err
	}

	h.awsClient = kaytuAws.client{
		Client: h.client,
	}
	h.azureClient = kaytuAzure.Client{
		Client: h.client,
	}
	fmt.Println("Initialized elasticSearch : ", h.client)

	h.logger = logger

	return h, nil
}
