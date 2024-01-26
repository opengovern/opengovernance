package describe

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"sync"
	"time"

	awsConfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
	envoyAuth "github.com/envoyproxy/go-control-plane/envoy/service/auth/v3"
	"github.com/kaytu-io/kaytu-engine/pkg/analytics"
	authAPI "github.com/kaytu-io/kaytu-engine/pkg/auth/api"
	"github.com/kaytu-io/kaytu-engine/pkg/checkup"
	checkupAPI "github.com/kaytu-io/kaytu-engine/pkg/checkup/api"
	"github.com/kaytu-io/kaytu-engine/pkg/compliance/client"
	"github.com/kaytu-io/kaytu-engine/pkg/compliance/runner"
	"github.com/kaytu-io/kaytu-engine/pkg/compliance/summarizer"
	"github.com/kaytu-io/kaytu-engine/pkg/describe/config"
	"github.com/kaytu-io/kaytu-engine/pkg/describe/db"
	"github.com/kaytu-io/kaytu-engine/pkg/describe/db/model"
	"github.com/kaytu-io/kaytu-engine/pkg/describe/schedulers/compliance"
	"github.com/kaytu-io/kaytu-engine/pkg/describe/schedulers/discovery"
	"github.com/kaytu-io/kaytu-engine/pkg/httpclient"
	"github.com/kaytu-io/kaytu-engine/pkg/httpserver"
	"github.com/kaytu-io/kaytu-engine/pkg/insight"
	inventoryClient "github.com/kaytu-io/kaytu-engine/pkg/inventory/client"
	"github.com/kaytu-io/kaytu-engine/pkg/jq"
	metadataClient "github.com/kaytu-io/kaytu-engine/pkg/metadata/client"
	"github.com/kaytu-io/kaytu-engine/pkg/metadata/models"
	onboardClient "github.com/kaytu-io/kaytu-engine/pkg/onboard/client"
	"github.com/kaytu-io/kaytu-engine/pkg/utils"
	workspaceClient "github.com/kaytu-io/kaytu-engine/pkg/workspace/client"
	"github.com/kaytu-io/kaytu-util/pkg/kaytu-es-sdk"
	"github.com/kaytu-io/kaytu-util/pkg/postgres"
	"github.com/kaytu-io/kaytu-util/pkg/ticker"
	"github.com/kaytu-io/kaytu-util/proto/src/golang"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

const (
	JobSchedulingInterval   = 1 * time.Minute
	JobSequencerInterval    = 1 * time.Minute
	JobTimeoutCheckInterval = 1 * time.Minute
	MaxJobInQueue           = 10000

	schedulerConsumerGroup = "describe-scheduler"
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

	describeIntervalHours      time.Duration
	fullDiscoveryIntervalHours time.Duration
	costDiscoveryIntervalHours time.Duration
	describeTimeoutHours       int64
	insightIntervalHours       time.Duration
	checkupIntervalHours       int64
	mustSummarizeIntervalHours int64
	analyticsIntervalHours     time.Duration
	complianceIntervalHours    time.Duration

	logger           *zap.Logger
	workspaceClient  workspaceClient.WorkspaceServiceClient
	metadataClient   metadataClient.MetadataServiceClient
	complianceClient client.ComplianceServiceClient
	onboardClient    onboardClient.OnboardServiceClient
	inventoryClient  inventoryClient.InventoryServiceClient
	authGrpcClient   envoyAuth.AuthorizationClient
	es               kaytu.Client

	jq *jq.JobQueue

	describeEndpoint string
	keyARN           string
	keyRegion        string

	WorkspaceName string

	DoDeleteOldResources bool
	OperationMode        OperationMode
	MaxConcurrentCall    int64

	LambdaClient *lambda.Client

	complianceScheduler *compliance.JobScheduler
	discoveryScheduler  *discovery.Scheduler
	conf                config.SchedulerConfig
}

