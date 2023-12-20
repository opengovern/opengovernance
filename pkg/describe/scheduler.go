package describe

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/config"
	config2 "github.com/kaytu-io/kaytu-engine/pkg/describe/config"
	"github.com/kaytu-io/kaytu-engine/pkg/describe/db"
	"github.com/kaytu-io/kaytu-engine/pkg/describe/db/model"
	"github.com/kaytu-io/kaytu-engine/pkg/describe/schedulers/compliance"
	"github.com/kaytu-io/kaytu-engine/pkg/httpclient"
	"github.com/kaytu-io/kaytu-engine/pkg/httpserver"
	inventoryClient "github.com/kaytu-io/kaytu-engine/pkg/inventory/client"
	"github.com/kaytu-io/kaytu-engine/pkg/utils"
	"github.com/kaytu-io/kaytu-util/pkg/kaytu-es-sdk"
	"github.com/kaytu-io/kaytu-util/pkg/ticker"
	"net"
	"strconv"
	"strings"
	"time"

	envoyauth "github.com/envoyproxy/go-control-plane/envoy/service/auth/v3"
	"github.com/kaytu-io/kaytu-engine/pkg/metadata/models"
	"github.com/kaytu-io/kaytu-util/pkg/postgres"
	"github.com/kaytu-io/kaytu-util/pkg/queue"
	"github.com/kaytu-io/kaytu-util/proto/src/golang"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	confluent_kafka "github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/go-redis/redis/v8"
	api2 "github.com/kaytu-io/kaytu-engine/pkg/auth/api"
	"github.com/kaytu-io/kaytu-engine/pkg/checkup"
	checkupapi "github.com/kaytu-io/kaytu-engine/pkg/checkup/api"
	"github.com/kaytu-io/kaytu-engine/pkg/compliance/client"
	metadataClient "github.com/kaytu-io/kaytu-engine/pkg/metadata/client"
	onboardClient "github.com/kaytu-io/kaytu-engine/pkg/onboard/client"
	workspaceClient "github.com/kaytu-io/kaytu-engine/pkg/workspace/client"

	"go.uber.org/zap"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

	"github.com/aws/aws-sdk-go-v2/service/lambda"
)

const (
	JobSchedulingInterval    = 1 * time.Minute
	JobSequencerInterval     = 1 * time.Minute
	JobTimeoutCheckInterval  = 1 * time.Minute
	MaxJobInQueue            = 10000
	ConcurrentDeletedSources = 1000

	schedulerConsumerGroup = "describe-scheduler"

	RedisKeyWorkspaceResourceRemaining = "workspace_resource_remaining"
)

var DescribePublishingBlocked = promauto.NewGaugeVec(prometheus.GaugeOpts{
	Namespace: "kaytu",
	Subsystem: "scheduler",
	Name:      "queue_job_publishing_blocked",
	Help:      "The gauge whether publishing tasks to a queue is blocked: 0 for resumed and 1 for blocked",
}, []string{"queue_name"})

var InsightJobsCount = promauto.NewCounterVec(prometheus.CounterOpts{
	Namespace: "kaytu",
	Subsystem: "scheduler",
	Name:      "schedule_insight_jobs_total",
	Help:      "Count of insight jobs in scheduler service",
}, []string{"status"})

var CheckupJobsCount = promauto.NewCounterVec(prometheus.CounterOpts{
	Namespace: "kaytu",
	Subsystem: "scheduler",
	Name:      "schedule_checkup_jobs_total",
	Help:      "Count of checkup jobs in scheduler service",
}, []string{"status"})

var AnalyticsJobsCount = promauto.NewCounterVec(prometheus.CounterOpts{
	Namespace: "kaytu",
	Subsystem: "scheduler",
	Name:      "schedule_analytics_jobs_total",
	Help:      "Count of analytics jobs in scheduler service",
}, []string{"status"})

