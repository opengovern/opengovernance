package discovery

import (
	"context"
	"github.com/opengovern/og-util/pkg/opengovernance-es-sdk"
	"github.com/opengovern/opengovernance/pkg/compliance/client"
	config2 "github.com/opengovern/opengovernance/pkg/describe/config"
	"github.com/opengovern/opengovernance/pkg/describe/db"
	onboardClient "github.com/opengovern/opengovernance/pkg/onboard/client"
	"github.com/opengovern/opengovernance/pkg/utils"
	"go.uber.org/zap"
)

type Scheduler struct {
	conf             config2.SchedulerConfig
	logger           *zap.Logger
	complianceClient client.ComplianceServiceClient
	onboardClient    onboardClient.OnboardServiceClient
	db               db.Database
	esClient         opengovernance.Client
}

func New(conf config2.SchedulerConfig, logger *zap.Logger, complianceClient client.ComplianceServiceClient, onboardClient onboardClient.OnboardServiceClient, db db.Database, esClient opengovernance.Client) *Scheduler {
	return &Scheduler{
		conf:             conf,
		logger:           logger,
		complianceClient: complianceClient,
		onboardClient:    onboardClient,
		db:               db,
		esClient:         esClient,
	}
}

func (s *Scheduler) Run(ctx context.Context) {
	utils.EnsureRunGoroutine(func() {
		s.OldResourceDeleter(ctx)
	})
}
