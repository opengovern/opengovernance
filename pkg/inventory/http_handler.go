package inventory

import (
	"fmt"
	metadataClient "github.com/kaytu-io/kaytu-engine/pkg/metadata/client"

	kaytuAws "github.com/kaytu-io/kaytu-aws-describer/pkg/kaytu-es-sdk"
	awsSteampipe "github.com/kaytu-io/kaytu-aws-describer/pkg/steampipe"
	kaytuAzure "github.com/kaytu-io/kaytu-azure-describer/pkg/kaytu-es-sdk"
	azureSteampipe "github.com/kaytu-io/kaytu-azure-describer/pkg/steampipe"
	complianceClient "github.com/kaytu-io/kaytu-engine/pkg/compliance/client"
	describeClient "github.com/kaytu-io/kaytu-engine/pkg/describe/client"
	onboardClient "github.com/kaytu-io/kaytu-engine/pkg/onboard/client"
	"github.com/kaytu-io/kaytu-util/pkg/config"
	"github.com/kaytu-io/kaytu-util/pkg/kaytu-es-sdk"
	"github.com/kaytu-io/kaytu-util/pkg/postgres"
	"github.com/kaytu-io/kaytu-util/pkg/steampipe"
	"github.com/turbot/steampipe-plugin-sdk/v5/plugin"
	"go.uber.org/zap"
)

type HttpHandler struct {
	client           kaytu.Client
	awsClient        kaytuAws.Client
	azureClient      kaytuAzure.Client
	db               Database
	steampipeConn    *steampipe.Database
	schedulerClient  describeClient.SchedulerServiceClient
	onboardClient    onboardClient.OnboardServiceClient
	complianceClient complianceClient.ComplianceServiceClient
	metadataClient   metadataClient.MetadataServiceClient

	logger *zap.Logger

	awsPlg, azurePlg, azureADPlg *plugin.Plugin
}

func InitializeHttpHandler(
	esConf config.ElasticSearch,
	postgresHost string, postgresPort string, postgresDb string, postgresUsername string, postgresPassword string, postgresSSLMode string,
	steampipeHost string, steampipePort string, steampipeDb string, steampipeUsername string, steampipePassword string,
	schedulerBaseUrl string, onboardBaseUrl string, complianceBaseUrl string, metadataBaseUrl string,
	logger *zap.Logger,
) (h *HttpHandler, err error) {
	h = &HttpHandler{}

	fmt.Println("Initializing http handler")

	// setup postgres connection
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

	h.client, err = kaytu.NewClient(kaytu.ClientConfig{
		Addresses:     []string{esConf.Address},
		Username:      &esConf.Username,
		Password:      &esConf.Password,
		IsOnAks:       &esConf.IsOnAks,
		IsOpenSearch:  &esConf.IsOpenSearch,
		AwsRegion:     &esConf.AwsRegion,
		AssumeRoleArn: &esConf.AssumeRoleArn,
	})
	if err != nil {
		return nil, err
	}
	h.awsClient = kaytuAws.Client{
		Client: h.client,
	}
	h.azureClient = kaytuAzure.Client{
		Client: h.client,
	}
	h.schedulerClient = describeClient.NewSchedulerServiceClient(schedulerBaseUrl)

	h.onboardClient = onboardClient.NewOnboardServiceClient(onboardBaseUrl)
	h.complianceClient = complianceClient.NewComplianceClient(complianceBaseUrl)
	h.metadataClient = metadataClient.NewMetadataServiceClient(metadataBaseUrl)

	h.logger = logger

	h.awsPlg = awsSteampipe.Plugin()
	h.azurePlg = azureSteampipe.Plugin()
	h.azureADPlg = azureSteampipe.ADPlugin()

	return h, nil
}
