package transactions

import (
	"fmt"
	api2 "github.com/opengovern/og-util/pkg/api"
	"github.com/opengovern/og-util/pkg/httpclient"
	"github.com/opengovern/og-util/pkg/source"
	api5 "github.com/opengovern/opengovernance/pkg/analytics/api"
	"github.com/opengovern/opengovernance/pkg/describe/api"
	client2 "github.com/opengovern/opengovernance/pkg/describe/client"
	client3 "github.com/opengovern/opengovernance/pkg/onboard/client"
	api4 "github.com/opengovern/opengovernance/pkg/workspace/api"
	"github.com/opengovern/opengovernance/pkg/workspace/config"
	"github.com/opengovern/opengovernance/pkg/workspace/db"
	"golang.org/x/net/context"
	"strings"
)

type EnsureJobsFinished struct {
	cfg config.Config
}

func NewEnsureJobsFinished(
	cfg config.Config,
) *EnsureJobsFinished {
	return &EnsureJobsFinished{
		cfg: cfg,
	}
}

func (t *EnsureJobsFinished) Requirements() []api4.TransactionID {
	return []api4.TransactionID{api4.Transaction_EnsureJobsRunning}
}

func (t *EnsureJobsFinished) ApplyIdempotent(ctx context.Context, workspace db.Workspace) error {
	hctx := &httpclient.Context{UserRole: api2.InternalRole}
	schedulerURL := strings.ReplaceAll(t.cfg.Scheduler.BaseURL, "%NAMESPACE%", t.cfg.KaytuOctopusNamespace)
	schedulerClient := client2.NewSchedulerServiceClient(schedulerURL)
	onboardURL := strings.ReplaceAll(t.cfg.Onboard.BaseURL, "%NAMESPACE%", t.cfg.KaytuOctopusNamespace)
	onboardClient := client3.NewOnboardServiceClient(onboardURL)

	job, err := schedulerClient.GetAnalyticsJob(hctx, workspace.AnalyticsJobID)
	if err != nil {
		return fmt.Errorf("getting analytics job failed: %v", err)
	}
	if job == nil {
		return ErrTransactionNeedsTime
	}

	if job.Status == api5.JobCreated || job.Status == api5.JobInProgress {
		return ErrTransactionNeedsTime
	}

	awsSrcs, err := onboardClient.ListSources(hctx, []source.Type{source.CloudAWS})
	if err != nil {
		return err
	}
	if len(awsSrcs) > 0 {
		complianceJob, err := schedulerClient.GetLatestComplianceJobForBenchmark(hctx, "aws_cis_v200")
		if err != nil {
			return err
		}

		if complianceJob.Status != api.ComplianceJobSucceeded && complianceJob.Status != api.ComplianceJobFailed {
			return ErrTransactionNeedsTime
		}
	}

	azureSrcs, err := onboardClient.ListSources(hctx, []source.Type{source.CloudAzure})
	if err != nil {
		return err
	}
	if len(azureSrcs) > 0 {
		complianceJob, err := schedulerClient.GetLatestComplianceJobForBenchmark(hctx, "azure_cis_v200")
		if err != nil {
			return err
		}

		if complianceJob.Status != api.ComplianceJobSucceeded && complianceJob.Status != api.ComplianceJobFailed {
			return ErrTransactionNeedsTime
		}
	}

	return nil
}

func (t *EnsureJobsFinished) RollbackIdempotent(ctx context.Context, workspace db.Workspace) error {
	return nil
}
