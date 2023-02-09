package describe

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azeventhubs"
	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azeventhubs/checkpoints"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/bloberror"
	"github.com/ProtonMail/gopenpgp/v2/crypto"
	"github.com/google/uuid"
	"github.com/hashicorp/vault/api/auth/kubernetes"
	"gitlab.com/keibiengine/keibi-engine/pkg/checkup"
	checkupapi "gitlab.com/keibiengine/keibi-engine/pkg/checkup/api"
	"gitlab.com/keibiengine/keibi-engine/pkg/compliance/client"
	"gitlab.com/keibiengine/keibi-engine/pkg/describe/enums"
	"gitlab.com/keibiengine/keibi-engine/pkg/internal/vault"
	"gitlab.com/keibiengine/keibi-engine/pkg/summarizer"
	summarizerapi "gitlab.com/keibiengine/keibi-engine/pkg/summarizer/api"
	"gopkg.in/Shopify/sarama.v1"
	"gorm.io/gorm"

	"gitlab.com/keibiengine/keibi-engine/pkg/azure"

	"github.com/go-redis/redis/v8"
	api2 "gitlab.com/keibiengine/keibi-engine/pkg/auth/api"
	"gitlab.com/keibiengine/keibi-engine/pkg/internal/httpclient"
	"gitlab.com/keibiengine/keibi-engine/pkg/keibi-es-sdk"
	onboardClient "gitlab.com/keibiengine/keibi-engine/pkg/onboard/client"
	workspaceClient "gitlab.com/keibiengine/keibi-engine/pkg/workspace/client"

	"gitlab.com/keibiengine/keibi-engine/pkg/insight"
	insightapi "gitlab.com/keibiengine/keibi-engine/pkg/insight/api"

	"gitlab.com/keibiengine/keibi-engine/pkg/source"

	complianceapi "gitlab.com/keibiengine/keibi-engine/pkg/compliance/api"

	"gitlab.com/keibiengine/keibi-engine/pkg/aws"
	compliancereport "gitlab.com/keibiengine/keibi-engine/pkg/compliance"
	"gitlab.com/keibiengine/keibi-engine/pkg/describe/api"
	"gitlab.com/keibiengine/keibi-engine/pkg/internal/httpserver"
	"gitlab.com/keibiengine/keibi-engine/pkg/internal/postgres"
	"gitlab.com/keibiengine/keibi-engine/pkg/internal/queue"
	"go.uber.org/zap"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

const (
	JobCompletionInterval    = 1 * time.Minute
	JobSchedulingInterval    = 1 * time.Minute
	JobTimeoutCheckInterval  = 15 * time.Minute
	MaxJobInQueue            = 10000
	ConcurrentDeletedSources = 1000

	RedisKeyWorkspaceResourceRemaining = "workspace_resource_remaining"
)

var DescribePublishingBlocked = promauto.NewGaugeVec(prometheus.GaugeOpts{
	Namespace: "keibi",
	Subsystem: "scheduler",
	Name:      "queue_job_publishing_blocked",
	Help:      "The gauge whether publishing tasks to a queue is blocked: 0 for resumed and 1 for blocked",
}, []string{"queue_name"})

var InsightJobsCount = promauto.NewCounterVec(prometheus.CounterOpts{
	Namespace: "keibi",
	Subsystem: "scheduler",
	Name:      "schedule_insight_jobs_total",
	Help:      "Count of insight jobs in scheduler service",
}, []string{"status"})

var CheckupJobsCount = promauto.NewCounterVec(prometheus.CounterOpts{
	Namespace: "keibi",
	Subsystem: "scheduler",
	Name:      "schedule_checkup_jobs_total",
	Help:      "Count of checkup jobs in scheduler service",
}, []string{"status"})

var SummarizerJobsCount = promauto.NewCounterVec(prometheus.CounterOpts{
	Namespace: "keibi",
	Subsystem: "scheduler",
	Name:      "schedule_summarizer_jobs_total",
	Help:      "Count of summarizer jobs in scheduler service",
}, []string{"status"})

var DescribeJobsCount = promauto.NewCounterVec(prometheus.CounterOpts{
	Namespace: "keibi",
	Subsystem: "scheduler",
	Name:      "schedule_describe_jobs_total",
	Help:      "Count of describe jobs in scheduler service",
}, []string{"status"})

var DescribeSourceJobsCount = promauto.NewCounterVec(prometheus.CounterOpts{
	Name: "keibi_scheduler_schedule_describe_source_jobs_total",
	Help: "Count of describe source jobs in scheduler service",
}, []string{"status"})

var DescribeCleanupJobsCount = promauto.NewCounterVec(prometheus.CounterOpts{
	Name: "keibi_scheduler_schedule_describe_cleanup_jobs_total",
	Help: "Count of describe jobs in scheduler service",
}, []string{"status"})

var DescribeCleanupSourceJobsCount = promauto.NewCounterVec(prometheus.CounterOpts{
	Name: "keibi_scheduler_schedule_describe_cleanup_source_jobs_total",
	Help: "Count of describe source jobs in scheduler service",
}, []string{"status"})

var ComplianceJobsCount = promauto.NewCounterVec(prometheus.CounterOpts{
	Name: "keibi_scheduler_schedule_compliance_job_total",
	Help: "Count of describe jobs in scheduler service",
}, []string{"status"})

var ComplianceSourceJobsCount = promauto.NewCounterVec(prometheus.CounterOpts{
	Name: "keibi_scheduler_schedule_compliance_source_job_total",
	Help: "Count of describe source jobs in scheduler service",
}, []string{"status"})

type Scheduler struct {
	id         string
	db         Database
	httpServer *HttpServer

	// describeJobQueue is used to publish describe jobs to be performed by the workers.
	describeJobQueue queue.Interface
	// describeJobResultQueue is used to consume the describe job results returned by the workers.
	describeJobResultQueue queue.Interface
	// describeCleanupJobQueue is used to publish describe cleanup jobs to be performed by the workers.
	describeCleanupJobQueue queue.Interface
	// describeConnectionJobQueue is used to publish describe jobs to be performed by the workers.
	describeConnectionJobQueue queue.Interface
	// describeConnectionJobResultQueue is used to consume the describe job results returned by the workers.
	describeConnectionJobResultQueue queue.Interface

	cloudNativeDescribeConnectionJobQueue          *azeventhubs.ProducerClient
	cloudNativeDescribeConnectionJobResultQueue    *azeventhubs.Processor
	cloudNativeDescribeConnectionJobResourcesQueue *azeventhubs.Processor

	// sourceQueue is used to consume source updates by the onboarding service.
	sourceQueue queue.Interface

	complianceReportJobQueue        queue.Interface
	complianceReportJobResultQueue  queue.Interface
	complianceReportCleanupJobQueue queue.Interface

	// insightJobQueue is used to publish insight jobs to be performed by the workers.
	insightJobQueue queue.Interface
	// insightJobResultQueue is used to consume the insight job results returned by the workers.
	insightJobResultQueue queue.Interface

	// checkupJobQueue is used to publish checkup jobs to be performed by the workers.
	checkupJobQueue queue.Interface
	// checkupJobResultQueue is used to consume the checkup job results returned by the workers.
	checkupJobResultQueue queue.Interface

	// summarizerJobQueue is used to publish summarizer jobs to be performed by the workers.
	summarizerJobQueue queue.Interface
	// summarizerJobResultQueue is used to consume the summarizer job results returned by the workers.
	summarizerJobResultQueue queue.Interface

	// watch the deleted source
	deletedSources chan string

	describeIntervalHours   int64
	complianceIntervalHours int64
	insightIntervalHours    int64
	checkupIntervalHours    int64
	summarizerIntervalHours int64

	logger              *zap.Logger
	workspaceClient     workspaceClient.WorkspaceServiceClient
	complianceClient    client.ComplianceServiceClient
	onboardClient       onboardClient.OnboardServiceClient
	es                  keibi.Client
	rdb                 *redis.Client
	vault               vault.SourceConfig
	azblobClient        *azblob.Client
	kafkaClient         sarama.Client
	kafkaResourcesTopic string
}