func InitializeScheduler(
	id string,
	conf config.SchedulerConfig,
	checkupJobQueueName string,
	checkupJobResultQueueName string,
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
		id:               id,
		OperationMode:    OperationMode(OperationModeConfig),
		describeEndpoint: DescribeDeliverEndpoint,
		keyARN:           KeyARN,
		keyRegion:        KeyRegion,
	}
	defer func() {
		if err != nil && s != nil {
			s.Stop()
		}
	}()

	lambdaCfg, err := awsConfig.LoadDefaultConfig(context.Background())
	lambdaCfg.Region = KeyRegion

	s.conf = conf
	s.LambdaClient = lambda.NewFromConfig(lambdaCfg)

	s.logger, err = zap.NewProduction()
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

	jq, err := jq.New(conf.NATS.URL, s.logger)
	if err != nil {
		return nil, err
	}
	s.jq = jq

	if err := s.jq.Stream(context.Background(), insight.StreamName, "insight job queue", []string{insight.ResultsQueueName, insight.JobsQueueName}, 1000); err != nil {
		return nil, err
	}

	if err := s.jq.Stream(context.Background(), summarizer.StreamName, "compliance summarizer job queues", []string{summarizer.JobQueueTopic, summarizer.ResultQueueTopic}, 1000); err != nil {
		return nil, err
	}

	if err := s.jq.Stream(context.Background(), runner.StreamName, "compliance runner job queues", []string{runner.JobQueueTopic, runner.ResultQueueTopic}, 1000000); err != nil {
		return nil, err
	}

	if err := s.jq.Stream(context.Background(), analytics.StreamName, "analytics job queue", []string{analytics.JobQueueTopic, analytics.JobResultQueueTopic}, 1000); err != nil {
		return nil, err
	}

	if err := s.jq.Stream(context.Background(), checkup.StreamName, "checkup job queue", []string{checkup.JobsQueueName, checkup.ResultsQueueName}, 1000); err != nil {
		return nil, err
	}

	if err := s.jq.Stream(context.Background(), DescribeStreamName, "describe job queue", []string{DescribeResultsQueueName}, 1000000); err != nil {
		return nil, err
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
	s.onboardClient = onboardClient.NewOnboardServiceClient(OnboardBaseURL)
	s.inventoryClient = inventoryClient.NewInventoryServiceClient(InventoryBaseURL)
	authGRPCConn, err := grpc.Dial(AuthGRPCURI, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{InsecureSkipVerify: true})))
	if err != nil {
		return nil, err
	}
	s.authGrpcClient = envoyAuth.NewAuthorizationClient(authGRPCConn)

	describeServer := NewDescribeServer(s.db, s.jq, s.authGrpcClient, s.logger, conf)
	s.grpcServer = grpc.NewServer(
		grpc.MaxRecvMsgSize(128*1024*1024),
		grpc.UnaryInterceptor(describeServer.grpcUnaryAuthInterceptor),
		grpc.StreamInterceptor(describeServer.grpcStreamAuthInterceptor),
	)

	golang.RegisterDescribeServiceServer(s.grpcServer, describeServer)

	workspace, err := s.workspaceClient.GetByID(&httpclient.Context{
		UserRole: authAPI.InternalRole,
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
		s.jq,
		s.es,
		s.complianceIntervalHours,
	)

	s.discoveryScheduler = discovery.New(
		conf,
		s.logger,
		s.complianceClient,
		s.onboardClient,
		s.db,
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

	var wg sync.WaitGroup

	httpCtx := &httpclient.Context{
		UserRole: authAPI.ViewerRole,
	}
	describeJobIntM, err := s.metadataClient.GetConfigMetadata(httpCtx, models.MetadataKeyDescribeJobInterval)
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

	fullDiscoveryJobIntM, err := s.metadataClient.GetConfigMetadata(httpCtx, models.MetadataKeyFullDiscoveryJobInterval)
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

	costDiscoveryJobIntM, err := s.metadataClient.GetConfigMetadata(httpCtx, models.MetadataKeyCostDiscoveryJobInterval)
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

	insightJobIntM, err := s.metadataClient.GetConfigMetadata(httpCtx, models.MetadataKeyInsightJobInterval)
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

	analyticsJobIntM, err := s.metadataClient.GetConfigMetadata(httpCtx, models.MetadataKeyMetricsJobInterval)
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

	complianceJobIntM, err := s.metadataClient.GetConfigMetadata(httpCtx, models.MetadataKeyComplianceJobInterval)
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

	s.logger.Info("starting scheduler")

	// Describe
	utils.EnsureRunGoroutine(func() {
		s.RunDescribeJobScheduler()
	})
	utils.EnsureRunGoroutine(func() {
		s.RunDescribeResourceJobs(ctx)
	})
	s.discoveryScheduler.Run()

	// Describe
	utils.EnsureRunGoroutine(func() {
		s.RunStackScheduler()
	})

	// Inventory summarizer
	utils.EnsureRunGoroutine(func() {
		s.RunAnalyticsJobScheduler()
	})

	wg.Add(1)
	utils.EnsureRunGoroutine(func() {
		s.logger.Fatal("AnalyticsJobResult consumer exited", zap.Error(s.RunAnalyticsJobResultsConsumer(ctx)))
		wg.Done()
	})

	// Compliance
	s.complianceScheduler.Run()
	utils.EnsureRunGoroutine(func() {
		s.RunJobSequencer()
	})

	// Insights
	utils.EnsureRunGoroutine(func() {
		s.RunInsightJobScheduler()
	})
	wg.Add(1)
	utils.EnsureRunGoroutine(func() {
		s.logger.Fatal("InsightJobResult consumer exited", zap.Error(s.RunInsightJobResultsConsumer(ctx)))
		wg.Done()
	})
	utils.EnsureRunGoroutine(func() {
		s.RunCheckupJobScheduler()
	})
	utils.EnsureRunGoroutine(func() {
		s.RunDisabledConnectionCleanup()
	})
	wg.Add(1)
	utils.EnsureRunGoroutine(func() {
		s.logger.Fatal("CheckupJobResult consumer exited", zap.Error(s.RunCheckupJobResultsConsumer(ctx)))
		wg.Done()
	})
	utils.EnsureRunGoroutine(func() {
		s.RunScheduledJobCleanup()
	})
	utils.EnsureRunGoroutine(func() {
		s.UpdateDescribedResourceCountScheduler()
	})
	utils.EnsureRunGoroutine(func() {
		s.UpdateDescribedResourceCountScheduler()
	})
	wg.Add(1)
	utils.EnsureRunGoroutine(func() {
		s.logger.Fatal("DescribeJobResults consumer exited", zap.Error(s.RunDescribeJobResultsConsumer(ctx)))
		wg.Done()
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

	go func() {
		if err := httpserver.RegisterAndStart(s.logger, s.httpServer.Address, s.httpServer); err != nil && !errors.Is(err, http.ErrServerClosed) {
			s.logger.Fatal("failed to serve http server", zap.Error(err))
		}
	}()

	wg.Wait()

	return nil
}

func (s *Scheduler) RunDisabledConnectionCleanup() {
	ticker := ticker.NewTicker(time.Hour, time.Second*10)
	defer ticker.Stop()

	for range ticker.C {
		connections, err := s.onboardClient.ListSources(&httpclient.Context{UserRole: authAPI.InternalRole}, nil)
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

func (s *Scheduler) Stop() {
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
		job := newCheckupJob()
		err = s.db.AddCheckupJob(&job)
		if err != nil {
			CheckupJobsCount.WithLabelValues("failure").Inc()
			s.logger.Error("Failed to create CheckupJob",
				zap.Uint("jobId", job.ID),
				zap.Error(err),
			)
		}

		bytes, err := json.Marshal(checkup.Job{
			JobID:      job.ID,
			ExecutedAt: job.CreatedAt.UnixMilli(),
		})
		if err != nil {
			CheckupJobsCount.WithLabelValues("failure").Inc()
			s.logger.Error("Failed to marshal a checkup job as json", zap.Error(err), zap.Uint("jobId", job.ID))
		}

		if err := s.jq.Produce(context.Background(), checkup.JobsQueueName, bytes, fmt.Sprintf("job-%d", job.ID)); err != nil {
			CheckupJobsCount.WithLabelValues("failure").Inc()
			s.logger.Error("Failed to enqueue CheckupJob",
				zap.Uint("jobId", job.ID),
				zap.Error(err),
			)
			job.Status = checkupAPI.CheckupJobFailed
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

// RunCheckupJobResultsConsumer consumes messages from the checkupJobResultQueue queue.
// It will update the status of the jobs in the database based on the message.
// It will also update the jobs status that are not completed in certain time to FAILED
func (s *Scheduler) RunCheckupJobResultsConsumer(ctx context.Context) error {
	s.logger.Info("Consuming messages from the CheckupJobResultQueue queue")

	consumeCtx, err := s.jq.Consume(
		ctx,
		"checkup-scheduler",
		checkup.StreamName,
		[]string{checkup.ResultsQueueName},
		"checkup-scheduler",
		func(msg jetstream.Msg) {
			var result checkup.JobResult

			if err := json.Unmarshal(msg.Data(), &result); err != nil {
				s.logger.Error("Failed to unmarshal CheckupJobResult results", zap.Error(err))

				// when message cannot be unmarshal, there is no need to consume it again.
				if err := msg.Ack(); err != nil {
					s.logger.Error("Failed to ack the message", zap.Error(err))
				}

				return
			}

			s.logger.Info("Processing CheckupJobResult for Job",
				zap.Uint("jobId", result.JobID),
				zap.String("status", string(result.Status)),
			)

			if err := s.db.UpdateCheckupJob(result.JobID, result.Status, result.Error); err != nil {
				s.logger.Error("Failed to update the status of CheckupJob",
					zap.Uint("jobId", result.JobID),
					zap.Error(err))

				if err = msg.Nak(); err != nil {
					s.logger.Error("Failed to not ack the message", zap.Error(err))
				}

				return
			}

			if err := msg.Ack(); err != nil {
				s.logger.Error("Failed to ack the message", zap.Error(err))
			}
		},
	)
	if err != nil {
		return err
	}

	t := ticker.NewTicker(JobTimeoutCheckInterval, time.Second*10)
	defer t.Stop()

	for {
		select {
		case <-t.C:
			if err := s.db.UpdateCheckupJobsTimedOut(s.checkupIntervalHours); err != nil {
				s.logger.Error("Failed to update timed out CheckupJob", zap.Error(err))
			}
		case <-ctx.Done():
			consumeCtx.Drain()
			consumeCtx.Stop()
			return nil
		}
	}
}

func newCheckupJob() model.CheckupJob {
	return model.CheckupJob{
		Status: checkupAPI.CheckupJobInProgress,
	}
}
