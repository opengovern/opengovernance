package compliance

import (
	"context"
	"fmt"
	helmv2 "github.com/fluxcd/helm-controller/api/v2beta1"
	metadataClient "github.com/opengovern/opengovernance/pkg/metadata/client"
	"github.com/opengovern/opengovernance/services/migrator/db/model"
	"github.com/sashabaranov/go-openai"
	v1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/opengovern/og-util/pkg/postgres"
	describeClient "github.com/opengovern/opengovernance/pkg/describe/client"
	inventoryClient "github.com/opengovern/opengovernance/pkg/inventory/client"
	onboardClient "github.com/opengovern/opengovernance/pkg/onboard/client"

	"github.com/opengovern/og-util/pkg/opengovernance-es-sdk"
	"github.com/opengovern/opengovernance/pkg/compliance/db"

	"go.uber.org/zap"
)

type HttpHandler struct {
	conf       ServerConfig
	client     opengovernance.Client
	db         db.Database
	migratorDb db.Database
	logger     *zap.Logger

	//s3Client *s3.Client

	schedulerClient describeClient.SchedulerServiceClient
	onboardClient   onboardClient.OnboardServiceClient
	inventoryClient inventoryClient.InventoryServiceClient
	metadataClient  metadataClient.MetadataServiceClient
	openAIClient    *openai.Client
	kubeClient      client.Client
}

func NewKubeClient() (client.Client, error) {
	scheme := runtime.NewScheme()
	if err := helmv2.AddToScheme(scheme); err != nil {
		return nil, err
	}
	if err := corev1.AddToScheme(scheme); err != nil {
		return nil, err
	}
	if err := v1.AddToScheme(scheme); err != nil {
		return nil, err
	}
	kubeClient, err := client.New(ctrl.GetConfigOrDie(), client.Options{Scheme: scheme})
	if err != nil {
		return nil, err
	}
	return kubeClient, nil
}

func InitializeHttpHandler(
	ctx context.Context,
	conf ServerConfig,
	//s3Region, s3AccessKey, s3AccessSecret string,
	logger *zap.Logger) (h *HttpHandler, err error) {
	h = &HttpHandler{
		conf:   conf,
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

	migratorDbCfg := postgres.Config{
		Host:    conf.PostgreSQL.Host,
		Port:    conf.PostgreSQL.Port,
		User:    conf.PostgreSQL.Username,
		Passwd:  conf.PostgreSQL.Password,
		DB:      "migrator",
		SSLMode: conf.PostgreSQL.SSLMode,
	}
	migratorOrm, err := postgres.NewClient(&migratorDbCfg, logger)
	if err != nil {
		return nil, fmt.Errorf("new postgres client: %w", err)
	}
	if err := migratorOrm.AutoMigrate(&model.Migration{}); err != nil {
		return nil, fmt.Errorf("gorm migrate: %w", err)
	}
	h.migratorDb = db.Database{Orm: migratorOrm}

	err = h.db.Initialize(ctx)
	if err != nil {
		return nil, err
	}
	fmt.Println("Initialized postgres database: ", conf.PostgreSQL.DB)

	h.client, err = opengovernance.NewClient(opengovernance.ClientConfig{
		Addresses:     []string{conf.ElasticSearch.Address},
		Username:      &conf.ElasticSearch.Username,
		Password:      &conf.ElasticSearch.Password,
		IsOnAks:       &conf.ElasticSearch.IsOnAks,
		IsOpenSearch:  &conf.ElasticSearch.IsOpenSearch,
		AwsRegion:     &conf.ElasticSearch.AwsRegion,
		AssumeRoleArn: &conf.ElasticSearch.AssumeRoleArn,
	})
	if err != nil {
		return nil, err
	}

	//if s3Region == "" {
	//	s3Region = "us-west-2"
	//}
	//var awsConfig aws.Config
	//if s3AccessKey == "" || s3AccessSecret == "" {
	//	//load default credentials
	//	awsConfig = aws.Config{
	//		Region: s3Region,
	//	}
	//} else {
	//	awsConfig = aws.Config{
	//		Region:      s3Region,
	//		Credentials: credentials.NewStaticCredentialsProvider(s3AccessKey, s3AccessSecret, ""),
	//	}
	//}
	//h.s3Client = s3.NewFromConfig(awsConfig)

	h.schedulerClient = describeClient.NewSchedulerServiceClient(conf.Scheduler.BaseURL)
	h.onboardClient = onboardClient.NewOnboardServiceClient(conf.Onboard.BaseURL)
	h.inventoryClient = inventoryClient.NewInventoryServiceClient(conf.Inventory.BaseURL)
	h.metadataClient = metadataClient.NewMetadataServiceClient(conf.Metadata.BaseURL)
	h.openAIClient = openai.NewClient(conf.OpenAI.Token)

	kubeClient, err := NewKubeClient()
	if err != nil {
		return nil, fmt.Errorf("new kube client: %w", err)
	}
	h.kubeClient = kubeClient

	return h, nil
}