var AnalyticsJobResultsCount = promauto.NewCounterVec(prometheus.CounterOpts{
	Namespace: "kaytu",
	Subsystem: "scheduler",
	Name:      "schedule_analytics_job_results_total",
	Help:      "Count of analytics job results in scheduler service",
}, []string{"status"})

var LargeDescribeResourceMessage = promauto.NewCounterVec(prometheus.CounterOpts{
	Name: "kaytu_scheduler_large_describe_resource_message",
	Help: "The gauge whether the describe resource message is too large: 0 for not large and 1 for large",
}, []string{"resource_type"})

type OperationMode string

const (
	OperationModeScheduler OperationMode = "scheduler"
	OperationModeReceiver  OperationMode = "receiver"
)

type Scheduler struct {
	id         string
	db         db.Database
	httpServer *HttpServer
	grpcServer *grpc.Server

	// describeJobResultQueue is used to consume the describe job results returned by the workers.
	describeJobResultQueue queue.Interface

	// sourceQueue is used to consume source updates by the onboarding service.
	sourceQueue queue.Interface

	// insightJobQueue is used to publish insight jobs to be performed by the workers.
	insightJobQueue queue.Interface
	// insightJobResultQueue is used to consume the insight job results returned by the workers.
	insightJobResultQueue queue.Interface

	// checkupJobQueue is used to publish checkup jobs to be performed by the workers.
	checkupJobQueue queue.Interface
	// checkupJobResultQueue is used to consume the checkup job results returned by the workers.
	checkupJobResultQueue queue.Interface

	describeIntervalHours      time.Duration
	fullDiscoveryIntervalHours time.Duration
	costDiscoveryIntervalHours time.Duration
	describeTimeoutHours       int64
	insightIntervalHours       time.Duration
	checkupIntervalHours       int64
	mustSummarizeIntervalHours int64
	analyticsIntervalHours     time.Duration
	complianceIntervalHours    time.Duration

	logger              *zap.Logger
	workspaceClient     workspaceClient.WorkspaceServiceClient
	metadataClient      metadataClient.MetadataServiceClient
	complianceClient    client.ComplianceServiceClient
	onboardClient       onboardClient.OnboardServiceClient
	inventoryClient     inventoryClient.InventoryServiceClient
	authGrpcClient      envoyauth.AuthorizationClient
	es                  kaytu.Client
	rdb                 *redis.Client
	kafkaProducer       *confluent_kafka.Producer
	kafkaResourcesTopic string
	kafkaConsumer       *confluent_kafka.Consumer
	kafkaServers        []string

	describeEndpoint string
	keyARN           string
	keyRegion        string

	WorkspaceName string

	DoDeleteOldResources bool
	OperationMode        OperationMode
	MaxConcurrentCall    int64

	LambdaClient *lambda.Client

	complianceScheduler *compliance.JobScheduler
	conf                config2.SchedulerConfig
}

func initRabbitQueue(queueName string) (queue.Interface, error) {
	qCfg := queue.Config{}
	qCfg.Server.Username = RabbitMQUsername
	qCfg.Server.Password = RabbitMQPassword
	qCfg.Server.Host = RabbitMQService
	qCfg.Server.Port = RabbitMQPort
	qCfg.Queue.Name = queueName
	qCfg.Queue.Durable = true
	qCfg.Producer.ID = "describe-scheduler"
	insightQueue, err := queue.New(qCfg)
	if err != nil {
		return nil, err
	}

	return insightQueue, nil
}

