package compliance

import (
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/kaytu-io/kaytu-util/pkg/postgres"
	"github.com/kaytu-io/kaytu-util/pkg/queue"

	describeClient "github.com/kaytu-io/kaytu-engine/pkg/describe/client"
	inventoryClient "github.com/kaytu-io/kaytu-engine/pkg/inventory/client"
	onboardClient "github.com/kaytu-io/kaytu-engine/pkg/onboard/client"

	"github.com/kaytu-io/kaytu-engine/pkg/compliance/db"
	"github.com/kaytu-io/kaytu-util/pkg/keibi-es-sdk"

	"go.uber.org/zap"
)

type HttpHandler struct {
	client keibi.Client
	db     db.Database
	logger *zap.Logger

	s3Client *s3.Client

	syncJobsQueue queue.Interface

	schedulerClient describeClient.SchedulerServiceClient
	onboardClient   onboardClient.OnboardServiceClient
	inventoryClient inventoryClient.InventoryServiceClient
}

func InitializeHttpHandler(
	conf ServerConfig,
	s3Region, s3AccessKey, s3AccessSecret string,
	logger *zap.Logger) (h *HttpHandler, err error) {
	h = &HttpHandler{
		logger: logger,
	}

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

	if s3Region == "" {
		s3Region = "us-west-2"
	}
	var awsConfig aws.Config
	if s3AccessKey == "" || s3AccessSecret == "" {
		//load default credentials
		awsConfig = aws.Config{
			Region: s3Region,
		}
	} else {
		awsConfig = aws.Config{
			Region:      s3Region,
			Credentials: credentials.NewStaticCredentialsProvider(s3AccessKey, s3AccessSecret, ""),
		}
	}
	h.s3Client = s3.NewFromConfig(awsConfig)

	qCfg := queue.Config{}
	qCfg.Server.Username = conf.RabbitMq.Username
	qCfg.Server.Password = conf.RabbitMq.Password
	qCfg.Server.Host = conf.RabbitMq.Service
	qCfg.Server.Port = 5672
	qCfg.Queue.Name = conf.MigratorJobQueueName
	qCfg.Queue.Durable = true
	qCfg.Producer.ID = "compliance"
	syncJobsQueue, err := queue.New(qCfg)
	if err != nil {
		return nil, fmt.Errorf("new queue: %w", err)
	}
	h.syncJobsQueue = syncJobsQueue

	h.schedulerClient = describeClient.NewSchedulerServiceClient(conf.Scheduler.BaseURL)
	h.onboardClient = onboardClient.NewOnboardServiceClient(conf.Onboard.BaseURL, nil)
	h.inventoryClient = inventoryClient.NewInventoryServiceClient(conf.Inventory.BaseURL)

	return h, nil
}
