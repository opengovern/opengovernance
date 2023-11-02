package compliance

import (
	confluent_kafka "github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/kaytu-io/kaytu-engine/pkg/compliance/client"
	config2 "github.com/kaytu-io/kaytu-engine/pkg/describe/config"
	"github.com/kaytu-io/kaytu-engine/pkg/describe/db"
	onboardClient "github.com/kaytu-io/kaytu-engine/pkg/onboard/client"
	"go.uber.org/zap"
	"time"
)

const JobSchedulingInterval = 1 * time.Minute

type JobScheduler struct {
	conf             config2.SchedulerConfig
	logger           *zap.Logger
	complianceClient client.ComplianceServiceClient
	onboardClient    onboardClient.OnboardServiceClient
	db               db.Database
	kafkaProducer    *confluent_kafka.Producer
}

func New(conf config2.SchedulerConfig, logger *zap.Logger, complianceClient client.ComplianceServiceClient, onboardClient onboardClient.OnboardServiceClient, db db.Database, kafkaProducer *confluent_kafka.Producer) *JobScheduler {
	return &JobScheduler{
		conf:             conf,
		logger:           logger,
		complianceClient: complianceClient,
		onboardClient:    onboardClient,
		db:               db,
		kafkaProducer:    kafkaProducer,
	}
}

func (s *JobScheduler) Run() {
	s.logger.Info("Scheduling compliance jobs on a timer")

	t := time.NewTicker(JobSchedulingInterval)
	defer t.Stop()

	for ; ; <-t.C {
		if err := s.runScheduler(); err != nil {
			s.logger.Error("failed to run scheduleComplianceJob", zap.Error(err))
			ComplianceJobsCount.WithLabelValues("failure").Inc()
			continue
		}

		if err := s.runPublisher(); err != nil {
			s.logger.Error("failed to run scheduleComplianceJob", zap.Error(err))
			ComplianceJobsCount.WithLabelValues("failure").Inc()
			continue
		}
	}
}