func InitializeScheduler(
	id string,
	rabbitMQUsername string,
	rabbitMQPassword string,
	rabbitMQHost string,
	rabbitMQPort int,
	describeJobQueueName string,
	describeJobResultQueueName string,
	describeConnectionJobQueueName string,
	describeConnectionJobResultQueueName string,
	cloudNativeConnectionJobTriggerEventHubConnectionString string,
	cloudNativeConnectionJobTriggerEventHubName string,
	cloudNativeConnectionJobOutputEventHubConnectionString string,
	cloudNativeConnectionJobOutputEventHubName string,
	cloudNativeConnectionJobResourcesEventHubName string,
	cloudNativeConnectionJobOutputCheckpointContainerName string,
	cloudNativeConnectionJobBlobStorageConnectionString string,
	describeCleanupJobQueueName string,
	complianceReportJobQueueName string,
	complianceReportJobResultQueueName string,
	complianceReportCleanupJobQueueName string,
	insightJobQueueName string,
	insightJobResultQueueName string,
	checkupJobQueueName string,
	checkupJobResultQueueName string,
	summarizerJobQueueName string,
	summarizerJobResultQueueName string,
	sourceQueueName string,
	postgresUsername string,
	postgresPassword string,
	postgresHost string,
	postgresPort string,
	postgresDb string,
	postgresSSLMode string,
	vaultAddress string,
	vaultRoleName string,
	vaultToken string,
	vaultCaPath string,
	vaultUseTLS bool,
	httpServerAddress string,
	describeIntervalHours string,
	complianceIntervalHours string,
	insightIntervalHours string,
	checkupIntervalHours string,
) (s *Scheduler, err error) {
	if id == "" {
		return nil, fmt.Errorf("'id' must be set to a non empty string")
	}

	s = &Scheduler{
		id:             id,
		deletedSources: make(chan string, ConcurrentDeletedSources),
	}
	defer func() {
		if err != nil && s != nil {
			s.Stop()
		}
	}()

	s.logger, err = zap.NewProduction()
	if err != nil {
		return nil, err
	}

	s.logger.Info("Initializing the scheduler")

	qCfg := queue.Config{}
	qCfg.Server.Username = rabbitMQUsername
	qCfg.Server.Password = rabbitMQPassword
	qCfg.Server.Host = rabbitMQHost
	qCfg.Server.Port = rabbitMQPort
	qCfg.Queue.Name = describeJobQueueName
	qCfg.Queue.Durable = true
	qCfg.Producer.ID = s.id
	describeQueue, err := queue.New(qCfg)
	if err != nil {
		return nil, err
	}

	s.logger.Info("Connected to the describe jobs queue", zap.String("queue", describeJobQueueName))
	s.describeJobQueue = describeQueue

	qCfg = queue.Config{}
	qCfg.Server.Username = rabbitMQUsername
	qCfg.Server.Password = rabbitMQPassword
	qCfg.Server.Host = rabbitMQHost
	qCfg.Server.Port = rabbitMQPort
	qCfg.Queue.Name = describeJobResultQueueName
	qCfg.Queue.Durable = true
	qCfg.Consumer.ID = s.id
	describeResultsQueue, err := queue.New(qCfg)
	if err != nil {
		return nil, err
	}

	s.logger.Info("Connected to the describe job results queue", zap.String("queue", describeJobResultQueueName))
	s.describeJobResultQueue = describeResultsQueue

	qCfg = queue.Config{}
	qCfg.Server.Username = rabbitMQUsername
	qCfg.Server.Password = rabbitMQPassword
	qCfg.Server.Host = rabbitMQHost
	qCfg.Server.Port = rabbitMQPort
	qCfg.Queue.Name = describeConnectionJobQueueName
	qCfg.Queue.Durable = true
	qCfg.Producer.ID = s.id
	describeConnectionQueue, err := queue.New(qCfg)
	if err != nil {
		return nil, err
	}

	s.logger.Info("Connected to the describe jobs queue", zap.String("queue", describeConnectionJobQueueName))
	s.describeConnectionJobQueue = describeConnectionQueue

	qCfg = queue.Config{}
	qCfg.Server.Username = rabbitMQUsername
	qCfg.Server.Password = rabbitMQPassword
	qCfg.Server.Host = rabbitMQHost
	qCfg.Server.Port = rabbitMQPort
	qCfg.Queue.Name = describeConnectionJobResultQueueName
	qCfg.Queue.Durable = true
	qCfg.Consumer.ID = s.id
	describeConnectionResultsQueue, err := queue.New(qCfg)
	if err != nil {
		return nil, err
	}

	s.logger.Info("Connected to the describe job results queue", zap.String("queue", describeConnectionJobResultQueueName))
	s.describeConnectionJobResultQueue = describeConnectionResultsQueue

	qCfg = queue.Config{}
	qCfg.Server.Username = rabbitMQUsername
	qCfg.Server.Password = rabbitMQPassword
	qCfg.Server.Host = rabbitMQHost
	qCfg.Server.Port = rabbitMQPort
	qCfg.Queue.Name = insightJobQueueName
	qCfg.Queue.Durable = true
	qCfg.Producer.ID = s.id
	insightQueue, err := queue.New(qCfg)
	if err != nil {
		return nil, err
	}

	s.logger.Info("Connected to the insight jobs queue", zap.String("queue", insightJobQueueName))
	s.insightJobQueue = insightQueue

	qCfg = queue.Config{}
	qCfg.Server.Username = rabbitMQUsername
	qCfg.Server.Password = rabbitMQPassword
	qCfg.Server.Host = rabbitMQHost
	qCfg.Server.Port = rabbitMQPort
	qCfg.Queue.Name = insightJobResultQueueName
	qCfg.Queue.Durable = true
	qCfg.Consumer.ID = s.id
	insightResultsQueue, err := queue.New(qCfg)
	if err != nil {
		return nil, err
	}

	s.logger.Info("Connected to the insight job results queue", zap.String("queue", insightJobResultQueueName))
	s.insightJobResultQueue = insightResultsQueue

	qCfg = queue.Config{}
	qCfg.Server.Username = rabbitMQUsername
	qCfg.Server.Password = rabbitMQPassword
	qCfg.Server.Host = rabbitMQHost
	qCfg.Server.Port = rabbitMQPort
	qCfg.Queue.Name = checkupJobQueueName
	qCfg.Queue.Durable = true
	qCfg.Producer.ID = s.id
	checkupQueue, err := queue.New(qCfg)
	if err != nil {
		return nil, err
	}

	s.logger.Info("Connected to the checkup jobs queue", zap.String("queue", checkupJobQueueName))
	s.checkupJobQueue = checkupQueue

	qCfg = queue.Config{}
	qCfg.Server.Username = rabbitMQUsername
	qCfg.Server.Password = rabbitMQPassword
	qCfg.Server.Host = rabbitMQHost
	qCfg.Server.Port = rabbitMQPort
	qCfg.Queue.Name = checkupJobResultQueueName
	qCfg.Queue.Durable = true
	qCfg.Consumer.ID = s.id
	checkupResultsQueue, err := queue.New(qCfg)
	if err != nil {
		return nil, err
	}

	s.logger.Info("Connected to the checkup job results queue", zap.String("queue", checkupJobResultQueueName))
	s.checkupJobResultQueue = checkupResultsQueue

	qCfg = queue.Config{}
	qCfg.Server.Username = rabbitMQUsername
	qCfg.Server.Password = rabbitMQPassword
	qCfg.Server.Host = rabbitMQHost
	qCfg.Server.Port = rabbitMQPort
	qCfg.Queue.Name = summarizerJobQueueName
	qCfg.Queue.Durable = true
	qCfg.Producer.ID = s.id
	summarizerQueue, err := queue.New(qCfg)
	if err != nil {
		return nil, err
	}

	s.logger.Info("Connected to the summarizer jobs queue", zap.String("queue", summarizerJobQueueName))
	s.summarizerJobQueue = summarizerQueue

	qCfg = queue.Config{}
	qCfg.Server.Username = rabbitMQUsername
	qCfg.Server.Password = rabbitMQPassword
	qCfg.Server.Host = rabbitMQHost
	qCfg.Server.Port = rabbitMQPort
	qCfg.Queue.Name = summarizerJobResultQueueName
	qCfg.Queue.Durable = true
	qCfg.Consumer.ID = s.id
	summarizerResultsQueue, err := queue.New(qCfg)
	if err != nil {
		return nil, err
	}

	s.logger.Info("Connected to the summarizer job results queue", zap.String("queue", summarizerJobResultQueueName))
	s.summarizerJobResultQueue = summarizerResultsQueue

	qCfg = queue.Config{}
	qCfg.Server.Username = rabbitMQUsername
	qCfg.Server.Password = rabbitMQPassword
	qCfg.Server.Host = rabbitMQHost
	qCfg.Server.Port = rabbitMQPort
	qCfg.Queue.Name = describeCleanupJobQueueName
	qCfg.Queue.Durable = true
	qCfg.Producer.ID = s.id
	describeCleanupJobQueue, err := queue.New(qCfg)
	if err != nil {
		return nil, err
	}

	s.logger.Info("Connected to the describe cleanup job queue", zap.String("queue", describeCleanupJobQueueName))
	s.describeCleanupJobQueue = describeCleanupJobQueue

	qCfg = queue.Config{}
	qCfg.Server.Username = rabbitMQUsername
	qCfg.Server.Password = rabbitMQPassword
	qCfg.Server.Host = rabbitMQHost
	qCfg.Server.Port = rabbitMQPort
	qCfg.Queue.Name = complianceReportCleanupJobQueueName
	qCfg.Queue.Durable = true
	qCfg.Producer.ID = s.id
	complianceReportCleanupJobQueue, err := queue.New(qCfg)
	if err != nil {
		return nil, err
	}

	s.logger.Info("Connected to the complianceReport cleanup job queue", zap.String("queue", complianceReportCleanupJobQueueName))
	s.complianceReportCleanupJobQueue = complianceReportCleanupJobQueue

	qCfg = queue.Config{}
	qCfg.Server.Username = rabbitMQUsername
	qCfg.Server.Password = rabbitMQPassword
	qCfg.Server.Host = rabbitMQHost
	qCfg.Server.Port = rabbitMQPort
	qCfg.Queue.Name = sourceQueueName
	qCfg.Queue.Durable = true
	qCfg.Consumer.ID = s.id
	sourceEventsQueue, err := queue.New(qCfg)
	if err != nil {
		return nil, err
	}

	s.logger.Info("Connected to the source events queue", zap.String("queue", sourceQueueName))
	s.sourceQueue = sourceEventsQueue

	qCfg = queue.Config{}
	qCfg.Server.Username = rabbitMQUsername
	qCfg.Server.Password = rabbitMQPassword
	qCfg.Server.Host = rabbitMQHost
	qCfg.Server.Port = rabbitMQPort
	qCfg.Queue.Name = complianceReportJobQueueName
	qCfg.Queue.Durable = true
	qCfg.Producer.ID = s.id
	complianceReportJobsQueue, err := queue.New(qCfg)
	if err != nil {
		return nil, err
	}

	s.logger.Info("Connected to the compliance report jobs queue", zap.String("queue", complianceReportJobQueueName))
	s.complianceReportJobQueue = complianceReportJobsQueue

	qCfg = queue.Config{}
	qCfg.Server.Username = rabbitMQUsername
	qCfg.Server.Password = rabbitMQPassword
	qCfg.Server.Host = rabbitMQHost
	qCfg.Server.Port = rabbitMQPort
	qCfg.Queue.Name = complianceReportJobResultQueueName
	qCfg.Queue.Durable = true
	qCfg.Consumer.ID = s.id
	complianceReportJobsResultQueue, err := queue.New(qCfg)
	if err != nil {
		return nil, err
	}

	s.logger.Info("Connected to the compliance report jobs result queue", zap.String("queue", complianceReportJobResultQueueName))
	s.complianceReportJobResultQueue = complianceReportJobsResultQueue

	cfg := postgres.Config{
		Host:    postgresHost,
		Port:    postgresPort,
		User:    postgresUsername,
		Passwd:  postgresPassword,
		DB:      postgresDb,
		SSLMode: postgresSSLMode,
	}
	orm, err := postgres.NewClient(&cfg, s.logger)
	if err != nil {
		return nil, fmt.Errorf("new postgres client: %w", err)
	}

	s.logger.Info("Connected to the postgres database: ", zap.String("db", postgresDb))
	s.db = Database{orm: orm}

	azblobClient, err := azblob.NewClientFromConnectionString(cloudNativeConnectionJobBlobStorageConnectionString, nil)
	if err != nil {
		return nil, fmt.Errorf("new azblob client: %w", err)
	}
	s.azblobClient = azblobClient
	s.logger.Info("Connected to the cloud native connection job blob storage")

	producerClient, err := azeventhubs.NewProducerClientFromConnectionString(cloudNativeConnectionJobTriggerEventHubConnectionString, cloudNativeConnectionJobTriggerEventHubName, nil)
	if err != nil {
		return nil, err
	}
	s.cloudNativeDescribeConnectionJobQueue = producerClient
	s.logger.Info("Connected to the cloud native describe connection job queue", zap.String("queue", cloudNativeConnectionJobTriggerEventHubName))

	_, err = azblobClient.CreateContainer(context.Background(), cloudNativeConnectionJobOutputCheckpointContainerName, nil)
	if err != nil && !bloberror.HasCode(err, bloberror.ContainerAlreadyExists) {
		return nil, err
	}
	checkpointContainerClient := azblobClient.ServiceClient().NewContainerClient(cloudNativeConnectionJobOutputCheckpointContainerName)
	checkpointStore, err := checkpoints.NewBlobStore(checkpointContainerClient, nil)
	if err != nil {
		return nil, err
	}
	triggerOutputConsumerClient, err := azeventhubs.NewConsumerClientFromConnectionString(cloudNativeConnectionJobOutputEventHubConnectionString, cloudNativeConnectionJobOutputEventHubName, azeventhubs.DefaultConsumerGroup, nil)
	if err != nil {
		return nil, err
	}
	triggerOutputProcessor, err := azeventhubs.NewProcessor(triggerOutputConsumerClient, checkpointStore, nil)
	if err != nil {
		return nil, err
	}
	s.cloudNativeDescribeConnectionJobResultQueue = triggerOutputProcessor
	s.logger.Info("Connected to the cloud native describe connection job result queue", zap.String("queue", cloudNativeConnectionJobOutputEventHubName))

	resourcesConsumerClient, err := azeventhubs.NewConsumerClientFromConnectionString(cloudNativeConnectionJobOutputEventHubConnectionString, cloudNativeConnectionJobResourcesEventHubName, azeventhubs.DefaultConsumerGroup, nil)
	if err != nil {
		return nil, err
	}
	resourcesProcessor, err := azeventhubs.NewProcessor(resourcesConsumerClient, checkpointStore, nil)
	if err != nil {
		return nil, err
	}
	s.cloudNativeDescribeConnectionJobResourcesQueue = resourcesProcessor

	kafkaClient, err := newKafkaClient(strings.Split(KafkaService, ","))
	if err != nil {
		return nil, err
	}
	s.kafkaClient = kafkaClient
	s.kafkaResourcesTopic = KafkaResourcesTopic

	s.httpServer = NewHTTPServer(httpServerAddress, s.db, s)
	s.describeIntervalHours, err = strconv.ParseInt(describeIntervalHours, 10, 64)
	if err != nil {
		return nil, err
	}
	s.complianceIntervalHours, err = strconv.ParseInt(complianceIntervalHours, 10, 64)
	if err != nil {
		return nil, err
	}
	s.insightIntervalHours, err = strconv.ParseInt(insightIntervalHours, 10, 64)
	if err != nil {
		return nil, err
	}
	s.checkupIntervalHours, err = strconv.ParseInt(checkupIntervalHours, 10, 64)
	if err != nil {
		return nil, err
	}

	s.workspaceClient = workspaceClient.NewWorkspaceClient(WorkspaceBaseURL)
	s.complianceClient = client.NewComplianceClient(ComplianceBaseURL)
	s.onboardClient = onboardClient.NewOnboardServiceClient(OnboardBaseURL, nil)
	defaultAccountID := "default"
	s.es, err = keibi.NewClient(keibi.ClientConfig{
		Addresses: []string{ElasticSearchAddress},
		Username:  &ElasticSearchUsername,
		Password:  &ElasticSearchPassword,
		AccountID: &defaultAccountID,
	})
	s.rdb = redis.NewClient(&redis.Options{
		Addr:     RedisAddress,
		Password: "", // no password set
		DB:       0,  // use default DB
	})

	k8sAuth, err := kubernetes.NewKubernetesAuth(
		vaultRoleName,
		kubernetes.WithServiceAccountToken(vaultToken),
	)
	if err != nil {
		return nil, err
	}
	// setup vault
	v, err := vault.NewSourceConfig(vaultAddress, vaultCaPath, k8sAuth, vaultUseTLS)
	if err != nil {
		return nil, err
	}
	s.vault = v

	return s, nil
}

