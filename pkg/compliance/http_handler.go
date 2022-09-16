package compliance

import (
	"fmt"

	client2 "gitlab.com/keibiengine/keibi-engine/pkg/onboard/client"
	"go.uber.org/zap"

	"gitlab.com/keibiengine/keibi-engine/pkg/describe/client"
	"gitlab.com/keibiengine/keibi-engine/pkg/internal/postgres"
	"gitlab.com/keibiengine/keibi-engine/pkg/keibi-es-sdk"
)

type HttpHandler struct {
	client          keibi.Client
	db              Database
	schedulerClient client.SchedulerServiceClient
	onboardClient   client2.OnboardServiceClient
}

func InitializeHttpHandler(conf ServerConfig, logger *zap.Logger) (h *HttpHandler, err error) {
	h = &HttpHandler{}

	fmt.Println("Initializing http handler")

	// setup postgres connection
	cfg := postgres.Config{
		Host:   conf.PostgreSQL.Host,
		Port:   conf.PostgreSQL.Port,
		User:   conf.PostgreSQL.Username,
		Passwd: conf.PostgreSQL.Password,
		DB:     conf.PostgreSQL.DB,
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

	return h, nil
}
