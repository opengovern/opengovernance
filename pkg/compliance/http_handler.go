package compliance

import (
	"fmt"

	describeClient "gitlab.com/keibiengine/keibi-engine/pkg/describe/client"
	inventoryClient "gitlab.com/keibiengine/keibi-engine/pkg/inventory/client"
	onboardClient "gitlab.com/keibiengine/keibi-engine/pkg/onboard/client"

	"gitlab.com/keibiengine/keibi-engine/pkg/compliance/db"
	"gitlab.com/keibiengine/keibi-engine/pkg/internal/postgres"
	"gitlab.com/keibiengine/keibi-engine/pkg/keibi-es-sdk"

	"go.uber.org/zap"
)

type HttpHandler struct {
	client          keibi.Client
	db              db.Database
	schedulerClient describeClient.SchedulerServiceClient
	onboardClient   onboardClient.OnboardServiceClient
	inventoryClient inventoryClient.InventoryServiceClient
}

func InitializeHttpHandler(conf ServerConfig, logger *zap.Logger) (h *HttpHandler, err error) {
	h = &HttpHandler{}

	fmt.Println("Initializing http handler")

	// setup postgres connection
	cfg := postgres.Config{
		Host:    conf.PostgreSQL.Host,
		Port:    conf.PostgreSQL.Port,
		User:    conf.PostgreSQL.Username,
		Passwd:  conf.PostgreSQL.Password,
		DB:      conf.PostgreSQL.DB,
		SSLMode: conf.PostgreSQL.SSLMode,
	}
	orm, err := postgres.NewClient(&cfg, logger)
	if err != nil {
		return nil, fmt.Errorf("new postgres client: %w", err)
	}

	h.db = db.Database{Orm: orm}
	fmt.Println("Connected to the postgres database: ", conf.PostgreSQL.DB)

	err = h.db.Initialize()
	if err != nil {
		return nil, err
	}
	fmt.Println("Initialized postgres database: ", conf.PostgreSQL.DB)

	defaultAccountID := "default"
	h.client, err = keibi.NewClient(keibi.ClientConfig{
		Addresses: []string{conf.ES.Address},
		Username:  &conf.ES.Username,
		Password:  &conf.ES.Password,
		AccountID: &defaultAccountID,
	})
	if err != nil {
		return nil, err
	}
	h.schedulerClient = describeClient.NewSchedulerServiceClient(conf.Scheduler.BaseURL)
	h.onboardClient = onboardClient.NewOnboardServiceClient(conf.Onboard.BaseURL, nil)
	h.inventoryClient = inventoryClient.NewInventoryServiceClient(conf.Inventory.BaseURL)

	return h, nil
}