func (s *Scheduler) Run() error {
	err := s.db.Initialize()
	if err != nil {
		return err
	}

	go s.RunDescribeJobCompletionUpdater()
	go s.RunScheduleJobCompletionUpdater()
	go s.RunDescribeJobScheduler()
	go s.RunInsightJobScheduler()
	go s.RunCheckupJobScheduler()
	go s.RunDescribeCleanupJobScheduler()
	//go s.RunComplianceReportScheduler()
	go s.RunDeletedSourceCleanup()
	go s.RunCloudNativeDescribeConnectionJobResultConsumer()
	go s.RunCloudNativeDescribeConnectionJobResourcesConsumer()

	// In order to have history of reports, we won't clean up compliance reports for now.
	//go s.RunComplianceReportCleanupJobScheduler()

	go func() {
		s.logger.Fatal("SourceEvent consumer exited", zap.Error(s.RunSourceEventsConsumer()))
	}()

	go func() {
		s.logger.Fatal("DescribeJobResult consumer exited", zap.Error(s.RunDescribeJobResultsConsumer()))
	}()

	go func() {
		s.logger.Fatal("DescribeConnectionJobResult consumer exited", zap.Error(s.RunDescribeConnectionJobResultsConsumer()))
	}()

	go func() {
		s.logger.Fatal("ComplianceReportJobResult consumer exited", zap.Error(s.RunComplianceReportJobResultsConsumer()))
	}()

	go func() {
		s.logger.Fatal("InsightJobResult consumer exited", zap.Error(s.RunInsightJobResultsConsumer()))
	}()

	go func() {
		s.logger.Fatal("InsightJobResult consumer exited", zap.Error(s.RunCheckupJobResultsConsumer()))
	}()

	go func() {
		s.logger.Fatal("SummarizerJobResult consumer exited", zap.Error(s.RunSummarizerJobResultsConsumer()))
	}()

	return httpserver.RegisterAndStart(s.logger, s.httpServer.Address, s.httpServer)
}

func (s *Scheduler) processCloudNativeDescribeConnectionJobResultEvents(partitionClient *azeventhubs.ProcessorPartitionClient) error {
	defer partitionClient.Close(context.Background())
	for {
		receiveCtx, receiveCtxCancel := context.WithTimeout(context.TODO(), 10*time.Second)
		events, err := partitionClient.ReceiveEvents(receiveCtx, 100, nil)
		receiveCtxCancel()

		if err != nil && !errors.Is(err, context.DeadlineExceeded) {
			return err
		}
		if len(events) == 0 {
			continue
		}

		s.logger.Info("Received events from cloud native describe connection job result queue", zap.Int("eventCount", len(events)))
		for _, event := range events {
			var message api.CloudNativeConnectionWorkerTriggerQueueMessage
			err := json.Unmarshal(event.Body, &message)
			if err != nil {
				s.logger.Error("Error unmarshalling event", zap.Error(err))
				continue
			}
			jobId, err := uuid.Parse(message.JobId)
			if err != nil {
				s.logger.Error("Error parsing job id", zap.Error(err))
				continue
			}
			if message.Status == http.StatusAccepted {
				var triggerResponse api.CloudNativeConnectionWorkerTriggerOutput
				err = json.Unmarshal([]byte(message.Body), &triggerResponse)
				if err != nil {
					s.logger.Error("Error unmarshalling trigger response", zap.Error(err))
					continue
				}
				job := CloudNativeDescribeSourceJob{
					JobID:        jobId,
					StatusURI:    &triggerResponse.StatusQueryGetURI,
					TerminateURI: &triggerResponse.TerminatePostURI,
				}
				err = s.db.UpdateCloudNativeDescribeSourceJobURIs(&job)
				if err != nil {
					s.logger.Error("Error updating cloud native describe source job URIs", zap.Error(err))
					continue
				}
			} else {
				cnSourceJob, err := s.db.GetCloudNativeDescribeSourceJob(jobId.String())
				if err != nil {
					s.logger.Error("Error getting cloud native describe source job", zap.Error(err))
					continue
				}
				sourceJob, err := s.db.GetDescribeSourceJob(cnSourceJob.SourceJobID)
				if err != nil {
					s.logger.Error("Error getting describe source job", zap.Error(err))
					continue
				}
				for _, drj := range sourceJob.DescribeResourceJobs {
					if err := s.db.UpdateDescribeResourceJobStatus(drj.ID, api.DescribeResourceJobFailed, message.Body); err != nil {
						s.logger.Error("Failed to update DescribeResourceJob",
							zap.Uint("jobId", drj.ID),
							zap.Error(err),
						)
					}
				}
			}
		}

		if len(events) != 0 {
			if err := partitionClient.UpdateCheckpoint(context.TODO(), events[len(events)-1]); err != nil {
				return err
			}
		}
		s.logger.Info("Processed events from cloud native describe connection job result queue", zap.Int("eventCount", len(events)))
	}
}

func (s *Scheduler) RunCloudNativeDescribeConnectionJobResultConsumer() {
	dispatchPartitionClients := func() {
		for {
			partitionClient := s.cloudNativeDescribeConnectionJobResultQueue.NextPartitionClient(context.TODO())
			if partitionClient == nil {
				break
			}
			go func() {
				if err := s.processCloudNativeDescribeConnectionJobResultEvents(partitionClient); err != nil {
					s.logger.Error("Error processing events", zap.Error(err))
				}
			}()
		}
	}

	go dispatchPartitionClients()

	processorCtx, processorCancel := context.WithCancel(context.TODO())
	defer processorCancel()

	if err := s.cloudNativeDescribeConnectionJobResultQueue.Run(processorCtx); err != nil {
		s.logger.Error("Error running cloud native describe connection job result queue", zap.Error(err))
		return
	}
}

func (s *Scheduler) processCloudNativeDescribeConnectionJobResourcesEvents(partitionClient *azeventhubs.ProcessorPartitionClient) error {
	defer partitionClient.Close(context.Background())
	for {
		receiveCtx, receiveCtxCancel := context.WithTimeout(context.TODO(), time.Minute)
		events, err := partitionClient.ReceiveEvents(receiveCtx, 100, nil)
		receiveCtxCancel()

		if err != nil && !errors.Is(err, context.DeadlineExceeded) {
			return err
		}
		if len(events) == 0 {
			continue
		}

		s.logger.Info("Received events from cloud native describe connection job resources queue", zap.Int("eventCount", len(events)))
		for _, event := range events {
			var connectionWorkerResourcesResult CloudNativeConnectionWorkerResult
			err := json.Unmarshal(event.Body, &connectionWorkerResourcesResult)
			if err != nil {
				s.logger.Error("Error unmarshalling event", zap.Error(err))
				continue
			}

			saramaMessages := make([]*sarama.ProducerMessage, 0, len(connectionWorkerResourcesResult.Resources))
			for _, message := range connectionWorkerResourcesResult.Resources {
				saramaMessages = append(saramaMessages, &sarama.ProducerMessage{
					Topic:   message.Topic,
					Key:     message.Key,
					Value:   message.Value,
					Headers: message.Headers,
				})
			}
			producer, err := sarama.NewSyncProducerFromClient(s.kafkaClient)
			if err != nil {
				s.logger.Error("Failed to create producer", zap.Error(err))
				continue
			}
			if err := producer.SendMessages(saramaMessages); err != nil {
				if errs, ok := err.(sarama.ProducerErrors); ok {
					for _, e := range errs {
						s.logger.Error("Failed calling SendMessages", zap.Error(fmt.Errorf("Failed to persist resource[%s] in kafka topic[%s]: %s\nMessage: %v\n", e.Msg.Key, e.Msg.Topic, e.Error(), e.Msg)))
					}
				}
				continue
			}
			if err := producer.Close(); err != nil {
				s.logger.Error("Failed to close producer", zap.Error(err))
				continue
			}
			if len(saramaMessages) != 0 {
				s.logger.Info("Successfully sent messages to kafka", zap.Int("count", len(saramaMessages)))
			}

			err = s.describeConnectionJobResultQueue.Publish(connectionWorkerResourcesResult.JobResult)
			if err != nil {
				s.logger.Error("Failed calling describeConnectionJobResultQueue.Publish", zap.Error(err))
				continue
			}
		}

		if err := partitionClient.UpdateCheckpoint(context.TODO(), events[len(events)-1]); err != nil {
			return err
		}

		s.logger.Info("Processed events from cloud native describe connection job resources queue", zap.Int("eventCount", len(events)))
	}
}

func (s *Scheduler) RunCloudNativeDescribeConnectionJobResourcesConsumer() {
	dispatchPartitionClients := func() {
		for {
			partitionClient := s.cloudNativeDescribeConnectionJobResourcesQueue.NextPartitionClient(context.TODO())
			if partitionClient == nil {
				break
			}
			go func() {
				if err := s.processCloudNativeDescribeConnectionJobResourcesEvents(partitionClient); err != nil {
					s.logger.Error("Error processing events for ConnectionJobResourcesEvents", zap.Error(err))
				}
			}()
		}
	}

	go dispatchPartitionClients()

	processorCtx, processorCancel := context.WithCancel(context.TODO())
	defer processorCancel()

	if err := s.cloudNativeDescribeConnectionJobResultQueue.Run(processorCtx); err != nil {
		s.logger.Error("Error running cloud native describe connection job resources queue", zap.Error(err))
		return
	}
}

func (s *Scheduler) RunScheduleJobCompletionUpdater() {
	defer func() {
		if r := recover(); r != nil {
			err := fmt.Errorf("paniced during RunScheduleJobCompletionUpdater: %v", r)
			s.logger.Error("Paniced, retry", zap.Error(err))
			go s.RunScheduleJobCompletionUpdater()
		}
	}()

	t := time.NewTicker(JobCompletionInterval)
	defer t.Stop()

	for ; ; <-t.C {
		scheduleJob, err := s.db.FetchLastScheduleJob()
		if err != nil {
			s.logger.Error("Failed to find ScheduleJobs", zap.Error(err))
			continue
		}

		if scheduleJob == nil || scheduleJob.Status != summarizerapi.SummarizerJobInProgress {
			continue
		}

		djs, err := s.db.QueryDescribeSourceJobsForScheduleJob(scheduleJob)
		if err != nil {
			s.logger.Error("Failed to find list of describe source jobs", zap.Error(err))
			continue
		}

		inProgress := false
		for _, j := range djs {
			if j.Status == api.DescribeSourceJobCreated || j.Status == api.DescribeSourceJobInProgress {
				inProgress = true
			}
		}

		if inProgress {
			continue
		}

		srcs, err := s.db.ListSources()
		if err != nil {
			s.logger.Error("Failed to find list of sources", zap.Error(err))
			continue
		}

		inProgress = false
		for _, src := range srcs {
			found := false
			for _, j := range djs {
				if src.ID == j.SourceID {
					found = true
					break
				}
			}

			if !found {
				inProgress = true
				break
			}
		}

		if inProgress {
			continue
		}

		j, err := s.db.GetSummarizerJobByScheduleID(scheduleJob.ID, summarizer.JobType_ResourceSummarizer)
		if err != nil {
			s.logger.Error("Failed to fetch SummarizerJob", zap.Error(err))
			continue
		}

		if j == nil {
			err = s.scheduleSummarizerJob(scheduleJob.ID)
			if err != nil {
				s.logger.Error("Failed to enqueue summarizer job\n",
					zap.Uint("jobId", scheduleJob.ID),
					zap.Error(err),
				)
			}
			continue
		}

		if j.Status == summarizerapi.SummarizerJobInProgress {
			continue
		}

		cjobs, err := s.db.GetComplianceReportJobsByScheduleID(scheduleJob.ID)
		if err != nil {
			s.logger.Error("Failed to get ComplianceJobs", zap.Error(err))
			continue
		}

		if cjobs == nil || len(cjobs) == 0 {
			createdJobCount, err := s.RunComplianceReport(scheduleJob)
			if err != nil {
				s.logger.Error("Failed to enqueue compliance job\n",
					zap.Uint("jobId", scheduleJob.ID),
					zap.Error(err),
				)
			}

			if createdJobCount > 0 {
				continue
			}
		}

		inProgress = false
		for _, j := range cjobs {
			if j.Status == complianceapi.ComplianceReportJobCreated ||
				j.Status == complianceapi.ComplianceReportJobInProgress {
				inProgress = true
			}
		}

		if inProgress {
			continue
		}

		j, err = s.db.GetSummarizerJobByScheduleID(scheduleJob.ID, summarizer.JobType_ComplianceSummarizer)
		if err != nil {
			s.logger.Error("Failed to fetch SummarizerJob", zap.Error(err))
			continue
		}

		if j == nil {
			err = s.scheduleComplianceSummarizerJob(scheduleJob.ID)
			if err != nil {
				s.logger.Error("Failed to enqueue summarizer job\n",
					zap.Uint("jobId", scheduleJob.ID),
					zap.Error(err),
				)
			}
		}

		if j.Status == summarizerapi.SummarizerJobInProgress {
			continue
		}

		err = s.db.UpdateScheduleJobStatus(scheduleJob.ID, summarizerapi.SummarizerJobSucceeded)
		if err != nil {
			s.logger.Error("Failed to update ScheduleJob's status", zap.Error(err))
			continue
		}
	}
}

