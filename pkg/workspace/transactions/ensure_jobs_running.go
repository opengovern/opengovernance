package transactions

import (
	api2 "github.com/kaytu-io/kaytu-engine/pkg/auth/api"
	"github.com/kaytu-io/kaytu-engine/pkg/compliance/client"
	client2 "github.com/kaytu-io/kaytu-engine/pkg/describe/client"
	"github.com/kaytu-io/kaytu-engine/pkg/httpclient"
	api3 "github.com/kaytu-io/kaytu-engine/pkg/insight/api"
	client3 "github.com/kaytu-io/kaytu-engine/pkg/onboard/client"
	"github.com/kaytu-io/kaytu-engine/pkg/workspace/api"
	"github.com/kaytu-io/kaytu-engine/pkg/workspace/config"
	"github.com/kaytu-io/kaytu-engine/pkg/workspace/db"
	"github.com/kaytu-io/kaytu-util/pkg/source"
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

func (t *EnsureJobsRunning) Apply(workspace db.Workspace) error {
	hctx := &httpclient.Context{UserRole: api2.InternalRole}
	schedulerURL := strings.ReplaceAll(t.cfg.Scheduler.BaseURL, "%NAMESPACE%", workspace.ID)
	schedulerClient := client2.NewSchedulerServiceClient(schedulerURL)
	complianceURL := strings.ReplaceAll(t.cfg.Compliance.BaseURL, "%NAMESPACE%", workspace.ID)
	complianceClient := client.NewComplianceClient(complianceURL)
	onboardURL := strings.ReplaceAll(t.cfg.Onboard.BaseURL, "%NAMESPACE%", workspace.ID)
	onboardClient := client3.NewOnboardServiceClient(onboardURL, nil)

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

	// run insight if not running
	if len(workspace.InsightJobsID) == 0 {
		ins, err := complianceClient.ListInsights(hctx)
		if err != nil {
			return err
		}

		var allJobIDs []uint
		for _, insight := range ins {
			insightJobs, err := schedulerClient.GetJobsByInsightID(hctx, insight.ID)
			if err != nil {
				return err
			}

			hasCreated := false
			for _, job := range insightJobs {
				if job.Status == api3.InsightJobCreated {
					hasCreated = true
					allJobIDs = append(allJobIDs, job.ID)
					break
				}
			}

			if !hasCreated {
				jobIDs, err := schedulerClient.TriggerInsightJob(hctx, insight.ID)
				if err != nil {
					return err
				}
				allJobIDs = append(allJobIDs, jobIDs...)
			}
		}

		err = t.db.SetWorkspaceInsightsJobIDs(workspace.ID, allJobIDs)
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

func (t *EnsureJobsRunning) Rollback(workspace db.Workspace) error {
	return nil
}
