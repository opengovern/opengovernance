package statemanager

import (
	"fmt"
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
	"go.uber.org/zap"
	"strings"
)

func (s *Service) runBootstrapping(workspace *db.Workspace) error {
	ok, err := s.ensureWorkspaceCreated(workspace)
	if err != nil {
		return err
	}
	if !ok {
		return nil
	}

	ok, err = s.ensureCredentialsOnboarded(workspace)
	if err != nil {
		return err
	}
	if !ok {
		return nil
	}

	if workspace.IsBootstrapInputFinished {
		ok, err = s.ensureDiscoveryIsFinished(workspace)
		if err != nil {
			return err
		}
		if !ok {
			return nil
		}

		ok, err = s.ensureJobsAreRunning(workspace)
		if err != nil {
			return err
		}
		if !ok {
			return nil
		}

		ok, err = s.ensureJobsAreFinished(workspace)
		if err != nil {
			return err
		}
		if !ok {
			return nil
		}

		s.logger.Info("workspace provisioned", zap.String("workspaceID", workspace.ID))
		err = s.db.UpdateWorkspaceStatus(workspace.ID, api4.StatusProvisioned)
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *Service) ensureWorkspaceCreated(workspace *db.Workspace) (bool, error) {
	if !workspace.IsCreated {
		creds, err := s.db.ListCredentialsByWorkspaceID(workspace.ID)
		if err != nil {
			return false, err
		}

		if len(creds) > 0 {
			s.logger.Info("creating workspace", zap.String("workspaceID", workspace.ID))
			return false, s.createWorkspace(workspace)
		}
		return false, nil
	}
	return true, nil
}

func (s *Service) ensureCredentialsOnboarded(workspace *db.Workspace) (bool, error) {
	creds, err := s.db.ListCredentialsByWorkspaceID(workspace.ID)
	if err != nil {
		return false, err
	}

	for _, cred := range creds {
		if !cred.IsCreated {
			s.logger.Info("adding credential", zap.String("workspaceID", workspace.ID), zap.Uint("credentialID", cred.ID))
			err := s.addCredentialToWorkspace(workspace, cred)
			if err != nil {
				return false, err
			}
		}
	}
	return true, nil
}

func (s *Service) ensureDiscoveryIsFinished(workspace *db.Workspace) (bool, error) {
	hctx := &httpclient.Context{UserRole: api2.InternalRole}
	schedulerURL := strings.ReplaceAll(s.cfg.Scheduler.BaseURL, "%NAMESPACE%", workspace.ID)
	schedulerClient := client2.NewSchedulerServiceClient(schedulerURL)

	status, err := schedulerClient.GetDescribeAllJobsStatus(hctx)
	if err != nil {
		return false, err
	}

	// waiting for all connections to finish
	if status == nil || *status != api.DescribeAllJobsStatusResourcesPublished {
		s.logger.Info("waiting for connections to finish describing", zap.String("workspaceID", workspace.ID))
		return false, nil
	}
	return true, nil
}

func (s *Service) ensureJobsAreRunning(workspace *db.Workspace) (bool, error) {
	hctx := &httpclient.Context{UserRole: api2.InternalRole}
	schedulerURL := strings.ReplaceAll(s.cfg.Scheduler.BaseURL, "%NAMESPACE%", workspace.ID)
	schedulerClient := client2.NewSchedulerServiceClient(schedulerURL)
	complianceURL := strings.ReplaceAll(s.cfg.Compliance.BaseURL, "%NAMESPACE%", workspace.ID)
	complianceClient := client.NewComplianceClient(complianceURL)
	onboardURL := strings.ReplaceAll(s.cfg.Onboard.BaseURL, "%NAMESPACE%", workspace.ID)
	onboardClient := client3.NewOnboardServiceClient(onboardURL, nil)

	// run analytics if not running
	if workspace.AnalyticsJobID <= 0 {
		s.logger.Info("running analytics", zap.String("workspaceID", workspace.ID))
		jobID, err := schedulerClient.TriggerAnalyticsJob(hctx)
		if err != nil {
			return false, err
		}
		err = s.db.SetWorkspaceAnalyticsJobID(workspace.ID, jobID)
		if err != nil {
			return false, err
		}
	}

	// run insight if not running
	if len(workspace.InsightJobsID) == 0 {
		s.logger.Info("running insights", zap.String("workspaceID", workspace.ID))
		ins, err := complianceClient.ListInsights(hctx)
		if err != nil {
			return false, err
		}

		var allJobIDs []uint
		for _, insight := range ins {
			insightJobs, err := schedulerClient.GetJobsByInsightID(hctx, insight.ID)
			if err != nil {
				return false, err
			}

			hasCreated := false
			for _, job := range insightJobs {
				if job.Status == api3.InsightJobCreated {
					hasCreated = true
				}
			}

			if !hasCreated {
				jobIDs, err := schedulerClient.TriggerInsightJob(hctx, insight.ID)
				if err != nil {
					return false, err
				}
				allJobIDs = append(allJobIDs, jobIDs...)
			}
		}

		err = s.db.SetWorkspaceInsightsJobIDs(workspace.ID, allJobIDs)
		if err != nil {
			return false, err
		}
	}

	// assign compliance for aws cis v2, azure cis v2 (jobs will be triggeredl
	if !workspace.ComplianceTriggered {
		awsSrcs, err := onboardClient.ListSources(hctx, []source.Type{source.CloudAWS})
		if err != nil {
			return false, err
		}
		azureSrcs, err := onboardClient.ListSources(hctx, []source.Type{source.CloudAzure})
		if err != nil {
			return false, err
		}

		s.logger.Info("running compliance", zap.String("workspaceID", workspace.ID))
		for _, src := range awsSrcs {
			_, err := complianceClient.CreateBenchmarkAssignment(hctx, "aws_cis_v200", src.ID.String())
			if err != nil {
				return false, err
			}
		}

		for _, src := range azureSrcs {
			_, err := complianceClient.CreateBenchmarkAssignment(hctx, "azure_cis_v200", src.ID.String())
			if err != nil {
				return false, err
			}
		}

		err = s.db.SetWorkspaceComplianceTriggered(workspace.ID)
		if err != nil {
			return false, err
		}
	}

	return true, nil
}

func (s *Service) ensureJobsAreFinished(workspace *db.Workspace) (bool, error) {
	hctx := &httpclient.Context{UserRole: api2.InternalRole}
	schedulerURL := strings.ReplaceAll(s.cfg.Scheduler.BaseURL, "%NAMESPACE%", workspace.ID)
	schedulerClient := client2.NewSchedulerServiceClient(schedulerURL)
	onboardURL := strings.ReplaceAll(s.cfg.Onboard.BaseURL, "%NAMESPACE%", workspace.ID)
	onboardClient := client3.NewOnboardServiceClient(onboardURL, nil)

	s.logger.Info("checking analytics job", zap.String("workspaceID", workspace.ID))
	job, err := schedulerClient.GetAnalyticsJob(hctx, workspace.AnalyticsJobID)
	if err != nil {
		return false, fmt.Errorf("getting analytics job failed: %v", err)
	}
	if job == nil {
		s.logger.Info("analytics job not found", zap.String("workspaceID", workspace.ID))
		return false, nil
	}
	if job.Status == api5.JobCreated || job.Status == api5.JobInProgress {
		s.logger.Info("analytics job is running", zap.String("workspaceID", workspace.ID), zap.Uint("jobID", job.ID))
		return false, nil
	}

	s.logger.Info("checking insight job", zap.String("workspaceID", workspace.ID))
	isInProgress := false
	for _, insJobID := range workspace.InsightJobsID {
		job, err := schedulerClient.GetInsightJob(hctx, uint(insJobID))
		if err != nil {
			return false, err
		}
		if job == nil {
			s.logger.Info("insight job not found", zap.String("workspaceID", workspace.ID))
			return false, nil
		}

		if job.Status == api3.InsightJobSucceeded {
			isInProgress = false
			break
		}

		if job.Status == api3.InsightJobCreated || job.Status == api3.InsightJobInProgress {
			isInProgress = true
		}
	}

	if isInProgress {
		s.logger.Info("insight job is running", zap.String("workspaceID", workspace.ID), zap.Uint("jobID", job.ID))
		return false, nil
	}

	awsSrcs, err := onboardClient.ListSources(hctx, []source.Type{source.CloudAWS})
	if err != nil {
		return false, err
	}
	if len(awsSrcs) > 0 {
		s.logger.Info("checking aws compliance job", zap.String("workspaceID", workspace.ID))
		complianceJob, err := schedulerClient.GetLatestComplianceJobForBenchmark(hctx, "aws_cis_v200")
		if err != nil {
			return false, err
		}

		if complianceJob.Status != api.ComplianceJobSucceeded && complianceJob.Status != api.ComplianceJobFailed {
			s.logger.Info("aws compliance job is running", zap.String("workspaceID", workspace.ID), zap.Uint("jobID", complianceJob.ID))
			return false, nil
		}
	}

	azureSrcs, err := onboardClient.ListSources(hctx, []source.Type{source.CloudAzure})
	if err != nil {
		return false, err
	}
	if len(azureSrcs) > 0 {
		s.logger.Info("checking azure compliance job", zap.String("workspaceID", workspace.ID))
		complianceJob, err := schedulerClient.GetLatestComplianceJobForBenchmark(hctx, "azure_cis_v200")
		if err != nil {
			return false, err
		}

		if complianceJob.Status != api.ComplianceJobSucceeded && complianceJob.Status != api.ComplianceJobFailed {
			s.logger.Info("azure compliance job is running", zap.String("workspaceID", workspace.ID), zap.Uint("jobID", complianceJob.ID))
			return false, nil
		}
	}

	return true, nil
}