func (s *Scheduler) RunDescribeJobCompletionUpdater() {
	t := time.NewTicker(JobCompletionInterval)
	defer t.Stop()

	for ; ; <-t.C {
		results, err := s.db.QueryInProgressDescribedSourceJobGroupByDescribeResourceJobStatus()
		if err != nil {
			s.logger.Error("Failed to find DescribeSourceJobs", zap.Error(err))
			continue
		}

		jobIDToStatus := make(map[uint]map[api.DescribeResourceJobStatus]int)
		for _, v := range results {
			if _, ok := jobIDToStatus[v.DescribeSourceJobID]; !ok {
				jobIDToStatus[v.DescribeSourceJobID] = map[api.DescribeResourceJobStatus]int{
					api.DescribeResourceJobCreated:   0,
					api.DescribeResourceJobQueued:    0,
					api.DescribeResourceJobFailed:    0,
					api.DescribeResourceJobSucceeded: 0,
				}
			}

			jobIDToStatus[v.DescribeSourceJobID][v.DescribeResourceJobStatus] = v.DescribeResourceJobCount
		}

		for id, status := range jobIDToStatus {
			// If any CREATED or QUEUED, job is still in progress
			if status[api.DescribeResourceJobCreated] > 0 ||
				status[api.DescribeResourceJobQueued] > 0 {
				continue
			}

			// If any FAILURE, job is completed with failure
			if status[api.DescribeResourceJobFailed] > 0 {
				err := s.db.UpdateDescribeSourceJob(id, api.DescribeSourceJobCompletedWithFailure)
				if err != nil {
					s.logger.Error("Failed to update DescribeSourceJob status\n",
						zap.Uint("jobId", id),
						zap.String("status", string(api.DescribeSourceJobCompletedWithFailure)),
						zap.Error(err),
					)
				}

				job, err := s.db.GetDescribeSourceJob(id)
				if err != nil {
					s.logger.Error("Failed to call summarizer\n",
						zap.Uint("jobId", id),
						zap.Error(err),
					)
				} else if job == nil {
					s.logger.Error("Failed to find the job for summarizer\n",
						zap.Uint("jobId", id),
						zap.Error(err),
					)
				} else {
				}
				continue
			}

			// If the rest is SUCCEEDED, job has completed with no failure
			if status[api.DescribeResourceJobSucceeded] > 0 {
				err := s.db.UpdateDescribeSourceJob(id, api.DescribeSourceJobCompleted)
				if err != nil {
					s.logger.Error("Failed to update DescribeSourceJob status\n",
						zap.Uint("jobId", id),
						zap.String("status", string(api.DescribeSourceJobCompleted)),
						zap.Error(err),
					)
				}

				continue
			}
		}
	}
}

func (s Scheduler) RunDescribeJobScheduler() {
	s.logger.Info("Scheduling describe jobs on a timer")

	t := time.NewTicker(JobSchedulingInterval)
	defer t.Stop()

	for ; ; <-t.C {
		s.scheduleDescribeJob()
	}
}

func (s Scheduler) createLocalDescribeSource(scheduleJob *ScheduleJob, source *Source) {
	if isPublishingBlocked(s.logger, s.describeConnectionJobQueue) {
		s.logger.Warn("The jobs in queue is over the threshold")
		return
	}
	src, err := s.db.GetLastDescribeSourceJob(source.ID)
	if err != nil {
		DescribeSourceJobsCount.WithLabelValues("failure").Inc()
		s.logger.Error("Failed to get last describe source job",
			zap.Uint("jobID", scheduleJob.ID),
			zap.String("sourceID", source.ID.String()),
			zap.Error(err))
		return
	}

	triggerType := enums.DescribeTriggerTypeScheduled
	if src == nil {
		triggerType = enums.DescribeTriggerTypeInitialDiscovery
	}

	s.logger.Info("Source is due for a describe. Creating a job now", zap.String("sourceId", source.ID.String()))
	daj := newDescribeSourceJob(*source, *scheduleJob)
	err = s.db.CreateDescribeSourceJob(&daj)
	if err != nil {
		DescribeSourceJobsCount.WithLabelValues("failure").Inc()
		s.logger.Error("Failed to create DescribeSourceJob",
			zap.Uint("jobId", daj.ID),
			zap.String("sourceId", source.ID.String()),
			zap.Error(err),
		)
		return
	}

	enqueueDescribeConnectionJob(s.logger, s.db, s.describeConnectionJobQueue, *source, daj, scheduleJob.ID, scheduleJob.CreatedAt, triggerType)

	isSuccessful := true

	err = s.db.UpdateDescribeSourceJob(daj.ID, api.DescribeSourceJobInProgress)
	if err != nil {
		DescribeSourceJobsCount.WithLabelValues("failure").Inc()
		s.logger.Error("Failed to update DescribeSourceJob",
			zap.Uint("jobId", daj.ID),
			zap.String("sourceId", source.ID.String()),
			zap.Error(err),
		)
		isSuccessful = false
	}
	daj.Status = api.DescribeSourceJobInProgress

	err = s.db.UpdateSourceDescribed(source.ID, scheduleJob.CreatedAt, time.Duration(s.describeIntervalHours)*time.Hour)
	if err != nil {
		DescribeSourceJobsCount.WithLabelValues("failure").Inc()
		s.logger.Error("Failed to update Source",
			zap.String("sourceId", source.ID.String()),
			zap.Error(err),
		)
		isSuccessful = false
	}

	if isSuccessful {
		DescribeSourceJobsCount.WithLabelValues("successful").Inc()
	}

	return
}

func (s Scheduler) createCloudNativeDescribeSource(scheduleJob *ScheduleJob, source *Source) {
	if isPublishingBlocked(s.logger, s.describeConnectionJobQueue) {
		s.logger.Warn("The jobs in queue is over the threshold")
		return
	}
	src, err := s.db.GetLastDescribeSourceJob(source.ID)
	if err != nil {
		DescribeSourceJobsCount.WithLabelValues("failure").Inc()
		s.logger.Error("Failed to get last describe source job",
			zap.Uint("jobID", scheduleJob.ID),
			zap.String("sourceID", source.ID.String()),
			zap.Error(err))
		return
	}

	triggerType := enums.DescribeTriggerTypeScheduled
	if src == nil {
		triggerType = enums.DescribeTriggerTypeInitialDiscovery
	}

	s.logger.Info("Source is due for a describe. Creating a job now", zap.String("sourceId", source.ID.String()))
	daj := newDescribeSourceJob(*source, *scheduleJob)
	err = s.db.CreateDescribeSourceJob(&daj)
	if err != nil {
		DescribeSourceJobsCount.WithLabelValues("failure").Inc()
		s.logger.Error("Failed to create DescribeSourceJob",
			zap.Uint("jobId", daj.ID),
			zap.String("sourceId", source.ID.String()),
			zap.Error(err),
		)
		return
	}
	cloudDaj, err := newCloudNativeDescribeSourceJob(daj)
	if err != nil {
		DescribeSourceJobsCount.WithLabelValues("failure").Inc()
		s.logger.Error("Failed to create CloudNativeDescribeSourceJob",
			zap.Uint("jobId", daj.ID),
			zap.String("sourceId", source.ID.String()),
			zap.Error(err),
		)
		return
	}
	err = s.db.CreateCloudNativeDescribeSourceJob(&cloudDaj)

	enqueueCloudNativeDescribeConnectionJob(s.logger, s.db, s.cloudNativeDescribeConnectionJobQueue, *source, cloudDaj, s.kafkaResourcesTopic, scheduleJob.ID, scheduleJob.CreatedAt, triggerType)

	isSuccessful := true

	err = s.db.UpdateDescribeSourceJob(daj.ID, api.DescribeSourceJobInProgress)
	if err != nil {
		DescribeSourceJobsCount.WithLabelValues("failure").Inc()
		s.logger.Error("Failed to update DescribeSourceJob",
			zap.Uint("jobId", daj.ID),
			zap.String("sourceId", source.ID.String()),
			zap.Error(err),
		)
		isSuccessful = false
	}
	daj.Status = api.DescribeSourceJobInProgress

	err = s.db.UpdateSourceDescribed(source.ID, scheduleJob.CreatedAt, time.Duration(s.describeIntervalHours)*time.Hour)
	if err != nil {
		DescribeSourceJobsCount.WithLabelValues("failure").Inc()
		s.logger.Error("Failed to update Source",
			zap.String("sourceId", source.ID.String()),
			zap.Error(err),
		)
		isSuccessful = false
	}

	if isSuccessful {
		DescribeSourceJobsCount.WithLabelValues("successful").Inc()
	}

	return
}

func (s Scheduler) scheduleDescribeJob() {
	scheduleJob, err := s.db.FetchLastScheduleJob()
	if err != nil {
		s.logger.Error("Failed to fetch last ScheduleJob", zap.Error(err))
		DescribeJobsCount.WithLabelValues("failure").Inc()
		return
	}

	if scheduleJob == nil ||
		(scheduleJob.CreatedAt.Before(time.Now().Add(time.Duration(-s.describeIntervalHours)*time.Hour)) && scheduleJob.Status != summarizerapi.SummarizerJobInProgress) {
		job := ScheduleJob{
			Model:          gorm.Model{},
			Status:         summarizerapi.SummarizerJobInProgress,
			FailureMessage: "",
		}
		err := s.db.AddScheduleJob(&job)
		if err != nil {
			s.logger.Error("Failed to add new ScheduleJob", zap.Error(err))
			DescribeJobsCount.WithLabelValues("failure").Inc()
			return
		}
		scheduleJob = &job
	}

	s.logger.Info("Checking sources due for this schedule job", zap.Uint("jobID", scheduleJob.ID))
	describeJobs, err := s.db.QueryDescribeSourceJobsForScheduleJob(scheduleJob)
	if err != nil {
		s.logger.Error("Failed to fetch related describe source jobs", zap.Error(err))
		DescribeJobsCount.WithLabelValues("failure").Inc()
		return
	}

	srcs, err := s.db.ListSources()
	if err != nil {
		s.logger.Error("Failed to find list of sources", zap.Error(err))
		DescribeJobsCount.WithLabelValues("failure").Inc()
		return
	}

	var sources []Source
	for _, src := range srcs {
		hasOne := false
		for _, j := range describeJobs {
			if src.ID == j.SourceID {
				hasOne = true
				break
			}
		}
		if hasOne {
			continue
		}

		sources = append(sources, src)
	}

	if len(sources) > 0 {
		s.logger.Info("There are some sources that need to be described", zap.Int("count", len(sources)))
	} else {
		DescribeJobsCount.WithLabelValues("successful").Inc()
		return
	}

	limit, err := s.workspaceClient.GetLimitsByID(&httpclient.Context{
		UserRole: api2.ViewerRole,
	}, CurrentWorkspaceID)
	if err != nil {
		DescribeSourceJobsCount.WithLabelValues("failure").Inc()
		s.logger.Error("Failed to get workspace limits",
			zap.String("workspace", CurrentWorkspaceID),
			zap.Error(err),
		)
		return
	}

	currentResourceCount, err := s.es.Count(context.Background(), InventorySummaryIndex)
	if err != nil {
		DescribeSourceJobsCount.WithLabelValues("failure").Inc()
		s.logger.Error("Failed to get count of current resources",
			zap.String("workspace", CurrentWorkspaceID),
			zap.Error(err),
		)
		return
	}

	if currentResourceCount >= limit.MaxResources {
		DescribeSourceJobsCount.WithLabelValues("failure").Inc()
		s.logger.Error("Workspace has reached its max resources limit",
			zap.String("workspace", CurrentWorkspaceID),
			zap.Error(err),
		)
		return
	}

	if err = s.rdb.Set(context.Background(), RedisKeyWorkspaceResourceRemaining,
		limit.MaxResources-currentResourceCount, 12*time.Hour).Err(); err != nil {
		DescribeSourceJobsCount.WithLabelValues("failure").Inc()
		s.logger.Error("Failed to set workspace resource remaining on redis",
			zap.String("workspace", CurrentWorkspaceID),
			zap.Error(err),
		)
		return
	}

	sourceIDs := make([]string, 0, len(sources))
	for _, src := range sources {
		sourceIDs = append(sourceIDs, src.ID.String())
	}
	onboardSources, err := s.onboardClient.GetSources(&httpclient.Context{
		UserRole: api2.ViewerRole,
	}, sourceIDs)
	if err != nil {
		DescribeSourceJobsCount.WithLabelValues("failure").Inc()
		s.logger.Error("Failed to get onboard sources",
			zap.Strings("sourceIDs", sourceIDs),
			zap.Error(err),
		)
		return
	}
	filteredSources := make([]Source, 0, len(sources))
	for _, src := range sources {
		for _, onboardSrc := range onboardSources {
			if src.ID.String() == onboardSrc.ID.String() {
				healthCheckedSrc, err := s.onboardClient.GetSourceHealthcheck(&httpclient.Context{
					UserRole: api2.EditorRole,
				}, onboardSrc.ID.String())
				if err != nil {
					s.logger.Error("Failed to get source healthcheck",
						zap.String("sourceID", onboardSrc.ID.String()),
						zap.Error(err),
					)
					continue
				}
				if healthCheckedSrc.AssetDiscoveryMethod == source.AssetDiscoveryMethodTypeScheduled &&
					healthCheckedSrc.HealthState != source.SourceHealthStateUnhealthy {
					filteredSources = append(filteredSources, src)
				}
				break
			}
		}
	}
	sources = filteredSources

	rand.Shuffle(len(sources), func(i, j int) { sources[i], sources[j] = sources[j], sources[i] })
	for _, source := range sources {
		//s.createLocalDescribeSource(scheduleJob, &source) // Uncomment this line to enable local describe
		s.createCloudNativeDescribeSource(scheduleJob, &source) // Comment this line to enable local describe
	}
	DescribeJobsCount.WithLabelValues("successful").Inc()
}
func (s *Scheduler) RunComplianceReportCleanupJobScheduler() {
	s.logger.Info("Running compliance report cleanup job scheduler")

	t := time.NewTicker(JobSchedulingInterval)
	defer t.Stop()

	for range t.C {
		s.cleanupComplianceReportJob()
	}
}

