package alerting

import (
	"fmt"
	"github.com/kaytu-io/kaytu-engine/pkg/compliance/client"
	onboardClient "github.com/kaytu-io/kaytu-engine/pkg/onboard/client"
	"github.com/kaytu-io/kaytu-util/pkg/postgres"
	"go.uber.org/zap"
)

type HttpHandler struct {
	db               Database
	onboardClient    onboardClient.OnboardServiceClient
	complianceClient client.ComplianceServiceClient
	logger           *zap.Logger
}

func InitializeHttpHandler(
	postgresHost string, postgresPort string, postgresDb string, postgresUsername string, postgresPassword string, postgresSSLMode string,
	complianceBaseUrl string, onboardBaseUrl string,
	logger *zap.Logger,
) (h *HttpHandler, err error) {

	httpHandler := &HttpHandler{
		logger: logger,
	}

	fmt.Println("Initializing http handler")

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
	httpHandler.logger.Info("Connected to the postgres database")

	db := NewDatabase(orm)
	err = db.Initialize()
	if err != nil {
		return nil, err
	}
	httpHandler.db = db
	httpHandler.logger.Info("Initialized postgres database")

	httpHandler.onboardClient = onboardClient.NewOnboardServiceClient(onboardBaseUrl, nil)
	httpHandler.complianceClient = client.NewComplianceClient(complianceBaseUrl)

	return httpHandler, nil
}