func InitializeScheduler(
	id string,
	conf config2.SchedulerConfig,
	insightJobQueueName string,
	insightJobResultQueueName string,
	checkupJobQueueName string,
	checkupJobResultQueueName string,
	sourceQueueName string,
	postgresUsername string,
	postgresPassword string,
	postgresHost string,
	postgresPort string,
	postgresDb string,
	postgresSSLMode string,
	httpServerAddress string,
	describeTimeoutHours string,
	checkupIntervalHours string,
	mustSummarizeIntervalHours string,
	kaytuHelmChartLocation string,
	fluxSystemNamespace string,
) (s *Scheduler, err error) {
	if id == "" {
		return nil, fmt.Errorf("'id' must be set to a non empty string")
	}

	s = &Scheduler{
		id:                  id,
		OperationMode:       OperationMode(OperationModeConfig),
		describeEndpoint:    DescribeDeliverEndpoint,
		keyARN:              KeyARN,
		keyRegion:           KeyRegion,
		kafkaResourcesTopic: conf.Kafka.Topic,
	}
	defer func() {
		if err != nil && s != nil {
			s.Stop()
		}
	}()

	lambdaCfg, err := config.LoadDefaultConfig(context.Background())
	lambdaCfg.Region = KeyRegion

	s.conf = conf
	s.LambdaClient = lambda.NewFromConfig(lambdaCfg)

	s.logger, err = zap.NewProduction()
	if err != nil {
		return nil, err
	}

	s.logger.Info("Initializing the scheduler")

	s.insightJobQueue, err = initRabbitQueue(insightJobQueueName)
	if err != nil {
		s.logger.Error("failed to init rabbit queue", zap.Error(err), zap.String("queue_name", insightJobQueueName))
		return nil, err
	}

	s.insightJobResultQueue, err = initRabbitQueue(insightJobResultQueueName)
	if err != nil {
		return nil, err
	}

	s.checkupJobQueue, err = initRabbitQueue(checkupJobQueueName)
	if err != nil {
		return nil, err
	}

	s.checkupJobResultQueue, err = initRabbitQueue(checkupJobResultQueueName)
	if err != nil {
		return nil, err
	}

	s.sourceQueue, err = initRabbitQueue(sourceQueueName)
	if err != nil {
		return nil, err
	}

	cfg := postgres.Config{
		Host:    postgresHost,
		Port:    postgresPort,
		User:    postgresUsername,
		Passwd:  postgresPassword,
		DB:      postgresDb,
		SSLMode: postgresSSLMode,
	}

	if s.OperationMode == OperationModeScheduler {
		cfg.Connection.MaxOpen = 50
		cfg.Connection.MaxIdle = 20
	}

	orm, err := postgres.NewClient(&cfg, s.logger)
	if err != nil {
		return nil, fmt.Errorf("new postgres client: %w", err)
	}

	s.logger.Info("Connected to the postgres database: ", zap.String("db", postgresDb))
	s.db = db.Database{ORM: orm}

	s.es, err = kaytu.NewClient(kaytu.ClientConfig{
		Addresses:     []string{conf.ElasticSearch.Address},
		Username:      &conf.ElasticSearch.Username,
		Password:      &conf.ElasticSearch.Password,
		IsOpenSearch:  &conf.ElasticSearch.IsOpenSearch,
		AwsRegion:     &conf.ElasticSearch.AwsRegion,
		AssumeRoleArn: &conf.ElasticSearch.AssumeRoleArn,
	})
	if err != nil {
		return nil, err
	}

	s.kafkaServers = strings.Split(conf.Kafka.Addresses, ",")

	kafkaProducer, err := newKafkaProducer(s.kafkaServers)
	if err != nil {
		return nil, err
	}
	s.kafkaProducer = kafkaProducer
	kafkaResourceSinkConsumer, err := newKafkaConsumer(s.kafkaServers, s.kafkaResourcesTopic)
	if err != nil {
		return nil, err
	}
	s.kafkaConsumer = kafkaResourceSinkConsumer

	helmConfig := HelmConfig{
		KaytuHelmChartLocation: kaytuHelmChartLocation,
		FluxSystemNamespace:    fluxSystemNamespace,
	}
	s.httpServer = NewHTTPServer(httpServerAddress, s.db, s, helmConfig)

	describeIntervalHours, err := strconv.ParseInt(DescribeIntervalHours, 10, 64)
	if err != nil {
		return nil, err
	}
	s.describeIntervalHours = time.Duration(describeIntervalHours) * time.Hour

	fullDiscoveryIntervalHours, err := strconv.ParseInt(FullDiscoveryIntervalHours, 10, 64)
	if err != nil {
		return nil, err
	}
	s.fullDiscoveryIntervalHours = time.Duration(fullDiscoveryIntervalHours) * time.Hour

	costDiscoveryIntervalHours, err := strconv.ParseInt(CostDiscoveryIntervalHours, 10, 64)
	if err != nil {
		return nil, err
	}
	s.costDiscoveryIntervalHours = time.Duration(costDiscoveryIntervalHours) * time.Hour

	s.describeTimeoutHours, err = strconv.ParseInt(describeTimeoutHours, 10, 64)
	if err != nil {
		return nil, err
	}

	insightIntervalHours, err := strconv.ParseInt(InsightIntervalHours, 10, 64)
	if err != nil {
		return nil, err
	}
	s.insightIntervalHours = time.Duration(insightIntervalHours) * time.Hour

	s.checkupIntervalHours, err = strconv.ParseInt(checkupIntervalHours, 10, 64)
	if err != nil {
		return nil, err
	}
	s.mustSummarizeIntervalHours, err = strconv.ParseInt(mustSummarizeIntervalHours, 10, 64)
	if err != nil {
		return nil, err
	}

	analyticsIntervalHours, err := strconv.ParseInt(AnalyticsIntervalHours, 10, 64)
	if err != nil {
		return nil, err
	}
	s.analyticsIntervalHours = time.Duration(analyticsIntervalHours) * time.Hour

	s.complianceIntervalHours = time.Duration(conf.ComplianceIntervalHours) * time.Hour

	s.metadataClient = metadataClient.NewMetadataServiceClient(MetadataBaseURL)
	s.workspaceClient = workspaceClient.NewWorkspaceClient(WorkspaceBaseURL)
	s.complianceClient = client.NewComplianceClient(ComplianceBaseURL)
	s.onboardClient = onboardClient.NewOnboardServiceClient(OnboardBaseURL, nil)
	s.inventoryClient = inventoryClient.NewInventoryServiceClient(InventoryBaseURL)
	authGRPCConn, err := grpc.Dial(AuthGRPCURI, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{InsecureSkipVerify: true})))
	if err != nil {
		return nil, err
	}
	s.authGrpcClient = envoyauth.NewAuthorizationClient(authGRPCConn)

	s.rdb = redis.NewClient(&redis.Options{
		Addr:     RedisAddress,
		Password: "", // no password set
		DB:       0,  // use default DB
	})

	describeServer := NewDescribeServer(s.db, s.rdb, s.kafkaProducer, s.kafkaResourcesTopic, s.authGrpcClient, s.logger, conf)
	s.grpcServer = grpc.NewServer(
		grpc.MaxRecvMsgSize(128*1024*1024),
		grpc.UnaryInterceptor(describeServer.grpcUnaryAuthInterceptor),
		grpc.StreamInterceptor(describeServer.grpcStreamAuthInterceptor),
	)
	golang.RegisterDescribeServiceServer(s.grpcServer, describeServer)

	workspace, err := s.workspaceClient.GetByID(&httpclient.Context{
		UserRole: api2.EditorRole,
	}, CurrentWorkspaceID)
	if err != nil {
		return nil, err
	}
	s.WorkspaceName = workspace.Name

	s.DoDeleteOldResources, _ = strconv.ParseBool(DoDeleteOldResources)
	describeServer.DoProcessReceivedMessages, _ = strconv.ParseBool(DoProcessReceivedMsgs)
	s.MaxConcurrentCall, _ = strconv.ParseInt(MaxConcurrentCall, 10, 64)
	if s.MaxConcurrentCall <= 0 {
		s.MaxConcurrentCall = 5000
	}

	s.complianceScheduler = compliance.New(
		conf,
		s.logger,
		s.complianceClient,
		s.onboardClient,
		s.db,
		s.kafkaProducer,
		s.es,
		s.complianceIntervalHours,
	)
	return s, nil
}

