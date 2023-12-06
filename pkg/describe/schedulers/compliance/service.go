package compliance

import (
	confluent_kafka "github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/kaytu-io/kaytu-engine/pkg/compliance/client"
	config2 "github.com/kaytu-io/kaytu-engine/pkg/describe/config"
	"github.com/kaytu-io/kaytu-engine/pkg/describe/db"
	onboardClient "github.com/kaytu-io/kaytu-engine/pkg/onboard/client"
	"github.com/kaytu-io/kaytu-engine/pkg/utils"
	"github.com/kaytu-io/kaytu-util/pkg/kaytu-es-sdk"
	"go.uber.org/zap"
	"time"
)

const JobSchedulingInterval = 1 * time.Minute

type JobScheduler struct {
	conf                    config2.SchedulerConfig
	logger                  *zap.Logger
	complianceClient        client.ComplianceServiceClient
	onboardClient           onboardClient.OnboardServiceClient
	db                      db.Database
	kafkaProducer           *confluent_kafka.Producer
	esClient                kaytu.Client
	complianceIntervalHours int64
}

func New(conf config2.SchedulerConfig,
	logger *zap.Logger,
	complianceClient client.ComplianceServiceClient,
	onboardClient onboardClient.OnboardServiceClient,
	db db.Database,
	kafkaProducer *confluent_kafka.Producer,
	esClient kaytu.Client,
	complianceIntervalHours int64) *JobScheduler {
	return &JobScheduler{
		conf:                    conf,
		logger:                  logger,
		complianceClient:        complianceClient,
		onboardClient:           onboardClient,
		db:                      db,
		kafkaProducer:           kafkaProducer,
		esClient:                esClient,
		complianceIntervalHours: complianceIntervalHours,
	}
}

func (s *JobScheduler) Run() {
	utils.EnsureRunGoroutin(func() {
		s.RunScheduler()
	})
	utils.EnsureRunGoroutin(func() {
		s.RunPublisher()
	})
	utils.EnsureRunGoroutin(func() {
		s.RunSummarizer()
	})
	utils.EnsureRunGoroutin(func() {
		s.logger.Fatal("ComplianceReportJobResult consumer exited", zap.Error(s.RunComplianceReportJobResultsConsumer()))
	})
	utils.EnsureRunGoroutin(func() {
		s.logger.Fatal("ComplianceSummarizerResult consumer exited", zap.Error(s.RunComplianceSummarizerResultsConsumer()))
	})
}

func (s *JobScheduler) RunScheduler() {
	s.logger.Info("Scheduling compliance jobs on a timer")

	t := time.NewTicker(JobSchedulingInterval)
	defer t.Stop()

	for ; ; <-t.C {
		if err := s.runScheduler(); err != nil {
			s.logger.Error("failed to run compliance scheduler", zap.Error(err))
			ComplianceJobsCount.WithLabelValues("failure").Inc()
			continue
		}
	}
}

func (s *JobScheduler) RunPublisher() {
	s.logger.Info("Scheduling publisher on a timer")

	t := time.NewTicker(JobSchedulingInterval)
	defer t.Stop()

	for ; ; <-t.C {
		if err := s.runPublisher(); err != nil {
			s.logger.Error("failed to run compliance publisher", zap.Error(err))
			ComplianceJobsCount.WithLabelValues("failure").Inc()
			continue
		}
	}
}

func (s *JobScheduler) RunSummarizer() {
	s.logger.Info("Scheduling compliance summarizer on a timer")

	t := time.NewTicker(SummarizerSchedulingInterval)
	defer t.Stop()

	for ; ; <-t.C {
		if err := s.runSummarizer(); err != nil {
			s.logger.Error("failed to run compliance summarizer", zap.Error(err))
			ComplianceJobsCount.WithLabelValues("failure").Inc()
			continue
		}
	}
}
