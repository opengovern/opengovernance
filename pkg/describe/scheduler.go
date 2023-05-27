package describe

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	envoyauth "github.com/envoyproxy/go-control-plane/envoy/service/auth/v3"
	"github.com/kaytu-io/kaytu-util/pkg/postgres"
	"github.com/kaytu-io/kaytu-util/pkg/queue"
	"github.com/kaytu-io/kaytu-util/proto/src/golang"
	"gitlab.com/keibiengine/keibi-engine/pkg/internal/httpclient"
	"gitlab.com/keibiengine/keibi-engine/pkg/internal/httpserver"
	"gitlab.com/keibiengine/keibi-engine/pkg/metadata/models"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	confluent_kafka "github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"gitlab.com/keibiengine/keibi-engine/pkg/checkup"
	checkupapi "gitlab.com/keibiengine/keibi-engine/pkg/checkup/api"
	"gitlab.com/keibiengine/keibi-engine/pkg/compliance/client"
	"gitlab.com/keibiengine/keibi-engine/pkg/summarizer"
	summarizerapi "gitlab.com/keibiengine/keibi-engine/pkg/summarizer/api"
	"gorm.io/gorm"

	"github.com/go-redis/redis/v8"
	"github.com/kaytu-io/kaytu-util/pkg/keibi-es-sdk"
	api2 "gitlab.com/keibiengine/keibi-engine/pkg/auth/api"
	metadataClient "gitlab.com/keibiengine/keibi-engine/pkg/metadata/client"
	onboardClient "gitlab.com/keibiengine/keibi-engine/pkg/onboard/client"
	workspaceClient "gitlab.com/keibiengine/keibi-engine/pkg/workspace/client"

	"github.com/kaytu-io/kaytu-util/pkg/source"

	complianceapi "gitlab.com/keibiengine/keibi-engine/pkg/compliance/api"

	complianceworker "gitlab.com/keibiengine/keibi-engine/pkg/compliance/worker"
	"gitlab.com/keibiengine/keibi-engine/pkg/describe/api"
	"go.uber.org/zap"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

