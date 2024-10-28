package inventory

import (
	"fmt"
	"github.com/opengovern/og-util/pkg/opengovernance-es-sdk"
	metadataClient "github.com/opengovern/opengovernance/pkg/metadata/client"

	opengovernanceAws "github.com/opengovern/og-aws-describer/pkg/opengovernance-es-sdk"
	awsSteampipe "github.com/opengovern/og-aws-describer/pkg/steampipe"
	opengovernanceAzure "github.com/opengovern/og-azure-describer/pkg/opengovernance-es-sdk"
	azureSteampipe "github.com/opengovern/og-azure-describer/pkg/steampipe"
	"github.com/opengovern/og-util/pkg/config"
	"github.com/opengovern/og-util/pkg/postgres"
	"github.com/opengovern/og-util/pkg/steampipe"
	complianceClient "github.com/opengovern/opengovernance/pkg/compliance/client"
	describeClient "github.com/opengovern/opengovernance/pkg/describe/client"
	onboardClient "github.com/opengovern/opengovernance/pkg/onboard/client"
	"github.com/turbot/steampipe-plugin-sdk/v5/plugin"
	"go.uber.org/zap"
)

type HttpHandler struct {
	client           opengovernance.Client
	awsClient        opengovernanceAws.Client
	azureClient      opengovernanceAzure.Client
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

	h.client, err = opengovernance.NewClient(opengovernance.ClientConfig{
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
	h.awsClient = opengovernanceAws.Client{
		Client: h.client,
	}
	h.azureClient = opengovernanceAzure.Client{
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
