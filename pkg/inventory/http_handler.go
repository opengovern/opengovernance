package inventory

import (
	"fmt"
	"time"

	awsSteampipe "github.com/kaytu-io/kaytu-aws-describer/pkg/steampipe"
	azureSteampipe "github.com/kaytu-io/kaytu-azure-describer/pkg/steampipe"
	"github.com/kaytu-io/kaytu-util/pkg/neo4j"
	"github.com/kaytu-io/kaytu-util/pkg/postgres"
	"github.com/turbot/steampipe-plugin-sdk/v4/plugin"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/go-redis/cache/v8"

	complianceClient "gitlab.com/keibiengine/keibi-engine/pkg/compliance/client"
	onboardClient "gitlab.com/keibiengine/keibi-engine/pkg/onboard/client"

	"github.com/go-redis/redis/v8"

	"github.com/kaytu-io/kaytu-util/pkg/steampipe"
	describeClient "gitlab.com/keibiengine/keibi-engine/pkg/describe/client"

	keibiaws "github.com/kaytu-io/kaytu-aws-describer/pkg/keibi-es-sdk"
	keibiazure "github.com/kaytu-io/kaytu-azure-describer/pkg/keibi-es-sdk"
	"github.com/kaytu-io/kaytu-util/pkg/keibi-es-sdk"
	"go.uber.org/zap"
)

type HttpHandler struct {
	client           keibi.Client
	awsClient        keibiaws.Client
	azureClient      keibiazure.Client
	db               Database
	graphDb          GraphDatabase
	steampipeConn    *steampipe.Database
	schedulerClient  describeClient.SchedulerServiceClient
	onboardClient    onboardClient.OnboardServiceClient
	complianceClient complianceClient.ComplianceServiceClient
	rdb              *redis.Client
	cache            *cache.Cache
	s3Downloader     *s3manager.Downloader
	s3Bucket         string

	logger *zap.Logger

	awsPlg, azurePlg, azureADPlg *plugin.Plugin
}

func InitializeHttpHandler(
	elasticSearchAddress string, elasticSearchUsername string, elasticSearchPassword string,
	postgresHost string, postgresPort string, postgresDb string, postgresUsername string, postgresPassword string, postgresSSLMode string,
	neo4jHost string, neo4jPort string, neo4jUsername string, neo4jPassword string,
	steampipeHost string, steampipePort string, steampipeDb string, steampipeUsername string, steampipePassword string,
	schedulerBaseUrl string, onboardBaseUrl string, complianceBaseUrl string,
	logger *zap.Logger,
	redisAddress string,
	s3Endpoint, s3AccessKey, s3AccessSecret, s3Region, s3Bucket string,
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

	neo4jCfg := neo4j.Config{
		Host:   neo4jHost,
		Port:   neo4jPort,
		User:   neo4jUsername,
		Passwd: neo4jPassword,
	}
	driver, err := neo4j.NewDriver(&neo4jCfg, logger)
	if err != nil {
		return nil, fmt.Errorf("new neo4j driver: %w", err)
	}
	h.graphDb, err = NewGraphDatabase(driver)
	if err != nil {
		return nil, fmt.Errorf("new graph database: %w", err)
	}
	fmt.Println("Connected to the neo4j database")

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
	h.awsClient = keibiaws.Client{
		Client: h.client,
	}
	h.azureClient = keibiazure.Client{
		Client: h.client,
	}
	h.schedulerClient = describeClient.NewSchedulerServiceClient(schedulerBaseUrl)

	h.rdb = redis.NewClient(&redis.Options{
		Addr:     redisAddress,
		Password: "", // no password set
		DB:       0,  // use default DB
	})
	h.cache = cache.New(&cache.Options{
		Redis:      h.rdb,
		LocalCache: cache.NewTinyLFU(100000, 5*time.Minute),
	})
	h.onboardClient = onboardClient.NewOnboardServiceClient(onboardBaseUrl, h.cache)
	h.complianceClient = complianceClient.NewComplianceClient(complianceBaseUrl)

	if s3Region == "" {
		s3Region = "us-west-2"
	}
	var awsConfig *aws.Config
	if s3AccessKey == "" || s3AccessSecret == "" {
		//load default credentials
		awsConfig = &aws.Config{
			Region: aws.String(s3Region),
		}
	} else {
		awsConfig = &aws.Config{
			Endpoint:    aws.String(s3Endpoint),
			Region:      aws.String(s3Region),
			Credentials: credentials.NewStaticCredentials(s3AccessKey, s3AccessSecret, ""),
		}
	}
	sess := session.Must(session.NewSession(awsConfig))
	h.s3Downloader = s3manager.NewDownloader(sess)
	h.s3Bucket = s3Bucket

	h.logger = logger

	h.awsPlg = awsSteampipe.Plugin()
	h.azurePlg = azureSteampipe.Plugin()
	h.azureADPlg = azureSteampipe.ADPlugin()
	return h, nil
}
