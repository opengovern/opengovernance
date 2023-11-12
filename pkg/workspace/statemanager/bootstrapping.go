package statemanager

import (
	"errors"
	api5 "github.com/kaytu-io/kaytu-engine/pkg/analytics/api"
	api2 "github.com/kaytu-io/kaytu-engine/pkg/auth/api"
	"github.com/kaytu-io/kaytu-engine/pkg/compliance/client"
	"github.com/kaytu-io/kaytu-engine/pkg/describe/api"
	client2 "github.com/kaytu-io/kaytu-engine/pkg/describe/client"
	api3 "github.com/kaytu-io/kaytu-engine/pkg/insight/api"
	"github.com/kaytu-io/kaytu-engine/pkg/internal/httpclient"
	client3 "github.com/kaytu-io/kaytu-engine/pkg/onboard/client"
	api4 "github.com/kaytu-io/kaytu-engine/pkg/workspace/api"
	"github.com/kaytu-io/kaytu-engine/pkg/workspace/db"
	"github.com/kaytu-io/kaytu-util/pkg/source"
	"strings"
)

func (s *Service) runBootstrapping(workspace *db.Workspace) error {
	hctx := &httpclient.Context{UserRole: api2.InternalRole}

	creds, err := s.db.ListCredentialsByWorkspaceID(workspace.ID)
	if err != nil {
		return err
	}

	if !workspace.IsCreated {
		if len(creds) > 0 {
			return s.createWorkspace(workspace)
		}
		return nil
	}

	for _, cred := range creds {
		if !cred.IsCreated {
			err := s.addCredentialToWorkspace(workspace, cred)
			if err != nil {
				return err
			}
		}
	}

	if workspace.IsBootstrapInputFinished {
		schedulerURL := strings.ReplaceAll(s.cfg.Scheduler.BaseURL, "%NAMESPACE%", workspace.ID)
		schedulerClient := client2.NewSchedulerServiceClient(schedulerURL)

		status, err := schedulerClient.GetDescribeAllJobsStatus(hctx)
		if err != nil {
			return err
		}

		// waiting for all connections to finish
		if status == nil || *status != api.DescribeAllJobsStatusResourcesPublished {
			return nil
		}

		// run analytics if not running
		if workspace.AnalyticsJobID <= 0 {
			jobID, err := schedulerClient.TriggerAnalyticsJob(hctx)
			if err != nil {
				return err
			}
			err = s.db.SetWorkspaceAnalyticsJobID(workspace.ID, jobID)
			if err != nil {
				return err
			}
			return nil
		}

		complianceURL := strings.ReplaceAll(s.cfg.Compliance.BaseURL, "%NAMESPACE%", workspace.ID)
		complianceClient := client.NewComplianceClient(complianceURL)

		// run insight if not running
		if len(workspace.InsightJobID) == 0 {
			ins, err := complianceClient.ListInsights(hctx)
			if err != nil {
				return err
			}

			var allJobIDs []uint
			for _, insight := range ins {
				jobIDs, err := schedulerClient.TriggerInsightJob(hctx, insight.ID)
				if err != nil {
					return err
				}
				allJobIDs = append(allJobIDs, jobIDs...)
			}

			err = s.db.SetWorkspaceInsightsJobIDs(workspace.ID, allJobIDs)
			if err != nil {
				return err
			}
			return nil
		}

		onboardURL := strings.ReplaceAll(s.cfg.Onboard.BaseURL, "%NAMESPACE%", workspace.ID)
		onboardClient := client3.NewOnboardServiceClient(onboardURL, nil)

		// assign compliance for aws cis v2, azure cis v2 (jobs will be triggeredl
		if !workspace.ComplianceTriggered {
			srcs, err := onboardClient.ListSources(hctx, []source.Type{source.CloudAWS})
			if err != nil {
				return err
			}

			for _, src := range srcs {
				_, err = complianceClient.CreateBenchmarkAssignment(hctx, "aws_cis_v200", src.ConnectionID)
				if err != nil {
					return err
				}
			}

			srcs, err = onboardClient.ListSources(hctx, []source.Type{source.CloudAzure})
			if err != nil {
				return err
			}

			for _, src := range srcs {
				_, err = complianceClient.CreateBenchmarkAssignment(hctx, "azure_cis_v200", src.ConnectionID)
				if err != nil {
					return err
				}
			}

			err = s.db.SetWorkspaceComplianceTriggered(workspace.ID)
			if err != nil {
				return err
			}
			return nil
		}

		job, err := schedulerClient.GetAnalyticsJob(hctx, workspace.AnalyticsJobID)
		if err != nil {
			return err
		}
		if job == nil {
			return errors.New("analytics job not found")
		}
		if job.Status == api5.JobCreated || job.Status == api5.JobInProgress {
			return nil
		}

		for _, insJobID := range workspace.InsightJobID {
			job, err := schedulerClient.GetInsightJob(hctx, uint(insJobID))
			if err != nil {
				return err
			}
			if job == nil {
				return errors.New("insight job not found")
			}
			if job.Status == api3.InsightJobInProgress {
				return nil
			}
		}

		complianceJob, err := schedulerClient.GetLatestComplianceJobForBenchmark(hctx, "aws_cis_v200")
		if err != nil {
			return err
		}

		if complianceJob.Status != api.ComplianceJobSucceeded && complianceJob.Status != api.ComplianceJobFailed {
			return nil
		}

		err = s.db.UpdateWorkspaceStatus(workspace.ID, api4.StatusProvisioned)
		if err != nil {
			return err
		}
	}
	return nil
}