func (s *Scheduler) RunDescribeCleanupJobScheduler() {
	s.logger.Info("Running describe cleanup job scheduler")

	t := time.NewTicker(JobSchedulingInterval)
	defer t.Stop()

	for range t.C {
		s.cleanupDescribeJob()
	}
}

func (s *Scheduler) RunDeletedSourceCleanup() {
	for id := range s.deletedSources {
		// cleanup describe job for deleted source
		s.cleanupDescribeJobForDeletedSource(id)
		// cleanup compliance report job for deleted source
		s.cleanupComplianceReportJobForDeletedSource(id)
	}
}

func (s Scheduler) cleanupDescribeJobForDeletedSource(sourceId string) {
	jobs, err := s.db.QueryDescribeSourceJobs(sourceId)
	if err != nil {
		s.logger.Error("Failed to find all completed DescribeSourceJobs for source",
			zap.String("sourceId", sourceId),
			zap.Error(err),
		)
		DescribeCleanupJobsCount.WithLabelValues("failure").Inc()
		return
	}

	s.handleDescribeJobs(jobs)

	DescribeCleanupJobsCount.WithLabelValues("successful").Inc()
}

func (s Scheduler) handleDescribeJobs(jobs []DescribeSourceJob) {
	for _, sj := range jobs {
		// I purposefully didn't embbed this query in the previous query to keep returned results count low.
		drj, err := s.db.ListDescribeResourceJobs(sj.ID)
		if err != nil {
			s.logger.Error("Failed to retrieve DescribeResourceJobs for DescribeSouceJob",
				zap.Uint("jobId", sj.ID),
				zap.Error(err),
			)
			DescribeCleanupSourceJobsCount.WithLabelValues("failure").Inc()
			continue
		}

		success := true
		for _, rj := range drj {
			if isPublishingBlocked(s.logger, s.describeCleanupJobQueue) {
				s.logger.Warn("The jobs in queue is over the threshold")
				return
			}

			if err := s.describeCleanupJobQueue.Publish(DescribeCleanupJob{
				JobID:        rj.ID,
				ResourceType: rj.ResourceType,
			}); err != nil {
				s.logger.Error("Failed to publish describe clean up job to queue for DescribeResourceJob",
					zap.Uint("jobId", rj.ID),
					zap.Error(err),
				)
				success = false
				DescribeCleanupSourceJobsCount.WithLabelValues("failure").Inc()
				continue
			}

			err = s.db.DeleteDescribeResourceJob(rj.ID)
			if err != nil {
				s.logger.Error("Failed to delete DescribeResourceJob",
					zap.Uint("jobId", rj.ID),
					zap.Error(err),
				)
				success = false
				DescribeCleanupSourceJobsCount.WithLabelValues("failure").Inc()
				continue
			}
		}

		if success {
			err := s.db.DeleteDescribeSourceJob(sj.ID)
			if err != nil {
				s.logger.Error("Failed to delete DescribeSourceJob",
					zap.Uint("jobId", sj.ID),
					zap.Error(err),
				)
				DescribeCleanupSourceJobsCount.WithLabelValues("failure").Inc()
			} else {
				DescribeCleanupSourceJobsCount.WithLabelValues("successful").Inc()
			}
		} else {
			DescribeCleanupSourceJobsCount.WithLabelValues("failure").Inc()
		}

		s.logger.Info("Successfully deleted DescribeSourceJob and its DescribeResourceJobs",
			zap.Uint("jobId", sj.ID),
		)
	}
}

func (s Scheduler) cleanupDescribeJob() {
	jobs, err := s.db.QueryOlderThanNRecentCompletedDescribeSourceJobs(50)
	if err != nil {
		s.logger.Error("Failed to find older than 5 recent completed DescribeSourceJob for each source",
			zap.Error(err),
		)
		DescribeCleanupJobsCount.WithLabelValues("failure").Inc()
		return
	}

	s.handleDescribeJobs(jobs)

	DescribeCleanupJobsCount.WithLabelValues("successful").Inc()
}

func (s Scheduler) cleanupComplianceReportJobForDeletedSource(sourceId string) {
	jobs, err := s.db.QueryComplianceReportJobs(sourceId)
	if err != nil {
		s.logger.Error("Failed to find all completed ComplianceReportJobs for source",
			zap.String("sourceId", sourceId),
			zap.Error(err),
		)
		return
	}
	s.handleComplianceReportJobs(jobs)
}

func (s Scheduler) handleComplianceReportJobs(jobs []ComplianceReportJob) {
	for _, job := range jobs {
		if err := s.complianceReportCleanupJobQueue.Publish(compliancereport.ComplianceReportCleanupJob{
			JobID: job.ID,
		}); err != nil {
			s.logger.Error("Failed to publish compliance report clean up job to queue for ComplianceReportJob",
				zap.Uint("jobId", job.ID),
				zap.Error(err),
			)
			return
		}

		if err := s.db.DeleteComplianceReportJob(job.ID); err != nil {
			s.logger.Error("Failed to delete ComplianceReportJob",
				zap.Uint("jobId", job.ID),
				zap.Error(err),
			)
		}
		s.logger.Info("Successfully deleted ComplianceReportJob", zap.Uint("jobId", job.ID))
	}
}

func (s Scheduler) cleanupComplianceReportJob() {
	jobs, err := s.db.QueryOlderThanNRecentCompletedComplianceReportJobs(5)
	if err != nil {
		s.logger.Error("Failed to find older than 5 recent completed ComplianceReportJobs for each source",
			zap.Error(err),
		)
		return
	}
	s.handleComplianceReportJobs(jobs)
}

// Consume events from the source queue. Based on the action of the event,
// update the list of sources that need to be described. Either create a source
// or update/delete the source.
func (s *Scheduler) RunSourceEventsConsumer() error {
	s.logger.Info("Consuming messages from SourceEvents queue")
	msgs, err := s.sourceQueue.Consume()
	if err != nil {
		return err
	}

	for msg := range msgs {
		var event SourceEvent
		if err := json.Unmarshal(msg.Body, &event); err != nil {
			s.logger.Error("Failed to unmarshal SourceEvent", zap.Error(err))
			err = msg.Nack(false, false)
			if err != nil {
				s.logger.Error("Failed nacking message", zap.Error(err))
			}
			continue
		}

		err := ProcessSourceAction(s.db, event)
		if err != nil {
			s.logger.Error("Failed to process event for Source",
				zap.String("sourceId", event.SourceID.String()),
				zap.Error(err),
			)
			err = msg.Nack(false, false)
			if err != nil {
				s.logger.Error("Failed nacking message", zap.Error(err))
			}
			continue
		}

		if err := msg.Ack(false); err != nil {
			s.logger.Error("Failed acking message", zap.Error(err))
		}

		if event.Action == SourceDelete {
			s.deletedSources <- event.SourceID.String()
		}
	}

	return fmt.Errorf("source events queue channel is closed")
}

// RunDescribeConnectionJobResultsConsumer consumes messages from the jobResult queue.
// It will update the status of the jobs in the database based on the message.
// It will also update the jobs status that are not completed in certain time to FAILED
func (s *Scheduler) RunDescribeConnectionJobResultsConsumer() error {
	s.logger.Info("Consuming messages from the JobResults queue")

	msgs, err := s.describeConnectionJobResultQueue.Consume()
	if err != nil {
		return err
	}

	t := time.NewTicker(JobTimeoutCheckInterval)
	defer t.Stop()

	for {
		select {
		case msg, ok := <-msgs:
			if !ok {
				return fmt.Errorf("tasks channel is closed")
			}

			var result DescribeConnectionJobResult
			if err := json.Unmarshal(msg.Body, &result); err != nil {
				s.logger.Error("Failed to unmarshal DescribeConnectionJobResult results\n", zap.Error(err))
				err = msg.Nack(false, false)
				if err != nil {
					s.logger.Error("Failed nacking message", zap.Error(err))
				}
				continue
			}

			failed := false
			for jobID, res := range result.Result {
				s.logger.Info("Processing JobResult for Job",
					zap.Uint("jobId", jobID),
					zap.String("status", string(res.Status)),
				)

				if strings.Contains(res.Error, "ThrottlingException") ||
					strings.Contains(res.Error, "Rate exceeded") ||
					strings.Contains(res.Error, "RateExceeded") {
					// sent it to describe jobs
					s.logger.Info("Needs to be retried",
						zap.Uint("jobId", jobID),
						zap.String("status", string(res.Status)),
					)
					if err := s.describeJobQueue.Publish(res.DescribeJob); err != nil {
						s.logger.Error("Failed to queue DescribeConnectionJob",
							zap.Uint("jobId", res.JobID),
							zap.Error(err),
						)
					} else {
						continue
					}
				}

				err := s.db.UpdateDescribeResourceJobStatus(res.JobID, res.Status, res.Error)
				if err != nil {
					failed = true
					s.logger.Error("Failed to update the status of DescribeResourceJob",
						zap.Uint("jobId", res.JobID),
						zap.Error(err),
					)
					err = msg.Nack(false, true)
					if err != nil {
						s.logger.Error("Failed nacking message", zap.Error(err))
					}
					break
				}
			}

			if failed {
				continue
			}

			if err := msg.Ack(false); err != nil {
				s.logger.Error("Failed acking message", zap.Error(err))
			}
		case <-t.C:
			err := s.db.UpdateDescribeResourceJobsTimedOut(s.describeIntervalHours)
			if err != nil {
				s.logger.Error("Failed to update timed out DescribeResourceJobs", zap.Error(err))
			}
		}
	}
}

// RunDescribeJobResultsConsumer consumes messages from the jobResult queue.
// It will update the status of the jobs in the database based on the message.
// It will also update the jobs status that are not completed in certain time to FAILED
func (s *Scheduler) RunDescribeJobResultsConsumer() error {
	s.logger.Info("Consuming messages from the JobResults queue")

	msgs, err := s.describeJobResultQueue.Consume()
	if err != nil {
		return err
	}

	t := time.NewTicker(JobTimeoutCheckInterval)
	defer t.Stop()

	for {
		select {
		case msg, ok := <-msgs:
			if !ok {
				return fmt.Errorf("tasks channel is closed")
			}

			var result DescribeJobResult
			if err := json.Unmarshal(msg.Body, &result); err != nil {
				s.logger.Error("Failed to unmarshal DescribeJobResult results\n", zap.Error(err))
				err = msg.Nack(false, false)
				if err != nil {
					s.logger.Error("Failed nacking message", zap.Error(err))
				}
				continue
			}

			s.logger.Info("Processing JobResult for Job",
				zap.Uint("jobId", result.JobID),
				zap.String("status", string(result.Status)),
			)
			err := s.db.UpdateDescribeResourceJobStatus(result.JobID, result.Status, result.Error)
			if err != nil {
				s.logger.Error("Failed to update the status of DescribeResourceJob",
					zap.Uint("jobId", result.JobID),
					zap.Error(err),
				)
				err = msg.Nack(false, true)
				if err != nil {
					s.logger.Error("Failed nacking message", zap.Error(err))
				}
				continue
			}

			if err := msg.Ack(false); err != nil {
				s.logger.Error("Failed acking message", zap.Error(err))
			}
		case <-t.C:
			err := s.db.UpdateDescribeResourceJobsTimedOut(s.describeIntervalHours)
			if err != nil {
				s.logger.Error("Failed to update timed out DescribeResourceJobs", zap.Error(err))
			}
		}
	}
}