func (s *Scheduler) Run(ctx context.Context) error {
	err := s.db.Initialize()
	if err != nil {
		return err
	}

	httpctx := &httpclient.Context{
		UserRole: api2.ViewerRole,
	}
	describeJobIntM, err := s.metadataClient.GetConfigMetadata(httpctx, models.MetadataKeyDescribeJobInterval)
	if err != nil {
		s.logger.Error("failed to set describe interval due to error", zap.Error(err))
	} else {
		if v, ok := describeJobIntM.GetValue().(int); ok {
			s.describeIntervalHours = time.Duration(v) * time.Hour
			s.logger.Info("set describe interval", zap.Int64("interval", int64(s.describeIntervalHours.Hours())))
		} else {
			s.logger.Error("failed to set describe interval due to invalid type", zap.String("type", string(describeJobIntM.GetType())))
		}
	}

	fullDiscoveryJobIntM, err := s.metadataClient.GetConfigMetadata(httpctx, models.MetadataKeyFullDiscoveryJobInterval)
	if err != nil {
		s.logger.Error("failed to set describe interval due to error", zap.Error(err))
	} else {
		if v, ok := fullDiscoveryJobIntM.GetValue().(int); ok {
			s.fullDiscoveryIntervalHours = time.Duration(v) * time.Hour
			s.logger.Info("set describe interval", zap.Int64("interval", int64(s.fullDiscoveryIntervalHours.Hours())))
		} else {
			s.logger.Error("failed to set describe interval due to invalid type", zap.String("type", string(fullDiscoveryJobIntM.GetType())))
		}
	}

	costDiscoveryJobIntM, err := s.metadataClient.GetConfigMetadata(httpctx, models.MetadataKeyCostDiscoveryJobInterval)
	if err != nil {
		s.logger.Error("failed to set describe interval due to error", zap.Error(err))
	} else {
		if v, ok := costDiscoveryJobIntM.GetValue().(int); ok {
			s.costDiscoveryIntervalHours = time.Duration(v) * time.Hour
			s.logger.Info("set describe interval", zap.Int64("interval", int64(s.costDiscoveryIntervalHours.Hours())))
		} else {
			s.logger.Error("failed to set describe interval due to invalid type", zap.String("type", string(costDiscoveryJobIntM.GetType())))
		}
	}

	insightJobIntM, err := s.metadataClient.GetConfigMetadata(httpctx, models.MetadataKeyInsightJobInterval)
	if err != nil {
		s.logger.Error("failed to set describe interval due to error", zap.Error(err))
	} else {
		if v, ok := insightJobIntM.GetValue().(int); ok {
			s.insightIntervalHours = time.Duration(v) * time.Hour
			s.logger.Info("set insight interval", zap.Int64("interval", int64(s.insightIntervalHours.Hours())))
		} else {
			s.logger.Error("failed to set insight interval due to invalid type", zap.String("type", string(insightJobIntM.GetType())))
		}
	}

	analyticsJobIntM, err := s.metadataClient.GetConfigMetadata(httpctx, models.MetadataKeyMetricsJobInterval)
	if err != nil {
		s.logger.Error("failed to set describe interval due to error", zap.Error(err))
	} else {
		if v, ok := analyticsJobIntM.GetValue().(int); ok {
			s.analyticsIntervalHours = time.Duration(v) * time.Hour
			s.logger.Info("set insight interval", zap.Int64("interval", int64(s.analyticsIntervalHours.Hours())))
		} else {
			s.logger.Error("failed to set insight interval due to invalid type", zap.String("type", string(analyticsJobIntM.GetType())))
		}
	}

	complianceJobIntM, err := s.metadataClient.GetConfigMetadata(httpctx, models.MetadataKeyComplianceJobInterval)
	if err != nil {
		s.logger.Error("failed to set describe interval due to error", zap.Error(err))
	} else {
		if v, ok := complianceJobIntM.GetValue().(int); ok {
			s.complianceIntervalHours = time.Duration(v) * time.Hour
			s.logger.Info("set insight interval", zap.Int64("interval", int64(s.complianceIntervalHours.Hours())))
		} else {
			s.logger.Error("failed to set insight interval due to invalid type", zap.String("type", string(complianceJobIntM.GetType())))
		}
	}

	switch s.OperationMode {
	case OperationModeScheduler:
		s.logger.Info("starting scheduler")
		// --------- describe
		utils.EnsureRunGoroutin(func() {
			s.RunDescribeJobScheduler()
		})
		utils.EnsureRunGoroutin(func() {
			s.RunDescribeResourceJobs(ctx)
		})
		// ---------

		// --------- describe
		utils.EnsureRunGoroutin(func() {
			s.RunStackScheduler()
		})
		// ---------

		// --------- inventory summarizer
		utils.EnsureRunGoroutin(func() {
			s.RunAnalyticsJobScheduler()
		})

		utils.EnsureRunGoroutin(func() {
			s.logger.Fatal("AnalyticsJobResult consumer exited", zap.Error(s.RunAnalyticsJobResultsConsumer()))
		})
		// ---------

		// --------- compliance
		s.complianceScheduler.Run()
		utils.EnsureRunGoroutin(func() {
			s.RunJobSequencer()
		})
		// ---------

		// --------- insights
		utils.EnsureRunGoroutin(func() {
			s.RunInsightJobScheduler()
		})
		utils.EnsureRunGoroutin(func() {
			s.logger.Fatal("InsightJobResult consumer exited", zap.Error(s.RunInsightJobResultsConsumer()))
		})
		// ---------

		//EnsureRunGoroutin(func() {
		//	s.RunScheduleJobCompletionUpdater()
		//})

		utils.EnsureRunGoroutin(func() {
			s.RunCheckupJobScheduler()
		})
		utils.EnsureRunGoroutin(func() {
			s.RunDisabledConnectionCleanup()
		})
		utils.EnsureRunGoroutin(func() {
			s.logger.Fatal("SourceEvents consumer exited", zap.Error(s.RunSourceEventsConsumer()))
		})
		utils.EnsureRunGoroutin(func() {
			s.logger.Fatal("InsightJobResult consumer exited", zap.Error(s.RunCheckupJobResultsConsumer()))
		})
		utils.EnsureRunGoroutin(func() {
			s.RunScheduledJobCleanup()
		})
		utils.EnsureRunGoroutin(func() {
			s.UpdateDescribedResourceCountScheduler()
		})
	case OperationModeReceiver:
		utils.EnsureRunGoroutin(func() {
			s.logger.Fatal("DescribeJobResults consumer exited", zap.Error(s.RunDescribeJobResultsConsumer()))
		})
		s.logger.Info("starting receiver")
		lis, err := net.Listen("tcp", GRPCServerAddress)
		if err != nil {
			s.logger.Fatal("failed to listen on grpc port", zap.Error(err))
		}
		go func() {
			err := s.grpcServer.Serve(lis)
			if err != nil {
				s.logger.Fatal("failed to serve grpc server", zap.Error(err))
			}
		}()
	}

	return httpserver.RegisterAndStart(s.logger, s.httpServer.Address, s.httpServer)
}