const (
	JobCompletionInterval    = 1 * time.Minute
	JobSchedulingInterval    = 1 * time.Minute
	JobTimeoutCheckInterval  = 1 * time.Minute
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

var ComplianceJobsCount = promauto.NewCounterVec(prometheus.CounterOpts{
	Name: "keibi_scheduler_schedule_compliance_job_total",
	Help: "Count of describe jobs in scheduler service",
}, []string{"status"})

var ComplianceSourceJobsCount = promauto.NewCounterVec(prometheus.CounterOpts{
	Name: "keibi_scheduler_schedule_compliance_source_job_total",
	Help: "Count of describe source jobs in scheduler service",
}, []string{"status"})

type OperationMode string

const (
	OperationModeScheduler OperationMode = "scheduler"
	OperationModeReceiver  OperationMode = "receiver"
	OperationModeKafkaSink OperationMode = "kafka-sink"
)

type Scheduler struct {
	id         string
	db         Database
	httpServer *HttpServer
	grpcServer *grpc.Server

	// describeJobResultQueue is used to consume the describe job results returned by the workers.
	describeJobResultQueue queue.Interface

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

	describeIntervalHours      int64
	describeTimeoutHours       int64
	complianceIntervalHours    int64
	complianceTimeoutHours     int64
	insightIntervalHours       int64
	checkupIntervalHours       int64
	summarizerIntervalHours    int64
	mustSummarizeIntervalHours int64

	logger              *zap.Logger
	workspaceClient     workspaceClient.WorkspaceServiceClient
	metadataClient      metadataClient.MetadataServiceClient
	complianceClient    client.ComplianceServiceClient
	onboardClient       onboardClient.OnboardServiceClient
	authGrpcClient      envoyauth.AuthorizationClient
	es                  keibi.Client
	rdb                 *redis.Client
	kafkaProducer       *confluent_kafka.Producer
	kafkaESSink         *KafkaEsSink
	kafkaResourcesTopic string
	kafkaDeletionTopic  string

	describeEndpoint string
	keyARN           string
	keyRegion        string

	cloudNativeAPIBaseURL string
	cloudNativeAPIAuthKey string

	WorkspaceName string

	DoDeleteOldResources bool
	OperationMode        OperationMode
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
	describeJobResultQueueName string,
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
	httpServerAddress string,
	describeIntervalHours string,
	describeTimeoutHours string,
	complianceIntervalHours string,
	complianceTimeoutHours string,
	insightIntervalHours string,
	checkupIntervalHours string,
	mustSummarizeIntervalHours string,
) (s *Scheduler, err error) {
	if id == "" {
		return nil, fmt.Errorf("'id' must be set to a non empty string")
	}

	s = &Scheduler{
		id:                  id,
		deletedSources:      make(chan string, ConcurrentDeletedSources),
		OperationMode:       OperationMode(OperationModeConfig),
		describeEndpoint:    DescribeDeliverEndpoint,
		keyARN:              KeyARN,
		keyRegion:           KeyRegion,
		kafkaResourcesTopic: KafkaResourcesTopic,
		kafkaDeletionTopic:  KafkaDeletionTopic,
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

	s.describeJobResultQueue, err = initRabbitQueue(describeJobResultQueueName)
	if err != nil {
		return nil, err
	}

	s.insightJobQueue, err = initRabbitQueue(insightJobQueueName)
	if err != nil {
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

	s.summarizerJobQueue, err = initRabbitQueue(summarizerJobQueueName)
	if err != nil {
		return nil, err
	}

	s.summarizerJobResultQueue, err = initRabbitQueue(summarizerJobResultQueueName)
	if err != nil {
		return nil, err
	}

	s.complianceReportCleanupJobQueue, err = initRabbitQueue(complianceReportCleanupJobQueueName)
	if err != nil {
		return nil, err
	}

	s.sourceQueue, err = initRabbitQueue(sourceQueueName)
	if err != nil {
		return nil, err
	}

	s.complianceReportJobQueue, err = initRabbitQueue(complianceReportJobQueueName)
	if err != nil {
		return nil, err
	}

	s.complianceReportJobResultQueue, err = initRabbitQueue(complianceReportJobResultQueueName)
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
	orm, err := postgres.NewClient(&cfg, s.logger)
	if err != nil {
		return nil, fmt.Errorf("new postgres client: %w", err)
	}

	s.logger.Info("Connected to the postgres database: ", zap.String("db", postgresDb))
	s.db = Database{orm: orm}

	defaultAccountID := "default"
	s.es, err = keibi.NewClient(keibi.ClientConfig{
		Addresses: []string{ElasticSearchAddress},
		Username:  &ElasticSearchUsername,
		Password:  &ElasticSearchPassword,
		AccountID: &defaultAccountID,
	})
	if err != nil {
		return nil, err
	}

	kafkaProducer, err := newKafkaProducer(strings.Split(KafkaService, ","))
	if err != nil {
		return nil, err
	}
	s.kafkaProducer = kafkaProducer

	kafkaResourceSinkConsumer, err := newKafkaConsumer(strings.Split(KafkaService, ","), s.kafkaResourcesTopic)
	if err != nil {
		return nil, err
	}
	s.kafkaESSink = NewKafkaEsSink(s.logger, kafkaResourceSinkConsumer, s.es)

	s.httpServer = NewHTTPServer(httpServerAddress, s.db, s)

	s.describeIntervalHours, err = strconv.ParseInt(describeIntervalHours, 10, 64)
	if err != nil {
		return nil, err
	}
	s.describeTimeoutHours, err = strconv.ParseInt(describeTimeoutHours, 10, 64)
	if err != nil {
		return nil, err
	}
	s.complianceIntervalHours, err = strconv.ParseInt(complianceIntervalHours, 10, 64)
	if err != nil {
		return nil, err
	}
	s.complianceTimeoutHours, err = strconv.ParseInt(complianceTimeoutHours, 10, 64)
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
	s.mustSummarizeIntervalHours, err = strconv.ParseInt(mustSummarizeIntervalHours, 10, 64)
	if err != nil {
		return nil, err
	}

	s.metadataClient = metadataClient.NewMetadataServiceClient(MetadataBaseURL)
	s.workspaceClient = workspaceClient.NewWorkspaceClient(WorkspaceBaseURL)
	s.complianceClient = client.NewComplianceClient(ComplianceBaseURL)
	s.onboardClient = onboardClient.NewOnboardServiceClient(OnboardBaseURL, nil)
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

	describeServer := NewDescribeServer(s.db, s.rdb, kafkaProducer, s.kafkaResourcesTopic, s.describeJobResultQueue, s.authGrpcClient, s.logger)
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

	return s, nil
}

func (s *Scheduler) Run() error {
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
			s.describeIntervalHours = int64(v * int(time.Minute) / int(time.Hour))
			s.logger.Info("set describe interval", zap.Int64("interval", s.describeIntervalHours))
		} else {
			s.logger.Error("failed to set describe interval due to invalid type", zap.String("type", string(describeJobIntM.GetType())))
		}
	}

	insightJobIntM, err := s.metadataClient.GetConfigMetadata(httpctx, models.MetadataKeyInsightJobInterval)
	if err != nil {
		s.logger.Error("failed to set describe interval due to error", zap.Error(err))
	} else {
		if v, ok := insightJobIntM.GetValue().(int); ok {
			s.insightIntervalHours = int64(v * int(time.Minute) / int(time.Hour))
			s.logger.Info("set insight interval", zap.Int64("interval", s.insightIntervalHours))
		} else {
			s.logger.Error("failed to set insight interval due to invalid type", zap.String("type", string(insightJobIntM.GetType())))
		}
	}

	summarizerJobIntM, err := s.metadataClient.GetConfigMetadata(httpctx, models.MetadataKeyMetricsJobInterval)
	if err != nil {
		s.logger.Error("failed to set describe interval due to error", zap.Error(err))
	} else {
		if v, ok := summarizerJobIntM.GetValue().(int); ok {
			s.summarizerIntervalHours = int64(v * int(time.Minute) / int(time.Hour))
			s.logger.Info("set summarizer interval", zap.Int64("interval", s.summarizerIntervalHours))
		} else {
			s.logger.Error("failed to set summarizer interval due to invalid type", zap.String("type", string(summarizerJobIntM.GetType())))
		}
	}

	switch s.OperationMode {
	case OperationModeScheduler:
		s.logger.Info("starting scheduler")
		// describe
		EnsureRunGoroutin(func() {
			s.RunDescribeJobScheduler()
		})
		EnsureRunGoroutin(func() {
			s.RunDescribeResourceJobs()
		})
		EnsureRunGoroutin(func() {
			s.RunDescribeJobCompletionUpdater()
		})

		// inventory summarizer
		EnsureRunGoroutin(func() {
			s.RunMustSummerizeJobScheduler()
		})
		//

		EnsureRunGoroutin(func() {
			s.logger.Fatal("DescribeJobResults consumer exited", zap.Error(s.RunDescribeJobResultsConsumer()))
		})

		// compliance
		EnsureRunGoroutin(func() {
			s.RunComplianceJobScheduler()
		})
		//

		//EnsureRunGoroutin(func() {
		//	s.RunScheduleJobCompletionUpdater()
		//})

		EnsureRunGoroutin(func() {
			s.RunInsightJobScheduler()
		})
		EnsureRunGoroutin(func() {
			s.logger.Fatal("InsightJobResult consumer exited", zap.Error(s.RunInsightJobResultsConsumer()))
		})

		EnsureRunGoroutin(func() {
			s.RunCheckupJobScheduler()
		})
		EnsureRunGoroutin(func() {
			s.RunDeletedSourceCleanup()
		})
		EnsureRunGoroutin(func() {
			s.logger.Fatal("SourceEvents consumer exited", zap.Error(s.RunSourceEventsConsumer()))
		})
		EnsureRunGoroutin(func() {
			s.logger.Fatal("ComplianceReportJobResult consumer exited", zap.Error(s.RunComplianceReportJobResultsConsumer()))
		})
		EnsureRunGoroutin(func() {
			s.logger.Fatal("InsightJobResult consumer exited", zap.Error(s.RunCheckupJobResultsConsumer()))
		})
		EnsureRunGoroutin(func() {
			s.logger.Fatal("SummarizerJobResult consumer exited", zap.Error(s.RunSummarizerJobResultsConsumer()))
		})
	case OperationModeReceiver:
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
	case OperationModeKafkaSink:
		s.logger.Info("starting kafka sink")
		s.kafkaESSink.Run()
	}

	return httpserver.RegisterAndStart(s.logger, s.httpServer.Address, s.httpServer)
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
		if djs != nil {
			for _, j := range djs {
				if j.Status == api.DescribeSourceJobCreated || j.Status == api.DescribeSourceJobInProgress {
					inProgress = true
				}
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

		sourceIDs := make([]string, 0, len(srcs))
		for _, src := range srcs {
			sourceIDs = append(sourceIDs, src.ID.String())
		}
		onboardSources, err := s.onboardClient.GetSources(&httpclient.Context{
			UserRole: api2.ViewerRole,
		}, sourceIDs)
		if err != nil {
			s.logger.Error("Failed to get onboard sources",
				zap.Strings("sourceIDs", sourceIDs),
				zap.Error(err),
			)
			return
		}
		var filteredSources []Source
		for _, src := range srcs {
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
						healthCheckedSrc.HealthState != source.HealthStatusUnhealthy {
						filteredSources = append(filteredSources, src)
					}
					break
				}
			}
		}
		srcs = filteredSources

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

		j, err := s.db.GetSummarizerJobByScheduleID(scheduleJob.ID, summarizer.JobType_ResourceMustSummarizer)
		if err != nil {
			s.logger.Error("Failed to fetch SummarizerJob", zap.Error(err))
			continue
		}

		if j == nil {
			err = s.scheduleMustSummarizerJob(&scheduleJob.ID)
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

func (s *Scheduler) RunComplianceReportCleanupJobScheduler() {
	s.logger.Info("Running compliance report cleanup job scheduler")

	t := time.NewTicker(JobSchedulingInterval)
	defer t.Stop()

	for range t.C {
		s.cleanupComplianceReportJob()
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
	//TODO-Saleh remove all of source resources

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
		if err := s.complianceReportCleanupJobQueue.Publish(complianceworker.ComplianceReportCleanupJob{
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

	for _, src := range sources {
		ctx := &httpclient.Context{
			UserRole: api2.ViewerRole,
		}
		benchmarks, err := s.complianceClient.GetAllBenchmarkAssignmentsBySourceId(ctx, src.ID)
		if err != nil {
			ComplianceJobsCount.WithLabelValues("failure").Inc()
			return createdJobCount, fmt.Errorf("error while getting benchmark assignments: %v", err)
		}

		for _, b := range benchmarks {
			crj := newComplianceReportJob(src.ID.String(), source.Type(src.Type), b.BenchmarkId, scheduleJob.ID)
			err := s.db.CreateComplianceReportJob(&crj)
			if err != nil {
				ComplianceJobsCount.WithLabelValues("failure").Inc()
				ComplianceSourceJobsCount.WithLabelValues("failure").Inc()
				return createdJobCount, fmt.Errorf("error while creating compliance job: %v", err)
			}

			enqueueComplianceReportJobs(s.logger, s.db, s.complianceReportJobQueue, src, &crj, scheduleJob)

			err = s.db.UpdateSourceReportGenerated(src.ID.String(), s.complianceIntervalHours)
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

			var result complianceworker.JobResult
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
			err := s.db.UpdateComplianceReportJobsTimedOut(s.complianceTimeoutHours)
			if err != nil {
				s.logger.Error("Failed to update timed out ComplianceReportJob", zap.Error(err))
			}
		}
	}
}

func (s *Scheduler) Stop() {
	queues := []queue.Interface{
		s.describeJobResultQueue,
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
}

func newComplianceReportJob(connectionID string, connector source.Type, benchmarkID string, scheduleJobID uint) ComplianceReportJob {
	return ComplianceReportJob{
		Model:           gorm.Model{},
		ScheduleJobID:   scheduleJobID,
		SourceID:        connectionID,
		SourceType:      connector,
		BenchmarkID:     benchmarkID,
		ReportCreatedAt: 0,
		Status:          complianceapi.ComplianceReportJobCreated,
		FailureMessage:  "",
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

func (s Scheduler) RunCheckupJobScheduler() {
	s.logger.Info("Scheduling insight jobs on a timer")

	t := time.NewTicker(JobSchedulingInterval)
	defer t.Stop()

	for ; ; <-t.C {
		s.scheduleCheckupJob()
	}
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

func newSummarizerJob(jobType summarizer.JobType, scheduleJobID uint) SummarizerJob {
	return SummarizerJob{
		Model:          gorm.Model{},
		ScheduleJobID:  &scheduleJobID,
		Status:         summarizerapi.SummarizerJobInProgress,
		JobType:        jobType,
		FailureMessage: "",
	}
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

func enqueueCheckupJobs(_ Database, q queue.Interface, job CheckupJob) error {
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

			if result.JobType == "" || result.JobType == summarizer.JobType_ResourceSummarizer || result.JobType == summarizer.JobType_ResourceMustSummarizer {
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

func newCheckupJob() CheckupJob {
	return CheckupJob{
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
