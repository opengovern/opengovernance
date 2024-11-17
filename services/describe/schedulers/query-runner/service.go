package query_runner

import (
	"context"
	"time"

	"github.com/opengovern/og-util/pkg/jq"
	metadataClient "github.com/opengovern/opengovernance/services/metadata/client"

	"github.com/opengovern/og-util/pkg/opengovernance-es-sdk"
	"github.com/opengovern/og-util/pkg/ticker"
	"github.com/opengovern/opengovernance/pkg/describe/config"
	"github.com/opengovern/opengovernance/pkg/describe/db"
	"github.com/opengovern/opengovernance/pkg/utils"
	complianceClient "github.com/opengovern/opengovernance/services/compliance/client"
	inventoryClient "github.com/opengovern/opengovernance/services/inventory/client"
	"go.uber.org/zap"
)

const JobSchedulingInterval = 10 * time.Second

type JobScheduler struct {
	runSetupNatsStreams func(context.Context) error
	conf                config.SchedulerConfig
	logger              *zap.Logger
	db                  db.Database
	jq                  *jq.JobQueue
	esClient            opengovernance.Client
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
	esClient opengovernance.Client,
	inventoryClient inventoryClient.InventoryServiceClient,
	complianceClient complianceClient.ComplianceServiceClient,
	metadataClient metadataClient.MetadataServiceClient,
) *JobScheduler {
	return &JobScheduler{
		runSetupNatsStreams: runSetupNatsStreams,
		conf:                conf,
		logger:              logger,
		db:                  db,
		jq:                  jq,
		esClient:            esClient,
		inventoryClient:     inventoryClient,
		complianceClient:    complianceClient,
		metadataClient:      metadataClient,
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