func (s *Scheduler) RunDisabledConnectionCleanup() {
	ticker := ticker.NewTicker(time.Hour, time.Second*10)
	defer ticker.Stop()

	for range ticker.C {
		connections, err := s.onboardClient.ListSources(&httpclient.Context{UserRole: api2.InternalRole}, nil)
		if err != nil {
			s.logger.Error("Failed to list sources", zap.Error(err))
			continue
		}
		disabledConnectionIds := make([]string, 0)
		for _, connection := range connections {
			if connection.IsEnabled() {
				continue
			}
			disabledConnectionIds = append(disabledConnectionIds, connection.ID.String())
		}

		if len(disabledConnectionIds) > 0 {
			s.cleanupDescribeResourcesForConnections(disabledConnectionIds)
		}

	}
}

func (s *Scheduler) RunScheduledJobCleanup() {
	ticker := ticker.NewTicker(time.Hour, time.Second*10)
	defer ticker.Stop()
	for range ticker.C {
		tOlder := time.Now().AddDate(0, 0, -7)
		err := s.db.CleanupDescribeConnectionJobsOlderThan(tOlder)
		if err != nil {
			s.logger.Error("Failed to cleanup describe resource jobs", zap.Error(err))
		}
		err = s.db.CleanupInsightJobsOlderThan(tOlder)
		if err != nil {
			s.logger.Error("Failed to cleanup insight jobs", zap.Error(err))
		}
		err = s.db.CleanupComplianceJobsOlderThan(tOlder)
		if err != nil {
			s.logger.Error("Failed to cleanup compliance report jobs", zap.Error(err))
		}
	}
}

