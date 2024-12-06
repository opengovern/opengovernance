package audit_job

import (
	"context"
	authApi "github.com/opengovern/og-util/pkg/api"
	"github.com/opengovern/og-util/pkg/es"
	"github.com/opengovern/og-util/pkg/httpclient"
	"github.com/opengovern/og-util/pkg/steampipe"
	"github.com/opengovern/opencomply/pkg/types"
	"github.com/opengovern/opencomply/services/describe/db/model"
	"go.uber.org/zap"
	"strconv"
	"time"
)

type AuditJob struct {
	JobID          uint
	FrameworkID    string
	IntegrationIDs []string

	AuditResult *types.AuditSummary
}

type JobResult struct {
	JobID          uint
	Status         model.AuditJobStatus
	FailureMessage string
}

func (w *Worker) RunJob(ctx context.Context, job *AuditJob) error {
	job.AuditResult = &types.AuditSummary{
		Controls:     make(map[string]types.AuditControlResult),
		AuditSummary: make(map[types.ComplianceStatus]uint64),
		JobSummary: types.JobSummary{
			JobID:          job.JobID,
			JobStartedAt:   time.Now(),
			IntegrationIDs: job.IntegrationIDs,
		},
	}
	if len(job.IntegrationIDs) > 0 {
		for _, integrationID := range job.IntegrationIDs {
			err := w.RunJobForIntegration(ctx, job, integrationID)
			if err != nil {
				w.logger.Error("failed to run audit job for integration", zap.String("integration_id", integrationID), zap.Error(err))
				return err
			}
			w.logger.Info("audit job for integration completed", zap.String("integration_id", integrationID))
		}
	} else {
		err := w.RunJobForIntegration(ctx, job, "all")
		if err != nil {
			w.logger.Error("failed to run audit job for all integrations", zap.Error(err))
			return err
		}
		w.logger.Info("audit job for all integration completed")
	}

	keys, idx := job.AuditResult.KeysAndIndex()
	job.AuditResult.EsID = es.HashOf(keys...)
	job.AuditResult.EsIndex = idx

	var doc []es.Doc
	doc = append(doc, *job.AuditResult)

	w.logger.Info("Job Finished Successfully", zap.Any("result", *job.AuditResult))

	if _, err := w.sinkClient.Ingest(&httpclient.Context{Ctx: ctx, UserRole: authApi.AdminRole}, doc); err != nil {
		w.logger.Error("Failed to sink Query Run Result", zap.String("ID", strconv.Itoa(int(job.JobID))),
			zap.String("FrameworkID", job.FrameworkID), zap.Error(err))
		return err
	}

	return nil
}

func (w *Worker) RunJobForIntegration(ctx context.Context, job *AuditJob, integrationId string) error {
	ctx2 := &httpclient.Context{Ctx: ctx, UserRole: authApi.AdminRole}

	err := w.Initialize(ctx, integrationId)
	if err != nil {
		return err
	}

	defer w.steampipeConn.UnsetConfigTableValue(ctx, steampipe.OpenGovernanceConfigKeyIntegrationID)
	defer w.steampipeConn.UnsetConfigTableValue(ctx, steampipe.OpenGovernanceConfigKeyClientType)

	framework, err := w.complianceClient.GetBenchmark(&httpclient.Context{Ctx: ctx, UserRole: authApi.AdminRole}, job.FrameworkID)
	if err != nil {
		return err
	}
	controls, err := w.complianceClient.ListControl(ctx2, framework.Controls, nil)
	if err != nil {
		return err
	}

	for _, control := range controls {
		if control.Query == nil {
			continue
		}
		queryJob := QueryJob{
			AuditJobID: job.JobID,
			ExecutionPlan: ExecutionPlan{
				Query:          *control.Query,
				IntegrationIDs: job.IntegrationIDs,
			},
		}
		queryResults, err := w.RunQuery(ctx, queryJob)
		if err != nil {
			w.logger.Error("failed to run query", zap.String("jobID", strconv.Itoa(int(job.JobID))),
				zap.String("frameworkID", job.FrameworkID), zap.String("integrationID", integrationId),
				zap.String("controlID", control.ID), zap.Error(err))
			continue
		}
		for _, qr := range queryResults {
			if _, ok := job.AuditResult.Controls[control.ID]; !ok {
				job.AuditResult.Controls[control.ID] = types.AuditControlResult{
					Severity:       control.Severity,
					ControlSummary: make(map[types.ComplianceStatus]uint64),
					Results:        make(map[types.ComplianceStatus][]types.AuditResourceFinding),
				}
			}
			if _, ok := job.AuditResult.AuditSummary[qr.ComplianceStatus]; !ok {
				job.AuditResult.AuditSummary[qr.ComplianceStatus] = 0
			}
			job.AuditResult.AuditSummary[qr.ComplianceStatus] += 1

			if _, ok := job.AuditResult.Controls[control.ID].ControlSummary[qr.ComplianceStatus]; !ok {
				job.AuditResult.Controls[control.ID].ControlSummary[qr.ComplianceStatus] = 0
			}
			if _, ok := job.AuditResult.Controls[control.ID].Results[qr.ComplianceStatus]; !ok {
				job.AuditResult.Controls[control.ID].Results[qr.ComplianceStatus] = make([]types.AuditResourceFinding, 0)
			}
			job.AuditResult.Controls[control.ID].ControlSummary[qr.ComplianceStatus] += 1
			job.AuditResult.Controls[control.ID].Results[qr.ComplianceStatus] = append(job.AuditResult.Controls[control.ID].Results[qr.ComplianceStatus],
				types.AuditResourceFinding{
					ResourceID:   qr.ResourceID,
					ResourceType: qr.ResourceType,
					Reason:       qr.Reason,
				})
		}
	}
	return nil
}

func (w *Worker) Initialize(ctx context.Context, integrationId string) error {
	err := w.steampipeConn.SetConfigTableValue(ctx, steampipe.OpenGovernanceConfigKeyIntegrationID, integrationId)
	if err != nil {
		w.logger.Error("failed to set account id", zap.Error(err))
		return err
	}
	err = w.steampipeConn.SetConfigTableValue(ctx, steampipe.OpenGovernanceConfigKeyClientType, "compliance")
	if err != nil {
		w.logger.Error("failed to set client type", zap.Error(err))
		return err
	}

	return nil
}
