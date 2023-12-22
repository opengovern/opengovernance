package discovery

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

type Scheduler struct {
	conf                    config2.SchedulerConfig
	logger                  *zap.Logger
	complianceClient        client.ComplianceServiceClient
	onboardClient           onboardClient.OnboardServiceClient
	db                      db.Database
	kafkaProducer           *confluent_kafka.Producer
	esClient                kaytu.Client
	complianceIntervalHours time.Duration
}

func New(conf config2.SchedulerConfig, logger *zap.Logger, complianceClient client.ComplianceServiceClient, onboardClient onboardClient.OnboardServiceClient, db db.Database, kafkaProducer *confluent_kafka.Producer, esClient kaytu.Client, complianceIntervalHours time.Duration) *Scheduler {
	return &Scheduler{
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

func (s *Scheduler) Run() {
	utils.EnsureRunGoroutin(func() {
		s.OldResourceDeleter()
	})
}
