package query_runner

import (
	"context"
	"github.com/kaytu-io/kaytu-util/pkg/jq"
	metadataClient "github.com/kaytu-io/open-governance/pkg/metadata/client"
	"time"

	"github.com/kaytu-io/kaytu-util/pkg/kaytu-es-sdk"
	"github.com/kaytu-io/kaytu-util/pkg/ticker"
	complianceClient "github.com/kaytu-io/open-governance/pkg/compliance/client"
	"github.com/kaytu-io/open-governance/pkg/describe/config"
	"github.com/kaytu-io/open-governance/pkg/describe/db"
	inventoryClient "github.com/kaytu-io/open-governance/pkg/inventory/client"
	"github.com/kaytu-io/open-governance/pkg/utils"
	"go.uber.org/zap"
)

const JobSchedulingInterval = 10 * time.Second

type JobScheduler struct {
	runSetupNatsStreams func(context.Context) error
	conf                config.SchedulerConfig
	logger              *zap.Logger
	db                  db.Database
	jq                  *jq.JobQueue
	esClient            kaytu.Client
	inventoryClient     inventoryClient.InventoryServiceClient
	complianceClient    complianceClient.ComplianceServiceClient
	metadataClient      metadataClient.MetadataServiceClient
}

func New(
	runSetupNatsStreams func(context.Context) error,
	conf config.SchedulerConfig,
	logger *zap.Logger,
	db db.Database,
	jq *jq.JobQueue,
	esClient kaytu.Client,
	inventoryClient inventoryClient.InventoryServiceClient,
) *JobScheduler {
	return &JobScheduler{
		runSetupNatsStreams: runSetupNatsStreams,
		conf:                conf,
		logger:              logger,
		db:                  db,
		jq:                  jq,
		esClient:            esClient,
		inventoryClient:     inventoryClient,
	}
}

func (s *JobScheduler) Run(ctx context.Context) {
	utils.EnsureRunGoroutine(func() {
		s.RunPublisher(ctx)
	})
	utils.EnsureRunGoroutine(func() {
		s.logger.Fatal("ComplianceReportJobResult consumer exited", zap.Error(s.RunQueryRunnerReportJobResultsConsumer(ctx)))
	})
}

func (s *JobScheduler) RunPublisher(ctx context.Context) {
	s.logger.Info("Scheduling publisher on a timer")

	t := ticker.NewTicker(JobSchedulingInterval, time.Second*10)
	defer t.Stop()

	for ; ; <-t.C {
		if err := s.runPublisher(ctx); err != nil {
			s.logger.Error("failed to run compliance publisher", zap.Error(err))
			continue
		}
	}
}
