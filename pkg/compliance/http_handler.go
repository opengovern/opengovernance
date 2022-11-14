package compliance

import (
	"fmt"

	"gitlab.com/keibiengine/keibi-engine/pkg/internal/postgres"

	client3 "gitlab.com/keibiengine/keibi-engine/pkg/inventory/client"

	client2 "gitlab.com/keibiengine/keibi-engine/pkg/onboard/client"
	"go.uber.org/zap"

	"gitlab.com/keibiengine/keibi-engine/pkg/describe/client"
	"gitlab.com/keibiengine/keibi-engine/pkg/keibi-es-sdk"
)

type HttpHandler struct {
	client          keibi.Client
	db              Database
	schedulerClient client.SchedulerServiceClient
	onboardClient   client2.OnboardServiceClient
	inventoryClient client3.InventoryServiceClient
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

	h.db = Database{orm: orm}
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
	h.schedulerClient = client.NewSchedulerServiceClient(conf.Scheduler.BaseURL)
	h.onboardClient = client2.NewOnboardServiceClient(conf.Onboard.BaseURL, nil)
	h.inventoryClient = client3.NewInventoryServiceClient(conf.Inventory.BaseURL)

	return h, nil
}
