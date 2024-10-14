package transactions

import (
	api2 "github.com/opengovern/og-util/pkg/api"
	"github.com/opengovern/og-util/pkg/httpclient"
	"github.com/opengovern/og-util/pkg/source"
	"github.com/opengovern/opengovernance/pkg/compliance/client"
	client2 "github.com/opengovern/opengovernance/pkg/describe/client"
	client3 "github.com/opengovern/opengovernance/pkg/onboard/client"
	"github.com/opengovern/opengovernance/pkg/workspace/api"
	"github.com/opengovern/opengovernance/pkg/workspace/config"
	"github.com/opengovern/opengovernance/pkg/workspace/db"
	"golang.org/x/net/context"
	"strings"
)

type EnsureJobsRunning struct {
	cfg config.Config
	db  *db.Database
}

func NewEnsureJobsRunning(
	cfg config.Config,
	db *db.Database,
) *EnsureJobsRunning {
	return &EnsureJobsRunning{
		cfg: cfg,
		db:  db,
	}
}

func (t *EnsureJobsRunning) Requirements() []api.TransactionID {
	return []api.TransactionID{api.Transaction_EnsureDiscoveryFinished}
}

func (t *EnsureJobsRunning) ApplyIdempotent(ctx context.Context, workspace db.Workspace) error {
	hctx := &httpclient.Context{UserRole: api2.InternalRole}
	schedulerURL := strings.ReplaceAll(t.cfg.Scheduler.BaseURL, "%NAMESPACE%", t.cfg.KaytuOctopusNamespace)
	schedulerClient := client2.NewSchedulerServiceClient(schedulerURL)
	complianceURL := strings.ReplaceAll(t.cfg.Compliance.BaseURL, "%NAMESPACE%", t.cfg.KaytuOctopusNamespace)
	complianceClient := client.NewComplianceClient(complianceURL)
	onboardURL := strings.ReplaceAll(t.cfg.Onboard.BaseURL, "%NAMESPACE%", t.cfg.KaytuOctopusNamespace)
	onboardClient := client3.NewOnboardServiceClient(onboardURL)

	// run analytics if not running
	if workspace.AnalyticsJobID <= 0 {
		jobID, err := schedulerClient.TriggerAnalyticsJob(hctx)
		if err != nil {
			return err
		}
		err = t.db.SetWorkspaceAnalyticsJobID(workspace.ID, jobID)
		if err != nil {
			return err
		}
	}

	// assign compliance for aws cis v2, azure cis v2 (jobs will be triggeredl
	if !workspace.ComplianceTriggered {
		awsSrcs, err := onboardClient.ListSources(hctx, []source.Type{source.CloudAWS})
		if err != nil {
			return err
		}
		azureSrcs, err := onboardClient.ListSources(hctx, []source.Type{source.CloudAzure})
		if err != nil {
			return err
		}

		for _, src := range awsSrcs {
			_, err := complianceClient.CreateBenchmarkAssignment(hctx, "aws_cis_v200", src.ID.String())
			if err != nil {
				return err
			}
		}

		for _, src := range azureSrcs {
			_, err := complianceClient.CreateBenchmarkAssignment(hctx, "azure_cis_v200", src.ID.String())
			if err != nil {
				return err
			}
		}

		err = t.db.SetWorkspaceComplianceTriggered(workspace.ID)
		if err != nil {
			return err
		}
	}

	return nil
}

func (t *EnsureJobsRunning) RollbackIdempotent(ctx context.Context, workspace db.Workspace) error {
	return nil
}
