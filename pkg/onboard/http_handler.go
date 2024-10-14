package onboard

import (
	"context"
	"fmt"
	"github.com/opengovern/og-util/pkg/postgres"
	"github.com/opengovern/og-util/pkg/steampipe"
	"github.com/opengovern/og-util/pkg/vault"
	describeClient "github.com/opengovern/opengovernance/pkg/describe/client"
	metadataClient "github.com/opengovern/opengovernance/pkg/metadata/client"
	"github.com/opengovern/opengovernance/pkg/onboard/db"

	"go.uber.org/zap"
	"gopkg.in/go-playground/validator.v9"

	inventory "github.com/opengovern/opengovernance/pkg/inventory/client"
)

type HttpHandler struct {
	db                               db.Database
	steampipeConn                    *steampipe.Database
	vaultSc                          vault.VaultSourceConfig
	inventoryClient                  inventory.InventoryServiceClient
	describeClient                   describeClient.SchedulerServiceClient
	metadataClient                   metadataClient.MetadataServiceClient
	validator                        *validator.Validate
	vaultKeyId                       string
	logger                           *zap.Logger
	masterAccessKey, masterSecretKey string
}

func InitializeHttpHandler(
	ctx context.Context,
	postgresUsername string, postgresPassword string, postgresHost string, postgresPort string, postgresDb string, postgresSSLMode string,
	steampipeHost string, steampipePort string, steampipeDb string, steampipeUsername string, steampipePassword string,
	logger *zap.Logger,
	vaultSc vault.VaultSourceConfig,
	vaultKeyId string,
	inventoryBaseURL string,
	describeBaseURL string,
	metadataBaseURL string,
	masterAccessKey, masterSecretKey string,
) (*HttpHandler, error) {

	logger.Info("Initializing http handler")

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
	logger.Info("Connected to the postgres database", zap.String("database", postgresDb))

	steampipeConn, err := steampipe.NewSteampipeDatabase(steampipe.Option{
		Host: steampipeHost,
		Port: steampipePort,
		User: steampipeUsername,
		Pass: steampipePassword,
		Db:   steampipeDb,
	})
	if err != nil {
		return nil, fmt.Errorf("new steampipe client: %w", err)
	}
	logger.Info("Connected to the steampipe database", zap.String("database", steampipeDb))

	onboardDB := db.NewDatabase(orm)
	err = onboardDB.Initialize()
	if err != nil {
		return nil, err
	}
	logger.Info("Initialized postgres database: ", zap.String("database", postgresDb))

	inventoryClient := inventory.NewInventoryServiceClient(inventoryBaseURL)
	describeCli := describeClient.NewSchedulerServiceClient(describeBaseURL)

	meta := metadataClient.NewMetadataServiceClient(metadataBaseURL)

	return &HttpHandler{
		db:              onboardDB,
		steampipeConn:   steampipeConn,
		vaultSc:         vaultSc,
		inventoryClient: inventoryClient,
		describeClient:  describeCli,
		validator:       validator.New(),
		vaultKeyId:      vaultKeyId,
		logger:          logger,
		masterAccessKey: masterAccessKey,
		masterSecretKey: masterSecretKey,
		metadataClient:  meta,
	}, nil
}
