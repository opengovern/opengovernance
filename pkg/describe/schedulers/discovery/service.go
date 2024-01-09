package discovery

import (
	"time"

	"github.com/kaytu-io/kaytu-engine/pkg/compliance/client"
	config2 "github.com/kaytu-io/kaytu-engine/pkg/describe/config"
	"github.com/kaytu-io/kaytu-engine/pkg/describe/db"
	onboardClient "github.com/kaytu-io/kaytu-engine/pkg/onboard/client"
	"github.com/kaytu-io/kaytu-engine/pkg/utils"
	"github.com/kaytu-io/kaytu-util/pkg/kaytu-es-sdk"
	"go.uber.org/zap"
)

type Scheduler struct {
	conf                    config2.SchedulerConfig
	logger                  *zap.Logger
	complianceClient        client.ComplianceServiceClient
	onboardClient           onboardClient.OnboardServiceClient
	db                      db.Database
	esClient                kaytu.Client
	complianceIntervalHours time.Duration
}

func New(conf config2.SchedulerConfig, logger *zap.Logger, complianceClient client.ComplianceServiceClient, onboardClient onboardClient.OnboardServiceClient, db db.Database, esClient kaytu.Client, complianceIntervalHours time.Duration) *Scheduler {
	return &Scheduler{
		conf:                    conf,
		logger:                  logger,
		complianceClient:        complianceClient,
		onboardClient:           onboardClient,
		db:                      db,
		esClient:                esClient,
		complianceIntervalHours: complianceIntervalHours,
	}
}

func (s *Scheduler) Run() {
	utils.EnsureRunGoroutine(func() {
		s.OldResourceDeleter()
	})
}
