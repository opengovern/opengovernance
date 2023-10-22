package cost_estimator

import (
	"fmt"
	kaytuAzure "github.com/kaytu-io/kaytu-azure-describer/pkg/kaytu-es-sdk"
	"github.com/kaytu-io/kaytu-engine/pkg/cost-estimator/db"
	"github.com/kaytu-io/kaytu-util/pkg/kaytu-es-sdk"
	"github.com/kaytu-io/kaytu-util/pkg/postgres"
	"go.uber.org/zap"
)

type HttpHandler struct {
	db     db.Database
	client kaytu.Client
	//awsClient   kaytuAws.Client
	azureClient kaytuAzure.Client

	logger *zap.Logger
}

func InitializeHttpHandler(
	postgresHost string, postgresPort string, postgresDb string, postgresUsername string, postgresPassword string, postgresSSLMode string,
	elasticSearchPassword, elasticSearchUsername, elasticSearchAddress string,
	logger *zap.Logger,
) (h *HttpHandler, err error) {
	h = &HttpHandler{}
	h.logger = logger

	h.logger.Info("Initializing http handler")

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

	//h.awsClient = kaytuAws.Client{
	//	Client: h.client,
	//}
	h.azureClient = kaytuAzure.Client{
		Client: h.client,
	}
	h.logger.Info("Initialized elasticSearch", zap.String("client", fmt.Sprintf("%v", h.client)))

	cfg := postgres.Config{
		Host:    postgresHost,
		Port:    postgresPort,
		User:    postgresUsername,
		Passwd:  postgresPassword,
		DB:      postgresDb,
		SSLMode: postgresSSLMode,
	}
	orm, err := postgres.NewClient(&cfg, logger)
	if err != nil {
		return nil, fmt.Errorf("new postgres client: %w", err)
	}
	h.logger.Info("Connected to the postgres database")

	db := db.NewDatabase(orm)
	err = db.Initialize()
	if err != nil {
		return nil, err
	}
	h.db = db
	h.logger.Info("Initialized postgres database")

	return h, nil
}