//
//func (s *Scheduler) RunComplianceReportScheduler() {
//	s.logger.Info("Scheduling ComplianceReport jobs on a timer")
//	t := time.NewTicker(JobComplianceReportInterval)
//	defer t.Stop()
//
//	for ; ; <-t.C {
//		sources, err := s.db.QuerySourcesDueForComplianceReport()
//		if err != nil {
//			s.logger.Error("Failed to find the next sources to create ComplianceReportJob", zap.Error(err))
//			ComplianceJobsCount.WithLabelValues("failure").Inc()
//			continue
//		}
//
//		for _, source := range sources {
//			if isPublishingBlocked(s.logger, s.complianceReportJobQueue) {
//				s.logger.Warn("The jobs in queue is over the threshold", zap.Error(err))
//				break
//			}
//
//			s.logger.Error("Source is due for a steampipe check. Creating a ComplianceReportJob now", zap.String("sourceId", source.ID.String()))
//			crj := newComplianceReportJob(source)
//			err := s.db.CreateComplianceReportJob(&crj)
//			if err != nil {
//				ComplianceSourceJobsCount.WithLabelValues("failure").Inc()
//				s.logger.Error("Failed to create ComplianceReportJob for Source",
//					zap.Uint("jobId", crj.ID),
//					zap.String("sourceId", source.ID.String()),
//					zap.Error(err),
//				)
//				continue
//			}
//
//			enqueueComplianceReportJobs(s.logger, s.db, s.complianceReportJobQueue, source, &crj)
//
//			err = s.db.UpdateSourceReportGenerated(source.ID, s.complianceIntervalHours)
//			if err != nil {
//				s.logger.Error("Failed to update report job of Source: %s\n", zap.String("sourceId", source.ID.String()), zap.Error(err))
//				ComplianceSourceJobsCount.WithLabelValues("failure").Inc()
//				continue
//			}
//			ComplianceSourceJobsCount.WithLabelValues("successful").Inc()
//		}
//		ComplianceJobsCount.WithLabelValues("successful").Inc()
//	}
//}

func (s *Scheduler) RunComplianceReport(scheduleJob *ScheduleJob) (int, error) {
	createdJobCount := 0

	sources, err := s.db.ListSources()
	if err != nil {
		ComplianceJobsCount.WithLabelValues("failure").Inc()
		return createdJobCount, fmt.Errorf("error while listing sources: %v", err)
	}

	for _, source := range sources {
		ctx := &httpclient.Context{
			UserRole: api2.ViewerRole,
		}
		benchmarks, err := s.complianceClient.GetAllBenchmarkAssignmentsBySourceId(ctx, source.ID)
		if err != nil {
			ComplianceJobsCount.WithLabelValues("failure").Inc()
			return createdJobCount, fmt.Errorf("error while getting benchmark assignments: %v", err)
		}

		for _, b := range benchmarks {
			crj := newComplianceReportJob(source, b.BenchmarkId, scheduleJob.ID)
			err := s.db.CreateComplianceReportJob(&crj)
			if err != nil {
				ComplianceJobsCount.WithLabelValues("failure").Inc()
				ComplianceSourceJobsCount.WithLabelValues("failure").Inc()
				return createdJobCount, fmt.Errorf("error while creating compliance job: %v", err)
			}

			enqueueComplianceReportJobs(s.logger, s.db, s.complianceReportJobQueue, source, &crj, scheduleJob)

			err = s.db.UpdateSourceReportGenerated(source.ID, s.complianceIntervalHours)
			if err != nil {
				ComplianceJobsCount.WithLabelValues("failure").Inc()
				ComplianceSourceJobsCount.WithLabelValues("failure").Inc()
				return createdJobCount, fmt.Errorf("error while updating compliance job: %v", err)
			}
			ComplianceSourceJobsCount.WithLabelValues("successful").Inc()
			createdJobCount++
		}
	}
	ComplianceJobsCount.WithLabelValues("successful").Inc()
	return createdJobCount, nil
}

// RunComplianceReportJobResultsConsumer consumes messages from the complianceReportJobResultQueue queue.
// It will update the status of the jobs in the database based on the message.
// It will also update the jobs status that are not completed in certain time to FAILED
func (s *Scheduler) RunComplianceReportJobResultsConsumer() error {
	s.logger.Info("Consuming messages from the ComplianceReportJobResultQueue queue")

	msgs, err := s.complianceReportJobResultQueue.Consume()
	if err != nil {
		return err
	}

	t := time.NewTicker(JobTimeoutCheckInterval)
	defer t.Stop()

	for {
		select {
		case msg, ok := <-msgs:
			if !ok {
				return fmt.Errorf("tasks channel is closed")
			}

			var result compliancereport.JobResult
			if err := json.Unmarshal(msg.Body, &result); err != nil {
				s.logger.Error("Failed to unmarshal ComplianceReportJob results", zap.Error(err))
				err = msg.Nack(false, false)
				if err != nil {
					s.logger.Error("Failed nacking message", zap.Error(err))
				}
				continue
			}

			s.logger.Info("Processing ReportJobResult for Job",
				zap.Uint("jobId", result.JobID),
				zap.String("status", string(result.Status)),
			)
			err := s.db.UpdateComplianceReportJob(result.JobID, result.Status, result.ReportCreatedAt, result.Error)
			if err != nil {
				s.logger.Error("Failed to update the status of ComplianceReportJob",
					zap.Uint("jobId", result.JobID),
					zap.Error(err))
				err = msg.Nack(false, true)
				if err != nil {
					s.logger.Error("Failed nacking message", zap.Error(err))
				}
				continue
			}

			if err := msg.Ack(false); err != nil {
				s.logger.Error("Failed acking message", zap.Error(err))
			}
		case <-t.C:
			err := s.db.UpdateComplianceReportJobsTimedOut(s.complianceIntervalHours)
			if err != nil {
				s.logger.Error("Failed to update timed out ComplianceReportJob", zap.Error(err))
			}
		}
	}
}

func (s *Scheduler) Stop() {
	queues := []queue.Interface{
		s.describeJobQueue,
		s.describeJobResultQueue,
		s.describeConnectionJobQueue,
		s.describeConnectionJobResultQueue,
		s.describeCleanupJobQueue,
		s.complianceReportJobQueue,
		s.complianceReportJobResultQueue,
		s.sourceQueue,
		s.insightJobQueue,
		s.insightJobResultQueue,
		s.summarizerJobQueue,
		s.summarizerJobResultQueue,
	}

	for _, queue := range queues {
		queue.Close()
	}

	s.kafkaClient.Close()
	s.cloudNativeDescribeConnectionJobQueue.Close(context.Background())
}

func newDescribeSourceJob(a Source, s ScheduleJob) DescribeSourceJob {
	daj := DescribeSourceJob{
		ScheduleJobID:        s.ID,
		DescribedAt:          s.CreatedAt,
		SourceID:             a.ID,
		AccountID:            a.AccountID,
		DescribeResourceJobs: []DescribeResourceJob{},
		Status:               api.DescribeSourceJobCreated,
	}
	switch sType := api.SourceType(a.Type); sType {
	case api.SourceCloudAWS:
		resourceTypes := aws.ListResourceTypes()
		rand.Shuffle(len(resourceTypes), func(i, j int) { resourceTypes[i], resourceTypes[j] = resourceTypes[j], resourceTypes[i] })
		for _, rType := range resourceTypes {
			daj.DescribeResourceJobs = append(daj.DescribeResourceJobs, DescribeResourceJob{
				ResourceType: rType,
				Status:       api.DescribeResourceJobCreated,
			})
		}
	case api.SourceCloudAzure:
		resourceTypes := azure.ListResourceTypes()
		rand.Shuffle(len(resourceTypes), func(i, j int) { resourceTypes[i], resourceTypes[j] = resourceTypes[j], resourceTypes[i] })
		for _, rType := range resourceTypes {
			daj.DescribeResourceJobs = append(daj.DescribeResourceJobs, DescribeResourceJob{
				ResourceType: rType,
				Status:       api.DescribeResourceJobCreated,
			})
		}
	default:
		panic(fmt.Errorf("unsupported source type: %s", sType))
	}

	return daj
}

func newCloudNativeDescribeSourceJob(j DescribeSourceJob) (CloudNativeDescribeSourceJob, error) {
	credentialsKeypair, err := crypto.GenerateKey(j.AccountID, j.SourceID.String(), "x25519", 0)
	if err != nil {
		return CloudNativeDescribeSourceJob{}, err
	}
	credentialsPrivateKey, err := credentialsKeypair.Armor()
	if err != nil {
		return CloudNativeDescribeSourceJob{}, err
	}
	credentialsPublicKey, err := credentialsKeypair.GetArmoredPublicKey()
	if err != nil {
		return CloudNativeDescribeSourceJob{}, err
	}

	resultEncryptionKeyPair, err := crypto.GenerateKey(j.AccountID, j.SourceID.String(), "x25519", 0)
	if err != nil {
		return CloudNativeDescribeSourceJob{}, err
	}
	resultEncryptionPrivateKey, err := resultEncryptionKeyPair.Armor()
	if err != nil {
		return CloudNativeDescribeSourceJob{}, err
	}
	resultEncryptionPublicKey, err := resultEncryptionKeyPair.GetArmoredPublicKey()
	if err != nil {
		return CloudNativeDescribeSourceJob{}, err
	}

	job := CloudNativeDescribeSourceJob{
		SourceJob:                      j,
		CredentialEncryptionPrivateKey: credentialsPrivateKey,
		CredentialEncryptionPublicKey:  credentialsPublicKey,
		ResultEncryptionPrivateKey:     resultEncryptionPrivateKey,
		ResultEncryptionPublicKey:      resultEncryptionPublicKey,
	}

	return job, nil
}

func newComplianceReportJob(a Source, benchmarkID string, scheduleJobID uint) ComplianceReportJob {
	return ComplianceReportJob{
		Model:           gorm.Model{},
		ScheduleJobID:   scheduleJobID,
		SourceID:        a.ID,
		BenchmarkID:     benchmarkID,
		ReportCreatedAt: 0,
		Status:          complianceapi.ComplianceReportJobCreated,
		FailureMessage:  "",
	}
}

func enqueueDescribeResourceJobs(logger *zap.Logger, db Database, q queue.Interface, a Source, daj DescribeSourceJob, describedAt time.Time) {
	var oldJobFailed error

	for i, drj := range daj.DescribeResourceJobs {
		nextStatus := api.DescribeResourceJobQueued
		errMsg := ""

		if oldJobFailed == nil {
			if err := q.Publish(DescribeJob{
				JobID:        drj.ID,
				ParentJobID:  daj.ID,
				ResourceType: drj.ResourceType,
				SourceID:     daj.SourceID.String(),
				AccountID:    daj.AccountID,
				DescribedAt:  describedAt.UnixMilli(),
				SourceType:   a.Type,
				ConfigReg:    a.ConfigRef,
			}); err != nil {
				logger.Error("Failed to queue DescribeResourceJob",
					zap.Uint("jobId", drj.ID),
					zap.Error(err),
				)

				nextStatus = api.DescribeResourceJobFailed
				errMsg = fmt.Sprintf("queue: %s", err.Error())
			}
		} else {
			nextStatus = api.DescribeResourceJobFailed
			errMsg = fmt.Sprintf("queue: %s", oldJobFailed.Error())
		}

		if err := db.UpdateDescribeResourceJobStatus(drj.ID, nextStatus, errMsg); err != nil {
			logger.Error("Failed to update DescribeResourceJob",
				zap.Uint("jobId", drj.ID),
				zap.Error(err),
			)
		}

		daj.DescribeResourceJobs[i].Status = nextStatus
	}
}

func enqueueDescribeConnectionJob(logger *zap.Logger, db Database, q queue.Interface, a Source, daj DescribeSourceJob, scheduleJobID uint, describedAt time.Time, triggerType enums.DescribeTriggerType) {
	nextStatus := api.DescribeResourceJobQueued
	errMsg := ""

	resourceJobs := map[uint]string{}
	for _, drj := range daj.DescribeResourceJobs {
		resourceJobs[drj.ID] = drj.ResourceType
	}
	if err := q.Publish(DescribeConnectionJob{
		JobID:         daj.ID,
		ScheduleJobID: scheduleJobID,
		ResourceJobs:  resourceJobs,
		SourceID:      daj.SourceID.String(),
		AccountID:     daj.AccountID,
		DescribedAt:   describedAt.UnixMilli(),
		SourceType:    a.Type,
		ConfigReg:     a.ConfigRef,
		TriggerType:   triggerType,
	}); err != nil {
		logger.Error("Failed to queue DescribeConnectionJob",
			zap.Uint("jobId", daj.ID),
			zap.Error(err),
		)

		nextStatus = api.DescribeResourceJobFailed
		errMsg = fmt.Sprintf("queue: %s", err.Error())
	}

	for i, drj := range daj.DescribeResourceJobs {
		if err := db.UpdateDescribeResourceJobStatus(drj.ID, nextStatus, errMsg); err != nil {
			logger.Error("Failed to update DescribeResourceJob",
				zap.Uint("jobId", drj.ID),
				zap.Error(err),
			)
		}
		daj.DescribeResourceJobs[i].Status = nextStatus
	}
}

