package inventory

import (
	"fmt"

	"github.com/opengovern/og-util/pkg/opengovernance-es-sdk"
	integrationClient "github.com/opengovern/opengovernance/services/integration/client"
	metadataClient "github.com/opengovern/opengovernance/services/metadata/client"

	"github.com/opengovern/og-util/pkg/config"
	"github.com/opengovern/og-util/pkg/postgres"
	"github.com/opengovern/og-util/pkg/steampipe"
	describeClient "github.com/opengovern/opengovernance/pkg/describe/client"
	complianceClient "github.com/opengovern/opengovernance/services/compliance/client"
	"go.uber.org/zap"
)

type HttpHandler struct {
	client            opengovernance.Client
	db                Database
	steampipeConn     *steampipe.Database
	schedulerClient   describeClient.SchedulerServiceClient
	integrationClient integrationClient.IntegrationServiceClient
	complianceClient  complianceClient.ComplianceServiceClient
	metadataClient    metadataClient.MetadataServiceClient

	logger *zap.Logger
}

func InitializeHttpHandler(
	esConf config.ElasticSearch,
	postgresHost string, postgresPort string, postgresDb string, postgresUsername string, postgresPassword string, postgresSSLMode string,
	steampipeHost string, steampipePort string, steampipeDb string, steampipeUsername string, steampipePassword string,
	schedulerBaseUrl string, integrationBaseUrl string, complianceBaseUrl string, metadataBaseUrl string,
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
	h.schedulerClient = describeClient.NewSchedulerServiceClient(schedulerBaseUrl)

	h.integrationClient = integrationClient.NewIntegrationServiceClient(integrationBaseUrl)
	h.complianceClient = complianceClient.NewComplianceClient(complianceBaseUrl)
	h.metadataClient = metadataClient.NewMetadataServiceClient(metadataBaseUrl)

	h.logger = logger

	return h, nil
}
