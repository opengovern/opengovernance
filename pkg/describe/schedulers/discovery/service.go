package discovery

import (
	"context"
	"github.com/kaytu-io/kaytu-util/pkg/kaytu-es-sdk"
	"github.com/kaytu-io/open-governance/pkg/compliance/client"
	config2 "github.com/kaytu-io/open-governance/pkg/describe/config"
	"github.com/kaytu-io/open-governance/pkg/describe/db"
	onboardClient "github.com/kaytu-io/open-governance/pkg/onboard/client"
	"github.com/kaytu-io/open-governance/pkg/utils"
	"go.uber.org/zap"
)

type Scheduler struct {
	conf             config2.SchedulerConfig
	logger           *zap.Logger
	complianceClient client.ComplianceServiceClient
	onboardClient    onboardClient.OnboardServiceClient
	db               db.Database
	esClient         kaytu.Client
}

func New(conf config2.SchedulerConfig, logger *zap.Logger, complianceClient client.ComplianceServiceClient, onboardClient onboardClient.OnboardServiceClient, db db.Database, esClient kaytu.Client) *Scheduler {
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
