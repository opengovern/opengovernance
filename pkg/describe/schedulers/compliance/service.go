package compliance

import (
	"time"

	"github.com/kaytu-io/kaytu-engine/pkg/compliance/client"
	"github.com/kaytu-io/kaytu-engine/pkg/describe/config"
	"github.com/kaytu-io/kaytu-engine/pkg/describe/db"
	"github.com/kaytu-io/kaytu-engine/pkg/jq"
	onboardClient "github.com/kaytu-io/kaytu-engine/pkg/onboard/client"
	"github.com/kaytu-io/kaytu-engine/pkg/utils"
	"github.com/kaytu-io/kaytu-util/pkg/kaytu-es-sdk"
	"github.com/kaytu-io/kaytu-util/pkg/ticker"
	"go.uber.org/zap"
)

const JobSchedulingInterval = 1 * time.Minute

type JobScheduler struct {
	conf                    config.SchedulerConfig
	logger                  *zap.Logger
	complianceClient        client.ComplianceServiceClient
	onboardClient           onboardClient.OnboardServiceClient
	db                      db.Database
	jq                      *jq.JobQueue
	esClient                kaytu.Client
	complianceIntervalHours time.Duration
}

func New(
	conf config.SchedulerConfig,
	logger *zap.Logger,
	complianceClient client.ComplianceServiceClient,
	onboardClient onboardClient.OnboardServiceClient,
	db db.Database,
	jq *jq.JobQueue,
	esClient kaytu.Client,
	complianceIntervalHours time.Duration,
) *JobScheduler {
	return &JobScheduler{
		conf:                    conf,
		logger:                  logger,
		complianceClient:        complianceClient,
		onboardClient:           onboardClient,
		db:                      db,
		jq:                      jq,
		esClient:                esClient,
		complianceIntervalHours: complianceIntervalHours,
	}
}

func (s *JobScheduler) Run() {
	utils.EnsureRunGoroutine(func() {
		s.RunScheduler()
	})
	utils.EnsureRunGoroutine(func() {
		s.RunPublisher()
	})
	utils.EnsureRunGoroutine(func() {
		s.RunSummarizer()
	})
	utils.EnsureRunGoroutine(func() {
		s.logger.Fatal("ComplianceReportJobResult consumer exited", zap.Error(s.RunComplianceReportJobResultsConsumer()))
	})
	utils.EnsureRunGoroutine(func() {
		s.logger.Fatal("ComplianceSummarizerResult consumer exited", zap.Error(s.RunComplianceSummarizerResultsConsumer()))
	})
}

func (s *JobScheduler) RunScheduler() {
	s.logger.Info("Scheduling compliance jobs on a timer")

	t := ticker.NewTicker(JobSchedulingInterval, time.Second*10)
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

	t := ticker.NewTicker(JobSchedulingInterval, time.Second*10)
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

	t := ticker.NewTicker(SummarizerSchedulingInterval, time.Second*10)
	defer t.Stop()

	for ; ; <-t.C {
		if err := s.runSummarizer(); err != nil {
			s.logger.Error("failed to run compliance summarizer", zap.Error(err))
			ComplianceJobsCount.WithLabelValues("failure").Inc()
			continue
		}
	}
}
