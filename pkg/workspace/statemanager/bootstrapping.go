package statemanager

import (
	authapi "github.com/kaytu-io/kaytu-engine/pkg/auth/api"
	"github.com/kaytu-io/kaytu-engine/pkg/compliance/client"
	"github.com/kaytu-io/kaytu-engine/pkg/describe/api"
	client2 "github.com/kaytu-io/kaytu-engine/pkg/describe/client"
	"github.com/kaytu-io/kaytu-engine/pkg/internal/httpclient"
	client3 "github.com/kaytu-io/kaytu-engine/pkg/onboard/client"
	"github.com/kaytu-io/kaytu-engine/pkg/workspace/db"
	"github.com/kaytu-io/kaytu-util/pkg/source"
	"strings"
)

func (s *Service) runBootstrapping(workspace *db.Workspace) error {
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

		status, err := schedulerClient.GetDescribeAllJobsStatus(&httpclient.Context{UserRole: authapi.InternalRole})
		if err != nil {
			return err
		}

		// waiting for all connections to finish
		if status == nil || *status != api.DescribeAllJobsStatusResourcesPublished {
			return nil
		}

		// run analytics if not running
		if !workspace.AnalyticsTriggered {
			err = schedulerClient.TriggerAnalyticsJob(&httpclient.Context{UserRole: authapi.InternalRole})
			if err != nil {
				return err
			}
			err = s.db.SetWorkspaceAnalyticsTriggered(workspace.ID)
			if err != nil {
				return err
			}
		}

		complianceURL := strings.ReplaceAll(s.cfg.Compliance.BaseURL, "%NAMESPACE%", workspace.ID)
		complianceClient := client.NewComplianceClient(complianceURL)

		// run insight if not running
		if !workspace.InsightTriggered {
			ins, err := complianceClient.ListInsights(&httpclient.Context{UserRole: authapi.InternalRole})
			if err != nil {
				return err
			}

			for _, insight := range ins {
				err = schedulerClient.TriggerInsightJob(&httpclient.Context{UserRole: authapi.InternalRole}, insight.ID)
				if err != nil {
					return err
				}
			}

			err = s.db.SetWorkspaceInsightsTriggered(workspace.ID)
			if err != nil {
				return err
			}
		}

		onboardURL := strings.ReplaceAll(s.cfg.Onboard.BaseURL, "%NAMESPACE%", workspace.ID)
		onboardClient := client3.NewOnboardServiceClient(onboardURL, nil)

		// assign compliance for aws cis v2, azure cis v2 (jobs will be triggeredl
		if !workspace.ComplianceTriggered {
			srcs, err := onboardClient.ListSources(&httpclient.Context{UserRole: authapi.InternalRole}, []source.Type{source.CloudAWS})
			if err != nil {
				return err
			}

			for _, src := range srcs {
				_, err = complianceClient.CreateBenchmarkAssignment(&httpclient.Context{UserRole: authapi.InternalRole}, "aws_cis_v200", src.ConnectionID)
				if err != nil {
					return err
				}
			}

			srcs, err = onboardClient.ListSources(&httpclient.Context{UserRole: authapi.InternalRole}, []source.Type{source.CloudAzure})
			if err != nil {
				return err
			}

			for _, src := range srcs {
				_, err = complianceClient.CreateBenchmarkAssignment(&httpclient.Context{UserRole: authapi.InternalRole}, "azure_cis_v200", src.ConnectionID)
				if err != nil {
					return err
				}
			}

			err = s.db.SetWorkspaceComplianceTriggered(workspace.ID)
			if err != nil {
				return err
			}
		}

		//TODO when jobs finished -> change to provisioned
	}
	return nil
}
