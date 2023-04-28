package onboard

import (
	"context"
	"fmt"

	"go.uber.org/zap"
	"gopkg.in/go-playground/validator.v9"

	"gitlab.com/keibiengine/keibi-engine/pkg/internal/postgres"
	"gitlab.com/keibiengine/keibi-engine/pkg/internal/queue"
	inventory "gitlab.com/keibiengine/keibi-engine/pkg/inventory/client"
	"gitlab.com/keibiengine/keibi-engine/pkg/vault"
)

type HttpHandler struct {
	db                    Database
	sourceEventsQueue     queue.Interface
	kms                   *vault.KMSVaultSourceConfig
	awsPermissionCheckURL string
	inventoryClient       inventory.InventoryServiceClient
	validator             *validator.Validate
	keyARN                string
}

func InitializeHttpHandler(
	rabbitMQUsername string,
	rabbitMQPassword string,
	rabbitMQHost string,
	rabbitMQPort int,
	sourceEventsQueueName string,
	postgresUsername string,
	postgresPassword string,
	postgresHost string,
	postgresPort string,
	postgresDb string,
	postgresSSLMode string,
	logger *zap.Logger,
	awsPermissionCheckURL string,
	keyARN string,
	inventoryBaseURL string,
) (*HttpHandler, error) {

	fmt.Println("Initializing http handler")

	// setup source events queue
	qCfg := queue.Config{}
	qCfg.Server.Username = rabbitMQUsername
	qCfg.Server.Password = rabbitMQPassword
	qCfg.Server.Host = rabbitMQHost
	qCfg.Server.Port = rabbitMQPort
	qCfg.Queue.Name = sourceEventsQueueName
	qCfg.Queue.Durable = true
	qCfg.Producer.ID = "onboard-service"
	sourceEventsQueue, err := queue.New(qCfg)
	if err != nil {
		return nil, err
	}

	fmt.Println("Connected to the source queue: ", sourceEventsQueueName)

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
	fmt.Println("Connected to the postgres database: ", postgresDb)

	kms, err := vault.NewKMSVaultSourceConfig(context.Background())
	if err != nil {
		return nil, err
	}

	db := Database{orm: orm}
	err = db.Initialize()
	if err != nil {
		return nil, err
	}
	fmt.Println("Initialized postgres database: ", postgresDb)

	inventoryClient := inventory.NewInventoryServiceClient(inventoryBaseURL)

	return &HttpHandler{
		kms:                   kms,
		db:                    db,
		sourceEventsQueue:     sourceEventsQueue,
		awsPermissionCheckURL: awsPermissionCheckURL,
		inventoryClient:       inventoryClient,
		keyARN:                keyARN,
		validator:             validator.New(),
	}, nil
}
