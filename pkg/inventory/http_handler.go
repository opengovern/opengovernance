package inventory

import (
	"fmt"
	"time"

	"github.com/go-redis/cache/v8"

	client2 "gitlab.com/keibiengine/keibi-engine/pkg/onboard/client"

	"github.com/go-redis/redis/v8"

	"gitlab.com/keibiengine/keibi-engine/pkg/describe/client"
	"gitlab.com/keibiengine/keibi-engine/pkg/internal/postgres"
	"gitlab.com/keibiengine/keibi-engine/pkg/steampipe"

	"gitlab.com/keibiengine/keibi-engine/pkg/keibi-es-sdk"
	"go.uber.org/zap"
)

type HttpHandler struct {
	client          keibi.Client
	db              Database
	steampipeConn   *steampipe.Database
	schedulerClient client.SchedulerServiceClient
	onboardClient   client2.OnboardServiceClient
	rdb             *redis.Client
	cache           *cache.Cache
}

func InitializeHttpHandler(
	elasticSearchAddress string,
	elasticSearchUsername string,
	elasticSearchPassword string,
	postgresHost string,
	postgresPort string,
	postgresDb string,
	postgresUsername string,
	postgresPassword string,
	steampipeHost string,
	steampipePort string,
	steampipeDb string,
	steampipeUsername string,
	steampipePassword string,
	schedulerBaseUrl string,
	onboardBaseUrl string,
	logger *zap.Logger,
	redisAddress string,
) (h *HttpHandler, err error) {

	h = &HttpHandler{}

	fmt.Println("Initializing http handler")

	// setup postgres connection
	cfg := postgres.Config{
		Host:   postgresHost,
		Port:   postgresPort,
		User:   postgresUsername,
		Passwd: postgresPassword,
		DB:     postgresDb,
	}
	orm, err := postgres.NewClient(&cfg, logger)
	if err != nil {
		return nil, fmt.Errorf("new postgres client: %w", err)
	}

	h.db = Database{orm: orm}
	fmt.Println("Connected to the postgres database: ", postgresDb)

	err = h.db.Initialize()
	if err != nil {
		return nil, err
	}
	fmt.Println("Initialized postgres database: ", postgresDb)

	// setup steampipe connection
	steampipeConn, err := steampipe.NewSteampipeDatabase(steampipe.Option{
		Host: steampipeHost,
		Port: steampipePort,
		User: steampipeUsername,
		Pass: steampipePassword,
		Db:   steampipeDb,
	})
	h.steampipeConn = steampipeConn
	if err != nil {
		return nil, err
	}
	fmt.Println("Initialized steampipe database: ", steampipeConn)

	defaultAccountID := "default"
	h.client, err = keibi.NewClient(keibi.ClientConfig{
		Addresses: []string{elasticSearchAddress},
		Username:  &elasticSearchUsername,
		Password:  &elasticSearchPassword,
		AccountID: &defaultAccountID,
	})
	if err != nil {
		return nil, err
	}
	h.schedulerClient = client.NewSchedulerServiceClient(schedulerBaseUrl)
	h.onboardClient = client2.NewOnboardServiceClient(onboardBaseUrl)
	h.rdb = redis.NewClient(&redis.Options{
		Addr:     redisAddress,
		Password: "", // no password set
		DB:       0,  // use default DB
	})
	h.cache = cache.New(&cache.Options{
		Redis:      h.rdb,
		LocalCache: cache.NewTinyLFU(1000, time.Minute),
	})
	return h, nil
}