func enqueueCloudNativeDescribeConnectionJob(logger *zap.Logger, db Database, producer *azeventhubs.ProducerClient, a Source, daj CloudNativeDescribeSourceJob, kafkaResourcesTopic string, scheduleJobID uint, describedAt time.Time, triggerType enums.DescribeTriggerType) {
	nextStatus := api.DescribeResourceJobQueued
	errMsg := ""

	resourceJobs := map[uint]string{}
	for _, drj := range daj.SourceJob.DescribeResourceJobs {
		resourceJobs[drj.ID] = drj.ResourceType
	}
	dcj := DescribeConnectionJob{
		JobID:         daj.SourceJob.ID,
		ScheduleJobID: scheduleJobID,
		ResourceJobs:  resourceJobs,
		SourceID:      daj.SourceJob.SourceID.String(),
		AccountID:     daj.SourceJob.AccountID,
		DescribedAt:   describedAt.UnixMilli(),
		SourceType:    a.Type,
		TriggerType:   triggerType,
	}
	dcjJson, err := json.Marshal(dcj)
	if err != nil {
		logger.Error("Failed to marshal DescribeConnectionJob",
			zap.Uint("jobId", daj.ID),
			zap.Error(err),
		)

		nextStatus = api.DescribeResourceJobFailed
		errMsg = fmt.Sprintf("marshal: %s", err.Error())
	}

	cloudTriggerInput := api.CloudNativeConnectionWorkerTriggerInput{
		JobID:                   daj.JobID.String(),
		JobJson:                 string(dcjJson),
		CredentialsCallbackURL:  fmt.Sprintf("%s/schedule/api/v1/jobs/%s/creds", IngressBaseURL, daj.JobID.String()),
		EndOfJobCallbackURL:     fmt.Sprintf("%s/schedule/api/v1/jobs/%s/callback", IngressBaseURL, daj.JobID.String()),
		CredentialDecryptionKey: daj.CredentialEncryptionPrivateKey,
		OutputEncryptionKey:     daj.ResultEncryptionPublicKey,
		ResourcesTopic:          kafkaResourcesTopic,
	}

	//call azure function to trigger describe connection job
	cloudTriggerInputJson, err := json.Marshal(cloudTriggerInput)
	if err != nil {
		logger.Error("Failed to marshal DescribeConnectionJob",
			zap.Uint("jobId", daj.ID),
			zap.Error(err),
		)

		nextStatus = api.DescribeResourceJobFailed
		errMsg = fmt.Sprintf("marshal: %s", err.Error())
	}
	//enqueue job to cloud native connection worker
	batch, err := producer.NewEventDataBatch(context.TODO(), nil)
	if err != nil {
		logger.Error("Failed to create event data batch",
			zap.Uint("jobId", daj.ID),
			zap.Error(err),
		)

		nextStatus = api.DescribeResourceJobFailed
		errMsg = fmt.Sprintf("create event data batch: %s", err.Error())
	}
	err = batch.AddEventData(&azeventhubs.EventData{Body: cloudTriggerInputJson}, nil)
	if err != nil {
		logger.Error("Failed to add event data",
			zap.Uint("jobId", daj.ID),
			zap.Error(err),
		)

		nextStatus = api.DescribeResourceJobFailed
		errMsg = fmt.Sprintf("add event data: %s", err.Error())
	}
	err = producer.SendEventDataBatch(context.TODO(), batch, nil)
	if err != nil {
		logger.Error("Failed to send event data batch",
			zap.Uint("jobId", daj.ID),
			zap.Error(err),
		)

		nextStatus = api.DescribeResourceJobFailed
		errMsg = fmt.Sprintf("send event data batch: %s", err.Error())
	}

	for i, drj := range daj.SourceJob.DescribeResourceJobs {
		if err := db.UpdateDescribeResourceJobStatus(drj.ID, nextStatus, errMsg); err != nil {
			logger.Error("Failed to update DescribeResourceJob",
				zap.Uint("jobId", drj.ID),
				zap.Error(err),
			)
		}
		daj.SourceJob.DescribeResourceJobs[i].Status = nextStatus
	}
}

func enqueueComplianceReportJobs(logger *zap.Logger, db Database, q queue.Interface,
	a Source, crj *ComplianceReportJob, scheduleJob *ScheduleJob) {
	nextStatus := complianceapi.ComplianceReportJobInProgress
	errMsg := ""

	if err := q.Publish(compliancereport.Job{
		JobID:         crj.ID,
		ScheduleJobID: scheduleJob.ID,
		SourceID:      crj.SourceID,
		BenchmarkID:   crj.BenchmarkID,
		SourceType:    source.Type(a.Type),
		ConfigReg:     a.ConfigRef,
		DescribedAt:   scheduleJob.CreatedAt.UnixMilli(),
	}); err != nil {
		logger.Error("Failed to queue ComplianceReportJob",
			zap.Uint("jobId", crj.ID),
			zap.Error(err),
		)

		nextStatus = complianceapi.ComplianceReportJobCompletedWithFailure
		errMsg = fmt.Sprintf("queue: %s", err.Error())
	}

	if err := db.UpdateComplianceReportJob(crj.ID, nextStatus, 0, errMsg); err != nil {
		logger.Error("Failed to update ComplianceReportJob",
			zap.Uint("jobId", crj.ID),
			zap.Error(err),
		)
	}

	crj.Status = nextStatus
}

func isPublishingBlocked(logger *zap.Logger, queue queue.Interface) bool {
	count, err := queue.Len()
	if err != nil {
		logger.Error("Failed to get queue len", zap.String("queueName", queue.Name()), zap.Error(err))
		DescribePublishingBlocked.WithLabelValues(queue.Name()).Set(0)
		return false
	}
	if count >= MaxJobInQueue {
		DescribePublishingBlocked.WithLabelValues(queue.Name()).Set(1)
		return true
	}
	DescribePublishingBlocked.WithLabelValues(queue.Name()).Set(0)
	return false
}

func (s Scheduler) RunInsightJobScheduler() {
	s.logger.Info("Scheduling insight jobs on a timer")

	t := time.NewTicker(JobSchedulingInterval)
	defer t.Stop()

	for ; ; <-t.C {
		s.scheduleInsightJob()
	}
}

func (s Scheduler) RunCheckupJobScheduler() {
	s.logger.Info("Scheduling insight jobs on a timer")

	t := time.NewTicker(JobSchedulingInterval)
	defer t.Stop()

	for ; ; <-t.C {
		s.scheduleCheckupJob()
	}
}

func (s Scheduler) scheduleInsightJob() {
	insightJob, err := s.db.FetchLastInsightJob()
	if err != nil {
		s.logger.Error("Failed to find the last job to check for InsightJob", zap.Error(err))
		InsightJobsCount.WithLabelValues("failure").Inc()
		return
	}

	if insightJob == nil ||
		insightJob.CreatedAt.Add(time.Duration(s.insightIntervalHours)*time.Hour).Before(time.Now()) {
		if isPublishingBlocked(s.logger, s.insightJobQueue) {
			s.logger.Warn("The jobs in queue is over the threshold", zap.Error(err))
			InsightJobsCount.WithLabelValues("failure").Inc()
			return
		}

		s.logger.Info("Workspace is due for a insight. Creating a job now")
		insights, err := s.db.ListInsightsWithFilters(nil)
		if err != nil {
			s.logger.Error("Failed to fetch list of insights", zap.Error(err))
			InsightJobsCount.WithLabelValues("failure").Inc()
			return
		}

		srcs, err := s.db.ListSources()
		if err != nil {
			s.logger.Error("Failed to fetch list of sources", zap.Error(err))
			InsightJobsCount.WithLabelValues("failure").Inc()
			return
		}

		scheduleUUID, err := uuid.NewUUID()
		if err != nil {
			s.logger.Error("Failed to fetch list of sources", zap.Error(err))
			InsightJobsCount.WithLabelValues("failure").Inc()
			return
		}

		for _, src := range srcs {
			for _, ins := range insights {
				job := newInsightJob(ins, src, scheduleUUID.String())
				err = s.db.AddInsightJob(&job)
				if err != nil {
					InsightJobsCount.WithLabelValues("failure").Inc()
					s.logger.Error("Failed to create InsightJob",
						zap.Uint("jobId", job.ID),
						zap.Error(err),
					)
					continue
				}

				err = enqueueInsightJobs(s.db, s.insightJobQueue, job)
				if err != nil {
					InsightJobsCount.WithLabelValues("failure").Inc()
					s.logger.Error("Failed to enqueue InsightJob",
						zap.Uint("jobId", job.ID),
						zap.Error(err),
					)
					job.Status = insightapi.InsightJobFailed
					err = s.db.UpdateInsightJobStatus(job)
					if err != nil {
						s.logger.Error("Failed to update InsightJob status",
							zap.Uint("jobId", job.ID),
							zap.Error(err),
						)
					}
					continue
				}
			}
		}
	}
	InsightJobsCount.WithLabelValues("successful").Inc()
}

func (s Scheduler) scheduleCheckupJob() {
	checkupJob, err := s.db.FetchLastCheckupJob()
	if err != nil {
		s.logger.Error("Failed to find the last job to check for CheckupJob", zap.Error(err))
		CheckupJobsCount.WithLabelValues("failure").Inc()
		return
	}

	if checkupJob == nil ||
		checkupJob.CreatedAt.Add(time.Duration(s.checkupIntervalHours)*time.Hour).Before(time.Now()) {
		if isPublishingBlocked(s.logger, s.checkupJobQueue) {
			s.logger.Warn("The jobs in queue is over the threshold", zap.Error(err))
			CheckupJobsCount.WithLabelValues("failure").Inc()
			return
		}

		job := newCheckupJob()
		err = s.db.AddCheckupJob(&job)
		if err != nil {
			CheckupJobsCount.WithLabelValues("failure").Inc()
			s.logger.Error("Failed to create CheckupJob",
				zap.Uint("jobId", job.ID),
				zap.Error(err),
			)
		}
		err = enqueueCheckupJobs(s.db, s.checkupJobQueue, job)
		if err != nil {
			CheckupJobsCount.WithLabelValues("failure").Inc()
			s.logger.Error("Failed to enqueue CheckupJob",
				zap.Uint("jobId", job.ID),
				zap.Error(err),
			)
			job.Status = checkupapi.CheckupJobFailed
			err = s.db.UpdateCheckupJobStatus(job)
			if err != nil {
				s.logger.Error("Failed to update CheckupJob status",
					zap.Uint("jobId", job.ID),
					zap.Error(err),
				)
			}
		}
		CheckupJobsCount.WithLabelValues("successful").Inc()
	}
}

func (s Scheduler) scheduleSummarizerJob(scheduleJobID uint) error {
	job := newSummarizerJob(summarizer.JobType_ResourceSummarizer, scheduleJobID)
	err := s.db.AddSummarizerJob(&job)
	if err != nil {
		SummarizerJobsCount.WithLabelValues("failure").Inc()
		s.logger.Error("Failed to create SummarizerJob",
			zap.Uint("jobId", job.ID),
			zap.Error(err),
		)
		return err
	}

	err = enqueueSummarizerJobs(s.db, s.summarizerJobQueue, job, scheduleJobID)
	if err != nil {
		SummarizerJobsCount.WithLabelValues("failure").Inc()
		s.logger.Error("Failed to enqueue SummarizerJob",
			zap.Uint("jobId", job.ID),
			zap.Error(err),
		)
		job.Status = summarizerapi.SummarizerJobFailed
		err = s.db.UpdateSummarizerJobStatus(job)
		if err != nil {
			s.logger.Error("Failed to update SummarizerJob status",
				zap.Uint("jobId", job.ID),
				zap.Error(err),
			)
		}
		return err
	}

	return nil
}

func enqueueSummarizerJobs(db Database, q queue.Interface, job SummarizerJob, scheduleJobID uint) error {
	var lastDayJobID, lastWeekJobID, lastQuarterJobID, lastYearJobID uint

	lastDay, err := db.GetOldCompletedScheduleJob(1)
	if err != nil {
		return err
	}
	if lastDay != nil {
		lastDayJobID = lastDay.ID
	}
	lastWeek, err := db.GetOldCompletedScheduleJob(7)
	if err != nil {
		return err
	}
	if lastWeek != nil {
		lastWeekJobID = lastWeek.ID
	}
	lastQuarter, err := db.GetOldCompletedScheduleJob(93)
	if err != nil {
		return err
	}
	if lastQuarter != nil {
		lastQuarterJobID = lastQuarter.ID
	}
	lastYear, err := db.GetOldCompletedScheduleJob(428)
	if err != nil {
		return err
	}
	if lastYear != nil {
		lastYearJobID = lastYear.ID
	}

	if err := q.Publish(summarizer.ResourceJob{
		JobID:                    job.ID,
		ScheduleJobID:            scheduleJobID,
		LastDayScheduleJobID:     lastDayJobID,
		LastWeekScheduleJobID:    lastWeekJobID,
		LastQuarterScheduleJobID: lastQuarterJobID,
		LastYearScheduleJobID:    lastYearJobID,
		JobType:                  summarizer.JobType_ResourceSummarizer,
	}); err != nil {
		return err
	}

	return nil
}

