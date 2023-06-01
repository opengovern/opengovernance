package compliance

import (
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/kaytu-io/kaytu-util/pkg/postgres"

	describeClient "gitlab.com/keibiengine/keibi-engine/pkg/describe/client"
	inventoryClient "gitlab.com/keibiengine/keibi-engine/pkg/inventory/client"
	onboardClient "gitlab.com/keibiengine/keibi-engine/pkg/onboard/client"

	"github.com/kaytu-io/kaytu-util/pkg/keibi-es-sdk"
	"gitlab.com/keibiengine/keibi-engine/pkg/compliance/db"

	"go.uber.org/zap"
)

type HttpHandler struct {
	client keibi.Client
	db     db.Database

	s3Client *s3.Client

	schedulerClient describeClient.SchedulerServiceClient
	onboardClient   onboardClient.OnboardServiceClient
	inventoryClient inventoryClient.InventoryServiceClient
}

func InitializeHttpHandler(
	conf ServerConfig,
	s3Region, s3AccessKey, s3AccessSecret, s3Bucket string,
	logger *zap.Logger) (h *HttpHandler, err error) {
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

	h.schedulerClient = describeClient.NewSchedulerServiceClient(conf.Scheduler.BaseURL)
	h.onboardClient = onboardClient.NewOnboardServiceClient(conf.Onboard.BaseURL, nil)
	h.inventoryClient = inventoryClient.NewInventoryServiceClient(conf.Inventory.BaseURL)

	return h, nil
}