// RunSourceEventsConsumer Consume events from the source queue. Based on the action of the event,
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

		if err := msg.Ack(false); err != nil {
			s.logger.Error("Failed acking message", zap.Error(err))
		}
	}

	return fmt.Errorf("source events queue channel is closed")
}

func (s *Scheduler) Stop() {
	queues := []queue.Interface{
		s.sourceQueue,
		s.insightJobQueue,
		s.insightJobResultQueue,
	}

	for _, openQueues := range queues {
		openQueues.Close()
	}
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

func (s *Scheduler) RunCheckupJobScheduler() {
	s.logger.Info("Scheduling insight jobs on a timer")

	t := ticker.NewTicker(JobSchedulingInterval, time.Second*10)
	defer t.Stop()

	for ; ; <-t.C {
		s.scheduleCheckupJob()
	}
}

func (s *Scheduler) scheduleCheckupJob() {
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

func enqueueCheckupJobs(_ db.Database, q queue.Interface, job model.CheckupJob) error {
	if err := q.Publish(checkup.Job{
		JobID:      job.ID,
		ExecutedAt: job.CreatedAt.UnixMilli(),
	}); err != nil {
		return err
	}
	return nil
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

	t := ticker.NewTicker(JobTimeoutCheckInterval, time.Second*10)
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

func newCheckupJob() model.CheckupJob {
	return model.CheckupJob{
		Status: checkupapi.CheckupJobInProgress,
	}
}

func newKafkaProducer(brokers []string) (*confluent_kafka.Producer, error) {
	return confluent_kafka.NewProducer(&confluent_kafka.ConfigMap{
		"bootstrap.servers":            strings.Join(brokers, ","),
		"linger.ms":                    100,
		"compression.type":             "lz4",
		"message.timeout.ms":           10000,
		"queue.buffering.max.messages": 100000,
		"message.max.bytes":            104857600,
	})
}

func newKafkaConsumer(brokers []string, topic string) (*confluent_kafka.Consumer, error) {
	consumer, err := confluent_kafka.NewConsumer(&confluent_kafka.ConfigMap{
		"bootstrap.servers":  strings.Join(brokers, ","),
		"group.id":           "describe-receiver",
		"auto.offset.reset":  "earliest",
		"enable.auto.commit": false,
		"fetch.min.bytes":    10000000,
		"fetch.wait.max.ms":  5000,
	})
	if err != nil {
		return nil, err
	}
	err = consumer.Subscribe(topic, nil)
	if err != nil {
		return nil, err
	}
	return consumer, nil
}