func (s Scheduler) scheduleComplianceSummarizerJob(scheduleJobID uint) error {
	job := newSummarizerJob(summarizer.JobType_ComplianceSummarizer, scheduleJobID)
	err := s.db.AddSummarizerJob(&job)
	if err != nil {
		SummarizerJobsCount.WithLabelValues("failure").Inc()
		s.logger.Error("Failed to create SummarizerJob",
			zap.Uint("jobId", job.ID),
			zap.Error(err),
		)
		return err
	}

	err = enqueueComplianceSummarizerJobs(s.db, s.summarizerJobQueue, job, scheduleJobID)
	if err != nil {
		SummarizerJobsCount.WithLabelValues("failure").Inc()
		s.logger.Error("Failed to enqueue SummarizerJob",
			zap.Uint("jobId", job.ID),
			zap.Error(err),
		)
		job.Status = summarizerapi.SummarizerJobFailed
		err = s.db.UpdateSummarizerJobStatus(job)
		if err != nil {
			s.logger.Error("Failed to update SummarizerJob status",
				zap.Uint("jobId", job.ID),
				zap.Error(err),
			)
		}
		return err
	}

	return nil
}

func enqueueComplianceSummarizerJobs(db Database, q queue.Interface, job SummarizerJob, scheduleJobID uint) error {
	var lastDayJobID, lastWeekJobID, lastQuarterJobID, lastYearJobID uint

	lastDay, err := db.GetOldCompletedScheduleJob(1)
	if err != nil {
		return err
	}
	if lastDay != nil {
		lastDayJobID = lastDay.ID
	}
	lastWeek, err := db.GetOldCompletedScheduleJob(7)
	if err != nil {
		return err
	}
	if lastWeek != nil {
		lastWeekJobID = lastWeek.ID
	}
	lastQuarter, err := db.GetOldCompletedScheduleJob(93)
	if err != nil {
		return err
	}
	if lastQuarter != nil {
		lastQuarterJobID = lastQuarter.ID
	}
	lastYear, err := db.GetOldCompletedScheduleJob(428)
	if err != nil {
		return err
	}
	if lastYear != nil {
		lastYearJobID = lastYear.ID
	}

	if err := q.Publish(summarizer.ComplianceJob{
		JobID:                    job.ID,
		ScheduleJobID:            scheduleJobID,
		LastDayScheduleJobID:     lastDayJobID,
		LastWeekScheduleJobID:    lastWeekJobID,
		LastQuarterScheduleJobID: lastQuarterJobID,
		LastYearScheduleJobID:    lastYearJobID,
		JobType:                  summarizer.JobType_ComplianceSummarizer,
	}); err != nil {
		return err
	}

	return nil
}

func enqueueInsightJobs(db Database, q queue.Interface, job InsightJob) error {
	ins, err := db.GetInsight(job.InsightID)
	if err != nil {
		return err
	}

	var lastDayJobID, lastWeekJobID, lastMonthJobID, lastQuarterJobID, lastYearJobID uint

	lastDay, err := db.GetOldCompletedInsightJob(job.InsightID, 1)
	if err != nil {
		return err
	}
	if lastDay != nil {
		lastDayJobID = lastDay.ID
	}

	lastWeek, err := db.GetOldCompletedInsightJob(job.InsightID, 7)
	if err != nil {
		return err
	}
	if lastWeek != nil {
		lastWeekJobID = lastWeek.ID
	}

	lastMonth, err := db.GetOldCompletedInsightJob(job.InsightID, 30)
	if err != nil {
		return err
	}
	if lastMonth != nil {
		lastMonthJobID = lastMonth.ID
	}

	lastQuarter, err := db.GetOldCompletedInsightJob(job.InsightID, 93)
	if err != nil {
		return err
	}
	if lastQuarter != nil {
		lastQuarterJobID = lastQuarter.ID
	}

	lastYear, err := db.GetOldCompletedInsightJob(job.InsightID, 428)
	if err != nil {
		return err
	}
	if lastYear != nil {
		lastYearJobID = lastYear.ID
	}

	sourceType, err := source.ParseType(ins.Provider)
	if err != nil {
		return err
	}

	if err := q.Publish(insight.Job{
		JobID:            job.ID,
		QueryID:          job.InsightID,
		SmartQueryID:     ins.SmartQueryID,
		SourceID:         job.SourceID,
		ScheduleJobUUID:  job.ScheduleUUID,
		AccountID:        job.AccountID,
		SourceType:       sourceType,
		Internal:         ins.Internal,
		Query:            ins.Query,
		Description:      ins.Description,
		Category:         ins.Category,
		ExecutedAt:       job.CreatedAt.UnixMilli(),
		LastDayJobID:     lastDayJobID,
		LastWeekJobID:    lastWeekJobID,
		LastMonthJobID:   lastMonthJobID,
		LastQuarterJobID: lastQuarterJobID,
		LastYearJobID:    lastYearJobID,
	}); err != nil {
		return err
	}
	return nil
}

func enqueueCheckupJobs(_ Database, q queue.Interface, job CheckupJob) error {
	if err := q.Publish(checkup.Job{
		JobID:      job.ID,
		ExecutedAt: job.CreatedAt.UnixMilli(),
	}); err != nil {
		return err
	}
	return nil
}

// RunInsightJobResultsConsumer consumes messages from the insightJobResultQueue queue.
// It will update the status of the jobs in the database based on the message.
// It will also update the jobs status that are not completed in certain time to FAILED
func (s *Scheduler) RunInsightJobResultsConsumer() error {
	s.logger.Info("Consuming messages from the InsightJobResultQueue queue")

	msgs, err := s.insightJobResultQueue.Consume()
	if err != nil {
		return err
	}

	t := time.NewTicker(JobTimeoutCheckInterval)
	defer t.Stop()

	for {
		select {
		case msg, ok := <-msgs:
			if !ok {
				return fmt.Errorf("tasks channel is closed")
			}

			var result insight.JobResult
			if err := json.Unmarshal(msg.Body, &result); err != nil {
				s.logger.Error("Failed to unmarshal InsightJobResult results", zap.Error(err))
				err = msg.Nack(false, false)
				if err != nil {
					s.logger.Error("Failed nacking message", zap.Error(err))
				}
				continue
			}

			s.logger.Info("Processing InsightJobResult for Job",
				zap.Uint("jobId", result.JobID),
				zap.String("status", string(result.Status)),
			)
			err := s.db.UpdateInsightJob(result.JobID, result.Status, result.Error)
			if err != nil {
				s.logger.Error("Failed to update the status of InsightJob",
					zap.Uint("jobId", result.JobID),
					zap.Error(err))
				err = msg.Nack(false, true)
				if err != nil {
					s.logger.Error("Failed nacking message", zap.Error(err))
				}
				continue
			}

			if err := msg.Ack(false); err != nil {
				s.logger.Error("Failed acking message", zap.Error(err))
			}
		case <-t.C:
			err := s.db.UpdateInsightJobsTimedOut(s.insightIntervalHours)
			if err != nil {
				s.logger.Error("Failed to update timed out InsightJob", zap.Error(err))
			}
		}
	}
}

// RunCheckupJobResultsConsumer consumes messages from the checkupJobResultQueue queue.
// It will update the status of the jobs in the database based on the message.
// It will also update the jobs status that are not completed in certain time to FAILED
func (s *Scheduler) RunCheckupJobResultsConsumer() error {
	s.logger.Info("Consuming messages from the CheckupJobResultQueue queue")

	msgs, err := s.checkupJobResultQueue.Consume()
	if err != nil {
		return err
	}

	t := time.NewTicker(JobTimeoutCheckInterval)
	defer t.Stop()

	for {
		select {
		case msg, ok := <-msgs:
			if !ok {
				return fmt.Errorf("tasks channel is closed")
			}

			var result checkup.JobResult
			if err := json.Unmarshal(msg.Body, &result); err != nil {
				s.logger.Error("Failed to unmarshal CheckupJobResult results", zap.Error(err))
				err = msg.Nack(false, false)
				if err != nil {
					s.logger.Error("Failed nacking message", zap.Error(err))
				}
				continue
			}

			s.logger.Info("Processing CheckupJobResult for Job",
				zap.Uint("jobId", result.JobID),
				zap.String("status", string(result.Status)),
			)
			err := s.db.UpdateCheckupJob(result.JobID, result.Status, result.Error)
			if err != nil {
				s.logger.Error("Failed to update the status of CheckupJob",
					zap.Uint("jobId", result.JobID),
					zap.Error(err))
				err = msg.Nack(false, true)
				if err != nil {
					s.logger.Error("Failed nacking message", zap.Error(err))
				}
				continue
			}

			if err := msg.Ack(false); err != nil {
				s.logger.Error("Failed acking message", zap.Error(err))
			}
		case <-t.C:
			err := s.db.UpdateCheckupJobsTimedOut(s.checkupIntervalHours)
			if err != nil {
				s.logger.Error("Failed to update timed out CheckupJob", zap.Error(err))
			}
		}
	}
}

// RunSummarizerJobResultsConsumer consumes messages from the summarizerJobResultQueue queue.
// It will update the status of the jobs in the database based on the message.
// It will also update the jobs status that are not completed in certain time to FAILED
func (s *Scheduler) RunSummarizerJobResultsConsumer() error {
	s.logger.Info("Consuming messages from the summarizerJobResultQueue queue")

	msgs, err := s.summarizerJobResultQueue.Consume()
	if err != nil {
		return err
	}

	t := time.NewTicker(JobTimeoutCheckInterval)
	defer t.Stop()

	for {
		select {
		case msg, ok := <-msgs:
			if !ok {
				return fmt.Errorf("tasks channel is closed")
			}

			var result summarizer.ResourceJobResult
			if err := json.Unmarshal(msg.Body, &result); err != nil {
				s.logger.Error("Failed to unmarshal SummarizerJobResult results", zap.Error(err))
				err = msg.Nack(false, false)
				if err != nil {
					s.logger.Error("Failed nacking message", zap.Error(err))
				}
				continue
			}

			if result.JobType == "" || result.JobType == summarizer.JobType_ResourceSummarizer {
				s.logger.Info("Processing SummarizerJobResult for Job",
					zap.Uint("jobId", result.JobID),
					zap.String("status", string(result.Status)),
				)
				err := s.db.UpdateSummarizerJob(result.JobID, result.Status, result.Error)
				if err != nil {
					s.logger.Error("Failed to update the status of SummarizerJob",
						zap.Uint("jobId", result.JobID),
						zap.Error(err))
					err = msg.Nack(false, true)
					if err != nil {
						s.logger.Error("Failed nacking message", zap.Error(err))
					}
					continue
				}
			} else {
				var result summarizer.ComplianceJobResult
				if err := json.Unmarshal(msg.Body, &result); err != nil {
					s.logger.Error("Failed to unmarshal SummarizerJobResult results", zap.Error(err))
					err = msg.Nack(false, false)
					if err != nil {
						s.logger.Error("Failed nacking message", zap.Error(err))
					}
					continue
				}

				s.logger.Info("Processing SummarizerJobResult for Job",
					zap.Uint("jobId", result.JobID),
					zap.String("status", string(result.Status)),
				)
				err := s.db.UpdateSummarizerJob(result.JobID, result.Status, result.Error)
				if err != nil {
					s.logger.Error("Failed to update the status of SummarizerJob",
						zap.Uint("jobId", result.JobID),
						zap.Error(err))
					err = msg.Nack(false, true)
					if err != nil {
						s.logger.Error("Failed nacking message", zap.Error(err))
					}
					continue
				}
			}

			if err := msg.Ack(false); err != nil {
				s.logger.Error("Failed acking message", zap.Error(err))
			}
		case <-t.C:
			err := s.db.UpdateSummarizerJobsTimedOut(s.summarizerIntervalHours)
			if err != nil {
				s.logger.Error("Failed to update timed out SummarizerJob", zap.Error(err))
			}
		}
	}
}

func newInsightJob(insight Insight, src Source, scheduleUUID string) InsightJob {
	srcType, _ := source.ParseType(string(src.Type))
	return InsightJob{
		InsightID:      insight.ID,
		SourceID:       src.ID.String(),
		AccountID:      src.AccountID,
		ScheduleUUID:   scheduleUUID,
		SourceType:     srcType,
		Status:         insightapi.InsightJobInProgress,
		FailureMessage: "",
	}
}

func newCheckupJob() CheckupJob {
	return CheckupJob{
		Status: checkupapi.CheckupJobInProgress,
	}
}

func newSummarizerJob(jobType summarizer.JobType, scheduleJobID uint) SummarizerJob {
	return SummarizerJob{
		Model:          gorm.Model{},
		ScheduleJobID:  scheduleJobID,
		Status:         summarizerapi.SummarizerJobInProgress,
		JobType:        jobType,
		FailureMessage: "",
	}
}
